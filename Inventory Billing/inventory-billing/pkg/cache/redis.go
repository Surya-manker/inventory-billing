package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/yourusername/inventory-billing/config"
)

const refreshTokenPrefix = "refresh_token:"

type RefreshTokenPayload struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

type TokenStore struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewRedisClient(cfg *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
}

func NewTokenStore(rdb *redis.Client, refreshExpiryDays int) *TokenStore {
	return &TokenStore{
		rdb: rdb,
		ttl: time.Duration(refreshExpiryDays) * 24 * time.Hour,
	}
}

// SaveRefreshToken generates a random token, persists the payload in Redis, and returns the token.
func (s *TokenStore) SaveRefreshToken(ctx context.Context, payload RefreshTokenPayload) (string, error) {
	token := uuid.New().String()
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	if err := s.rdb.Set(ctx, refreshTokenPrefix+token, data, s.ttl).Err(); err != nil {
		return "", err
	}
	return token, nil
}

func (s *TokenStore) GetRefreshToken(ctx context.Context, token string) (*RefreshTokenPayload, error) {
	data, err := s.rdb.Get(ctx, refreshTokenPrefix+token).Bytes()
	if err != nil {
		return nil, err
	}
	var payload RefreshTokenPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func (s *TokenStore) DeleteRefreshToken(ctx context.Context, token string) error {
	return s.rdb.Del(ctx, refreshTokenPrefix+token).Err()
}
