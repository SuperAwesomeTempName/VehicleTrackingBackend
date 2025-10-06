package db

import (
	"context"
)

func InsertPosition(ctx context.Context, busId string, ts int64, lat, lon, speed float64, raw map[string]interface{}) error {
	// Use ST_SetSRID(ST_MakePoint(lon, lat),4326)
	_, err := pool.Exec(ctx, `
		INSERT INTO positions (bus_id, route_id, ts, speed_kph, heading, geom, raw)
		VALUES ($1,$2,to_timestamp($3),$4,$5,ST_SetSRID(ST_MakePoint($6,$7),4326),$8)
	`, busId, nil, ts, speed, nil, lon, lat, raw)
	return err
}
