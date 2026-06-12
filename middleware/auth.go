// Package middleware provides HTTP middlewares for authentication, rate limiting, logging, timeout, and CORS.
package middleware

import (
	"context"
	"net/http"
	"strings"
	"to-do/config"
	"to-do/utils"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey - A.10 — Custom type untuk context key (hindari collision)
type contextKey string

// UserIDKey is the context key for storing the authenticated user's ID.
const UserIDKey contextKey = "userID"

// AuthJWT - C.32 — JWT: validasi token
// B.19 — Middleware pattern
func AuthJWT(cfg *config.Config) func(http.Handler) http.Handler {
	// A.21 — Closure: menangkap cfg dari scope luar
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Ambil token dari header Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				utils.Fail(w, http.StatusUnauthorized, "Token tidak ditemukan")
				return
			}

			// A.44 — Fungsi String: split "Bearer <token>"
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				utils.Fail(w, http.StatusUnauthorized, "Format token tidak valid")
				return
			}

			tokenStr := parts[1]

			// C.32 — JWT: parse dan validasi token
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(cfg.JWTSecret), nil
			})

			// A.37 — Error handling
			if err != nil || !token.Valid {
				utils.Fail(w, http.StatusUnauthorized, "Token tidak valid atau sudah expired")
				return
			}

			// A.28 — interface{}: extract claims
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				utils.Fail(w, http.StatusUnauthorized, "Claims token tidak valid")
				return
			}

			userID, ok := claims["user_id"].(string)
			if !ok {
				utils.Fail(w, http.StatusUnauthorized, "User ID tidak ditemukan di token")
				return
			}

			// Simpan userID ke context — diteruskan ke handler
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
