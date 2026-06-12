// Package middleware provides HTTP middlewares for authentication, rate limiting, logging, timeout, and CORS.
package middleware

import (
    "net/http"
    "net/http/httptest"
    "testing"
    "strings"

)

// TestRecoverMiddleware ensures that a panic in a downstream handler is recovered and
// results in a 500 Internal Server Error response.
func TestRecoverMiddleware(t *testing.T) {
    // Handler that deliberately panics.
    panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        panic("boom")
    })

    // Wrap the panic handler with the Recover middleware.
    handler := Recover(panicHandler)

    // Perform a request.
    req := httptest.NewRequest(http.MethodGet, "/test", nil)
    rr := httptest.NewRecorder()

    // Serve.
    handler.ServeHTTP(rr, req)

    // Expect a 500 status code.
    if rr.Code != http.StatusInternalServerError {
        t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
    }

    // Verify the response body contains the expected error message.
    if !strings.Contains(rr.Body.String(), "Internal server error") {
        t.Fatalf("response body does not contain expected error message: %s", rr.Body.String())
    }
}
