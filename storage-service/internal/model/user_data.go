package model

import (
	"github.com/google/uuid"
	"time"
)

type UserDataResponse struct {
	ID           uuid.UUID `json:"id"`
	MaxBytesSize int64     `json:"max_bytes_size"`
	UsedBytes    int64     `json:"used_bytes"`
}

type UserDataDB struct {
	ID           uuid.UUID `db:"id"`
	MaxBytesSize int64     `db:"max_bytes_size"`
	UsedBytes    int64     `db:"used_bytes"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}
