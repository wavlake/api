package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"github.com/wavlake/api/internal/models"
	"google.golang.org/api/iterator"
)

type NostrTrackService struct {
	firestoreClient *firestore.Client
	storageService  *StorageService
}

func NewNostrTrackService(firestoreClient *firestore.Client, storageService *StorageService) *NostrTrackService {
	return &NostrTrackService{
		firestoreClient: firestoreClient,
		storageService:  storageService,
	}
}

// CreateTrack creates a new NostrTrack record and returns a presigned upload URL
func (s *NostrTrackService) CreateTrack(ctx context.Context, pubkey, firebaseUID, extension string) (*models.NostrTrack, error) {
	trackID := uuid.New().String()
	now := time.Now()

	// Generate storage object names
	originalObjectName := fmt.Sprintf("tracks/original/%s.%s", trackID, extension)
	
	// Generate presigned URL for upload (valid for 1 hour)
	presignedURL, err := s.storageService.GeneratePresignedURL(ctx, originalObjectName, time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	// Create the track record
	track := &models.NostrTrack{
		ID:           trackID,
		FirebaseUID:  firebaseUID,
		Pubkey:       pubkey,
		OriginalURL:  s.storageService.GetPublicURL(originalObjectName),
		PresignedURL: presignedURL,
		Extension:    extension,
		IsProcessing: true,
		IsCompressed: false,
		Deleted:      false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Save to Firestore
	_, err = s.firestoreClient.Collection("nostr_tracks").Doc(trackID).Set(ctx, track)
	if err != nil {
		return nil, fmt.Errorf("failed to save track to firestore: %w", err)
	}

	log.Printf("Created new Nostr track with ID: %s for pubkey: %s", trackID, pubkey)
	return track, nil
}

// GetTrack retrieves a track by ID
func (s *NostrTrackService) GetTrack(ctx context.Context, trackID string) (*models.NostrTrack, error) {
	doc, err := s.firestoreClient.Collection("nostr_tracks").Doc(trackID).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get track: %w", err)
	}

	var track models.NostrTrack
	if err := doc.DataTo(&track); err != nil {
		return nil, fmt.Errorf("failed to decode track: %w", err)
	}

	return &track, nil
}

// GetTracksByPubkey retrieves all tracks for a given pubkey
func (s *NostrTrackService) GetTracksByPubkey(ctx context.Context, pubkey string) ([]*models.NostrTrack, error) {
	query := s.firestoreClient.Collection("nostr_tracks").
		Where("pubkey", "==", pubkey).
		Where("deleted", "==", false).
		OrderBy("created_at", firestore.Desc)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var tracks []*models.NostrTrack
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate tracks: %w", err)
		}

		var track models.NostrTrack
		if err := doc.DataTo(&track); err != nil {
			log.Printf("Failed to decode track %s: %v", doc.Ref.ID, err)
			continue
		}

		tracks = append(tracks, &track)
	}

	return tracks, nil
}

// GetTracksByFirebaseUID retrieves all tracks for a given Firebase UID
func (s *NostrTrackService) GetTracksByFirebaseUID(ctx context.Context, firebaseUID string) ([]*models.NostrTrack, error) {
	query := s.firestoreClient.Collection("nostr_tracks").
		Where("firebase_uid", "==", firebaseUID).
		Where("deleted", "==", false).
		OrderBy("created_at", firestore.Desc)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var tracks []*models.NostrTrack
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate tracks: %w", err)
		}

		var track models.NostrTrack
		if err := doc.DataTo(&track); err != nil {
			log.Printf("Failed to decode track %s: %v", doc.Ref.ID, err)
			continue
		}

		tracks = append(tracks, &track)
	}

	return tracks, nil
}

// UpdateTrack updates track metadata
func (s *NostrTrackService) UpdateTrack(ctx context.Context, trackID string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	var updatePaths []firestore.Update
	for path, value := range updates {
		updatePaths = append(updatePaths, firestore.Update{Path: path, Value: value})
	}
	
	_, err := s.firestoreClient.Collection("nostr_tracks").Doc(trackID).Update(ctx, updatePaths)
	if err != nil {
		return fmt.Errorf("failed to update track: %w", err)
	}

	return nil
}

// MarkTrackAsProcessed updates track status after processing
func (s *NostrTrackService) MarkTrackAsProcessed(ctx context.Context, trackID string, size int64, duration int) error {
	updates := map[string]interface{}{
		"is_processing": false,
		"size":         size,
		"duration":     duration,
		"updated_at":   time.Now(),
	}

	return s.UpdateTrack(ctx, trackID, updates)
}

// MarkTrackAsCompressed updates track with compressed file info
func (s *NostrTrackService) MarkTrackAsCompressed(ctx context.Context, trackID, compressedURL string) error {
	updates := map[string]interface{}{
		"compressed_url": compressedURL,
		"is_compressed":  true,
		"updated_at":     time.Now(),
	}

	return s.UpdateTrack(ctx, trackID, updates)
}

// DeleteTrack soft deletes a track
func (s *NostrTrackService) DeleteTrack(ctx context.Context, trackID string) error {
	updates := map[string]interface{}{
		"deleted":    true,
		"updated_at": time.Now(),
	}

	return s.UpdateTrack(ctx, trackID, updates)
}

// HardDeleteTrack permanently deletes a track and its files
func (s *NostrTrackService) HardDeleteTrack(ctx context.Context, trackID string) error {
	// Get track first to know which files to delete
	track, err := s.GetTrack(ctx, trackID)
	if err != nil {
		return fmt.Errorf("failed to get track for deletion: %w", err)
	}

	// Delete files from storage
	originalObjectName := fmt.Sprintf("tracks/original/%s.%s", trackID, track.Extension)
	if err := s.storageService.DeleteObject(ctx, originalObjectName); err != nil {
		log.Printf("Failed to delete original file for track %s: %v", trackID, err)
	}

	if track.CompressedURL != "" {
		compressedObjectName := fmt.Sprintf("tracks/compressed/%s.mp3", trackID)
		if err := s.storageService.DeleteObject(ctx, compressedObjectName); err != nil {
			log.Printf("Failed to delete compressed file for track %s: %v", trackID, err)
		}
	}

	// Delete from Firestore
	_, err = s.firestoreClient.Collection("nostr_tracks").Doc(trackID).Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete track from firestore: %w", err)
	}

	log.Printf("Hard deleted track %s", trackID)
	return nil
}

// UpdateCompressionVisibility updates which compression versions are public
func (s *NostrTrackService) UpdateCompressionVisibility(ctx context.Context, trackID string, updates []struct {
	VersionID string `json:"version_id"`
	IsPublic  bool   `json:"is_public"`
}) error {
	// Get current track
	track, err := s.GetTrack(ctx, trackID)
	if err != nil {
		return fmt.Errorf("failed to get track: %w", err)
	}

	// Update visibility for specified versions
	for i, version := range track.CompressionVersions {
		for _, update := range updates {
			if version.ID == update.VersionID {
				track.CompressionVersions[i].IsPublic = update.IsPublic
				break
			}
		}
	}

	// Save updated track
	_, err = s.firestoreClient.Collection("nostr_tracks").Doc(trackID).Set(ctx, track)
	if err != nil {
		return fmt.Errorf("failed to update track: %w", err)
	}

	log.Printf("Updated compression visibility for track %s", trackID)
	return nil
}

// AddCompressionVersion adds a new compression version to a track
func (s *NostrTrackService) AddCompressionVersion(ctx context.Context, trackID string, version models.CompressionVersion) error {
	// Get current track
	track, err := s.GetTrack(ctx, trackID)
	if err != nil {
		return fmt.Errorf("failed to get track: %w", err)
	}

	// Check if version with same ID already exists
	for i, existing := range track.CompressionVersions {
		if existing.ID == version.ID {
			// Update existing version
			track.CompressionVersions[i] = version
			log.Printf("Updated existing compression version %s for track %s", version.ID, trackID)
			
			// Save updated track
			_, err = s.firestoreClient.Collection("nostr_tracks").Doc(trackID).Set(ctx, track)
			return err
		}
	}

	// Add new version
	track.CompressionVersions = append(track.CompressionVersions, version)
	track.HasPendingCompression = false // Clear pending flag

	// Save updated track
	_, err = s.firestoreClient.Collection("nostr_tracks").Doc(trackID).Set(ctx, track)
	if err != nil {
		return fmt.Errorf("failed to update track: %w", err)
	}

	log.Printf("Added compression version %s for track %s", version.ID, trackID)
	return nil
}

// SetPendingCompression marks a track as having pending compression requests
func (s *NostrTrackService) SetPendingCompression(ctx context.Context, trackID string, pending bool) error {
	updates := map[string]interface{}{
		"has_pending_compression": pending,
		"updated_at":              time.Now(),
	}

	_, err := s.firestoreClient.Collection("nostr_tracks").Doc(trackID).Update(ctx, updates)
	if err != nil {
		return fmt.Errorf("failed to update pending compression status: %w", err)
	}

	return nil
}