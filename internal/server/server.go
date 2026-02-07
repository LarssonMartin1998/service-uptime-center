// Package server
package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	service "service-uptime-center/internal/service"
	mw "service-uptime-center/middleware"
)

func ServeAndAwaitTermination(port uint16) {
	server := http.Server{Addr: fmt.Sprintf(":%d", port)}
	go func() {
		slog.Info("Starting HTTP server", "port", port)
		server.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	server.Close()
}

func SetupEndpoints(authToken string, mapper service.Mapper) {
	if mapper == nil {
		panic("mapper cannot be passed as nil")
	}

	globalMiddleware := []mw.Middleware{
		mw.MiddlewareLogger,
		mw.CreateAuthMiddleware(authToken),
	}

	const base = "/api/v1"
	endpoints := []struct {
		pattern    string
		middleware []mw.Middleware
		handler    func(http.ResponseWriter, *http.Request)
	}{
		{
			"/health",
			[]mw.Middleware{
				mw.MiddlewareMethodGet,
			},
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, "OK")
			},
		},
		{
			"/status",
			[]mw.Middleware{
				mw.MiddlewareMethodGet,
			},
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotImplemented)
				fmt.Fprint(w, "Missing implementation")
			},
		},
		{
			"/pulse",
			[]mw.Middleware{
				mw.MiddlewareMethodPost,
				mw.MiddlewareContentTypeJSON,
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

				if _, ok := mapper[body.ServiceName]; !ok {
					slog.Warn("ServiceName doesn't exist in Mapper", "endpoint", "/pulse", "body", r.Body)
					http.Error(w, "Invalid Service Name", http.StatusBadRequest)
					return
				}

				mapper[body.ServiceName].LastPulse = time.Now()
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, "Service '%s' pulsed successfully", body.ServiceName)
				slog.Info("Pulse request successfully executed.", "service", body.ServiceName)
			},
		},
	}

	for _, endpoint := range endpoints {
		http.HandleFunc(base+endpoint.pattern, func(w http.ResponseWriter, r *http.Request) {
			if !mw.ApplyMiddleware(w, r, globalMiddleware) {
				return
			}

			if !mw.ApplyMiddleware(w, r, endpoint.middleware) {
				return
			}

			endpoint.handler(w, r)
		})
	}
}
