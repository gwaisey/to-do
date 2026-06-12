// Package middleware provides HTTP middlewares for authentication, rate limiting, logging, timeout, and CORS.
package middleware

import (
    "log/slog"
    "net/http"
    "to-do/utils"
)

// Recover is a middleware that catches panics from downstream handlers,
// logs the panic, and returns a 500 Internal Server Error response.
func Recover(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if rec := recover(); rec != nil {
                // Log the panic with stack trace if available
                slog.Error("panic recovered", slog.Any("panic", rec), slog.String("path", r.URL.Path))
                // Respond with a generic internal server error to avoid leaking details
                utils.InternalServerError(w, "Internal server error")
            }
        }()
        next.ServeHTTP(w, r)
    })
}
