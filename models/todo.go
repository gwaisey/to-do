// models/todo.go
package models

import (
	"errors"
	"strings"
	"time"
)

// A.10 — Tipe Data: custom type dari string
type Priority string
type Status string

// A.11 — Konstanta dengan tipe kustom
const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"

	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

// User - A.24 — Struct: representasi data user
// A.26 — Field yang diawali huruf kapital = Public (Exported)
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // A.26 — field ini tidak dikirim ke JSON response
	CreatedAt time.Time `json:"created_at"` // A.40 — Time
	UpdatedAt time.Time `json:"updated_at"`
}

// Todo - A.24 — Struct untuk To-Do
type Todo struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Priority    Priority  `json:"priority"`
	Status      Status    `json:"status"`
	DueDate     *time.Time `json:"due_date,omitempty"` // A.23 — Pointer: bisa nil
	Tags        []string  `json:"tags"` // A.16 — Slice
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Validator - A.27 — Interface: kontrak method yang harus diimplementasikan
type Validator interface {
	Validate() error
}

// Request structs (dipisah dari model utama — praktik bersih)
type CreateTodoRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    Priority `json:"priority"`
	DueDate     string   `json:"due_date"` // diterima sebagai string, diparse ke time
	Tags        []string `json:"tags"` // A.16 — Slice
}

type UpdateTodoRequest struct {
	Title       *string   `json:"title"`       // A.23 — Pointer: opsional
	Description *string   `json:"description"`
	Priority    *Priority `json:"priority"`
	Status      *Status   `json:"status"`
	DueDate     *string   `json:"due_date"`
	Tags        []string  `json:"tags"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// A.27 — Interface: implementasi Validate() untuk CreateTodoRequest
func (r *CreateTodoRequest) Validate() error {
	return validateTodoInput(r.Title, string(r.Priority))
}

// Validate untuk UpdateTodoRequest
func (r *UpdateTodoRequest) Validate() error {
	if r.Title != nil {
		if strings.TrimSpace(*r.Title) == "" {
			return ErrEmptyTitle
		}
		if len(*r.Title) > 200 {
			return ErrTitleTooLong
		}
	}
	if r.Priority != nil {
		valid := map[Priority]bool{PriorityLow: true, PriorityMedium: true, PriorityHigh: true}
		if !valid[*r.Priority] {
			return ErrInvalidPriority
		}
	}
	return nil
}


// A.65 — Go Generics: fungsi generik untuk filter slice
func FilterByStatus[T any](items []T, predicate func(T) bool) []T {
	// A.16 — Slice: make dengan length 0
	result := make([]T, 0)
	for _, item := range items {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// A.17 — Map: grouping todos berdasarkan priority
func GroupByPriority(todos []Todo) map[Priority][]Todo {
	// A.17 — Map: inisialisasi
	grouped := make(map[Priority][]Todo)
	for _, todo := range todos {
		grouped[todo.Priority] = append(grouped[todo.Priority], todo)
	}
	return grouped
}

// A.42 — Time Duration: hitung sisa waktu ke due date
func (t *Todo) TimeUntilDue() *time.Duration {
	if t.DueDate == nil {
		return nil // A.23 — Pointer nil
	}
	remaining := time.Until(*t.DueDate) // A.42 — time.Duration
	return &remaining
}

// A.40 — Time: cek apakah todo sudah overdue
func (t *Todo) IsOverdue() bool {
	if t.DueDate == nil || t.Status == StatusDone {
		return false
	}
	return time.Now().After(*t.DueDate) // A.40 — time.Now()
}

// Error definitions relocated here from utils to avoid circular dependency:
// models -> utils -> models
var (
	ErrEmptyTitle      = errors.New("judul tidak boleh kosong")
	ErrTitleTooLong    = errors.New("judul terlalu panjang (maks 200 karakter)")
	ErrInvalidPriority = errors.New("priority harus: low, medium, atau high")
)

// validateTodoInput - validation logic relocated to models package
func validateTodoInput(title, priority string) error {
	if strings.TrimSpace(title) == "" {
		return ErrEmptyTitle
	}
	if len(title) > 200 {
		return ErrTitleTooLong
	}
	valid := map[string]bool{"low": true, "medium": true, "high": true, "": true}
	if !valid[priority] {
		return ErrInvalidPriority
	}
	return nil
}
