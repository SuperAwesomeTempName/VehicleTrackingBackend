package auth

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/alexedwards/argon2id"
)

func HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

func ComparePassword(hash, password string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}

func GenerateRandom(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
