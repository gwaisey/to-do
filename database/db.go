// Package database provides SQLite connection, migration, and helper methods.
package database

import (
	"database/sql"
	"fmt"
	"log"
	"to-do/config"

	_ "modernc.org/sqlite" // A.56 — driver SQL di-import sebagai side-effect
)

// DB - A.24 — Struct: representasi koneksi database
type DB struct {
	Conn *sql.DB
}

// New - A.25 — Method: fungsi yang melekat pada struct
func New(cfg *config.Config) *DB {
	// A.56 — SQL: membuka koneksi database
	conn, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		// A.37 — Panic: error fatal yang menghentikan program
		log.Fatalf("❌ Gagal membuka database: %v", err)
	}

	// A.37 — Recover: pastikan koneksi valid
	if err = conn.Ping(); err != nil {
		log.Fatalf("❌ Database tidak bisa di-ping: %v", err)
	}

	// C.11 — SQLite Optimizations for Concurrency
	// 1. Set busy_timeout to 5000ms to handle concurrent locks gracefully
	if _, err := conn.Exec("PRAGMA busy_timeout = 5000;"); err != nil {
		log.Printf("⚠️  Gagal menyetel busy_timeout: %v", err)
	}

	// 2. Enable WAL (Write-Ahead Logging) mode to allow concurrent reads and writes
	if _, err := conn.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		log.Printf("⚠️  Gagal menyetel journal_mode WAL: %v", err)
	}

	// 3. Limit connection pool size for SQLite
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)

	db := &DB{Conn: conn}
	db.migrate()

	fmt.Println("✅ Database SQLite terhubung (WAL mode):", cfg.DBPath)
	return db
}

// migrate - A.25 — Method pada struct DB
func (db *DB) migrate() {
	// A.56 — SQL: membuat tabel jika belum ada
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id          TEXT PRIMARY KEY,
			username    TEXT UNIQUE NOT NULL,
			email       TEXT UNIQUE NOT NULL,
			password    TEXT NOT NULL,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS todos (
			id          TEXT PRIMARY KEY,
			user_id     TEXT NOT NULL,
			title       TEXT NOT NULL,
			description TEXT,
			priority    TEXT DEFAULT 'medium',
			status      TEXT DEFAULT 'pending',
			due_date    DATETIME,
			tags        TEXT DEFAULT '[]',
			sub_tasks   TEXT DEFAULT '[]',
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		// Index untuk performa query (A.56)
		`CREATE INDEX IF NOT EXISTS idx_todos_user_id ON todos(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_todos_status ON todos(status)`,
		`CREATE INDEX IF NOT EXISTS idx_todos_sort_order ON todos(sort_order)`,
	}

	// A.14 — Perulangan: for range
	for _, q := range queries {
		if _, err := db.Conn.Exec(q); err != nil {
			log.Fatalf("❌ Migrasi gagal: %v\nQuery: %s", err, q)
		}
	}

	// Migrasi tambahan: tambahkan kolom sub_tasks dan sort_order ke database yang sudah ada jika belum terdaftar
	var subTasksExists bool
	var sortOrderExists bool
	rows, err := db.Conn.Query("PRAGMA table_info(todos)")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cid int
			var name, ctype string
			var notnull int
			var dfltValue interface{}
			var pk int
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err == nil {
				if name == "sub_tasks" {
					subTasksExists = true
				}
				if name == "sort_order" {
					sortOrderExists = true
				}
			}
		}
	}
	if !subTasksExists {
		_, err = db.Conn.Exec("ALTER TABLE todos ADD COLUMN sub_tasks TEXT DEFAULT '[]'")
		if err != nil {
			log.Printf("⚠️ Gagal menambahkan kolom sub_tasks ke tabel todos: %v", err)
		} else {
			fmt.Println("🚀 Berhasil menambahkan kolom sub_tasks ke tabel todos")
		}
	}
	if !sortOrderExists {
		_, err = db.Conn.Exec("ALTER TABLE todos ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0")
		if err != nil {
			log.Printf("⚠️ Gagal menambahkan kolom sort_order ke tabel todos: %v", err)
		} else {
			fmt.Println("🚀 Berhasil menambahkan kolom sort_order ke tabel todos")
			db.Conn.Exec("CREATE INDEX IF NOT EXISTS idx_todos_sort_order ON todos(sort_order)")
		}
	}

	fmt.Println("✅ Migrasi database selesai")
}
