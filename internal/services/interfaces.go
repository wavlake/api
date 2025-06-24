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
}

// Ensure UserService implements the interface
var _ UserServiceInterface = (*UserService)(nil)