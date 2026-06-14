package http

import (
	"context"

	"auth-service/internal/model"
	"auth-service/internal/service"
)

type authService interface {
	Create(ctx context.Context, user model.User) error
	Login(ctx context.Context, email, password string) (*service.TokenPair, error)
	Refresh(ctx context.Context, oldRefreshToken string) (*service.TokenPair, error)
}
