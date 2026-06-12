// middleware/cors.go
// Package middleware provides HTTP middleware utilities such as CORS handling.
package middleware

import (
	"net/http"
	"os"
	"strings"
)

// CORSConfig holds configuration for the CORS middleware.
type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

// NewCORS returns a CORS middleware using environment variables for configuration.
// If CORS_ALLOWED_ORIGINS is not set, it defaults to "*" (allow all origins).
func NewCORS(next http.Handler) http.Handler {
	cfg := loadConfig()
	return corsHandler(cfg, next)
}

func loadConfig() CORSConfig {
	origins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if origins == "" {
		origins = "*"
	}
	methods := os.Getenv("CORS_ALLOWED_METHODS")
	if methods == "" {
		methods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	}
	headers := os.Getenv("CORS_ALLOWED_HEADERS")
	if headers == "" {
		headers = "Content-Type, Authorization"
	}
	return CORSConfig{
		AllowedOrigins: strings.Split(origins, ","),
		AllowedMethods: strings.Split(methods, ","),
		AllowedHeaders: strings.Split(headers, ","),
	}
}

func corsHandler(cfg CORSConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Join slices back to strings (trimming spaces)
		join := func(parts []string) string {
			var cleaned []string
			for _, p := range parts {
				cleaned = append(cleaned, strings.TrimSpace(p))
			}
			return strings.Join(cleaned, ", ")
		}
		w.Header().Set("Access-Control-Allow-Origin", strings.TrimSpace(strings.Join(cfg.AllowedOrigins, ", ")))
		w.Header().Set("Access-Control-Allow-Methods", join(cfg.AllowedMethods))
		w.Header().Set("Access-Control-Allow-Headers", join(cfg.AllowedHeaders))
		// Vary header helps caches differentiate based on Origin
		w.Header().Add("Vary", "Origin")

		if r.Method == http.MethodOptions {
			// Preflight request – respond with no body
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
