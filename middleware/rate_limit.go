// middleware/rate_limit.go
package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
	"to-do/utils"
)

// client - Menyimpan data token bucket untuk tiap IP
type client struct {
	tokens     float64
	lastActive time.Time
}

// RateLimiter - B.19 — Middleware untuk membatasi jumlah request per IP
func RateLimiter(rate float64, burst float64) func(http.Handler) http.Handler {
	// A.60 — sync.Mutex untuk keamanan akses konkuren pada map
	var mu sync.Mutex
	clients := make(map[string]*client)

	// A.30 — Goroutine pembersih map berkala untuk mencegah kebocoran memori (memory leak)
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			mu.Lock()
			for ip, c := range clients {
				// Hapus client yang tidak aktif lebih dari 3 menit
				if time.Since(c.lastActive) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// A.44 — Fungsi String: ekstrak IP dari RemoteAddr
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}

			mu.Lock()
			c, exists := clients[ip]
			if !exists {
				c = &client{
					tokens:     burst,
					lastActive: time.Now(),
				}
				clients[ip] = c
			}

			// Tambah token berdasarkan waktu berlalu (refill rate)
			now := time.Now()
			elapsed := now.Sub(c.lastActive).Seconds()
			c.lastActive = now

			c.tokens += elapsed * rate
			if c.tokens > burst {
				c.tokens = burst
			}

			// Jika token kurang dari 1, tolak request
			if c.tokens < 1.0 {
				mu.Unlock()
				utils.Fail(w, http.StatusTooManyRequests, "Terlalu banyak permintaan. Silakan coba beberapa saat lagi.")
				return
			}

			// Ambil 1 token untuk request ini
			c.tokens -= 1.0
			mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}
