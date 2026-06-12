// middleware/cors_test.go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCORS(t *testing.T) {
	// Simple handler that returns OK
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	// Wrap with CORS middleware
	wrapped := NewCORS(handler)

	// Test GET request
	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	// Check CORS headers are set
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got == "" {
		t.Fatalf("Access-Control-Allow-Origin header missing")
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatalf("Access-Control-Allow-Methods header missing")
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatalf("Access-Control-Allow-Headers header missing")
	}

	// Test OPTIONS preflight request
	optsReq := httptest.NewRequest(http.MethodOptions, "http://example.com", nil)
	optsRec := httptest.NewRecorder()
	wrapped.ServeHTTP(optsRec, optsReq)
	if optsRec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d for OPTIONS, got %d", http.StatusNoContent, optsRec.Code)
	}
	// Ensure headers present in OPTIONS response as well
	if got := optsRec.Header().Get("Access-Control-Allow-Origin"); got == "" {
		t.Fatalf("Access-Control-Allow-Origin header missing in OPTIONS response")
	}
}
