package service

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"storage-service/internal/model"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)

type StorageService struct {
	repo  storageRepository
	pydio pydioClient
}

func NewStorageService(repo storageRepository, pydio pydioClient) *StorageService {
	return &StorageService{
		repo:  repo,
		pydio: pydio,
	}
}

func (s *StorageService) CreateStorage(ctx context.Context, userID uuid.UUID) error {
	_, err := s.repo.GetByID(ctx, userID)

	if err == nil {
		return ErrAlreadyExists
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	err = s.pydio.CreateUserFolder(ctx, userID)
	if err != nil {
		return err
	}

	defaultMaxSite := int64(1 * 1024 * 1024 * 1024)

	newStorage := model.UserDataDB{
		ID:           userID,
		MaxBytesSize: defaultMaxSite,
		UsedBytes:    0,
	}

	return s.repo.Create(ctx, newStorage)
}
