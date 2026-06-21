package service

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
	"storage-service/internal/model"
)

type storageRepository interface {
	CreateStorage(ctx context.Context, s model.Storage) error
	GetStorage(ctx context.Context, id uuid.UUID) (model.Storage, error)
	ListStorages(ctx context.Context, userID uuid.UUID) ([]model.Storage, error)
	UpsertFileWithQuota(ctx context.Context, f model.File) error
	GetFile(ctx context.Context, storageID uuid.UUID, name string) (model.File, error)
	GetFileByID(ctx context.Context, id uuid.UUID) (model.File, error)
	ListFiles(ctx context.Context, storageID uuid.UUID) ([]model.File, error)
	DeleteFile(ctx context.Context, storageID uuid.UUID, name string) (model.File, error)
	GrantAccess(ctx context.Context, a model.StorageAccess) error
	RevokeAccess(ctx context.Context, storageID, userID uuid.UUID) error
	GetAccess(ctx context.Context, storageID, userID uuid.UUID) (model.StorageAccess, error)
	ListAccess(ctx context.Context, storageID uuid.UUID) ([]model.StorageAccess, error)
	CreateShare(ctx context.Context, sh model.FileShare) error
	GetShare(ctx context.Context, token string) (model.FileShare, error)
	DeleteShare(ctx context.Context, token string) error
}

type pydioClient interface {
	CreateStorageFolder(ctx context.Context, storageID uuid.UUID) error
	UploadFile(ctx context.Context, storageID uuid.UUID, fileName string, body io.Reader, size int64, contentType string) error
	DownloadFile(ctx context.Context, storageID uuid.UUID, fileName string) (io.ReadCloser, int64, string, error)
	DeleteFile(ctx context.Context, storageID uuid.UUID, fileName string) error
	CreateUploadLink(ctx context.Context, storageID uuid.UUID, fileName string, expires time.Duration) (string, error)
}
