package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	redisclient "github.com/SuperAwesomeTempName/VehicleTrackingBackend/internal/redis"
	db "github.com/SuperAwesomeTempName/VehicleTrackingBackend/internal/db"
	"github.com/redis/go-redis/v9"
)

func main() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" { redisAddr = "redis:6379" }
	dsn := os.Getenv("DATABASE_DSN") // e.g. postgres://transport:transport123@postgres:5432/vehicletracking?sslmode=disable

	// init DB
	ctx := context.Background()
	if err := db.Connect(ctx, dsn); err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer db.Close()

	r := redisclient.New(redisAddr)
	defer r.Close()
	consumerGroup := "workers"
	stream := "stream:positions"
	consumerName := fmt.Sprintf("worker-%d", time.Now().UnixNano())

	// Ensure group exists
	_, err := r.rdb.XGroupCreateMkStream(ctx, stream, consumerGroup, "$").Result()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		log.Printf("xgroup create: %v", err)
	}

	// graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
loop:
	for {
		select {
		case <-sig:
			log.Println("shutting down")
			break loop
		default:
			// read with XREADGROUP
			streams, err := r.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    consumerGroup,
				Consumer: consumerName,
				Streams:  []string{stream, ">"},
				Count:    200,
				Block:    5000 * time.Millisecond,
			}).Result()
			if err != nil && err != redis.Nil {
				log.Printf("xreadgroup error: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}
			// process
			for _, s := range streams {
				for _, msg := range s.Messages {
					if err := processMessage(ctx, r, msg); err != nil {
						log.Printf("process err: %v, msg: %v", err, msg.ID)
						// do not ack; message remains pending for retry/dlq
						continue
					}
					// ack
					if err := r.rdb.XAck(ctx, stream, consumerGroup, msg.ID).Err(); err != nil {
						log.Printf("xack failed: %v", err)
					}
				}
			}
		}
	}
}

func processMessage(ctx context.Context, r *redisclient.Client, msg redis.XMessage) error {
	// Extract fields safely
	busId, ok := msg.Values["busId"].(string)
	if !ok || busId == "" { return fmt.Errorf("invalid busId") }
	lat, _ := parseFloatFromMsg(msg.Values["lat"])
	lon, _ := parseFloatFromMsg(msg.Values["lon"])
	tsInt, _ := parseInt64FromMsg(msg.Values["ts"])
	speed, _ := parseFloatFromMsg(msg.Values["speed"])

	// Insert into Postgres positions table
	if err := db.InsertPosition(ctx, busId, tsInt, lat, lon, speed, msg.Values); err != nil {
		return err
	}
	// publish to pubsub channel for websockets (optional; can use Redis.Publish)
	_ = r.Publish(ctx, "vehicle:"+busId, map[string]interface{}{
		"busId": busId, "lat": lat, "lon": lon, "ts": tsInt,
	})
	return nil
}
