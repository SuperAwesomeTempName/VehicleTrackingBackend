package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Health returns the health status of the application
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "VehicleTrackingBackend",
	})
}

// Ready returns the readiness status of the application
func (h *HealthHandler) Ready(c *gin.Context) {
	// Add any readiness checks here (database connectivity, etc.)
	c.JSON(http.StatusOK, gin.H{
		"status":  "ready",
		"service": "VehicleTrackingBackend",
	})
}
