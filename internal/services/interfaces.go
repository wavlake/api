package services

import (
	"context"

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

// Ensure UserService implements the interface
var _ UserServiceInterface = (*UserService)(nil)
