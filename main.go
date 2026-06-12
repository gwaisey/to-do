// main.go
package main

import (
	"context"
	"flag" // A.48 — Arguments & Flag
	"fmt"
	"to-do/utils"
	"net/http"
	"os"
	"os/signal"
	"sync" // A.59 — sync.WaitGroup, A.60 — sync.Mutex
	"syscall"
	"time"
	"to-do/config"
	"to-do/database"
	"to-do/handlers"
	"to-do/middleware"
)

// A.60 — sync.Mutex: untuk request counter thread-safe
var (
	mu           sync.Mutex
	requestCount int
)

func main() {
	// Inisialisasi structured logger (slog) JSON format ke stdout
	utils.InitLogger()

	// A.8 — Komentar: dokumentasi kode
	// A.48 — Arguments & Flag: flag CLI
	port := flag.String("port", "", "Port server (override .env)")
	flag.Parse()

	// Load konfigurasi
	cfg := config.Load()

	// Validasi keamanan JWT_SECRET di environment production
	if cfg.AppEnv == "production" && (cfg.JWTSecret == "" || cfg.JWTSecret == "default-secret") {
		utils.Error("Keamanan Kritis: JWT_SECRET wajib disetel di environment production dan tidak boleh bernilai default!")
		os.Exit(1)
	}

	// A.13 — Seleksi kondisi: override port dari flag jika ada
	if *port != "" {
		cfg.AppPort = *port
	}

	// Inisialisasi database
	db := database.New(cfg)

	// A.36 — Defer: tutup DB saat program selesai
	defer db.Conn.Close()

	// Inisialisasi handlers
	authHandler := handlers.NewAuthHandler(db, cfg)
	todoHandler := handlers.NewTodoHandler(db)

	// B.20 — Custom Multiplexer: pakai ServeMux sendiri
	mux := http.NewServeMux()

	// B.2 — Routing
	fs := http.FileServer(http.Dir("./public"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "./public/index.html")
			return
		}

		// Check if it is a file request that exists in public folder
		filePath := "./public" + r.URL.Path
		if _, err := os.Stat(filePath); err == nil {
			fs.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"success":false,"error":"Rute tidak ditemukan"}`)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// A.17 — Map: response health check
		fmt.Fprintf(w, `{"status":"ok","time":"%s"}`, time.Now().Format(time.RFC3339))
	})
	mux.HandleFunc("/register", authHandler.Register)
	mux.HandleFunc("/login", authHandler.Login)

	// ── Protected routes (perlu JWT) ──
	// B.19 — Middleware: chain middleware
	authMiddleware := middleware.AuthJWT(cfg)

	// A.52 — URL Parsing: routing berdasarkan method dan path
	mux.Handle("/todos", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			todoHandler.GetAll(w, r)
		case http.MethodPost:
			todoHandler.Create(w, r)
		default:
			http.Error(w, "Method tidak diizinkan", http.StatusMethodNotAllowed)
		}
	})))

	// D.2 — Timeout Pattern Route
	mux.Handle("/todos/search", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			todoHandler.SearchWithTimeout(w, r)
		} else {
			http.Error(w, "Method tidak diizinkan", http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/todos/bulk-complete", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			todoHandler.BulkComplete(w, r)
		} else {
			http.Error(w, "Method tidak diizinkan", http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/todos/bulk-delete", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			todoHandler.BulkDelete(w, r)
		} else {
			http.Error(w, "Method tidak diizinkan", http.StatusMethodNotAllowed)
		}
	})))

	// Add reorder route
	mux.Handle("/todos/reorder", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			todoHandler.Reorder(w, r)
		} else {
			http.Error(w, "Method tidak diizinkan", http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/todos/", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A.44 — Fungsi String: cek suffix untuk routing
		path := r.URL.Path
		switch {
		case len(path) > len("/todos/") && path[len(path)-len("/complete"):] == "/complete":
			if r.Method == http.MethodPatch {
				todoHandler.MarkDone(w, r)
			} else {
				http.Error(w, "Method tidak diizinkan", http.StatusMethodNotAllowed)
			}
		default:
			switch r.Method {
			case http.MethodGet:
				todoHandler.GetOne(w, r)
			case http.MethodPut:
				todoHandler.Update(w, r)
			case http.MethodDelete:
				todoHandler.Delete(w, r)
			default:
				http.Error(w, "Method tidak diizinkan", http.StatusMethodNotAllowed)
			}
		}
	})))

	// B.19 — Middleware stack: CORS + Rate Limiter + Timeout + Logger + Request Counter
	// Batasi rata-rata 10 request/detik dengan kapasitas burst 20 request per IP
	rateLimiter := middleware.RateLimiter(10.0, 20.0)

	// C.14 — CORS + Timeout
	finalHandler := middleware.Recover(middleware.NewCORS(rateLimiter(middleware.Timeout(middleware.Logger(countRequests(mux)), 10*time.Second))))

	server := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      finalHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// A.30 — Goroutine: jalankan server di goroutine terpisah
	// A.31 — Channel: channel untuk sinyal shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// A.30 — Goroutine: server berjalan tanpa memblokir main goroutine
	go func() {
		utils.Info("Server running", "url", fmt.Sprintf("http://localhost:%s", cfg.AppPort))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			utils.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	// A.33 — Channel Select: tunggu sinyal shutdown
	<-quit
	utils.Info("Shutting down server...")

	// A.59 — sync.WaitGroup: tunggu request yang sedang berjalan selesai
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		utils.Error("Shutdown error", "error", err)
	}

	// A.60 — sync.Mutex: baca counter dengan aman
	mu.Lock()
	fmt.Printf("✅ Server selesai. Total request dilayani: %d\n", requestCount)
	mu.Unlock()
}

// CORS middleware moved to middleware.NewCORS; local implementation removed

// countRequests - A.60 — sync.Mutex: middleware untuk counting request secara thread-safe
func countRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()
		next.ServeHTTP(w, r)
	})
}
