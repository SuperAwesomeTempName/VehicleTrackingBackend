package handlers

import (
	"context"
	"net/http"
	"time"

	"encoding/json"

	redisclient "github.com/SuperAwesomeTempName/VehicleTrackingBackend/internal/redis"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GLocationRequest struct {
	BusID     string  `json:"busId"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timestamp int64   `json:"timestamp"`
	SpeedKph  float64 `json:"speedKph"`
	Heading   float64 `json:"heading"`
}

type LocationsGinHandler struct {
	redis *redisclient.Client
}

func NewLocationsGinHandler(r *redisclient.Client) *LocationsGinHandler {
	return &LocationsGinHandler{redis: r}
}

func (h *LocationsGinHandler) Post(c *gin.Context) {
	var req GLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if req.BusID == "" || req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 || req.Timestamp == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fields"})
		return
	}
	t := time.Unix(req.Timestamp, 0)
	if t.After(time.Now().Add(5 * time.Minute)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "timestamp invalid"})
		return
	}

	msgId := uuid.New().String()
	values := map[string]interface{}{
		"msgId":   msgId,
		"busId":   req.BusID,
		"lat":     req.Latitude,
		"lon":     req.Longitude,
		"ts":      req.Timestamp,
		"speed":   req.SpeedKph,
		"heading": req.Heading,
	}
	ctx := context.Background()
	if _, err := h.redis.XAdd(ctx, "stream:positions", values); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ingest failed"})
		return
	}
	if err := h.redis.GeoAdd(ctx, "live:vehicles", req.Longitude, req.Latitude, req.BusID); err == nil {
		_ = h.redis.HSet(ctx, "vehicle:"+req.BusID+":last", map[string]interface{}{
			"lat": req.Latitude, "lon": req.Longitude, "ts": req.Timestamp, "speed": req.SpeedKph,
		})
	}
	event := map[string]interface{}{
		"msgId":   msgId,
		"busId":   req.BusID,
		"lat":     req.Latitude,
		"lon":     req.Longitude,
		"ts":      req.Timestamp,
		"speed":   req.SpeedKph,
		"heading": req.Heading,
	}
	if b, err := json.Marshal(event); err == nil {
		_ = h.redis.Publish(ctx, "vehicle:"+req.BusID, string(b))
	}

	c.Status(http.StatusNoContent)
}
