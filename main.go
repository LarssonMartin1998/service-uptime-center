package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

var (
	ErrCodeInvalidCliArgument        = 1
	ErrCodeFailedReadingPasswordFile = 2
)

var (
	ErrNoServiceGroups    = errors.New("config doesn't contain any service groups, this is not allowed")
	ErrInvalidGroupName   = errors.New("invalid service group name length")
	ErrAuthTokenTooLong   = errors.New("auth token too long")
	ErrHeartbeatTooShort  = errors.New("heartbeat frequency too short")
	ErrNoServices         = errors.New("service group has no services")
	ErrInvalidServiceName = errors.New("invalid service name length")
)

var context globalContext

type globalContext struct {
	authToken string
}

type Middleware func(http.ResponseWriter, *http.Request) bool

func middlewareMethodCheck(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		slog.Warn("Middleware BLOCKED request - Invalid Method!", "expected", method, "got", r.Method)
		w.Header().Set("Allow", method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

var (
	MiddlewareAuth = func(w http.ResponseWriter, r *http.Request) bool {
		writeResponse := func(msg string, code int) {
			slog.Warn(fmt.Sprintf("Middleware BLOCKED request - %s", msg))
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "Unauthorized", code)
		}

		if len(context.authToken) == 0 {
			return true
		}

		authHeader := r.Header.Get("Authorization")
		if len(authHeader) == 0 {
			writeResponse("Missing Authorization Header!", http.StatusBadRequest)
			return false
		}

		rawToken := strings.TrimPrefix(authHeader, "Bearer ")
		if len(authHeader) == len(rawToken) {
			writeResponse("Invalid Auth Token Format!", http.StatusUnauthorized)
			return false
		}

		trimmedToken := strings.TrimSpace(rawToken)
		if trimmedToken != context.authToken {
			writeResponse("Invalid Auth Token!", http.StatusUnauthorized)
			return false
		}

		return true
	}
	MiddlewareMethodPost = func(w http.ResponseWriter, r *http.Request) bool {
		return middlewareMethodCheck(w, r, http.MethodPost)
	}
	MiddlewareMethodGet = func(w http.ResponseWriter, r *http.Request) bool {
		return middlewareMethodCheck(w, r, http.MethodGet)
	}
	MiddlewareLogger = func(w http.ResponseWriter, r *http.Request) bool {
		slog.Info("HTTP Request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent())
		return true
	}
)

type cliArgs struct {
	configPath *string
	port       uint16
	pwFilePath *string
}

type service struct {
	Name string `toml:"name"`
}

type serviceGroup struct {
	Name             string        `toml:"name"`
	Services         []service     `toml:"services"`
	AuthToken        string        `toml:"auth_token"`
	MaxHeartbeatFreq time.Duration `toml:"max_heartbeat_freq"`
}

type config struct {
	ServiceGroups []serviceGroup `toml:"service_groups"`
}

func handleCliArgs() cliArgs {
	var args cliArgs
	args.configPath = flag.String("config-path", "config.toml", "Path to the configuration file, defaults to './config.toml'")
	args.pwFilePath = flag.String("pw-file", "", "Path to the password file, if run without a password file, auth token middleware will be disabled.")

	portFlag := *flag.Uint64("port", 8080, "The port that the HTTP server will listen on")
	flag.Parse()

	if portFlag > math.MaxUint16 {
		slog.Error("--port flag is too high.", "max value", math.MaxUint16, "got", portFlag)
		os.Exit(ErrCodeInvalidCliArgument)
	}
	args.port = uint16(portFlag)

	return args
}

func createConfigFromPath(configPath string) (config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return config{}, err
	}

	return createConfig(string(data))
}

func createConfig(data string) (config, error) {
	var cfg config
	_, err := toml.Decode(string(data), &cfg)
	if err != nil {
		return cfg, err
	}

	err = validateConfig(&cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, err
}

func validateConfig(cfg *config) error {
	if len(cfg.ServiceGroups) == 0 {
		return ErrNoServiceGroups
	}

	for _, grp := range cfg.ServiceGroups {
		const MinNameLen = 2
		const MaxNameLen = 64
		if len(grp.Name) < MinNameLen || len(grp.Name) > MaxNameLen {
			return fmt.Errorf("%w (min: %d, max: %d): %s", ErrInvalidGroupName, MinNameLen, MaxNameLen, grp.Name)
		}

		const MaxAuthLen = 255
		if len(grp.AuthToken) > MaxAuthLen {
			return fmt.Errorf("%w (max: %d): %s", ErrAuthTokenTooLong, MaxAuthLen, grp.AuthToken)
		}

		const MinHeartbeatFreq = time.Second * 60
		if grp.MaxHeartbeatFreq < MinHeartbeatFreq {
			return fmt.Errorf("%w (min: %v): %v", ErrHeartbeatTooShort, MinHeartbeatFreq, grp.MaxHeartbeatFreq)
		}

		if len(grp.Services) == 0 {
			return fmt.Errorf("%w: %s", ErrNoServices, grp.Name)
		}

		for _, service := range grp.Services {
			if len(service.Name) < MinNameLen || len(service.Name) > MaxNameLen {
				return fmt.Errorf("%w (min: %d, max: %d): %s", ErrInvalidServiceName, MinNameLen, MaxNameLen, service.Name)
			}
		}
	}

	return nil
}

func applyMiddleware(w http.ResponseWriter, r *http.Request, middlewares []Middleware) bool {
	for _, middleware := range middlewares {
		if !middleware(w, r) {
			return false
		}
	}

	return true
}

func parsePasswordFile(path string) string {
	if len(path) == 0 {
		slog.Warn("Running without a password file, this is supported but might not be what you intended to do, see --help for more info")
		return ""
	}

	data, err := os.ReadFile(path)
	if err != nil {
		slog.Error("failed to read password file", "path", path, "error", err)
		os.Exit(ErrCodeFailedReadingPasswordFile)
	}

	return strings.TrimSpace(string(data))
}

func setupEndpoints() {
	globalMiddleware := []Middleware{
		MiddlewareLogger,
		MiddlewareAuth,
	}

	const base = "/api/v1"
	endpoints := []struct {
		pattern    string
		middleware []Middleware
		handler    func(http.ResponseWriter, *http.Request)
	}{
		{
			"/health",
			[]Middleware{
				MiddlewareMethodGet,
			},
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, "OK")
			},
		},
		{
			"/status",
			[]Middleware{
				MiddlewareMethodGet,
			},
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotImplemented)
				fmt.Fprint(w, "Missing implementation")
			},
		},
		{
			"/pulse",
			[]Middleware{
				MiddlewareMethodPost,
			},
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotImplemented)
				fmt.Fprint(w, "Missing implementation")
			},
		},
	}

	for _, endpoint := range endpoints {
		http.HandleFunc(base+endpoint.pattern, func(w http.ResponseWriter, r *http.Request) {
			if !applyMiddleware(w, r, globalMiddleware) {
				return
			}

			if !applyMiddleware(w, r, endpoint.middleware) {
				return
			}

			endpoint.handler(w, r)
		})
	}
}

func main() {
	args := handleCliArgs()
	_, err := createConfigFromPath(*args.configPath)
	if err != nil {
		slog.Error("failed to parse toml config", "error", err)
		return
	}

	context.authToken = parsePasswordFile(*args.pwFilePath)
	setupEndpoints()

	slog.Info("Starting HTTP server", "port", args.port)
	http.ListenAndServe(fmt.Sprintf(":%d", args.port), nil)
}
