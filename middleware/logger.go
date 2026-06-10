// middleware/logger.go
package middleware

import (
	"log/slog"
	"net/http"
	"time" // A.40 — Time
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

		// C.8 — Logging: log setiap request menggunakan slog
		slog.Info("HTTP Request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", wrapped.statusCode),
			slog.Duration("duration", duration),
			slog.String("remote_addr", r.RemoteAddr),
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
