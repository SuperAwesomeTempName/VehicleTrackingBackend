package handlers

import (
	"context"
	"net/http"
	"time"

	redisclient "github.com/SuperAwesomeTempName/VehicleTrackingBackend/internal/redis"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type LocationRequest struct {
	BusID     string  `json:"busId" validate:"required,uuid"`
	Latitude  float64 `json:"latitude" validate:"required,gt=-90,lt=90"`
	Longitude float64 `json:"longitude" validate:"required,gt=-180,lt=180"`
	Timestamp int64   `json:"timestamp" validate:"required"`
	SpeedKph  float64 `json:"speedKph"`
	Heading   float64 `json:"heading"`
}

func PostLocationHandler(rdb *redisclient.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req LocationRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
		}
		if err := c.Validate(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		// Basic sanity check on timestamp (not > 5 minutes in future)
		t := time.Unix(req.Timestamp, 0)
		if t.After(time.Now().Add(5 * time.Minute)) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "timestamp invalid"})
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
		if _, err := rdb.XAdd(ctx, "stream:positions", values); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "ingest failed"})
		}
		// Update Redis GEO and last known
		if err := rdb.GeoAdd(ctx, "live:vehicles", req.Longitude, req.Latitude, req.BusID); err == nil {
			_ = rdb.HSet(ctx, "vehicle:"+req.BusID+":last", map[string]interface{}{
				"lat": req.Latitude, "lon": req.Longitude, "ts": req.Timestamp, "speed": req.SpeedKph,
			})
		}
		// Publish light event for websocket subscribers
		_ = rdb.Publish(ctx, "vehicle:"+req.BusID, map[string]interface{}{
			"msgId": msgId, "busId": req.BusID, "lat": req.Latitude, "lon": req.Longitude, "ts": req.Timestamp,
		})

		return c.NoContent(http.StatusOK)
	}
}
