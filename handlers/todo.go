// Package handlers provides HTTP handlers for the Todo API, including CRUD operations and business logic.
package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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

// NewTodoHandler creates a new TodoHandler with the given database connection.
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
	tag := query.Get("tag")
	sortBy := query.Get("sort_by")
	order := strings.ToLower(query.Get("order"))

	// A.44 — Fungsi String:// Build and execute query SQL dinamis
	whereClause := `FROM todos WHERE user_id = ?`
	args := []interface{}{userID}

	// A.13 — Seleksi kondisi: filter opsional
	if status != "" {
		whereClause += " AND status = ?"
		args = append(args, status)
	}
	if priority != "" {
		whereClause += " AND priority = ?"
		args = append(args, priority)
	}
	if search != "" {
		whereClause += " AND (title LIKE ? OR description LIKE ?)"
		args = append(args, "%"+search+"%", "%"+search+"%")
	}
	if tag != "" {
		whereClause += " AND EXISTS (SELECT 1 FROM json_each(todos.tags) WHERE json_each.value = ?)"
		args = append(args, tag)
	}

	// 1. Ambil total data tersaring (filtered total) sebelum di-sorting dan di-limit
	var filteredTotal int
	countQuery := "SELECT COUNT(*) " + whereClause
	err := h.db.Conn.QueryRowContext(r.Context(), countQuery, args...).Scan(&filteredTotal)
	if err != nil {
		utils.Fail(w, http.StatusInternalServerError, "Gagal mengambil data statistik")
		return
	}
	// Ensure sort_order column exists (SQLite will ignore if already present)
	_, _ = h.db.Conn.ExecContext(r.Context(), "ALTER TABLE todos ADD COLUMN sort_order INTEGER DEFAULT 0")

	// Validasi whitelist untuk menghindari SQL Injection pada ORDER BY
	var orderClause string
	if sortBy == "" || sortBy == "custom" {
		// Default custom order: sort by explicit sort_order then created_at descending
		orderClause = "ORDER BY sort_order ASC, created_at DESC"
	} else {
		// Normalize order direction
		if order != "asc" && order != "desc" {
			order = "desc"
		}
		orderUpper := strings.ToUpper(order)
		switch sortBy {
		case "due_date":
			// UX: tasks without due date (NULL) appear at bottom
			orderClause = fmt.Sprintf("ORDER BY due_date IS NULL ASC, due_date %s", orderUpper)
		case "priority":
			// Priority ordering: high > medium > low
			orderClause = fmt.Sprintf(`ORDER BY CASE priority 
				WHEN 'high' THEN 3 
				WHEN 'medium' THEN 2 
				WHEN 'low' THEN 1 
				ELSE 0 
				END %s`, orderUpper)
		case "title":
			orderClause = fmt.Sprintf("ORDER BY title %s", orderUpper)
		case "created_at":
			fallthrough
		default:
			orderClause = fmt.Sprintf("ORDER BY created_at %s", orderUpper)
		}
	}

	sqlQuery := `SELECT id, user_id, title, description, priority, status, due_date, tags, sub_tasks, sort_order, created_at, updated_at ` + whereClause + " " + orderClause

	// Parsing parameter paginasi
	pageStr := query.Get("page")
	limitStr := query.Get("limit")
	var page, limit int
	if pageStr != "" {
		page, _ = strconv.Atoi(pageStr)
	}
	if limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
	}

	isPaginated := page > 0
	if isPaginated {
		if limit <= 0 {
			limit = 10 // default limit per halaman
		}
		offset := (page - 1) * limit
		sqlQuery += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	// A.56 — SQL: query dengan multiple args dengan context
	rows, err := h.db.Conn.QueryContext(r.Context(), sqlQuery, args...)
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
		var subTasksJSON string
		var dueDate sql.NullString

		err := rows.Scan(
			&todo.ID, &todo.UserID, &todo.Title, &todo.Description,
			&todo.Priority, &todo.Status, &dueDate, &tagsJSON, &subTasksJSON,
			&todo.SortOrder, &todo.CreatedAt, &todo.UpdatedAt,
		)
		if err != nil {
			continue
		}

		// A.53 — JSON: parse tags dari string JSON di database
		json.Unmarshal([]byte(tagsJSON), &todo.Tags)
		if todo.Tags == nil {
			todo.Tags = []string{}
		}

		json.Unmarshal([]byte(subTasksJSON), &todo.SubTasks)
		if todo.SubTasks == nil {
			todo.SubTasks = []models.SubTask{}
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

	// Jalankan pipeline generator -> enricher secara konkuren
	todoChan := utils.TodoGenerator(todos)
	enrichedChan := utils.EnrichWithOverdue(todoChan)

	todos = make([]models.Todo, 0)
	for t := range enrichedChan {
		todos = append(todos, t)
	}

	// Get true statistics langsung dari database agar selalu akurat secara global
	var total, pendingCount, inProgressCount, completedCount int
	var highCount, mediumCount, lowCount int
	h.db.Conn.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM todos WHERE user_id = ?", userID).Scan(&total)
	h.db.Conn.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM todos WHERE user_id = ? AND status = 'pending'", userID).Scan(&pendingCount)
	h.db.Conn.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM todos WHERE user_id = ? AND status = 'in_progress'", userID).Scan(&inProgressCount)
	h.db.Conn.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM todos WHERE user_id = ? AND status = 'done'", userID).Scan(&completedCount)
	h.db.Conn.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM todos WHERE user_id = ? AND priority = 'high'", userID).Scan(&highCount)
	h.db.Conn.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM todos WHERE user_id = ? AND priority = 'medium'", userID).Scan(&mediumCount)
	h.db.Conn.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM todos WHERE user_id = ? AND priority = 'low'", userID).Scan(&lowCount)

	// Ambil semua tag unik milik user
	uniqueTags := make([]string, 0)
	rowsTags, err := h.db.Conn.QueryContext(r.Context(), `SELECT DISTINCT json_each.value FROM todos, json_each(todos.tags) WHERE todos.user_id = ?`, userID)
	if err == nil {
		defer rowsTags.Close()
		for rowsTags.Next() {
			var t string
			if err := rowsTags.Scan(&t); err == nil {
				uniqueTags = append(uniqueTags, t)
			}
		}
	}

	stats := map[string]interface{}{
		"total":       total,
		"pending":     pendingCount,
		"in_progress": inProgressCount,
		"completed":   completedCount,
		"by_priority": map[string]int{
			"high":   highCount,
			"medium": mediumCount,
			"low":    lowCount,
		},
		"tags": uniqueTags,
	}

	pagination := map[string]interface{}{
		"page":           page,
		"limit":          limit,
		"filtered_total": filteredTotal,
		"has_more":       isPaginated && (page*limit) < filteredTotal,
	}

	utils.Success(w, http.StatusOK, "Berhasil mengambil todos", map[string]interface{}{
		"todos":      todos,
		"stats":      stats,
		"pagination": pagination,
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

	if req.SubTasks == nil {
		req.SubTasks = []models.SubTask{}
	}
	subTasksJSON, _ := json.Marshal(req.SubTasks)

	// Shift existing todos' sort_order to make room for the new item at top
	_, err := h.db.Conn.ExecContext(r.Context(), `UPDATE todos SET sort_order = sort_order + 1 WHERE user_id = ?`, userID)
	if err != nil {
		utils.Fail(w, http.StatusInternalServerError, "Gagal memperbarui urutan todo")
		return
	}

	// Insert new todo with sort_order = 0 (top position)
	_, err = h.db.Conn.ExecContext(
		r.Context(),
		`INSERT INTO todos (id, user_id, title, description, priority, status, due_date, tags, sub_tasks, sort_order)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		todoID, userID, strings.TrimSpace(req.Title), req.Description,
		req.Priority, models.StatusPending, dueDate, string(tagsJSON), string(subTasksJSON), 0,
	)
	if err != nil {
		utils.Fail(w, http.StatusInternalServerError, "Gagal menyimpan todo")
		return
	}

	// Ambil todo yang baru dibuat
	todo := h.getByID(r.Context(), todoID)
	utils.Success(w, http.StatusCreated, "Todo berhasil dibuat", todo)
}

// GetOne - GET /todos/{id} — ambil satu todo
func (h *TodoHandler) GetOne(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)
	todoID := extractID(r.URL.Path, "/todos/")

	todo := h.getByID(r.Context(), todoID)
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

	existing := h.getByID(r.Context(), todoID)
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
	if req.SubTasks != nil {
		existing.SubTasks = req.SubTasks
	}

	tagsJSON, _ := json.Marshal(existing.Tags)
	subTasksJSON, _ := json.Marshal(existing.SubTasks)

	// A.56 — SQL: update dengan context
	_, err := h.db.Conn.ExecContext(
		r.Context(),
		`UPDATE todos SET title=?, description=?, priority=?, status=?, tags=?, sub_tasks=?, updated_at=CURRENT_TIMESTAMP
		 WHERE id=? AND user_id=?`,
		existing.Title, existing.Description, existing.Priority,
		existing.Status, string(tagsJSON), string(subTasksJSON), todoID, userID,
	)
	if err != nil {
		utils.Fail(w, http.StatusInternalServerError, "Gagal update todo")
		return
	}

	utils.Success(w, http.StatusOK, "Todo berhasil diupdate", h.getByID(r.Context(), todoID))
}

// Delete - DELETE /todos/{id} — hapus todo
func (h *TodoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)
	todoID := extractID(r.URL.Path, "/todos/")

	result, err := h.db.Conn.ExecContext(
		r.Context(),
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

	_, err := h.db.Conn.ExecContext(
		r.Context(),
		`UPDATE todos SET status='done', updated_at=CURRENT_TIMESTAMP WHERE id=? AND user_id=?`,
		todoID, userID,
	)
	if err != nil {
		utils.Fail(w, http.StatusInternalServerError, "Gagal update status")
		return
	}

	utils.Success(w, http.StatusOK, "Todo ditandai selesai", h.getByID(r.Context(), todoID))
}

// SearchWithTimeout - A.35 — Channel Timeout: contoh penggunaan di handler
func (h *TodoHandler) SearchWithTimeout(w http.ResponseWriter, r *http.Request) {
	// A.35 — Channel Timeout
	// Buat context dengan timeout 5 detik dari request context
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	done := make(chan []models.Todo, 1)

	go func() {
		// Gunakan QueryContext(ctx, ...) agar database dibatalkan secara otomatis jika timeout
		rows, err := h.db.Conn.QueryContext(ctx, `SELECT id, user_id, title, description, priority, status, due_date, tags, sub_tasks, created_at, updated_at FROM todos WHERE title LIKE ?`,
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
			var subTasksJSON string
			var dueDate sql.NullString

			err := rows.Scan(
				&todo.ID, &todo.UserID, &todo.Title, &todo.Description,
				&todo.Priority, &todo.Status, &dueDate, &tagsJSON, &subTasksJSON,
				&todo.CreatedAt, &todo.UpdatedAt,
			)
			if err != nil {
				continue
			}

			json.Unmarshal([]byte(tagsJSON), &todo.Tags)
			if todo.Tags == nil {
				todo.Tags = []string{}
			}

			json.Unmarshal([]byte(subTasksJSON), &todo.SubTasks)
			if todo.SubTasks == nil {
				todo.SubTasks = []models.SubTask{}
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
		// Jalankan pipeline generator -> enricher secara konkuren
		todoChan := utils.TodoGenerator(result)
		enrichedChan := utils.EnrichWithOverdue(todoChan)

		processed := make([]models.Todo, 0)
		for t := range enrichedChan {
			processed = append(processed, t)
		}
		utils.Success(w, http.StatusOK, "Hasil pencarian", processed)
	case <-ctx.Done(): // A.35 — Timeout dipicu dari context.Done()
		utils.Fail(w, http.StatusGatewayTimeout, "Pencarian timeout")
	}
}

// Reorder - POST /todos/reorder — update sort_order based on client order
func (h *TodoHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)
	var req models.ReorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Fail(w, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}
	if err := req.Validate(); err != nil {
		utils.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	// Verify ownership of each ID
	placeholders := strings.Repeat("?,", len(req.OrderedIDs))
	placeholders = strings.TrimSuffix(placeholders, ",")
	query := fmt.Sprintf("SELECT id FROM todos WHERE user_id = ? AND id IN (%s)", placeholders)
	args := []interface{}{userID}
	for _, id := range req.OrderedIDs {
		args = append(args, id)
	}
	rows, err := h.db.Conn.QueryContext(r.Context(), query, args...)
	if err != nil {
		utils.Fail(w, http.StatusInternalServerError, "Gagal memverifikasi todo")
		return
	}
	defer rows.Close()
	found := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			found[id] = true
		}
	}
	for _, id := range req.OrderedIDs {
		if !found[id] {
			utils.Fail(w, http.StatusForbidden, "Todo tidak milik user atau tidak ditemukan: "+id)
			return
		}
	}
	// Transaction – atomic update of sort_order
	tx, err := h.db.Conn.BeginTx(r.Context(), nil)
	if err != nil {
		utils.Fail(w, http.StatusInternalServerError, "Gagal memulai transaksi")
		return
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(r.Context(), "UPDATE todos SET sort_order = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?")
	if err != nil {
		utils.Fail(w, http.StatusInternalServerError, "Gagal menyiapkan pernyataan")
		return
	}
	defer stmt.Close()
	for idx, id := range req.OrderedIDs {
		if _, err := stmt.ExecContext(r.Context(), idx, id, userID); err != nil {
			utils.Fail(w, http.StatusInternalServerError, "Gagal memperbarui urutan todo")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		utils.Fail(w, http.StatusInternalServerError, "Gagal menyelesaikan transaksi")
		return
	}
	utils.Success(w, http.StatusOK, "Urutan todo berhasil diperbarui", nil)
}

// BulkComplete - POST /todos/bulk-complete — tandai selesai beberapa todo secara massal
func (h *TodoHandler) BulkComplete(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)
	var req models.BulkActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Fail(w, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}
	if err := req.Validate(); err != nil {
		utils.Fail(w, http.StatusBadRequest, err.Error())
		return
	}

	// Buat placeholders dinamis (?, ?, ...)
	placeholders := make([]string, len(req.IDs))
	for i := range req.IDs {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(`UPDATE todos SET status='done', updated_at=CURRENT_TIMESTAMP 
	                       WHERE user_id = ? AND id IN (%s)`, strings.Join(placeholders, ","))

	args := make([]interface{}, 0, len(req.IDs)+1)
	args = append(args, userID)
	for _, id := range req.IDs {
		args = append(args, id)
	}

	_, err := h.db.Conn.ExecContext(r.Context(), query, args...)
	if err != nil {
		utils.Fail(w, http.StatusInternalServerError, "Gagal memperbarui data secara massal")
		return
	}

	utils.Success(w, http.StatusOK, fmt.Sprintf("%d tugas berhasil diselesaikan", len(req.IDs)), nil)
}

// BulkDelete - POST /todos/bulk-delete — hapus beberapa todo secara massal
func (h *TodoHandler) BulkDelete(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)
	var req models.BulkActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Fail(w, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}
	if err := req.Validate(); err != nil {
		utils.Fail(w, http.StatusBadRequest, err.Error())
		return
	}

	placeholders := make([]string, len(req.IDs))
	for i := range req.IDs {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf("DELETE FROM todos WHERE user_id = ? AND id IN (%s)", strings.Join(placeholders, ","))

	args := make([]interface{}, 0, len(req.IDs)+1)
	args = append(args, userID)
	for _, id := range req.IDs {
		args = append(args, id)
	}

	_, err := h.db.Conn.ExecContext(r.Context(), query, args...)
	if err != nil {
		utils.Fail(w, http.StatusInternalServerError, "Gagal menghapus data secara massal")
		return
	}

	utils.Success(w, http.StatusOK, fmt.Sprintf("%d tugas berhasil dihapus", len(req.IDs)), nil)
}

// A.19 — Multiple Return: helper yang bisa return nil dengan context
func (h *TodoHandler) getByID(ctx context.Context, id string) *models.Todo {
	var todo models.Todo
	var tagsJSON string
	var dueDate sql.NullString

	var subTasksJSON string
	err := h.db.Conn.QueryRowContext(
		ctx,
		`SELECT id, user_id, title, description, priority, status, due_date, tags, sub_tasks, created_at, updated_at
		 FROM todos WHERE id=?`, id,
	).Scan(&todo.ID, &todo.UserID, &todo.Title, &todo.Description,
		&todo.Priority, &todo.Status, &dueDate, &tagsJSON, &subTasksJSON,
		&todo.CreatedAt, &todo.UpdatedAt)

	if err != nil {
		return nil // A.23 — Pointer: return nil jika tidak ditemukan
	}

	json.Unmarshal([]byte(tagsJSON), &todo.Tags)
	if todo.Tags == nil {
		todo.Tags = []string{}
	}

	json.Unmarshal([]byte(subTasksJSON), &todo.SubTasks)
	if todo.SubTasks == nil {
		todo.SubTasks = []models.SubTask{}
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
