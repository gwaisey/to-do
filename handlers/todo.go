// handlers/todo.go
package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"to-do/database"
	"to-do/middleware"
	"to-do/models"
	"to-do/utils"
)

// TodoHandler - A.24 — Struct: handler untuk Todo
type TodoHandler struct {
	db *database.DB
}

func NewTodoHandler(db *database.DB) *TodoHandler {
	return &TodoHandler{db: db}
}

// GetAll - GET /todos — ambil semua todo milik user yang login
func (h *TodoHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)

	// A.52 — URL Parsing: ambil query params untuk filter
	query := r.URL.Query()
	status := query.Get("status")
	priority := query.Get("priority")
	search := query.Get("search")

	// A.44 — Fungsi String: build query SQL dinamis
	sqlQuery := `SELECT id, user_id, title, description, priority, status, due_date, tags, created_at, updated_at
	             FROM todos WHERE user_id = ?`
	args := []interface{}{userID}

	// A.13 — Seleksi kondisi: filter opsional
	if status != "" {
		sqlQuery += " AND status = ?"
		args = append(args, status)
	}
	if priority != "" {
		sqlQuery += " AND priority = ?"
		args = append(args, priority)
	}
	if search != "" {
		sqlQuery += " AND (title LIKE ? OR description LIKE ?)"
		args = append(args, "%"+search+"%", "%"+search+"%")
	}
	sqlQuery += " ORDER BY created_at DESC"

	// A.56 — SQL: query dengan multiple args
	rows, err := h.db.Conn.Query(sqlQuery, args...)
	if err != nil {
		utils.Fail(w, http.StatusInternalServerError, "Gagal mengambil data")
		return
	}
	defer rows.Close()

	// A.16 — Slice: kumpulkan hasil query
	todos := make([]models.Todo, 0)

	// A.14 — Perulangan: iterasi rows
	for rows.Next() {
		var todo models.Todo
		var tagsJSON string
		var dueDate sql.NullString

		err := rows.Scan(
			&todo.ID, &todo.UserID, &todo.Title, &todo.Description,
			&todo.Priority, &todo.Status, &dueDate, &tagsJSON,
			&todo.CreatedAt, &todo.UpdatedAt,
		)
		if err != nil {
			continue
		}

		// A.53 — JSON: parse tags dari string JSON di database
		json.Unmarshal([]byte(tagsJSON), &todo.Tags)
		if todo.Tags == nil {
			todo.Tags = []string{}
		}

		// A.23 — Pointer: handle nullable due_date
		if dueDate.Valid {
			t, err := time.Parse(time.RFC3339, dueDate.String)
			if err == nil {
				todo.DueDate = &t
			}
		}

		todos = append(todos, todo)
	}

	// Get true statistics directly from the database for the user
	var total, pendingCount, inProgressCount, completedCount int
	h.db.Conn.QueryRow("SELECT COUNT(*) FROM todos WHERE user_id = ?", userID).Scan(&total)
	h.db.Conn.QueryRow("SELECT COUNT(*) FROM todos WHERE user_id = ? AND status = 'pending'", userID).Scan(&pendingCount)
	h.db.Conn.QueryRow("SELECT COUNT(*) FROM todos WHERE user_id = ? AND status = 'in_progress'", userID).Scan(&inProgressCount)
	h.db.Conn.QueryRow("SELECT COUNT(*) FROM todos WHERE user_id = ? AND status = 'done'", userID).Scan(&completedCount)

	// A.17 — Map: tambahkan statistik (group by priority)
	grouped := models.GroupByPriority(todos)
	stats := map[string]interface{}{
		"total":       total,
		"pending":     pendingCount,
		"in_progress": inProgressCount,
		"completed":   completedCount,
		"by_priority": map[string]int{
			"high":   len(grouped[models.PriorityHigh]),
			"medium": len(grouped[models.PriorityMedium]),
			"low":    len(grouped[models.PriorityLow]),
		},
	}

	utils.Success(w, http.StatusOK, "Berhasil mengambil todos", map[string]interface{}{
		"todos": todos,
		"stats": stats,
	})
}

// Create - POST /todos — buat todo baru
func (h *TodoHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)

	var req models.CreateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Fail(w, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}

	// A.27 — Interface: panggil Validate()
	if err := req.Validate(); err != nil {
		utils.Fail(w, http.StatusBadRequest, err.Error())
		return
	}

	// A.13 — Seleksi kondisi: set default priority
	if req.Priority == "" {
		req.Priority = models.PriorityMedium
	}

	// A.16 — Slice: default tags jika nil
	if req.Tags == nil {
		req.Tags = []string{}
	}

	// A.53 — JSON: encode tags ke JSON string untuk SQLite
	tagsJSON, _ := json.Marshal(req.Tags)

	// A.40 — Time: parse due date
	var dueDate *time.Time
	if req.DueDate != "" {
		t, err := time.Parse(time.RFC3339, req.DueDate)
		if err == nil {
			dueDate = &t
		}
	}

	// A.39 — Random: UUID untuk ID baru
	todoID := utils.GenerateID()

	// A.56 — SQL: insert
	_, err := h.db.Conn.Exec(
		`INSERT INTO todos (id, user_id, title, description, priority, status, due_date, tags)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		todoID, userID, strings.TrimSpace(req.Title), req.Description,
		req.Priority, models.StatusPending, dueDate, string(tagsJSON),
	)
	if err != nil {
		utils.Fail(w, http.StatusInternalServerError, "Gagal menyimpan todo")
		return
	}

	// Ambil todo yang baru dibuat
	todo := h.getByID(todoID)
	utils.Success(w, http.StatusCreated, "Todo berhasil dibuat", todo)
}

// GetOne - GET /todos/{id} — ambil satu todo
func (h *TodoHandler) GetOne(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)
	todoID := extractID(r.URL.Path, "/todos/")

	todo := h.getByID(todoID)
	if todo == nil {
		utils.Fail(w, http.StatusNotFound, "Todo tidak ditemukan")
		return
	}

	// A.13 — Seleksi kondisi: pastikan todo milik user yang login
	if todo.UserID != userID {
		utils.Fail(w, http.StatusForbidden, "Tidak punya akses ke todo ini")
		return
	}

	utils.Success(w, http.StatusOK, "Berhasil", todo)
}

// Update - PUT /todos/{id} — update todo
func (h *TodoHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)
	todoID := extractID(r.URL.Path, "/todos/")

	existing := h.getByID(todoID)
	if existing == nil || existing.UserID != userID {
		utils.Fail(w, http.StatusNotFound, "Todo tidak ditemukan")
		return
	}

	var req models.UpdateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Fail(w, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}

	if err := req.Validate(); err != nil {
		utils.Fail(w, http.StatusBadRequest, err.Error())
		return
	}

	// A.23 — Pointer: update hanya field yang dikirim (partial update)
	if req.Title != nil {
		existing.Title = *req.Title
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Priority != nil {
		existing.Priority = *req.Priority
	}
	if req.Status != nil {
		existing.Status = *req.Status
	}
	if req.Tags != nil {
		existing.Tags = req.Tags
	}

	tagsJSON, _ := json.Marshal(existing.Tags)

	// A.56 — SQL: update
	_, err := h.db.Conn.Exec(
		`UPDATE todos SET title=?, description=?, priority=?, status=?, tags=?, updated_at=CURRENT_TIMESTAMP
		 WHERE id=? AND user_id=?`,
		existing.Title, existing.Description, existing.Priority,
		existing.Status, string(tagsJSON), todoID, userID,
	)
	if err != nil {
		utils.Fail(w, http.StatusInternalServerError, "Gagal update todo")
		return
	}

	utils.Success(w, http.StatusOK, "Todo berhasil diupdate", h.getByID(todoID))
}

// Delete - DELETE /todos/{id} — hapus todo
func (h *TodoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)
	todoID := extractID(r.URL.Path, "/todos/")

	result, err := h.db.Conn.Exec(
		`DELETE FROM todos WHERE id=? AND user_id=?`, todoID, userID,
	)
	if err != nil {
		utils.Fail(w, http.StatusInternalServerError, "Gagal menghapus todo")
		return
	}

	// A.56 — SQL: cek apakah ada row yang terhapus
	affected, _ := result.RowsAffected()
	if affected == 0 {
		utils.Fail(w, http.StatusNotFound, "Todo tidak ditemukan")
		return
	}

	utils.Success(w, http.StatusOK, "Todo berhasil dihapus", nil)
}

// MarkDone - PATCH /todos/{id}/complete — tandai selesai
func (h *TodoHandler) MarkDone(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)

	// A.44 — Fungsi String: extract ID dari URL
	path := strings.TrimSuffix(r.URL.Path, "/complete")
	todoID := extractID(path, "/todos/")

	_, err := h.db.Conn.Exec(
		`UPDATE todos SET status='done', updated_at=CURRENT_TIMESTAMP WHERE id=? AND user_id=?`,
		todoID, userID,
	)
	if err != nil {
		utils.Fail(w, http.StatusInternalServerError, "Gagal update status")
		return
	}

	utils.Success(w, http.StatusOK, "Todo ditandai selesai", h.getByID(todoID))
}

// SearchWithTimeout - A.35 — Channel Timeout: contoh penggunaan di handler
func (h *TodoHandler) SearchWithTimeout(w http.ResponseWriter, r *http.Request) {
	// A.35 — Channel Timeout
	done := make(chan []models.Todo, 1)

	go func() {
		// Simulasi query yang mungkin lama
		rows, err := h.db.Conn.Query(`SELECT id, user_id, title, description, priority, status, due_date, tags, created_at, updated_at FROM todos WHERE title LIKE ?`,
			"%"+r.URL.Query().Get("q")+"%")
		if err != nil {
			done <- nil
			return
		}
		defer rows.Close()
		var todos []models.Todo
		for rows.Next() {
			var todo models.Todo
			var tagsJSON string
			var dueDate sql.NullString

			err := rows.Scan(
				&todo.ID, &todo.UserID, &todo.Title, &todo.Description,
				&todo.Priority, &todo.Status, &dueDate, &tagsJSON,
				&todo.CreatedAt, &todo.UpdatedAt,
			)
			if err != nil {
				continue
			}

			json.Unmarshal([]byte(tagsJSON), &todo.Tags)
			if todo.Tags == nil {
				todo.Tags = []string{}
			}

			if dueDate.Valid {
				t, err := time.Parse(time.RFC3339, dueDate.String)
				if err == nil {
					todo.DueDate = &t
				}
			}

			todos = append(todos, todo)
		}
		done <- todos
	}()

	// A.33 — Channel Select: tunggu hasil atau timeout
	select {
	case result := <-done:
		utils.Success(w, http.StatusOK, "Hasil pencarian", result)
	case <-time.After(5 * time.Second): // A.35 — Timeout
		utils.Fail(w, http.StatusGatewayTimeout, "Pencarian timeout")
	}
}

// A.19 — Multiple Return: helper yang bisa return nil
func (h *TodoHandler) getByID(id string) *models.Todo {
	var todo models.Todo
	var tagsJSON string
	var dueDate sql.NullString

	err := h.db.Conn.QueryRow(
		`SELECT id, user_id, title, description, priority, status, due_date, tags, created_at, updated_at
		 FROM todos WHERE id=?`, id,
	).Scan(&todo.ID, &todo.UserID, &todo.Title, &todo.Description,
		&todo.Priority, &todo.Status, &dueDate, &tagsJSON,
		&todo.CreatedAt, &todo.UpdatedAt)

	if err != nil {
		return nil // A.23 — Pointer: return nil jika tidak ditemukan
	}

	json.Unmarshal([]byte(tagsJSON), &todo.Tags)
	if todo.Tags == nil {
		todo.Tags = []string{}
	}

	if dueDate.Valid {
		t, _ := time.Parse(time.RFC3339, dueDate.String)
		todo.DueDate = &t
	}

	return &todo
}

// A.44 — Fungsi String: helper extract ID dari path
func extractID(path, prefix string) string {
	return strings.TrimPrefix(path, prefix)
}
