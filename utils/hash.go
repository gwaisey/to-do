// utils/hash.go
package utils

import "golang.org/x/crypto/bcrypt"

// HashPassword menggunakan bcrypt (lebih aman dari SHA1 untuk password)
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword memverifikasi password dengan hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
