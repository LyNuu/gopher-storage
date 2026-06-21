package service

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"io"
	"storage-service/internal/model"
	"time"
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

func (s *StorageService) UploadUserFile(ctx context.Context, userID uuid.UUID, fileName string, fileStream io.Reader, fileSize int64, contentType string) error {
	err := s.pydio.UploadUserFile(ctx, userID, fileName, fileStream, fileSize, contentType)
	if err != nil {
		return err
	}
	return nil
}

func (s *StorageService) DownloadUserFile(ctx context.Context, userID uuid.UUID, fileName string) (io.ReadCloser, int64, string, error) {
	p, n, st, err := s.pydio.DownloadUserFile(ctx, userID, fileName)
	if err != nil {
		return nil, 0, "", err
	}
	return p, n, st, nil
}

func (s *StorageService) DeleteUserFile(ctx context.Context, userID uuid.UUID, fileName string) error {
	err := s.pydio.DeleteUserFile(ctx, userID, fileName)
	if err != nil {
		return err
	}
	return nil
}

func (s *StorageService) GetShareableLink(ctx context.Context, userID uuid.UUID, fileName string, duration time.Duration) (string, error) {
	str, err := s.pydio.GetShareableLink(ctx, userID, fileName, duration)
	if err != nil {
		return "", err
	}
	return str, nil
}

func (s *StorageService) CreateUploadLink(ctx context.Context, userID uuid.UUID, fileName string, expires time.Duration) (string, error) {
	str, err := s.pydio.CreateUploadLink(ctx, userID, fileName, expires)
	if err != nil {
		return "", err
	}
	return str, nil
}
