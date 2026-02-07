package middleware

import (
	"fmt"
	"log/slog"
	"mime"
	"net/http"
)

func MiddlewareContentTypeCheck(w http.ResponseWriter, r *http.Request, expectedType string) bool {
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

func middlewareMethodCheck(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		slog.Warn("Middleware BLOCKED request - Invalid Method!", "expected", method, "got", r.Method)
		w.Header().Set("Allow", method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}
