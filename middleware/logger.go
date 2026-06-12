// Package middleware provides HTTP middlewares for authentication, rate limiting, logging, timeout, and CORS.
package middleware

import (

	"net/http"
	"time"

	"to-do/utils"

)

// Logger - B.19 — Middleware: fungsi yang membungkus handler
// A.21 — Closure: mengembalikan fungsi
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now() // A.40 — time.Now()

		// Bungkus ResponseWriter untuk menangkap status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start) // A.42 — Time Duration

		// Logging each request using utils
		utils.Info("HTTP Request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration", duration,
			"remote_addr", r.RemoteAddr,
		)
	})
}

// responseWriter - A.24 — Struct: custom ResponseWriter untuk capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader - A.25 — Method: override WriteHeader
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
