package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
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

func (r *StorageRepository) GetByID(ctx context.Context, userID uuid.UUID) (model.UserDataDB, error) {
	query := `
		SELECT id, max_size_bytes, used_bytes, created_at, updated_at 
		FROM user_storage 
		WHERE id = $1
	`

	var res model.UserDataDB

	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&res.ID,
		&res.MaxBytesSize,
		&res.UsedBytes,
		&res.CreatedAt,
		&res.UpdatedAt,
	)

	if err != nil {
		return model.UserDataDB{}, err
	}
	return res, nil
}

func (r *StorageRepository) Create(ctx context.Context, storage model.UserDataDB) error {
	query := `
		INSERT INTO user_storage (id, max_size_bytes, used_bytes, created_at, updated_at) 
		VALUES ($1, $2, $3)
	`

	_, err := r.pool.Exec(ctx, query,
		storage.ID,
		storage.MaxBytesSize,
		storage.UsedBytes,
	)
	if err != nil {
		return fmt.Errorf("failed to insert storage info to db: %w", err)
	}
	return nil
}
