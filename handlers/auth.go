// handlers/auth.go
package handlers

import (
	"encoding/json"
	"net/http"
	"time"
	"to-do/config"
	"to-do/database"
	"to-do/models"
	"to-do/utils"

	"github.com/golang-jwt/jwt/v5"
)

// AuthHandler - A.24 — Struct: handler dengan dependency
type AuthHandler struct {
	db  *database.DB
	cfg *config.Config
}

// NewAuthHandler - A.18 — Fungsi: constructor
func NewAuthHandler(db *database.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{db: db, cfg: cfg}
}

// Register - POST /register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	// A.53 — JSON: decode request body
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Fail(w, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}

	// A.45 — Validasi dengan regexp
	if err := utils.ValidateEmail(req.Email); err != nil {
		utils.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := utils.ValidatePass(req.Password); err != nil {
		utils.Fail(w, http.StatusBadRequest, err.Error())
		return
	}

	// A.47 — Hash password sebelum disimpan
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.Fail(w, http.StatusInternalServerError, "Gagal memproses password")
		return
	}

	// A.39 — Random: generate unique ID
	userID := utils.GenerateID()

	// A.56 — SQL: insert ke database
	_, err = h.db.Conn.Exec(
		`INSERT INTO users (id, username, email, password) VALUES (?, ?, ?, ?)`,
		userID, req.Username, req.Email, hashedPassword,
	)
	if err != nil {
		utils.Fail(w, http.StatusConflict, "Username atau email sudah terdaftar")
		return
	}

	user := models.User{
		ID:       userID,
		Username: req.Username,
		Email:    req.Email,
	}

	utils.Success(w, http.StatusCreated, "Registrasi berhasil", user)
}

// Login - POST /login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Fail(w, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}

	// A.56 — SQL: query user berdasarkan email
	var user models.User
	var hashedPassword string
	err := h.db.Conn.QueryRow(
		`SELECT id, username, email, password FROM users WHERE email = ?`,
		req.Email,
	).Scan(&user.ID, &user.Username, &user.Email, &hashedPassword)

	if err != nil {
		utils.Fail(w, http.StatusUnauthorized, "Email atau password salah")
		return
	}

	// Verifikasi password
	if !utils.CheckPassword(req.Password, hashedPassword) {
		utils.Fail(w, http.StatusUnauthorized, "Email atau password salah")
		return
	}

	// C.32 — JWT: buat token
	// A.40 — Time: set expiry
	expiryTime := time.Now().Add(time.Duration(h.cfg.JWTExpiryHours) * time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"exp":     expiryTime.Unix(),
	})

	// A.19 — Multiple Return: Sign mengembalikan (string, error)
	tokenStr, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		utils.Fail(w, http.StatusInternalServerError, "Gagal membuat token")
		return
	}

	utils.Success(w, http.StatusOK, "Login berhasil", models.LoginResponse{
		Token: tokenStr,
		User:  user,
	})
}
