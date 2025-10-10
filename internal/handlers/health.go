package handlers

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	ServiceName string
}

func NewHealthHandler(serviceName string) *HealthHandler {
	return &HealthHandler{ServiceName: serviceName}
}

func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": h.ServiceName})
}

func (h *HealthHandler) Ready(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ready", "service": h.ServiceName})
}
