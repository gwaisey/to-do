// Package config provides application configuration loading and defaults.
package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// DefaultPort - A.11 — Konstanta: nilai yang tidak berubah
const DefaultPort = "8080"

// Config - A.24 — Struct: kumpulan field bertipe data
type Config struct {
	AppEnv         string
	AppPort        string
	DBPath         string
	JWTSecret      string
	JWTExpiryHours int
}

// Load - A.18 — Fungsi: mengembalikan nilai
// A.37 — Error handling
func Load() *Config {
	// C.11 — Best Practice: load dari environment variable
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  File .env tidak ditemukan, menggunakan environment variable sistem")
	}

	// A.9 — Variabel dengan tipe eksplisit
	appEnv := getEnv("APP_ENV", "development")
	port := getEnv("APP_PORT", DefaultPort)
	dbPath := getEnv("DB_PATH", "./todo.db")
	jwtSecret := getEnv("JWT_SECRET", "default-secret")

	// A.43 — Konversi string ke int
	expiryStr := getEnv("JWT_EXPIRY_HOURS", "24")
	expiry, err := strconv.Atoi(expiryStr)
	if err != nil {
		expiry = 24
	}

	return &Config{
		AppEnv:         appEnv,
		AppPort:        port,
		DBPath:         dbPath,
		JWTSecret:      jwtSecret,
		JWTExpiryHours: expiry,
	}
}

// A.21 — Closure: fungsi di dalam fungsi
func getEnv(key, fallback string) string {
	// A.13 — Seleksi kondisi: if-else
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
