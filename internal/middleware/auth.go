package middleware

import (
	"net/http"
	"strings"

	"github.com/SuperAwesomeTempName/VehicleTrackingBackend/internal/auth"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates JWT access token and attaches uid & role to Gin context
func AuthMiddleware(jwtMgr *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}
		parts := strings.SplitN(h, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}
		token := parts[1]
		claims, err := jwtMgr.ValidateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		// attach to context
		c.Set("uid", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}
