// Package models provides data structures, request/response types, and validation logic for the To-Do API.
package models

import (
    "errors"
    "strings"
    "time"
)

// Priority represents the importance level of a to‑do item.
type Priority string

// Status represents the current state of a to‑do item.
type Status string

// Predefined priority values.
const (
    PriorityLow    Priority = "low"
    PriorityMedium Priority = "medium"
    PriorityHigh   Priority = "high"
)

// Predefined status values.
const (
    StatusPending    Status = "pending"
    StatusInProgress Status = "in_progress"
    StatusDone       Status = "done"
)

// User represents an account holder.
type User struct {
    ID        string    `json:"id"`
    Username  string    `json:"username"`
    Email     string    `json:"email"`
    Password  string    `json:"-"`          // Field omitted from JSON responses.
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// SubTask represents a child task of a to‑do item.
type SubTask struct {
    ID     string `json:"id"`
    Title  string `json:"title"`
    Status Status `json:"status"` // pending, done
}

// Todo is the main to‑do entity.
type Todo struct {
    ID            string     `json:"id"`
    UserID        string     `json:"user_id"`
    Title         string     `json:"title"`
    Description   string     `json:"description,omitempty"`
    Priority      Priority   `json:"priority"`
    Status        Status     `json:"status"`
    DueDate       *time.Time `json:"due_date,omitempty"`
    Tags          []string   `json:"tags"`
    SubTasks      []SubTask  `json:"sub_tasks"`
    SortOrder     int        `json:"sort_order"`
    Overdue       bool       `json:"is_overdue"`
    TimeRemaining string     `json:"time_remaining,omitempty"`
    CreatedAt     time.Time  `json:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at"`
}

// Validator defines a contract for request validation.
type Validator interface {
    Validate() error
}

// CreateTodoRequest captures fields required to create a new to‑do.
type CreateTodoRequest struct {
    Title       string    `json:"title"`
    Description string    `json:"description"`
    Priority    Priority  `json:"priority"`
    DueDate     string    `json:"due_date"` // Expected as ISO‑8601 string.
    Tags        []string  `json:"tags"`
    SubTasks    []SubTask `json:"sub_tasks"`
}

// UpdateTodoRequest captures optional fields for updating a to‑do.
type UpdateTodoRequest struct {
    Title       *string   `json:"title"`
    Description *string   `json:"description"`
    Priority    *Priority `json:"priority"`
    Status      *Status   `json:"status"`
    DueDate     *string   `json:"due_date"`
    Tags        []string  `json:"tags"`
    SubTasks    []SubTask `json:"sub_tasks"`
}

// RegisterRequest holds data needed for user registration.
type RegisterRequest struct {
    Username string `json:"username"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

// LoginRequest holds credentials for authentication.
type LoginRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

// LoginResponse returns a JWT token and the authenticated user.
type LoginResponse struct {
    Token string `json:"token"`
    User  User   `json:"user"`
}

// BulkActionRequest contains a list of IDs for bulk operations.
type BulkActionRequest struct {
    IDs []string `json:"ids"`
}

// ReorderRequest is used to reorder to‑dos via drag‑and‑drop.
type ReorderRequest struct {
    OrderedIDs []string `json:"ordered_ids"`
}

// Validate checks that the reorder payload is not empty and contains unique IDs.
func (r *ReorderRequest) Validate() error {
    if len(r.OrderedIDs) == 0 {
        return errors.New("daftar ID tidak boleh kosong")
    }
    seen := make(map[string]bool)
    for _, id := range r.OrderedIDs {
        if id == "" {
            return errors.New("ID todo tidak boleh kosong")
        }
        if seen[id] {
            return errors.New("ID duplikat ditemukan dalam urutan")
        }
        seen[id] = true
    }
    return nil
}

// Validate ensures required fields for creating a to‑do are present.
func (r *CreateTodoRequest) Validate() error {
    return validateTodoInput(r.Title, string(r.Priority))
}

// Validate checks that BulkActionRequest contains at least one ID.
func (r *BulkActionRequest) Validate() error {
    if len(r.IDs) == 0 {
        return errors.New("daftar ID tidak boleh kosong")
    }
    return nil
}

// Validate checks optional fields of UpdateTodoRequest.
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

// FilterByStatus returns a slice containing items that satisfy the predicate.
func FilterByStatus[T any](items []T, predicate func(T) bool) []T {
    result := make([]T, 0)
    for _, item := range items {
        if predicate(item) {
            result = append(result, item)
        }
    }
    return result
}

// GroupByPriority groups to‑dos by their priority.
func GroupByPriority(todos []Todo) map[Priority][]Todo {
    grouped := make(map[Priority][]Todo)
    for _, todo := range todos {
        grouped[todo.Priority] = append(grouped[todo.Priority], todo)
    }
    return grouped
}

// TimeUntilDue returns the remaining duration until the due date, or nil if none is set.
func (t *Todo) TimeUntilDue() *time.Duration {
    if t.DueDate == nil {
        return nil
    }
    remaining := time.Until(*t.DueDate)
    return &remaining
}

// IsOverdue reports whether the to‑do's due date has passed and it is not completed.
func (t *Todo) IsOverdue() bool {
    if t.DueDate == nil || t.Status == StatusDone {
        return false
    }
    return time.Now().After(*t.DueDate)
}

// Error variables used for validation.
var (
    ErrEmptyTitle      = errors.New("judul tidak boleh kosong")
    ErrTitleTooLong    = errors.New("judul terlalu panjang (maks 200 karakter)")
    ErrInvalidPriority = errors.New("priority harus: low, medium, atau high")
)

// validateTodoInput validates title and priority fields.
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
