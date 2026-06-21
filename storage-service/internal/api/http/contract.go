package http

import (
	"context"
	"github.com/google/uuid"
	"io"
	"time"
)

type storageService interface {
	CreateStorage(ctx context.Context, userID uuid.UUID) error
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
