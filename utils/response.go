// Package utils provides helper utilities for the Todo API, including response handling, hashing, validation, and pipelines.
package utils

import (
    "errors"
	"encoding/json" // A.53 — JSON
	"net/http"
	"time"
)

// APIError represents a custom error with an associated HTTP status code.
type APIError struct {
    Msg  string
    Code int
}

// Error implements the error interface.
func (e *APIError) Error() string { return e.Msg }

// NewAPIError creates a new APIError with the given message and status code.
func NewAPIError(msg string, code int) error { return &APIError{Msg: msg, Code: code} }

// getStatus returns the HTTP status code for a given error, using the custom APIError if present.
func getStatus(err error) int {
    if apiErr, ok := err.(*APIError); ok {
        return apiErr.Code
    }
    if status, ok := errStatusMap[err]; ok {
        return status
    }
    return http.StatusInternalServerError
}

// WriteError writes a JSON error response, determining the status via getStatus.
func WriteError(w http.ResponseWriter, err error) {
    status := getStatus(err)
    WriteJSON(w, status, APIResponse{Success: false, Error: err.Error(), Timestamp: time.Now()})
}
// ----- Custom Errors -----
var (
    ErrNotFound   = errors.New("resource not found")
    ErrForbidden  = errors.New("forbidden")
    ErrBadRequest = errors.New("bad request")
    ErrConflict   = errors.New("conflict")
    ErrInternal   = errors.New("internal server error")
)

// map error to HTTP status code
var errStatusMap = map[error]int{
    ErrNotFound:   http.StatusNotFound,
    ErrForbidden:  http.StatusForbidden,
    ErrBadRequest: http.StatusBadRequest,
    ErrConflict:   http.StatusConflict,
    ErrInternal:   http.StatusInternalServerError,
}

type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"` // A.28 — interface{} / Any
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

// Success writes a successful API response.
func Success(w http.ResponseWriter, status int, message string, data interface{}) {
	WriteJSON(w, status, APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(), // A.40 — time.Now()
	})
}


// Fail writes an error API response with the given status and error message.
func Fail(w http.ResponseWriter, status int, errMsg string) {
	WriteJSON(w, status, APIResponse{
		Success:   false,
		Error:     errMsg,
		Timestamp: time.Now(),
	})
}
// Helper shortcuts for common HTTP errors.
func BadRequest(w http.ResponseWriter, msg string) {
    WriteError(w, NewAPIError(msg, http.StatusBadRequest))
}
func NotFound(w http.ResponseWriter, msg string) {
    WriteError(w, NewAPIError(msg, http.StatusNotFound))
}
func Unauthorized(w http.ResponseWriter, msg string) {
    WriteError(w, NewAPIError(msg, http.StatusUnauthorized))
}
func Forbidden(w http.ResponseWriter, msg string) {
    WriteError(w, NewAPIError(msg, http.StatusForbidden))
}
func Conflict(w http.ResponseWriter, msg string) {
    WriteError(w, NewAPIError(msg, http.StatusConflict))
}
func InternalServerError(w http.ResponseWriter, msg string) {
    WriteError(w, NewAPIError(msg, http.StatusInternalServerError))
}
