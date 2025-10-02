package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type APIHandler struct {
	logger *zap.Logger
}

func NewAPIHandler(logger *zap.Logger) *APIHandler {
	return &APIHandler{
		logger: logger,
	}
}

// Ping responds with pong
func (h *APIHandler) Ping(c *gin.Context) {
	h.logger.Info("Ping endpoint called")
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

// Version returns the API version
func (h *APIHandler) Version(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": "1.0.0",
		"service": "VehicleTrackingBackend",
	})
}
