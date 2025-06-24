package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/wavlake/api/internal/utils"
)

type ProcessingService struct {
	storageService    *StorageService
	nostrTrackService *NostrTrackService
	audioProcessor    *utils.AudioProcessor
	tempDir          string
}

func NewProcessingService(storageService *StorageService, nostrTrackService *NostrTrackService, audioProcessor *utils.AudioProcessor, tempDir string) *ProcessingService {
	return &ProcessingService{
		storageService:    storageService,
		nostrTrackService: nostrTrackService,
		audioProcessor:    audioProcessor,
		tempDir:          tempDir,
	}
}

// ProcessTrack downloads, analyzes, and compresses an uploaded track
func (p *ProcessingService) ProcessTrack(ctx context.Context, trackID string) error {
	log.Printf("Starting processing for track %s", trackID)

	// Get track info
	track, err := p.nostrTrackService.GetTrack(ctx, trackID)
	if err != nil {
		return fmt.Errorf("failed to get track: %w", err)
	}

	// Create temp files
	originalPath := filepath.Join(p.tempDir, fmt.Sprintf("%s_original.%s", trackID, track.Extension))
	compressedPath := filepath.Join(p.tempDir, fmt.Sprintf("%s_compressed.mp3", trackID))
	
	defer func() {
		os.Remove(originalPath)
		os.Remove(compressedPath)
	}()

	// Download original file from GCS
	if err := p.downloadFile(ctx, track.OriginalURL, originalPath); err != nil {
		return p.markProcessingFailed(ctx, trackID, fmt.Sprintf("download failed: %v", err))
	}

	// Validate it's a valid audio file
	if err := p.audioProcessor.ValidateAudioFile(ctx, originalPath); err != nil {
		return p.markProcessingFailed(ctx, trackID, fmt.Sprintf("invalid audio file: %v", err))
	}

	// Get audio metadata
	audioInfo, err := p.audioProcessor.GetAudioInfo(ctx, originalPath)
	if err != nil {
		log.Printf("Warning: Could not get audio info for %s: %v", trackID, err)
		// Continue processing even if we can't get metadata
	}

	// Compress the audio
	if err := p.audioProcessor.CompressAudio(ctx, originalPath, compressedPath); err != nil {
		return p.markProcessingFailed(ctx, trackID, fmt.Sprintf("compression failed: %v", err))
	}

	// Upload compressed file to GCS
	compressedObjectName := fmt.Sprintf("tracks/compressed/%s.mp3", trackID)
	compressedFile, err := os.Open(compressedPath)
	if err != nil {
		return p.markProcessingFailed(ctx, trackID, fmt.Sprintf("failed to open compressed file: %v", err))
	}
	defer compressedFile.Close()

	if err := p.storageService.UploadObject(ctx, compressedObjectName, compressedFile, "audio/mpeg"); err != nil {
		return p.markProcessingFailed(ctx, trackID, fmt.Sprintf("failed to upload compressed file: %v", err))
	}

	compressedURL := p.storageService.GetPublicURL(compressedObjectName)

	// Update track with processing results
	updates := map[string]interface{}{
		"is_processing":  false,
		"is_compressed":  true,
		"compressed_url": compressedURL,
	}

	if audioInfo != nil {
		updates["size"] = audioInfo.Size
		updates["duration"] = audioInfo.Duration
	}

	if err := p.nostrTrackService.UpdateTrack(ctx, trackID, updates); err != nil {
		log.Printf("Failed to update track %s after processing: %v", trackID, err)
		// Don't return error since processing succeeded
	}

	log.Printf("Successfully processed track %s", trackID)
	return nil
}

// downloadFile downloads a file from a URL to local path
func (p *ProcessingService) downloadFile(ctx context.Context, url, filePath string) error {
	// For GCS URLs, we can use the storage client directly
	// This is more efficient than HTTP download for files in the same project
	
	// Create temp file
	tempFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	// Extract object name from URL
	// URL format: https://storage.googleapis.com/bucket/object
	// We need to get the object name part
	objectName := ""
	if len(url) > 0 {
		// Simple extraction - in production you might want more robust parsing
		parts := filepath.Base(url)
		if track, err := p.nostrTrackService.GetTrack(ctx, parts[:len(parts)-len(filepath.Ext(parts))]); err == nil {
			objectName = fmt.Sprintf("tracks/original/%s.%s", track.ID, track.Extension)
		}
	}

	if objectName == "" {
		return fmt.Errorf("could not determine object name from URL")
	}

	// Download from GCS
	reader, err := p.storageService.GetClient().Bucket(p.storageService.GetBucketName()).Object(objectName).NewReader(ctx)
	if err != nil {
		return fmt.Errorf("failed to create GCS reader: %w", err)
	}
	defer reader.Close()

	// Copy to temp file
	if _, err := tempFile.ReadFrom(reader); err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	return nil
}

// markProcessingFailed marks a track as failed processing
func (p *ProcessingService) markProcessingFailed(ctx context.Context, trackID, errorMsg string) error {
	log.Printf("Processing failed for track %s: %s", trackID, errorMsg)
	
	updates := map[string]interface{}{
		"is_processing": false,
		"error":        errorMsg,
	}
	
	return p.nostrTrackService.UpdateTrack(ctx, trackID, updates)
}

// ProcessTrackAsync starts track processing in a goroutine
func (p *ProcessingService) ProcessTrackAsync(ctx context.Context, trackID string) {
	go func() {
		// Create a background context with timeout
		processCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := p.ProcessTrack(processCtx, trackID); err != nil {
			log.Printf("Async processing failed for track %s: %v", trackID, err)
		}
	}()
}