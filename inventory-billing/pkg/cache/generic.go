package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache is a generic Redis cache with JSON serialisation.
// Use it as a cache-aside layer: try Get → on miss, fetch from DB → Set.
type Cache struct {
	rdb *redis.Client
}

func NewCache(rdb *redis.Client) *Cache {
	return &Cache{rdb: rdb}
}

// Get deserialises the cached value into dst. Returns ErrCacheMiss when the key
// does not exist or has expired — callers should treat this as "go fetch from DB".
var ErrCacheMiss = errors.New("cache miss")

func (c *Cache) Get(ctx context.Context, key string, dst any) error {
	data, err := c.rdb.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return ErrCacheMiss
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

// Set serialises value and stores it with the given TTL.
func (c *Cache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, data, ttl).Err()
}

// Del removes one or more keys. Used to invalidate on write operations.
func (c *Cache) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

// GetOrSet is the cache-aside helper: it tries the cache first, and if it
// misses it calls fetch(), stores the result, and returns it.
func GetOrSet[T any](ctx context.Context, c *Cache, key string, ttl time.Duration, fetch func() (T, error)) (T, error) {
	var dst T
	if err := c.Get(ctx, key, &dst); err == nil {
		return dst, nil // cache hit
	}
	val, err := fetch()
	if err != nil {
		return dst, err
	}
	_ = c.Set(ctx, key, val, ttl) // best-effort; a Set failure doesn't break the response
	return val, nil
}
