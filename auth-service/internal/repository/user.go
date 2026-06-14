package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"auth-service/internal/model"
)

type AuthRepository struct {
	pool *pgxpool.Pool
}

func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{
		pool: pool,
	}
}

func (r *AuthRepository) Create(ctx context.Context, user model.UserDB) error {
	_, err := r.pool.Exec(ctx, `
		insert into users (id, email, password_hash, role)
		values ($1, $2, $3, $4)
		`, user.ID, user.Email, user.PasswordHash, user.Role)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return fmt.Errorf("email already exist %w", ErrEmailAlreadyExists)
		}
		return err
	}
	return nil
}

func (r *AuthRepository) GetByEmail(ctx context.Context, email string) (*model.UserDB, error) {
	var user model.UserDB

	err := r.pool.QueryRow(ctx, `
	select id, email, password_hash, role
	from users
	where email = $1
	`, email).Scan(&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}
