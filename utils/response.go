// utils/response.go
package utils

import (
	"encoding/json" // A.53 — JSON
	"net/http"
	"time"
)

// APIResponse - A.24 — Struct response standar API
type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`      // A.28 — interface{} / Any
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"` // A.40 — Time
}

// WriteJSON - A.28 — interface{}: bisa menerima tipe data apapun
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// A.53 — JSON: encode struct ke JSON
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Gagal encode JSON", http.StatusInternalServerError)
	}
}

func Success(w http.ResponseWriter, status int, message string, data interface{}) {
	WriteJSON(w, status, APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(), // A.40 — time.Now()
	})
}

func Fail(w http.ResponseWriter, status int, errMsg string) {
	WriteJSON(w, status, APIResponse{
		Success:   false,
		Error:     errMsg,
		Timestamp: time.Now(),
	})
}
