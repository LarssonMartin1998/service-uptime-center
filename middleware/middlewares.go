// Package middleware contains general broad functionality that attaches to a http request to perform validation or functionality like logging.
package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

type Middleware func(http.ResponseWriter, *http.Request) bool

var (
	MiddlewareMethodPost = func(w http.ResponseWriter, r *http.Request) bool {
		return middlewareMethodCheck(w, r, http.MethodPost)
	}
	MiddlewareMethodGet = func(w http.ResponseWriter, r *http.Request) bool {
		return middlewareMethodCheck(w, r, http.MethodGet)
	}
	MiddlewareLogger = func(_ http.ResponseWriter, r *http.Request) bool {
		slog.Info("HTTP Request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent())
		return true
	}
	MiddlewareContentTypeJSON = func(w http.ResponseWriter, r *http.Request) bool {
		return MiddlewareContentTypeCheck(w, r, "application/json")
	}
)

func ApplyMiddleware(w http.ResponseWriter, r *http.Request, middlewares []Middleware) bool {
	for _, middleware := range middlewares {
		if !middleware(w, r) {
			return false
		}
	}

	return true
}

func CreateAuthMiddleware(authToken string) Middleware {
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
