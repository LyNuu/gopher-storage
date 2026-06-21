package http

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
	"storage-service/internal/model"
)

type storageService interface {
	CreateStorage(ctx context.Context, ownerID uuid.UUID, in model.CreateStorageInput) (model.Storage, error)
	ListStorages(ctx context.Context, userID uuid.UUID) ([]model.Storage, error)
	GetStorage(ctx context.Context, userID, storageID uuid.UUID) (model.Storage, error)
	UploadFile(ctx context.Context, userID, storageID uuid.UUID, name string, body io.Reader, size int64, contentType string) (model.File, error)
	DownloadFile(ctx context.Context, userID, storageID uuid.UUID, name string) (io.ReadCloser, int64, string, error)
	ListFiles(ctx context.Context, userID, storageID uuid.UUID) ([]model.File, error)
	DeleteFile(ctx context.Context, userID, storageID uuid.UUID, name string) error
	GrantAccess(ctx context.Context, ownerID, storageID, targetUserID uuid.UUID, level model.AccessLevel) error
	RevokeAccess(ctx context.Context, ownerID, storageID, targetUserID uuid.UUID) error
	ListAccess(ctx context.Context, ownerID, storageID uuid.UUID) ([]model.StorageAccess, error)
	ShareFile(ctx context.Context, userID, storageID uuid.UUID, name string, ttl time.Duration) (model.FileShare, error)
	GetSharedFile(ctx context.Context, token string) (io.ReadCloser, int64, string, string, error)
	RevokeShare(ctx context.Context, userID uuid.UUID, token string) error
	CreateUploadLink(ctx context.Context, userID, storageID uuid.UUID, name string, expires time.Duration) (string, error)
}
