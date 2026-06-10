// tests/todo_test.go
package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing" // A.58 — Unit Test
	"todo-api/models"
	"todo-api/utils"
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
	password := "password123"

	hash, err := utils.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword gagal: %v", err)
	}

	if hash == password {
		t.Error("Hash tidak boleh sama dengan password asli")
	}

	if !utils.CheckPassword(password, hash) {
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

