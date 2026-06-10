// utils/validator.go
package utils

import (
	"errors"
	"regexp"   // A.45 — Regexp
)

// A.45 — Regexp: pola untuk validasi email
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// A.37 — Error: custom error type
var (
	ErrInvalidEmail = errors.New("format email tidak valid")
	ErrWeakPassword = errors.New("password minimal 8 karakter")
)

// ValidateEmail menggunakan A.45 — Regexp
func ValidateEmail(email string) error {
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}
	return nil
}

// ValidatePass: A.44 — Fungsi String
func ValidatePass(pass string) error {
	if len(pass) < 8 {
		return ErrWeakPassword
	}
	return nil
}
