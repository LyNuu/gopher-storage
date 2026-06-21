package model

import (
	"time"

	"github.com/google/uuid"
)

type StorageType string

const (
	StorageTypePersonal StorageType = "personal"
	StorageTypeGlobal   StorageType = "global"
)

type AccessLevel string

const (
	AccessRead  AccessLevel = "read"
	AccessWrite AccessLevel = "write"
)

type Storage struct {
	ID               uuid.UUID   `db:"id" json:"id"`
	OwnerID          uuid.UUID   `db:"owner_id" json:"owner_id"`
	Name             string      `db:"name" json:"name"`
	Type             StorageType `db:"type" json:"type"`
	MaxSizeBytes     int64       `db:"max_size_bytes" json:"max_size_bytes"`
	UsedBytes        int64       `db:"used_bytes" json:"used_bytes"`
	MaxFileSizeBytes int64       `db:"max_file_size_bytes" json:"max_file_size_bytes"`
	AllowedMimeTypes []string    `db:"allowed_mime_types" json:"allowed_mime_types"`
	CreatedAt        time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time   `db:"updated_at" json:"updated_at"`
}

type StorageAccess struct {
	StorageID uuid.UUID   `db:"storage_id" json:"storage_id"`
	UserID    uuid.UUID   `db:"user_id" json:"user_id"`
	Level     AccessLevel `db:"level" json:"level"`
	GrantedBy uuid.UUID   `db:"granted_by" json:"granted_by"`
	CreatedAt time.Time   `db:"created_at" json:"created_at"`
}

type File struct {
	ID          uuid.UUID `db:"id" json:"id"`
	StorageID   uuid.UUID `db:"storage_id" json:"storage_id"`
	Name        string    `db:"name" json:"name"`
	SizeBytes   int64     `db:"size_bytes" json:"size_bytes"`
	ContentType string    `db:"content_type" json:"content_type"`
	UploadedBy  uuid.UUID `db:"uploaded_by" json:"uploaded_by"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type FileShare struct {
	Token     string    `db:"token" json:"token"`
	FileID    uuid.UUID `db:"file_id" json:"file_id"`
	CreatedBy uuid.UUID `db:"created_by" json:"created_by"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type CreateStorageInput struct {
	Name             string      `json:"name" validate:"required,max=255"`
	Type             StorageType `json:"type" validate:"required,oneof=personal global"`
	MaxSizeBytes     int64       `json:"max_size_bytes" validate:"gte=0"`
	MaxFileSizeBytes int64       `json:"max_file_size_bytes" validate:"gte=0"`
	AllowedMimeTypes []string    `json:"allowed_mime_types"`
}
