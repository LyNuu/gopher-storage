package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"storage-service/internal/model"
)

type StorageRepository struct {
	pool *pgxpool.Pool
}

func NewStorageRepository(pool *pgxpool.Pool) *StorageRepository {
	return &StorageRepository{
		pool: pool,
	}
}

func (r *StorageRepository) CreateStorage(ctx context.Context, s model.Storage) error {
	query := `
		insert into storages (id, owner_id, name, type, max_size_bytes, used_bytes, max_file_size_bytes, allowed_mime_types)
		values ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, query,
		s.ID, s.OwnerID, s.Name, s.Type, s.MaxSizeBytes, s.UsedBytes, s.MaxFileSizeBytes, s.AllowedMimeTypes,
	)
	if err != nil {
		return fmt.Errorf("failed to insert storage: %w", err)
	}
	return nil
}

func (r *StorageRepository) GetStorage(ctx context.Context, id uuid.UUID) (model.Storage, error) {
	query := `
		select id, owner_id, name, type, max_size_bytes, used_bytes, max_file_size_bytes, allowed_mime_types, created_at, updated_at
		from storages
		where id = $1
	`
	var s model.Storage
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.OwnerID, &s.Name, &s.Type, &s.MaxSizeBytes, &s.UsedBytes, &s.MaxFileSizeBytes, &s.AllowedMimeTypes, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return model.Storage{}, err
	}
	return s, nil
}

func (r *StorageRepository) ListStorages(ctx context.Context, userID uuid.UUID) ([]model.Storage, error) {
	query := `
		select s.id, s.owner_id, s.name, s.type, s.max_size_bytes, s.used_bytes, s.max_file_size_bytes, s.allowed_mime_types, s.created_at, s.updated_at
		from storages s
		where s.owner_id = $1
		   or s.type = 'global'
		   or exists (select 1 from storage_access a where a.storage_id = s.id and a.user_id = $1)
		order by s.created_at
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []model.Storage
	for rows.Next() {
		var s model.Storage
		if err := rows.Scan(
			&s.ID, &s.OwnerID, &s.Name, &s.Type, &s.MaxSizeBytes, &s.UsedBytes, &s.MaxFileSizeBytes, &s.AllowedMimeTypes, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		res = append(res, s)
	}
	return res, rows.Err()
}

func (r *StorageRepository) UpsertFileWithQuota(ctx context.Context, f model.File) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var maxSize, usedBytes int64
	err = tx.QueryRow(ctx, `select max_size_bytes, used_bytes from storages where id = $1 for update`, f.StorageID).
		Scan(&maxSize, &usedBytes)
	if err != nil {
		return err
	}

	var oldSize int64
	err = tx.QueryRow(ctx, `select size_bytes from files where storage_id = $1 and name = $2`, f.StorageID, f.Name).
		Scan(&oldSize)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	delta := f.SizeBytes - oldSize
	if usedBytes+delta > maxSize {
		return ErrQuotaExceeded
	}

	_, err = tx.Exec(ctx, `
		insert into files (id, storage_id, name, size_bytes, content_type, uploaded_by)
		values ($1, $2, $3, $4, $5, $6)
		on conflict (storage_id, name) do update
		set size_bytes = excluded.size_bytes,
		    content_type = excluded.content_type,
		    uploaded_by = excluded.uploaded_by,
		    updated_at = now()
	`, f.ID, f.StorageID, f.Name, f.SizeBytes, f.ContentType, f.UploadedBy)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `update storages set used_bytes = used_bytes + $1, updated_at = now() where id = $2`,
		delta, f.StorageID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *StorageRepository) GetFile(ctx context.Context, storageID uuid.UUID, name string) (model.File, error) {
	query := `
		select id, storage_id, name, size_bytes, content_type, uploaded_by, created_at, updated_at
		from files
		where storage_id = $1 and name = $2
	`
	var f model.File
	err := r.pool.QueryRow(ctx, query, storageID, name).Scan(
		&f.ID, &f.StorageID, &f.Name, &f.SizeBytes, &f.ContentType, &f.UploadedBy, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return model.File{}, err
	}
	return f, nil
}

func (r *StorageRepository) GetFileByID(ctx context.Context, id uuid.UUID) (model.File, error) {
	query := `
		select id, storage_id, name, size_bytes, content_type, uploaded_by, created_at, updated_at
		from files
		where id = $1
	`
	var f model.File
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&f.ID, &f.StorageID, &f.Name, &f.SizeBytes, &f.ContentType, &f.UploadedBy, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return model.File{}, err
	}
	return f, nil
}

func (r *StorageRepository) ListFiles(ctx context.Context, storageID uuid.UUID) ([]model.File, error) {
	query := `
		select id, storage_id, name, size_bytes, content_type, uploaded_by, created_at, updated_at
		from files
		where storage_id = $1
		order by name
	`
	rows, err := r.pool.Query(ctx, query, storageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []model.File
	for rows.Next() {
		var f model.File
		if err := rows.Scan(
			&f.ID, &f.StorageID, &f.Name, &f.SizeBytes, &f.ContentType, &f.UploadedBy, &f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			return nil, err
		}
		res = append(res, f)
	}
	return res, rows.Err()
}

func (r *StorageRepository) DeleteFile(ctx context.Context, storageID uuid.UUID, name string) (model.File, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.File{}, err
	}
	defer tx.Rollback(ctx)

	var f model.File
	err = tx.QueryRow(ctx, `
		delete from files
		where storage_id = $1 and name = $2
		returning id, storage_id, name, size_bytes, content_type, uploaded_by, created_at, updated_at
	`, storageID, name).Scan(
		&f.ID, &f.StorageID, &f.Name, &f.SizeBytes, &f.ContentType, &f.UploadedBy, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return model.File{}, err
	}

	_, err = tx.Exec(ctx, `update storages set used_bytes = greatest(used_bytes - $1, 0), updated_at = now() where id = $2`,
		f.SizeBytes, storageID)
	if err != nil {
		return model.File{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.File{}, err
	}
	return f, nil
}

func (r *StorageRepository) GrantAccess(ctx context.Context, a model.StorageAccess) error {
	query := `
		insert into storage_access (storage_id, user_id, level, granted_by)
		values ($1, $2, $3, $4)
		on conflict (storage_id, user_id) do update
		set level = excluded.level, granted_by = excluded.granted_by
	`
	_, err := r.pool.Exec(ctx, query, a.StorageID, a.UserID, a.Level, a.GrantedBy)
	if err != nil {
		return fmt.Errorf("failed to grant access: %w", err)
	}
	return nil
}

func (r *StorageRepository) RevokeAccess(ctx context.Context, storageID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `delete from storage_access where storage_id = $1 and user_id = $2`, storageID, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke access: %w", err)
	}
	return nil
}

func (r *StorageRepository) GetAccess(ctx context.Context, storageID, userID uuid.UUID) (model.StorageAccess, error) {
	query := `
		select storage_id, user_id, level, granted_by, created_at
		from storage_access
		where storage_id = $1 and user_id = $2
	`
	var a model.StorageAccess
	err := r.pool.QueryRow(ctx, query, storageID, userID).Scan(&a.StorageID, &a.UserID, &a.Level, &a.GrantedBy, &a.CreatedAt)
	if err != nil {
		return model.StorageAccess{}, err
	}
	return a, nil
}

func (r *StorageRepository) ListAccess(ctx context.Context, storageID uuid.UUID) ([]model.StorageAccess, error) {
	query := `
		select storage_id, user_id, level, granted_by, created_at
		from storage_access
		where storage_id = $1
		order by created_at
	`
	rows, err := r.pool.Query(ctx, query, storageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []model.StorageAccess
	for rows.Next() {
		var a model.StorageAccess
		if err := rows.Scan(&a.StorageID, &a.UserID, &a.Level, &a.GrantedBy, &a.CreatedAt); err != nil {
			return nil, err
		}
		res = append(res, a)
	}
	return res, rows.Err()
}

func (r *StorageRepository) CreateShare(ctx context.Context, sh model.FileShare) error {
	query := `
		insert into file_shares (token, file_id, created_by, expires_at)
		values ($1, $2, $3, $4)
	`
	_, err := r.pool.Exec(ctx, query, sh.Token, sh.FileID, sh.CreatedBy, sh.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to create share: %w", err)
	}
	return nil
}

func (r *StorageRepository) GetShare(ctx context.Context, token string) (model.FileShare, error) {
	query := `
		select token, file_id, created_by, expires_at, created_at
		from file_shares
		where token = $1
	`
	var sh model.FileShare
	err := r.pool.QueryRow(ctx, query, token).Scan(&sh.Token, &sh.FileID, &sh.CreatedBy, &sh.ExpiresAt, &sh.CreatedAt)
	if err != nil {
		return model.FileShare{}, err
	}
	return sh, nil
}

func (r *StorageRepository) DeleteShare(ctx context.Context, token string) error {
	_, err := r.pool.Exec(ctx, `delete from file_shares where token = $1`, token)
	if err != nil {
		return fmt.Errorf("failed to delete share: %w", err)
	}
	return nil
}
