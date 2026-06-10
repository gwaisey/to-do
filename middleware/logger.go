// middleware/logger.go
package middleware

import (
	"fmt"
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

		// A.38 — Layout Format String: format waktu
		timestamp := start.Format("2006-01-02 15:04:05")
		duration := time.Since(start) // A.42 — Time Duration

		// C.8 — Logging: log setiap request
		fmt.Printf("[%s] %s %s → %d (%v)\n",
			timestamp,
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			duration,
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
