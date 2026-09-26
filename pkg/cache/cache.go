package cache

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const ChunkSize = 1000

// Service wraps a Redis client and exposes cache operations used across the application.
type Service struct {
	redis *redis.Client
}

// New constructs a Service from an existing Redis client.
func New(redis *redis.Client) *Service {
	return &Service{redis: redis}
}

// NewClient creates and pings a Redis client.
// Set tls=true for AWS ElastiCache Serverless (TLS is mandatory there).
// For local Redis leave tls=false.
func NewClient(addr, password string, db int, tlsEnabled bool) (*redis.Client, error) {
	opts := &redis.Options{
		Addr:           addr,
		Password:       password,
		DB:             db,
		MaxActiveConns: 50,
		MaxIdleConns:   20,
	}
	if tlsEnabled {
		// ServerName strips the port so tls.Config gets just the hostname.
		// InsecureSkipVerify is false — ElastiCache presents a valid AWS-signed cert.
		host, _, _ := strings.Cut(addr, ":")
		opts.TLSConfig = &tls.Config{
			ServerName: host,
		}
	}
	client := redis.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return client, nil
}

func (c *Service) Delete(key string) error {
	return c.redis.Del(context.Background(), key).Err()
}

func (c *Service) DeleteMultipleKeys(keys ...string) error {
	return c.redis.Del(context.Background(), keys...).Err()
}

// DeleteStartWithKeys deletes all keys whose names begin with the given prefix.
// Uses SCAN to avoid blocking the server on large keyspaces.
func (c *Service) DeleteStartWithKeys(prefix string) {
	if prefix == "" {
		return
	}
	keys, _ := c.GetAllKeys()
	for _, key := range keys {
		if strings.HasPrefix(key, prefix) {
			_ = c.Delete(key)
		}
	}
}

func (c *Service) DeleteAll() error {
	return c.redis.FlushAll(context.Background()).Err()
}

// GetDel atomically reads and deletes a key. Returns ("", redis.Nil) when the key does not exist.
func (c *Service) GetDel(key string) (string, error) {
	return c.redis.GetDel(context.Background(), key).Result()
}

func (c *Service) Get(key string) (string, error) {
	return c.redis.Get(context.Background(), key).Result()
}

// IsExists reports whether the given key exists in Redis.
func (c *Service) IsExists(key string) (bool, error) {
	val, err := c.redis.Exists(context.Background(), key).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
}

func (c *Service) GetAllKeys() ([]string, error) {
	var cursor uint64
	var keys []string
	for {
		ks, next, err := c.redis.Scan(context.Background(), cursor, "", 1000).Result()
		if err != nil {
			return keys, err
		}
		keys = append(keys, ks...)
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}

func (c *Service) SetWithExpireDuration(key string, value any, duration time.Duration) error {
	return c.redis.Set(context.Background(), key, value, duration).Err()
}

// UpdateKeyValue updates the value of an existing key while preserving its TTL.
func (c *Service) UpdateKeyValue(key string, value any) error {
	return c.redis.SetArgs(context.Background(), key, value, redis.SetArgs{
		KeepTTL: true,
	}).Err()
}

func (c *Service) ExtendTTL(key string, penalty time.Duration) error {
	ctx := context.Background()
	ttl, err := c.redis.TTL(ctx, key).Result()
	if err != nil {
		return err
	}
	if ttl <= 0 {
		return errors.New("key does not exist or has no TTL")
	}
	return c.redis.Expire(ctx, key, ttl+penalty).Err()
}

func (c *Service) SetMultipleKeyValues(keyValues map[string]any, bulkSize ...int) error {
	chunkSize := ChunkSize
	if len(bulkSize) > 0 {
		chunkSize = bulkSize[0]
	}
	chunked := make([]interface{}, 0, chunkSize*2)
	i := 0
	for key, value := range keyValues {
		chunked = append(chunked, key, value)
		i++
		if i == chunkSize {
			if err := c.redis.MSet(context.Background(), chunked...).Err(); err != nil {
				return err
			}
			chunked = chunked[:0]
			i = 0
		}
	}
	if i > 0 {
		return c.redis.MSet(context.Background(), chunked...).Err()
	}
	return nil
}
