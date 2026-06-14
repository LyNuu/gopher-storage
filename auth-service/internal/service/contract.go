package service

import (
	"context"
	"time"

	"auth-service/internal/model"
)

type authRepository interface {
	Create(ctx context.Context, user model.UserDB) error
	GetByEmail(ctx context.Context, email string) (*model.UserDB, error)
}

type cacheRepository interface {
	SetRefreshToken(ctx context.Context, token string, userID string, ttl time.Duration) error
	GetUserIDByRefresh(ctx context.Context, token string) (string, error)
	DelRefreshToken(ctx context.Context, token string) error
}
