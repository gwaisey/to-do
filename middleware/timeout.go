// middleware/timeout.go
// Package middleware provides HTTP middleware utilities such as timeout handling.
package middleware

import (
	"context"
	"net/http"
	"time"
)

// Timeout creates a middleware that sets a request‑level timeout.
// If the handler does not finish before the deadline, the client receives
// HTTP 504 Gateway Timeout. The underlying handler can still continue, but
// the response will be discarded.
func Timeout(next http.Handler, d time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), d)
		defer cancel()
		// Replace request with the new context containing the deadline.
		r = r.WithContext(ctx)
		// Run the next handler.
		done := make(chan struct{})
		go func() {
			next.ServeHTTP(w, r)
			close(done)
		}()
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				http.Error(w, "Request timeout", http.StatusGatewayTimeout)
				// Optionally, you could log the timeout here.
			}
		case <-done:
			// Completed within timeout.
		}
	})
}
