package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/SuperAwesomeTempName/VehicleTrackingBackend/internal/db"
	"github.com/alexedwards/argon2id"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=2"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// Register handler
func RegisterHandler(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	// Hash password with Argon2id
	hash, err := argon2id.CreateHash(req.Password, argon2id.DefaultParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failed"})
		return
	}
	id := uuid.New().String()
	if err := db.InsertUser(ctx, id, req.Name, req.Email, hash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db insert failed"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "email": req.Email})
}

// Login handler
func LoginHandler(jwtMgr *JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx := c.Request.Context()
		u, err := db.FindUserByEmail(ctx, req.Email)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		ok, err := argon2id.ComparePasswordAndHash(req.Password, u.PasswordHash)
		if err != nil || !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		// Generate access token
		access, err := jwtMgr.GenerateToken(u.ID, u.Role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token gen failed"})
			return
		}
		// Create refresh token (random)
		refreshRaw := uuid.New().String() + "-" + time.Now().Format(time.RFC3339Nano)
		h := sha256.Sum256([]byte(refreshRaw))
		refreshHash := base64.RawURLEncoding.EncodeToString(h[:])
		// Persist refresh token hash in DB
		exp := time.Now().Add(30 * 24 * time.Hour) // 30 days
		if err := db.StoreRefreshToken(ctx, u.ID, refreshHash, exp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "store refresh failed"})
			return
		}
		resp := TokenResponse{
			AccessToken:  access,
			RefreshToken: refreshRaw,
			TokenType:    "Bearer",
			ExpiresIn:    int64(jwtMgr.ttl.Seconds()),
		}
		c.JSON(http.StatusOK, resp)
	}
}

// Refresh handler: rotate refresh token
func RefreshHandler(jwtMgr *JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			RefreshToken string `json:"refresh_token" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx := c.Request.Context()
		// Lookup refresh token hash in DB and check not revoked and not expired
		h := sha256.Sum256([]byte(body.RefreshToken))
		refreshHash := base64.RawURLEncoding.EncodeToString(h[:])
		userID, ok, err := db.GetUserIDByRefreshHash(ctx, refreshHash)
		if err != nil || !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
			return
		}
		// rotate: revoke old token and create a new refresh token
		if err := db.RevokeRefreshTokenByHash(ctx, refreshHash); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "revoke failed"})
			return
		}
		// new refresh token
		newRaw := uuid.New().String() + "-" + time.Now().Format(time.RFC3339Nano)
		h2 := sha256.Sum256([]byte(newRaw))
		newHash := base64.RawURLEncoding.EncodeToString(h2[:])
		exp := time.Now().Add(30 * 24 * time.Hour)
		if err := db.StoreRefreshToken(ctx, userID, newHash, exp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "store refresh failed"})
			return
		}
		// issue new access token
		user, err := db.GetUserByID(ctx, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user lookup failed"})
			return
		}
		access, err := jwtMgr.GenerateToken(user.ID, user.Role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token create failed"})
			return
		}
		c.JSON(http.StatusOK, TokenResponse{
			AccessToken:  access,
			RefreshToken: newRaw,
			TokenType:    "Bearer",
			ExpiresIn:    int64(jwtMgr.ttl.Seconds()),
		})
	}
}

// Logout (revoke a refresh token)
func LogoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			RefreshToken string `json:"refresh_token" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx := c.Request.Context()
		h := sha256.Sum256([]byte(body.RefreshToken))
		refreshHash := base64.RawURLEncoding.EncodeToString(h[:])
		if err := db.RevokeRefreshTokenByHash(ctx, refreshHash); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "revoke failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

// Me handler
func MeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.GetString("uid")
		if uid == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}
		user, err := db.GetUserByID(c.Request.Context(), uid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user lookup failed"})
			return
		}
		// hide password hash
		c.JSON(http.StatusOK, gin.H{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
			"role":  user.Role,
		})
	}
}
