// tests/todo_test.go
package tests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing" // A.58 — Unit Test
	"time"
	"to-do/database"
	"to-do/handlers"
	"to-do/middleware"
	"to-do/models"
	"to-do/utils"
)

// TestGenerateID - A.58 — Unit Test: test fungsi GenerateID
func TestGenerateID(t *testing.T) {
	id1 := utils.GenerateID()
	id2 := utils.GenerateID()

	// A.13 — Seleksi kondisi di test
	if id1 == "" {
		t.Error("GenerateID() tidak boleh menghasilkan string kosong")
	}

	if id1 == id2 {
		t.Error("GenerateID() harus menghasilkan ID yang berbeda setiap kali")
	}

	t.Logf("ID1: %s", id1)
	t.Logf("ID2: %s", id2)
}

// TestValidateEmail - A.58 — Unit Test: test validasi email
func TestValidateEmail(t *testing.T) {
	// A.15 — Array: test cases
	testCases := []struct {
		email   string
		wantErr bool
	}{
		{"user@example.com", false},
		{"invalid-email", true},
		{"user@", true},
		{"test.user+tag@domain.co.id", false},
	}

	// A.14 — Perulangan: iterasi test cases
	for _, tc := range testCases {
		err := utils.ValidateEmail(tc.email)
		if (err != nil) != tc.wantErr {
			t.Errorf("ValidateEmail(%q): dapat error=%v, mau error=%v", tc.email, err != nil, tc.wantErr)
		}
	}
}

// TestHashPassword - A.58 — Unit Test: test hash password
func TestHashPassword(t *testing.T) {
	testPlaintext := "samplePlaintext123!"

	hash, err := utils.HashPassword(testPlaintext)
	if err != nil {
		t.Fatalf("HashPassword gagal: %v", err)
	}

	if hash == testPlaintext {
		t.Error("Hash tidak boleh sama dengan password asli")
	}

	if !utils.CheckPassword(testPlaintext, hash) {
		t.Error("CheckPassword harus return true untuk password yang benar")
	}

	if utils.CheckPassword("wrongpassword", hash) {
		t.Error("CheckPassword harus return false untuk password yang salah")
	}
}

// TestHealthEndpoint - A.58 — Unit Test: test HTTP handler dengan httptest
func TestHealthEndpoint(t *testing.T) {
	// A.21 — Closure: handler inline untuk test
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// A.53 — JSON: test response
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var resp map[string]string
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", resp["status"])
	}
}

// TestWriteJSON - A.58 — Unit Test: test response writer helper
func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()

	// A.53 — JSON: test encode
	data := map[string]string{"key": "value"}
	utils.WriteJSON(rec, http.StatusOK, data)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected application/json, got %s", contentType)
	}
}

// BenchmarkGenerateID - A.58 — Benchmark test: ukur performa GenerateID
func BenchmarkGenerateID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		utils.GenerateID()
	}
}

// TestConcurrentRequests - A.59 — sync.WaitGroup: test concurrent requests
func TestConcurrentRequests(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// A.30 — Goroutine: launch concurrent test requests
	done := make(chan bool, 10) // A.32 — Buffered Channel

	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/health", bytes.NewBuffer(nil))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			done <- rec.Code == http.StatusOK // A.31 — Channel: kirim hasil
		}()
	}

	// A.34 — Channel Range & Close: tunggu semua goroutine
	for i := 0; i < 10; i++ {
		if !<-done {
			t.Error("Salah satu concurrent request gagal")
		}
	}
}

// TestUpdateTodoRequestValidation - test input validation for updates
func TestUpdateTodoRequestValidation(t *testing.T) {
	titleVal := "Valid Title"
	emptyTitle := ""
	longTitle := strings.Repeat("a", 201)

	priorityVal := models.PriorityHigh
	invalidPriority := models.Priority("invalid")

	testCases := []struct {
		name    string
		req     models.UpdateTodoRequest
		wantErr bool
	}{
		{
			name: "All valid field pointers",
			req: models.UpdateTodoRequest{
				Title:    &titleVal,
				Priority: &priorityVal,
			},
			wantErr: false,
		},
		{
			name: "Nil pointers",
			req: models.UpdateTodoRequest{
				Title:    nil,
				Priority: nil,
			},
			wantErr: false,
		},
		{
			name: "Empty title error",
			req: models.UpdateTodoRequest{
				Title: &emptyTitle,
			},
			wantErr: true,
		},
		{
			name: "Title too long error",
			req: models.UpdateTodoRequest{
				Title: &longTitle,
			},
			wantErr: true,
		},
		{
			name: "Invalid priority error",
			req: models.UpdateTodoRequest{
				Priority: &invalidPriority,
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// TestRateLimiter - Unit test untuk middleware RateLimiter
func TestRateLimiter(t *testing.T) {
	// Buat handler dummy
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Setup RateLimiter dengan rate = 1 token/detik, burst = 2 token
	limiter := middleware.RateLimiter(1.0, 2.0)
	handler := limiter(dummyHandler)

	// Kirim 2 request pertama (harusnya OK karena burst = 2)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Request %d: diharapkan status 200, didapat %d", i+1, rec.Code)
		}
	}

	// Request ke-3 langsung setelahnya harusnya ditolak (Too Many Requests - 429)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Request 3: diharapkan status 429, didapat %d", rec.Code)
	}
}

// TestTodoPipeline - Unit test untuk pipeline konkurensi (generator, filter, dan enricher)
func TestTodoPipeline(t *testing.T) {
	pastTime := time.Now().Add(-24 * time.Hour)
	futureTime := time.Now().Add(24 * time.Hour)

	todos := []models.Todo{
		{
			ID:      "1",
			Title:   "Task 1 (Overdue Pending)",
			Status:  models.StatusPending,
			DueDate: &pastTime,
		},
		{
			ID:      "2",
			Title:   "Task 2 (Future Pending)",
			Status:  models.StatusPending,
			DueDate: &futureTime,
		},
		{
			ID:      "3",
			Title:   "Task 3 (Done)",
			Status:  models.StatusDone,
			DueDate: &pastTime, // Past time but status is Done, so not overdue
		},
	}

	// 1. Jalankan pipeline: Generator -> FilterPending -> EnrichWithOverdue
	genChan := utils.TodoGenerator(todos)
	filteredChan := utils.FilterPending(genChan)
	enrichedChan := utils.EnrichWithOverdue(filteredChan)

	var result []models.Todo
	for item := range enrichedChan {
		result = append(result, item)
	}

	// Verifikasi hasil filter (hanya status pending/in_progress yang lolos, status done disaring)
	if len(result) != 2 {
		t.Errorf("Harusnya ada 2 tugas pending, dapat %d", len(result))
	}

	// Verifikasi hasil enrichment
	for _, item := range result {
		if item.ID == "1" {
			if !item.Overdue {
				t.Error("Tugas 1 dengan due date di masa lalu harusnya terdeteksi overdue (true)")
			}
			if item.TimeRemaining == "" || item.TimeRemaining[0] != '-' {
				t.Errorf("Tugas 1 harusnya memiliki TimeRemaining negatif, dapat %s", item.TimeRemaining)
			}
		}
		if item.ID == "2" {
			if item.Overdue {
				t.Error("Tugas 2 dengan due date di masa depan tidak boleh overdue (false)")
			}
			if item.TimeRemaining == "" || item.TimeRemaining[0] == '-' {
				t.Errorf("Tugas 2 harusnya memiliki TimeRemaining positif, dapat %s", item.TimeRemaining)
			}
		}
	}
}

// TestTagFilteringHandler - Unit test untuk filter tag dan statistik tag pada handler
func TestTagFilteringHandler(t *testing.T) {
	// 1. Setup database in-memory
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Gagal membuka DB in-memory: %v", err)
	}
	defer conn.Close()

	// Jalankan migrasi tabel dengan default values mirip seperti production
	queries := []string{
		`CREATE TABLE users (
			id          TEXT PRIMARY KEY,
			username    TEXT UNIQUE NOT NULL,
			email       TEXT UNIQUE NOT NULL,
			password    TEXT NOT NULL,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE todos (
			id          TEXT PRIMARY KEY,
			user_id     TEXT NOT NULL,
			title       TEXT NOT NULL,
			description TEXT DEFAULT '',
			priority    TEXT DEFAULT 'medium',
			status      TEXT DEFAULT 'pending',
			due_date    DATETIME,
			tags        TEXT DEFAULT '[]',
			sub_tasks   TEXT DEFAULT '[]',
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, q := range queries {
		if _, err := conn.Exec(q); err != nil {
			t.Fatalf("Migrasi gagal: %v", err)
		}
	}

	// Insert mock data
	userID := "test-user-123"
	_, err = conn.Exec(`INSERT INTO users (id, username, email, password) VALUES (?, 'tester', 'tester@example.com', 'hashed')`, userID)
	if err != nil {
		t.Fatalf("Gagal insert user: %v", err)
	}

	// Insert 2 todos dengan tag yang berbeda
	_, err = conn.Exec(`INSERT INTO todos (id, user_id, title, description, tags) VALUES 
		('t1', ?, 'Task 1', 'Desc 1', '["kerja","penting"]'),
		('t2', ?, 'Task 2', 'Desc 2', '["pribadi"]')`, userID, userID)
	if err != nil {
		t.Fatalf("Gagal insert todos: %v", err)
	}

	// 2. Setup handler
	dbWrapper := &database.DB{Conn: conn}
	handler := handlers.NewTodoHandler(dbWrapper)

	// 3. Test GetAll filter by tag=kerja
	req := httptest.NewRequest(http.MethodGet, "/todos?tag=kerja", nil)
	// Inject userID ke context
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.GetAll(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Diharapkan status 200, dapat %d. Response: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			Todos []models.Todo `json:"todos"`
			Stats struct {
				Total int      `json:"total"`
				Tags  []string `json:"tags"`
			} `json:"stats"`
		} `json:"data"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Gagal unmarshal response: %v", err)
	}

	// Verifikasi todos yang dikembalikan (hanya 'Task 1' dengan tag 'kerja')
	if len(resp.Data.Todos) != 1 {
		t.Errorf("Harusnya hanya 1 todo yang lolos filter, dapat %d", len(resp.Data.Todos))
	} else if resp.Data.Todos[0].ID != "t1" {
		t.Errorf("Harusnya todo t1 yang lolos, dapat %s", resp.Data.Todos[0].ID)
	}

	// Verifikasi stats.tags memiliki semua tag unik (kerja, penting, pribadi)
	tagMap := make(map[string]bool)
	for _, tg := range resp.Data.Stats.Tags {
		tagMap[tg] = true
	}
	expectedTags := []string{"kerja", "penting", "pribadi"}
	for _, expected := range expectedTags {
		if !tagMap[expected] {
			t.Errorf("Tag %q tidak ditemukan di stats.tags", expected)
		}
	}
}

// TestTodoSortingHandler - Unit test untuk memverifikasi fitur pengurutan tugas (sorting)
func TestTodoSortingHandler(t *testing.T) {
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Gagal membuka DB in-memory: %v", err)
	}
	defer conn.Close()

	// Migrasi tabel
	queries := []string{
		`CREATE TABLE users (id TEXT PRIMARY KEY, username TEXT, email TEXT, password TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE todos (
			id          TEXT PRIMARY KEY,
			user_id     TEXT NOT NULL,
			title       TEXT NOT NULL,
			description TEXT DEFAULT '',
			priority    TEXT DEFAULT 'medium',
			status      TEXT DEFAULT 'pending',
			due_date    DATETIME,
			tags        TEXT DEFAULT '[]',
			sub_tasks   TEXT DEFAULT '[]',
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, q := range queries {
		if _, err := conn.Exec(q); err != nil {
			t.Fatalf("Migrasi gagal: %v", err)
		}
	}

	userID := "test-user-123"
	_, err = conn.Exec(`INSERT INTO users (id, username, email, password) VALUES (?, 'tester', 'tester@example.com', 'hashed')`, userID)
	if err != nil {
		t.Fatalf("Gagal insert user: %v", err)
	}

	// Insert 3 todos dengan variasi prioritas dan tenggat waktu
	_, err = conn.Exec(`INSERT INTO todos (id, user_id, title, priority, due_date) VALUES 
		('t_low', ?, 'Task A', 'low', '2026-06-20T10:00:00Z'),
		('t_high', ?, 'Task B', 'high', '2026-06-15T10:00:00Z'),
		('t_med', ?, 'Task C', 'medium', NULL)`, userID, userID, userID)
	if err != nil {
		t.Fatalf("Gagal insert todos: %v", err)
	}

	dbWrapper := &database.DB{Conn: conn}
	handler := handlers.NewTodoHandler(dbWrapper)

	// Sub-test 1: Sort by priority desc (high, medium, low)
	t.Run("SortByPriorityDesc", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/todos?sort_by=priority&order=desc", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.GetAll(rec, req)

		var resp struct {
			Data struct {
				Todos []models.Todo `json:"todos"`
			} `json:"data"`
		}
		json.Unmarshal(rec.Body.Bytes(), &resp)

		if len(resp.Data.Todos) != 3 {
			t.Fatalf("Harusnya mengembalikan 3 todos, dapat %d", len(resp.Data.Todos))
		}

		// Urutan prioritas desc: t_high (high), t_med (medium), t_low (low)
		if resp.Data.Todos[0].ID != "t_high" || resp.Data.Todos[1].ID != "t_med" || resp.Data.Todos[2].ID != "t_low" {
			t.Errorf("Urutan salah untuk priority desc. Dapat: %s, %s, %s",
				resp.Data.Todos[0].ID, resp.Data.Todos[1].ID, resp.Data.Todos[2].ID)
		}
	})

	// Sub-test 2: Sort by due_date asc (t_high, t_low, t_med) -> t_med (NULL) harus di paling akhir
	t.Run("SortByDueDateAscWithNullLast", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/todos?sort_by=due_date&order=asc", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.GetAll(rec, req)

		var resp struct {
			Data struct {
				Todos []models.Todo `json:"todos"`
			} `json:"data"`
		}
		json.Unmarshal(rec.Body.Bytes(), &resp)

		// Urutan due_date asc (NULL di akhir): 2026-06-15 (t_high), 2026-06-20 (t_low), NULL (t_med)
		if resp.Data.Todos[0].ID != "t_high" || resp.Data.Todos[1].ID != "t_low" || resp.Data.Todos[2].ID != "t_med" {
			t.Errorf("Urutan salah untuk due_date asc dengan NULL di akhir. Dapat: %s, %s, %s",
				resp.Data.Todos[0].ID, resp.Data.Todos[1].ID, resp.Data.Todos[2].ID)
		}
	})
}

// TestBulkActionsHandler - Unit test untuk memverifikasi fitur aksi massal (bulk complete dan delete)
func TestBulkActionsHandler(t *testing.T) {
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Gagal membuka DB in-memory: %v", err)
	}
	defer conn.Close()

	// Migrasi tabel
	queries := []string{
		`CREATE TABLE users (
			id          TEXT PRIMARY KEY,
			username    TEXT UNIQUE NOT NULL,
			email       TEXT UNIQUE NOT NULL,
			password    TEXT NOT NULL,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE todos (
			id          TEXT PRIMARY KEY,
			user_id     TEXT NOT NULL,
			title       TEXT NOT NULL,
			description TEXT DEFAULT '',
			priority    TEXT DEFAULT 'medium',
			status      TEXT DEFAULT 'pending',
			due_date    DATETIME,
			tags        TEXT DEFAULT '[]',
			sub_tasks   TEXT DEFAULT '[]',
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, q := range queries {
		if _, err := conn.Exec(q); err != nil {
			t.Fatalf("Migrasi gagal: %v", err)
		}
	}

	userID := "test-user-123"
	_, err = conn.Exec(`INSERT INTO users (id, username, email, password) VALUES (?, 'tester', 'tester@example.com', 'hashed')`, userID)
	if err != nil {
		t.Fatalf("Gagal insert user: %v", err)
	}

	dbWrapper := &database.DB{Conn: conn}
	handler := handlers.NewTodoHandler(dbWrapper)

	t.Run("BulkComplete", func(t *testing.T) {
		// Reset database/todos untuk test ini
		_, _ = conn.Exec("DELETE FROM todos")
		_, err = conn.Exec(`INSERT INTO todos (id, user_id, title, status) VALUES 
			('t1', ?, 'Task 1', 'pending'),
			('t2', ?, 'Task 2', 'pending'),
			('t3', ?, 'Task 3', 'pending')`, userID, userID, userID)
		if err != nil {
			t.Fatalf("Gagal insert todos: %v", err)
		}

		payload := models.BulkActionRequest{
			IDs: []string{"t1", "t2"},
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/todos/bulk-complete", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.BulkComplete(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Diharapkan status 200, dapat %d. Response: %s", rec.Code, rec.Body.String())
		}

		// Cek DB apakah t1 dan t2 done, dan t3 pending
		var status1, status2, status3 string
		conn.QueryRow("SELECT status FROM todos WHERE id = 't1'").Scan(&status1)
		conn.QueryRow("SELECT status FROM todos WHERE id = 't2'").Scan(&status2)
		conn.QueryRow("SELECT status FROM todos WHERE id = 't3'").Scan(&status3)

		if status1 != "done" {
			t.Errorf("t1 harusnya done, didapat %s", status1)
		}
		if status2 != "done" {
			t.Errorf("t2 harusnya done, didapat %s", status2)
		}
		if status3 != "pending" {
			t.Errorf("t3 harusnya pending, didapat %s", status3)
		}
	})

	t.Run("BulkDelete", func(t *testing.T) {
		// Reset database/todos untuk test ini
		_, _ = conn.Exec("DELETE FROM todos")
		_, err = conn.Exec(`INSERT INTO todos (id, user_id, title, status) VALUES 
			('t1', ?, 'Task 1', 'pending'),
			('t2', ?, 'Task 2', 'pending'),
			('t3', ?, 'Task 3', 'pending')`, userID, userID, userID)
		if err != nil {
			t.Fatalf("Gagal insert todos: %v", err)
		}

		payload := models.BulkActionRequest{
			IDs: []string{"t1", "t3"},
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/todos/bulk-delete", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.BulkDelete(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Diharapkan status 200, dapat %d. Response: %s", rec.Code, rec.Body.String())
		}

		// Cek DB apakah t1 dan t3 terhapus, dan t2 tetap ada
		var count1, count2, count3 int
		conn.QueryRow("SELECT COUNT(*) FROM todos WHERE id = 't1'").Scan(&count1)
		conn.QueryRow("SELECT COUNT(*) FROM todos WHERE id = 't2'").Scan(&count2)
		conn.QueryRow("SELECT COUNT(*) FROM todos WHERE id = 't3'").Scan(&count3)

		if count1 != 0 {
			t.Errorf("t1 harusnya terhapus, didapat count %d", count1)
		}
		if count2 != 1 {
			t.Errorf("t2 harusnya tetap ada, didapat count %d", count2)
		}
		if count3 != 0 {
			t.Errorf("t3 harusnya terhapus, didapat count %d", count3)
		}
	})

	t.Run("ValidationFailure", func(t *testing.T) {
		payload := models.BulkActionRequest{
			IDs: []string{}, // Empty IDs
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/todos/bulk-complete", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.BulkComplete(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Diharapkan status 400 bad request, dapat %d. Response: %s", rec.Code, rec.Body.String())
		}
	})
}
