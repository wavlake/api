package services

import (
	"context"
	"io"
	"time"

	"github.com/wavlake/api/internal/models"
)

// UserServiceInterface defines the interface for user operations
type UserServiceInterface interface {
	LinkPubkeyToUser(ctx context.Context, pubkey, firebaseUID string) error
	UnlinkPubkeyFromUser(ctx context.Context, pubkey, firebaseUID string) error
	GetLinkedPubkeys(ctx context.Context, firebaseUID string) ([]models.NostrAuth, error)
	GetFirebaseUIDByPubkey(ctx context.Context, pubkey string) (string, error)
}

// PostgresServiceInterface defines the interface for PostgreSQL operations
type PostgresServiceInterface interface {
	GetUserByFirebaseUID(ctx context.Context, firebaseUID string) (*models.LegacyUser, error)
	GetUserTracks(ctx context.Context, firebaseUID string) ([]models.LegacyTrack, error)
	GetUserArtists(ctx context.Context, firebaseUID string) ([]models.LegacyArtist, error)
	GetUserAlbums(ctx context.Context, firebaseUID string) ([]models.LegacyAlbum, error)
	GetTracksByArtist(ctx context.Context, artistID string) ([]models.LegacyTrack, error)
	GetTracksByAlbum(ctx context.Context, albumID string) ([]models.LegacyTrack, error)
}

// StorageServiceInterface defines the interface for storage operations
type StorageServiceInterface interface {
	GeneratePresignedURL(ctx context.Context, objectName string, expiration time.Duration) (string, error)
	GetPublicURL(objectName string) string
	UploadObject(ctx context.Context, objectName string, data io.Reader, contentType string) error
	CopyObject(ctx context.Context, srcObject, dstObject string) error
	DeleteObject(ctx context.Context, objectName string) error
	GetObjectMetadata(ctx context.Context, objectName string) (interface{}, error)
	GetObjectReader(ctx context.Context, objectName string) (io.ReadCloser, error)
	GetBucketName() string
	Close() error
}

// Ensure services implement their interfaces
var _ UserServiceInterface = (*UserService)(nil)
var _ StorageServiceInterface = (*StorageService)(nil)
