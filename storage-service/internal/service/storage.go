package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"storage-service/internal/model"
	"storage-service/internal/repository"
)

const defaultMaxStorageSize = int64(1 * 1024000 * 1024000 * 1024000)

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

func (s *StorageService) CreateStorage(ctx context.Context, ownerID uuid.UUID, in model.CreateStorageInput) (model.Storage, error) {
	id := uuid.New()
	if in.Type == model.StorageTypePersonal {
		id = ownerID
	}

	_, err := s.repo.GetStorage(ctx, id)
	if err == nil {
		return model.Storage{}, ErrAlreadyExists
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return model.Storage{}, err
	}

	maxSize := in.MaxSizeBytes
	if maxSize <= 0 {
		maxSize = defaultMaxStorageSize
	}
	allowed := in.AllowedMimeTypes
	if allowed == nil {
		allowed = []string{}
	}

	if err := s.pydio.CreateStorageFolder(ctx, id); err != nil {
		return model.Storage{}, err
	}

	st := model.Storage{
		ID:               id,
		OwnerID:          ownerID,
		Name:             in.Name,
		Type:             in.Type,
		MaxSizeBytes:     maxSize,
		UsedBytes:        0,
		MaxFileSizeBytes: in.MaxFileSizeBytes,
		AllowedMimeTypes: allowed,
	}
	if err := s.repo.CreateStorage(ctx, st); err != nil {
		return model.Storage{}, err
	}
	return st, nil
}

func (s *StorageService) ListStorages(ctx context.Context, userID uuid.UUID) ([]model.Storage, error) {
	return s.repo.ListStorages(ctx, userID)
}

func (s *StorageService) GetStorage(ctx context.Context, userID, storageID uuid.UUID) (model.Storage, error) {
	return s.storageWithAccess(ctx, userID, storageID, model.AccessRead)
}

func (s *StorageService) UploadFile(ctx context.Context, userID, storageID uuid.UUID, name string, body io.Reader, size int64, contentType string) (model.File, error) {
	st, err := s.storageWithAccess(ctx, userID, storageID, model.AccessWrite)
	if err != nil {
		return model.File{}, err
	}
	if err := validateFileName(name); err != nil {
		return model.File{}, err
	}
	if st.MaxFileSizeBytes > 0 && size > st.MaxFileSizeBytes {
		return model.File{}, ErrFileTooLarge
	}
	if !mimeAllowed(st.AllowedMimeTypes, contentType) {
		return model.File{}, ErrMimeNotAllowed
	}

	if err := s.pydio.UploadFile(ctx, storageID, name, body, size, contentType); err != nil {
		return model.File{}, err
	}

	f := model.File{
		ID:          uuid.New(),
		StorageID:   storageID,
		Name:        name,
		SizeBytes:   size,
		ContentType: contentType,
		UploadedBy:  userID,
	}
	if err := s.repo.UpsertFileWithQuota(ctx, f); err != nil {
		if errors.Is(err, repository.ErrQuotaExceeded) {
			_ = s.pydio.DeleteFile(ctx, storageID, name)
			return model.File{}, ErrQuotaExceeded
		}
		return model.File{}, err
	}
	return f, nil
}

func (s *StorageService) DownloadFile(ctx context.Context, userID, storageID uuid.UUID, name string) (io.ReadCloser, int64, string, error) {
	if _, err := s.storageWithAccess(ctx, userID, storageID, model.AccessRead); err != nil {
		return nil, 0, "", err
	}
	if _, err := s.repo.GetFile(ctx, storageID, name); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, "", ErrNotFound
		}
		return nil, 0, "", err
	}
	return s.pydio.DownloadFile(ctx, storageID, name)
}

func (s *StorageService) ListFiles(ctx context.Context, userID, storageID uuid.UUID) ([]model.File, error) {
	if _, err := s.storageWithAccess(ctx, userID, storageID, model.AccessRead); err != nil {
		return nil, err
	}
	return s.repo.ListFiles(ctx, storageID)
}

func (s *StorageService) DeleteFile(ctx context.Context, userID, storageID uuid.UUID, name string) error {
	if _, err := s.storageWithAccess(ctx, userID, storageID, model.AccessWrite); err != nil {
		return err
	}
	if _, err := s.repo.GetFile(ctx, storageID, name); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if err := s.pydio.DeleteFile(ctx, storageID, name); err != nil {
		return err
	}
	if _, err := s.repo.DeleteFile(ctx, storageID, name); err != nil {
		return err
	}
	return nil
}

func (s *StorageService) GrantAccess(ctx context.Context, ownerID, storageID, targetUserID uuid.UUID, level model.AccessLevel) error {
	st, err := s.repo.GetStorage(ctx, storageID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if st.OwnerID != ownerID {
		return ErrForbidden
	}
	if targetUserID == ownerID {
		return nil
	}
	return s.repo.GrantAccess(ctx, model.StorageAccess{
		StorageID: storageID,
		UserID:    targetUserID,
		Level:     level,
		GrantedBy: ownerID,
	})
}

func (s *StorageService) RevokeAccess(ctx context.Context, ownerID, storageID, targetUserID uuid.UUID) error {
	st, err := s.repo.GetStorage(ctx, storageID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if st.OwnerID != ownerID {
		return ErrForbidden
	}
	return s.repo.RevokeAccess(ctx, storageID, targetUserID)
}

func (s *StorageService) ListAccess(ctx context.Context, ownerID, storageID uuid.UUID) ([]model.StorageAccess, error) {
	st, err := s.repo.GetStorage(ctx, storageID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if st.OwnerID != ownerID {
		return nil, ErrForbidden
	}
	return s.repo.ListAccess(ctx, storageID)
}

func (s *StorageService) ShareFile(ctx context.Context, userID, storageID uuid.UUID, name string, ttl time.Duration) (model.FileShare, error) {
	if _, err := s.storageWithAccess(ctx, userID, storageID, model.AccessWrite); err != nil {
		return model.FileShare{}, err
	}
	f, err := s.repo.GetFile(ctx, storageID, name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.FileShare{}, ErrNotFound
		}
		return model.FileShare{}, err
	}
	token, err := newShareToken()
	if err != nil {
		return model.FileShare{}, err
	}
	sh := model.FileShare{
		Token:     token,
		FileID:    f.ID,
		CreatedBy: userID,
		ExpiresAt: time.Now().Add(ttl),
	}
	if err := s.repo.CreateShare(ctx, sh); err != nil {
		return model.FileShare{}, err
	}
	return sh, nil
}

func (s *StorageService) GetSharedFile(ctx context.Context, token string) (io.ReadCloser, int64, string, string, error) {
	sh, err := s.repo.GetShare(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, "", "", ErrNotFound
		}
		return nil, 0, "", "", err
	}
	if time.Now().After(sh.ExpiresAt) {
		_ = s.repo.DeleteShare(ctx, token)
		return nil, 0, "", "", ErrShareExpired
	}
	f, err := s.repo.GetFileByID(ctx, sh.FileID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, "", "", ErrNotFound
		}
		return nil, 0, "", "", err
	}
	stream, size, contentType, err := s.pydio.DownloadFile(ctx, f.StorageID, f.Name)
	if err != nil {
		return nil, 0, "", "", err
	}
	return stream, size, contentType, f.Name, nil
}

func (s *StorageService) RevokeShare(ctx context.Context, userID uuid.UUID, token string) error {
	sh, err := s.repo.GetShare(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	f, err := s.repo.GetFileByID(ctx, sh.FileID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	st, err := s.repo.GetStorage(ctx, f.StorageID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if sh.CreatedBy != userID && st.OwnerID != userID {
		return ErrForbidden
	}
	return s.repo.DeleteShare(ctx, token)
}

func (s *StorageService) CreateUploadLink(ctx context.Context, userID, storageID uuid.UUID, name string, expires time.Duration) (string, error) {
	if _, err := s.storageWithAccess(ctx, userID, storageID, model.AccessWrite); err != nil {
		return "", err
	}
	if err := validateFileName(name); err != nil {
		return "", err
	}
	return s.pydio.CreateUploadLink(ctx, storageID, name, expires)
}

func (s *StorageService) storageWithAccess(ctx context.Context, userID, storageID uuid.UUID, need model.AccessLevel) (model.Storage, error) {
	st, err := s.repo.GetStorage(ctx, storageID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Storage{}, ErrNotFound
		}
		return model.Storage{}, err
	}
	if st.OwnerID == userID {
		return st, nil
	}
	if st.Type == model.StorageTypeGlobal && need == model.AccessRead {
		return st, nil
	}
	acc, err := s.repo.GetAccess(ctx, storageID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Storage{}, ErrForbidden
		}
		return model.Storage{}, err
	}
	if need == model.AccessWrite && acc.Level != model.AccessWrite {
		return model.Storage{}, ErrForbidden
	}
	return st, nil
}

func validateFileName(name string) error {
	if name == "" || len(name) > 255 {
		return ErrInvalidFileName
	}
	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		return ErrInvalidFileName
	}
	for _, r := range name {
		if r < 0x20 {
			return ErrInvalidFileName
		}
	}
	return nil
}

func mimeAllowed(allowed []string, contentType string) bool {
	if len(allowed) == 0 {
		return true
	}
	ct := contentType
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	ct = strings.TrimSpace(strings.ToLower(ct))
	for _, a := range allowed {
		if strings.ToLower(strings.TrimSpace(a)) == ct {
			return true
		}
	}
	return false
}

func newShareToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
