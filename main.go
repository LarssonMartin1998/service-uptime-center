package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
)

var (
	errCodeInvalidCliArgument        = 1
	errCodeInvalidConfig             = 2
	errCodeFailedReadingPasswordFile = 3
)

var (
	errNoServiceGroups    = errors.New("config doesn't contain any service groups, this is not allowed")
	errInvalidGroupName   = errors.New("invalid service group name length")
	errAuthTokenTooLong   = errors.New("auth token too long")
	errHeartbeatTooShort  = errors.New("heartbeat frequency too short")
	errNoServices         = errors.New("service group has no services")
	errInvalidServiceName = errors.New("invalid service name length")
)

type middleware func(http.ResponseWriter, *http.Request) bool

func middlewareMethodCheck(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		slog.Warn("Middleware BLOCKED request - Invalid Method!", "expected", method, "got", r.Method)
		w.Header().Set("Allow", method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

func middlewareContentTypeCheck(w http.ResponseWriter, r *http.Request, expectedType string) bool {
	contentTypeHeader := r.Header.Get("Content-Type")
	writeError := func(msg string, code int) {
		slog.Warn(fmt.Sprintf("Middleware BLOCKED request - %s", msg), "expected", expectedType, "got", contentTypeHeader)
		http.Error(w, "Invalid Content-Type", code)
	}

	if len(contentTypeHeader) == 0 {
		writeError("Missing Content-Type Header!", http.StatusBadRequest)
		return false
	}

	mediaType, _, err := mime.ParseMediaType(contentTypeHeader)
	if err != nil {
		writeError("Invalid Content-Type Format", http.StatusBadRequest)
		return false
	}

	if mediaType != expectedType {
		writeError("Unexpected Content-Type", http.StatusUnsupportedMediaType)
		return false
	}

	return true
}

func createAuthMiddleware(authToken string) middleware {
	return func(w http.ResponseWriter, r *http.Request) bool {
		writeResponse := func(msg string, code int) {
			slog.Warn(fmt.Sprintf("Middleware BLOCKED request - %s", msg))
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "Unauthorized", code)
		}

		if len(authToken) == 0 {
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
		if trimmedToken != authToken {
			writeResponse("Invalid Auth Token!", http.StatusUnauthorized)
			return false
		}

		return true
	}
}

var (
	middlewareMethodPost = func(w http.ResponseWriter, r *http.Request) bool {
		return middlewareMethodCheck(w, r, http.MethodPost)
	}
	middlewareMethodGet = func(w http.ResponseWriter, r *http.Request) bool {
		return middlewareMethodCheck(w, r, http.MethodGet)
	}
	middlewareLogger = func(w http.ResponseWriter, r *http.Request) bool {
		slog.Info("HTTP Request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent())
		return true
	}
	middlewareContentTypeJSON = func(w http.ResponseWriter, r *http.Request) bool {
		return middlewareContentTypeCheck(w, r, "application/json")
	}
)

type cliArgs struct {
	configPath    *string
	port          uint16
	pwFilePath    *string
	sleepDuration time.Duration
}

type serviceCfg struct {
	Name string `toml:"name"`
}

type serviceGroupCfg struct {
	Name             string        `toml:"name"`
	Services         []serviceCfg  `toml:"services"`
	AuthToken        string        `toml:"auth_token"`
	MaxHeartbeatFreq time.Duration `toml:"max_heartbeat_freq"`
}

type config struct {
	ServiceGroups []serviceGroupCfg `toml:"service_groups"`
}

type pulseRequestBody struct {
	ServiceName string `json:"service_name"`
}

type service struct {
	lastPulse time.Time
}

type serviceContext struct {
	service    *service
	serviceCfg *serviceCfg
}

type contextProvider struct {
	cfg           config
	serviceCtx    map[string]serviceContext
	sleepDuration time.Duration
}

func handleCliArgs() cliArgs {
	var args cliArgs
	args.configPath = flag.String("config-path", "config.toml", "Path to the configuration file, defaults to './config.toml'")
	args.pwFilePath = flag.String("pw-file", "", "Path to the password file, if run without a password file, auth token middleware will be disabled.")
	sleepDurationStr := flag.String("sleep-duration", "3h", "How frequently to check for missing pulses from services.")

	portFlag := *flag.Uint64("port", 8080, "The port that the HTTP server will listen on")
	flag.Parse()

	if portFlag > math.MaxUint16 {
		slog.Error("--port flag is too high.", "max value", math.MaxUint16, "got", portFlag)
		os.Exit(errCodeInvalidCliArgument)
	}

	if duration, err := time.ParseDuration(*sleepDurationStr); err != nil {
		slog.Error("--sleep-duration has invalid format (supported format: ns, us, ms, s, m, h)")
		os.Exit(errCodeInvalidCliArgument)
	} else {
		args.sleepDuration = duration
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
		return errNoServiceGroups
	}

	for _, grp := range cfg.ServiceGroups {
		const MinNameLen = 2
		const MaxNameLen = 64
		if len(grp.Name) < MinNameLen || len(grp.Name) > MaxNameLen {
			return fmt.Errorf("%w (min: %d, max: %d): %s", errInvalidGroupName, MinNameLen, MaxNameLen, grp.Name)
		}

		const MaxAuthLen = 255
		if len(grp.AuthToken) > MaxAuthLen {
			return fmt.Errorf("%w (max: %d): %s", errAuthTokenTooLong, MaxAuthLen, grp.AuthToken)
		}

		const MinHeartbeatFreq = time.Second * 60
		if grp.MaxHeartbeatFreq < MinHeartbeatFreq {
			return fmt.Errorf("%w (min: %v): %v", errHeartbeatTooShort, MinHeartbeatFreq, grp.MaxHeartbeatFreq)
		}

		if len(grp.Services) == 0 {
			return fmt.Errorf("%w: %s", errNoServices, grp.Name)
		}

		for _, service := range grp.Services {
			if len(service.Name) < MinNameLen || len(service.Name) > MaxNameLen {
				return fmt.Errorf("%w (min: %d, max: %d): %s", errInvalidServiceName, MinNameLen, MaxNameLen, service.Name)
			}
		}
	}

	return nil
}

func applyMiddleware(w http.ResponseWriter, r *http.Request, middlewares []middleware) bool {
	for _, middleware := range middlewares {
		if !middleware(w, r) {
			return false
		}
	}

	return true
}

func parsePasswordFile(path string) (string, error) {
	if len(path) == 0 {
		slog.Warn("Running without a password file, this is supported but might not be what you intended to do, see --help for more info")
		return "", nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func setupEndpoints(authToken string, ctx *contextProvider) {
	if ctx == nil {
		panic("contextProvider cannot be passed as nil")
	}

	globalMiddleware := []middleware{
		middlewareLogger,
		createAuthMiddleware(authToken),
	}

	const base = "/api/v1"
	endpoints := []struct {
		pattern    string
		middleware []middleware
		handler    func(http.ResponseWriter, *http.Request)
	}{
		{
			"/health",
			[]middleware{
				middlewareMethodGet,
			},
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, "OK")
			},
		},
		{
			"/status",
			[]middleware{
				middlewareMethodGet,
			},
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotImplemented)
				fmt.Fprint(w, "Missing implementation")
			},
		},
		{
			"/pulse",
			[]middleware{
				middlewareMethodPost,
				middlewareContentTypeJSON,
			},
			func(w http.ResponseWriter, r *http.Request) {
				var body pulseRequestBody
				decoder := json.NewDecoder(r.Body)
				decoder.DisallowUnknownFields()
				if err := decoder.Decode(&body); err != nil {
					slog.Warn("Failed to decode json from request body", "endpoint", "/pulse", "body", r.Body, "error", err)
					http.Error(w, "Invalid JSON in Request", http.StatusBadRequest)
					return
				}

				if _, ok := ctx.serviceCtx[body.ServiceName]; !ok {
					slog.Warn("ServiceName doesn't exist in serviceMapper", "endpoint", "/pulse", "body", r.Body)
					http.Error(w, "Invalid Service Name", http.StatusBadRequest)
					return
				}

				ctx.serviceCtx[body.ServiceName].service.lastPulse = time.Now()
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, "Service '%s' pulsed successfully", body.ServiceName)
				slog.Info("Pulse request successfully executed.", "service", body.ServiceName)
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

func createServiceContext(ctx *contextProvider) (map[string]serviceContext, error) {
	if ctx == nil {
		panic("contextProvider cannot be passed as nil")
	}

	now := time.Now()
	serviceCtx := make(map[string]serviceContext)
	for _, group := range ctx.cfg.ServiceGroups {
		for _, serviceCfg := range group.Services {
			_, ok := serviceCtx[serviceCfg.Name]
			if ok {
				return nil, fmt.Errorf("service with this name already exist '%s', all service names must be unique", serviceCfg.Name)
			}

			serviceCtx[serviceCfg.Name] = serviceContext{
				service: &service{
					lastPulse: now,
				},
				serviceCfg: &serviceCfg,
			}
		}
	}

	return serviceCtx, nil
}

func main() {
	args := handleCliArgs()

	var err error
	contextProvider := contextProvider{}
	if contextProvider.cfg, err = createConfigFromPath(*args.configPath); err != nil {
		slog.Error("failed to parse toml config", "error", err)
		os.Exit(errCodeInvalidConfig)
	}

	contextProvider.serviceCtx, err = createServiceContext(&contextProvider)
	if err != nil {
		slog.Error("failed to create service mapper from config", "error", err)
		os.Exit(errCodeInvalidConfig)
	}

	if pw, err := parsePasswordFile(*args.pwFilePath); err != nil {
		slog.Error("failed to read password file", "path", *args.pwFilePath, "error", err)
		os.Exit(errCodeFailedReadingPasswordFile)
	} else {
		setupEndpoints(pw, &contextProvider)
	}

	server := http.Server{Addr: fmt.Sprintf(":%d", args.port)}
	go func() {
		slog.Info("Starting HTTP server", "port", args.port)
		server.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	server.Close()
}
