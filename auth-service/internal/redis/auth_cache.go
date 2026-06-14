package auth_redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrTokenNotFound = errors.New("key not found")
)

type AuthCacheRepository struct {
	client *redis.Client
}

func NewAuthCacheRepository(client *redis.Client) *AuthCacheRepository {
	return &AuthCacheRepository{
		client: client,
	}
}

func (r *AuthCacheRepository) SetRefreshToken(ctx context.Context, refToken string,
	userID string, ttl time.Duration) error {

	key := fmt.Sprintf("refresh_token:%s", refToken)
	err := r.client.Set(ctx, key, userID, ttl).Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *AuthCacheRepository) GetUserIDByRefresh(ctx context.Context, token string) (string, error) {
	key := fmt.Sprintf("refresh_token:%s", token)
	userID, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrTokenNotFound
		}
		return "", err
	}
	return userID, nil
}

func (r *AuthCacheRepository) DelRefreshToken(ctx context.Context, token string) error {
	key := fmt.Sprintf("refresh_token:%s", token)
	return r.client.Del(ctx, key).Err()
}
