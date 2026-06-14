package http

import (
	"context"
	"github.com/google/uuid"
)

type storageService interface {
	CreateStorage(ctx context.Context, userID uuid.UUID) error
}
