package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiterMiddleware creates a simple fixed-window rate limiter using Redis.
// It increments a per-client key and rejects requests exceeding the provided limit
// within the given window duration.
func RateLimiterMiddleware(rdb *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()

		clientID := c.ClientIP()
		if busID := c.GetHeader("X-Bus-ID"); busID != "" {
			clientID = busID
		}
		if clientID == "" {
			clientID = "unknown"
		}

		// Use a windowed key bucket per client
		bucket := time.Now().Unix() / int64(window.Seconds())
		key := fmt.Sprintf("ratelimit:%s:%d", clientID, bucket)

		// Increment request count atomically
		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "rate limiter unavailable"})
			return
		}
		if count == 1 {
			_ = rdb.Expire(ctx, key, window).Err()
		}

		if count > int64(limit) {
			c.Header("Retry-After", fmt.Sprintf("%d", int(window.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":  "rate limit exceeded",
				"limit":  limit,
				"window": window.Seconds(),
			})
			return
		}

		c.Next()
	}
}
