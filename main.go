// main.go
package main

import (
	"context"
	"flag"       // A.48 — Arguments & Flag
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"       // A.59 — sync.WaitGroup, A.60 — sync.Mutex
	"syscall"
	"time"
	"todo-api/config"
	"todo-api/database"
	"todo-api/handlers"
	"todo-api/middleware"
)

// A.60 — sync.Mutex: untuk request counter thread-safe
var (
	mu           sync.Mutex
	requestCount int
)

func main() {
	// A.8 — Komentar: dokumentasi kode
	// A.48 — Arguments & Flag: flag CLI
	port := flag.String("port", "", "Port server (override .env)")
	flag.Parse()

	// Load konfigurasi
	cfg := config.Load()

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

	// B.19 — Middleware stack: CORS + Logger + Request Counter
	// C.14 — CORS
	finalHandler := corsMiddleware(middleware.Logger(countRequests(mux)))

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
		fmt.Printf("\n🚀 Server berjalan di http://localhost:%s\n", cfg.AppPort)
		fmt.Println("   Tekan Ctrl+C untuk berhenti")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server error: %v", err)
		}
	}()

	// A.33 — Channel Select: tunggu sinyal shutdown
	<-quit
	fmt.Println("\n⏳ Mematikan server...")

	// A.59 — sync.WaitGroup: tunggu request yang sedang berjalan selesai
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("❌ Shutdown error: %v", err)
	}

	// A.60 — sync.Mutex: baca counter dengan aman
	mu.Lock()
	fmt.Printf("✅ Server selesai. Total request dilayani: %d\n", requestCount)
	mu.Unlock()
}

// corsMiddleware - C.14 — CORS Middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// countRequests - A.60 — sync.Mutex: middleware untuk counting request secara thread-safe
func countRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()
		next.ServeHTTP(w, r)
	})
}
