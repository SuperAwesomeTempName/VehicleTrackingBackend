package auth

import (
	"crypto/rsa"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTManager manages RSA-signed JWTs
type JWTManager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	issuer     string
	ttl        time.Duration
}

// TTL returns the token time-to-live duration
func (m *JWTManager) TTL() time.Duration {
	return m.ttl
}

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// NewJWTManagerFromFiles loads keys from file paths
func NewJWTManagerFromFiles(privPath, pubPath, issuer string, ttl time.Duration) (*JWTManager, error) {
	// Read and parse private key
	privBytes, err := os.ReadFile(privPath)
	if err != nil {
		return nil, errors.New("failed to read private key file: " + err.Error())
	}

	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(privBytes)
	if err != nil {
		// Try without password if first attempt fails
		privKey, err = jwt.ParseRSAPrivateKeyFromPEMWithPassword(privBytes, "")
		if err != nil {
			return nil, errors.New("failed to parse private key: " + err.Error())
		}
	}

	// Read and parse public key
	pubBytes, err := os.ReadFile(pubPath)
	if err != nil {
		return nil, errors.New("failed to read public key file: " + err.Error())
	}

	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubBytes)
	if err != nil {
		return nil, errors.New("failed to parse public key: " + err.Error())
	}

	return &JWTManager{privateKey: privKey, publicKey: pubKey, issuer: issuer, ttl: ttl}, nil
}

// GenerateToken generates a new JWT token for a user
func (m *JWTManager) GenerateToken(userID, role string) (string, error) {
	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    m.issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(m.privateKey)
}

// ValidateToken validates the given token string
func (m *JWTManager) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, errors.New("unexpected token signing method")
			}
			return m.publicKey, nil
		},
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}