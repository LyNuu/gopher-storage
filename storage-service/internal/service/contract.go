package service

import (
	"context"
	"github.com/google/uuid"
	"io"
	"storage-service/internal/model"
	"time"
)

type storageRepository interface {
	GetByID(ctx context.Context, userID uuid.UUID) (model.UserDataDB, error)
	Create(ctx context.Context, storage model.UserDataDB) error
}

type pydioClient interface {
	CreateUserFolder(ctx context.Context, userID uuid.UUID) error
	UploadUserFile(ctx context.Context, userID uuid.UUID, fileName string, fileStream io.Reader, fileSize int64, contentType string) error
	DownloadUserFile(ctx context.Context, userID uuid.UUID, fileName string) (io.ReadCloser, int64, string, error)
	DeleteUserFile(ctx context.Context, userID uuid.UUID, fileName string) error
	GetShareableLink(ctx context.Context, userID uuid.UUID, fileName string, duration time.Duration) (string, error)
	CreateUploadLink(
		ctx context.Context,
		userID uuid.UUID,
		fileName string,
		expires time.Duration,
	) (string, error)
}
