package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *redis.Client
}

func New(addr string) *Client {
	opts := &redis.Options{
		Addr:         addr,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		DialTimeout:  3 * time.Second,
	}
	rdb := redis.NewClient(opts)
	return &Client{rdb}
}

func (c *Client) Close() error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Close()
}

func (c *Client) XAdd(ctx context.Context, stream string, values map[string]interface{}) (string, error) {
	xargs := &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}
	return c.rdb.XAdd(ctx, xargs).Result()
}

func (c *Client) GeoAdd(ctx context.Context, key string, lon, lat float64, member string) error {
	_, err := c.rdb.GeoAdd(ctx, key, &redis.GeoLocation{
		Longitude: lon,
		Latitude:  lat,
		Name:      member,
	}).Result()
	return err
}

func (c *Client) HSet(ctx context.Context, key string, values map[string]interface{}) error {
	return c.rdb.HSet(ctx, key, values).Err()
}

func (c *Client) Publish(ctx context.Context, channel string, msg interface{}) error {
	return c.rdb.Publish(ctx, channel, msg).Err()
}
