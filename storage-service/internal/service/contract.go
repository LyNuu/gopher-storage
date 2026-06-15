package service

import (
	"context"
	"github.com/google/uuid"
	"storage-service/internal/model"
)

type storageRepository interface {
	GetByID(ctx context.Context, userID uuid.UUID) (model.UserDataDB, error)
	Create(ctx context.Context, storage model.UserDataDB) error
}

type pydioClient interface {
	CreateUserFolder(ctx context.Context, userID uuid.UUID) error
}
