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
	ErrCodeInvalidCliArgument = 1
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
		w.Header().Set("Allow", method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

var (
	MiddlewareAuth = func(w http.ResponseWriter, r *http.Request) bool {
		writeResponse := func() {
			w.Header().Set("WWW-Authenticate", "Bearer")
			w.WriteHeader(http.StatusUnauthorized)
		}

		if len(context.authToken) == 0 {
			return true
		}

		authHeader := r.Header.Get("Authorization")
		rawToken := strings.TrimPrefix(authHeader, "Bearer ")
		if len(authHeader) == len(rawToken) {
			writeResponse()
			return false
		}

		trimmedToken := strings.TrimSpace(rawToken)
		if trimmedToken == context.authToken {
			return true
		}

		writeResponse()
		return false
	}
	MiddlewareMethodPost = func(w http.ResponseWriter, r *http.Request) bool {
		return middlewareMethodCheck(w, r, http.MethodPost)
	}
	MiddlewareMethodGet = func(w http.ResponseWriter, r *http.Request) bool {
		return middlewareMethodCheck(w, r, http.MethodGet)
	}
)

type cliArgs struct {
	configPath string
	port       uint16
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
	args.configPath = *flag.String("config-path", "config.toml", "description")
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

func setupEndpoints() {
	globalMiddleware := []Middleware{
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
	_, err := createConfigFromPath(args.configPath)
	if err != nil {
		slog.Error("failed to parse toml config", "error", err)
		return
	}

	setupEndpoints()

	slog.Info("Starting HTTP server", "port", args.port)
	http.ListenAndServe(fmt.Sprintf(":%d", args.port), nil)
}
