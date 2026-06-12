// utils/random.go
// Package utils provides helper utilities for the Todo API, including response handling, hashing, validation, and pipelines.
package utils

import (
	"crypto/rand"
	"crypto/sha1"     // A.47 — Hash SHA1
	"encoding/base64" // A.46 — Base64
	"encoding/hex"
	"fmt"
	"math/big"
	mathrand "math/rand" // A.39 — Random
	"strings"            // A.44 — Fungsi String
	"time"
)

// GenerateID - A.39 — Random: generate UUID sederhana menggunakan crypto/rand
func GenerateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// HashSHA1 - A.47 — Hash SHA1
func HashSHA1(input string) string {
	h := sha1.New()
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil))
}

// EncodeBase64 - A.46 — Encode Base64
func EncodeBase64(input string) string {
	return base64.StdEncoding.EncodeToString([]byte(input))
}

// DecodeBase64 - A.46 — Decode Base64
func DecodeBase64(encoded string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// GenerateOTP - A.39 — Random: generate kode OTP 6 digit
func GenerateOTP() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("%06d", n.Int64())
}

// SanitizeInput - A.44 — Fungsi String: sanitasi input
func SanitizeInput(s string) string {
	s = strings.TrimSpace(s)             // hapus spasi di awal/akhir
	s = strings.ToLower(s)               // lowercase
	s = strings.ReplaceAll(s, "  ", " ") // hapus double space
	return s
}

// A.41 — Timer & Ticker: seed random (untuk demo)
func init() {
	mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
}
