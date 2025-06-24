package services

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/wavlake/api/internal/models"
	"google.golang.org/api/iterator"
)

type UserService struct {
	firestoreClient *firestore.Client
}

func NewUserService(firestoreClient *firestore.Client) *UserService {
	return &UserService{
		firestoreClient: firestoreClient,
	}
}

// LinkPubkeyToUser links a Nostr pubkey to a Firebase user
func (s *UserService) LinkPubkeyToUser(ctx context.Context, pubkey, firebaseUID string) error {
	now := time.Now()

	// Check if pubkey is already linked to a different user
	existingAuth, err := s.getNostrAuth(ctx, pubkey)
	if err == nil && existingAuth.FirebaseUID != firebaseUID && existingAuth.Active {
		return fmt.Errorf("pubkey is already linked to a different user")
	}

	// Start a transaction
	err = s.firestoreClient.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// Create or update User record
		userRef := s.firestoreClient.Collection("users").Doc(firebaseUID)
		userDoc, err := tx.Get(userRef)

		var user models.User
		if err != nil {
			// Create new user
			user = models.User{
				FirebaseUID:   firebaseUID,
				CreatedAt:     now,
				UpdatedAt:     now,
				ActivePubkeys: []string{pubkey},
			}
		} else {
			// Update existing user
			if err := userDoc.DataTo(&user); err != nil {
				return fmt.Errorf("failed to parse user data: %w", err)
			}

			// Add pubkey if not already present
			if !contains(user.ActivePubkeys, pubkey) {
				user.ActivePubkeys = append(user.ActivePubkeys, pubkey)
			}
			user.UpdatedAt = now
		}

		if err := tx.Set(userRef, user); err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}

		// Create or update NostrAuth record
		nostrAuthRef := s.firestoreClient.Collection("nostr_auth").Doc(pubkey)
		nostrAuth := models.NostrAuth{
			Pubkey:        pubkey,
			FirebaseUID:   firebaseUID,
			Active:        true,
			CreatedAt:     now,
			LastUsedAt:    now,
			LinkedAt:      now,
			DisplayPubkey: formatDisplayPubkey(pubkey),
		}

		if err := tx.Set(nostrAuthRef, nostrAuth); err != nil {
			return fmt.Errorf("failed to create nostr auth: %w", err)
		}

		return nil
	})

	return err
}

// UnlinkPubkeyFromUser unlinks a pubkey from a Firebase user
func (s *UserService) UnlinkPubkeyFromUser(ctx context.Context, pubkey, firebaseUID string) error {
	// Verify the pubkey belongs to this user
	nostrAuth, err := s.getNostrAuth(ctx, pubkey)
	if err != nil {
		return fmt.Errorf("pubkey not found")
	}

	if nostrAuth.FirebaseUID != firebaseUID {
		return fmt.Errorf("pubkey does not belong to this user")
	}

	if !nostrAuth.Active {
		return fmt.Errorf("pubkey is already unlinked")
	}

	// Start a transaction
	return s.firestoreClient.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// Update NostrAuth to inactive
		nostrAuthRef := s.firestoreClient.Collection("nostr_auth").Doc(pubkey)
		nostrAuth.Active = false
		if err := tx.Set(nostrAuthRef, nostrAuth); err != nil {
			return fmt.Errorf("failed to update nostr auth: %w", err)
		}

		// Update User to remove pubkey from active list
		userRef := s.firestoreClient.Collection("users").Doc(firebaseUID)
		userDoc, err := tx.Get(userRef)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

		var user models.User
		if err := userDoc.DataTo(&user); err != nil {
			return fmt.Errorf("failed to parse user data: %w", err)
		}

		user.ActivePubkeys = removeString(user.ActivePubkeys, pubkey)
		user.UpdatedAt = time.Now()

		if err := tx.Set(userRef, user); err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}

		return nil
	})
}

// GetLinkedPubkeys returns all active pubkeys for a Firebase user
func (s *UserService) GetLinkedPubkeys(ctx context.Context, firebaseUID string) ([]models.NostrAuth, error) {
	query := s.firestoreClient.Collection("nostr_auth").
		Where("firebase_uid", "==", firebaseUID).
		Where("active", "==", true).
		OrderBy("linked_at", firestore.Asc)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var pubkeys []models.NostrAuth
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate pubkeys: %w", err)
		}

		var nostrAuth models.NostrAuth
		if err := doc.DataTo(&nostrAuth); err != nil {
			return nil, fmt.Errorf("failed to parse nostr auth: %w", err)
		}

		pubkeys = append(pubkeys, nostrAuth)
	}

	return pubkeys, nil
}

// getNostrAuth retrieves a NostrAuth record by pubkey
func (s *UserService) getNostrAuth(ctx context.Context, pubkey string) (*models.NostrAuth, error) {
	doc, err := s.firestoreClient.Collection("nostr_auth").Doc(pubkey).Get(ctx)
	if err != nil {
		return nil, err
	}

	var nostrAuth models.NostrAuth
	if err := doc.DataTo(&nostrAuth); err != nil {
		return nil, err
	}

	return &nostrAuth, nil
}

// Helper functions
func formatDisplayPubkey(pubkey string) string {
	if len(pubkey) <= 16 {
		return pubkey
	}
	return pubkey[:8] + "..." + pubkey[len(pubkey)-8:]
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func removeString(slice []string, item string) []string {
	var result []string
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}
