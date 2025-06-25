package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/wavlake/api/internal/models"
	"github.com/wavlake/api/internal/utils"
)

type ProcessingService struct {
	storageService    *StorageService
	nostrTrackService *NostrTrackService
	audioProcessor    *utils.AudioProcessor
	tempDir           string
}

func NewProcessingService(storageService *StorageService, nostrTrackService *NostrTrackService, audioProcessor *utils.AudioProcessor, tempDir string) *ProcessingService {
	return &ProcessingService{
		storageService:    storageService,
		nostrTrackService: nostrTrackService,
		audioProcessor:    audioProcessor,
		tempDir:           tempDir,
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

	// Update track with processing results (legacy fields for backwards compatibility)
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

	// Also add as a compression version for new system compatibility
	defaultVersion := models.CompressionVersion{
		ID:         "default-128k-mp3",
		URL:        compressedURL,
		Bitrate:    128,
		Format:     "mp3",
		Quality:    "medium",
		SampleRate: 44100,
		Size:       0,    // Will be updated if we can get file info
		IsPublic:   true, // Default compressed version is public for backwards compatibility
		CreatedAt:  time.Now(),
		Options: models.CompressionOption{
			Bitrate:    128,
			Format:     "mp3",
			Quality:    "medium",
			SampleRate: 44100,
		},
	}

	// Try to get compressed file size
	if compressedInfo, err := os.Stat(compressedPath); err == nil {
		defaultVersion.Size = compressedInfo.Size()
	}

	// Add default compression version (ignore errors to maintain backwards compatibility)
	if err := p.nostrTrackService.AddCompressionVersion(ctx, trackID, defaultVersion); err != nil {
		log.Printf("Warning: Failed to add default compression version for track %s: %v", trackID, err)
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
		"error":         errorMsg,
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

// RequestCompressionVersions queues multiple compression jobs for a track
func (p *ProcessingService) RequestCompressionVersions(ctx context.Context, trackID string, compressionOptions []models.CompressionOption) error {
	log.Printf("Requesting compression versions for track %s with %d options", trackID, len(compressionOptions))

	// Mark track as having pending compression
	if err := p.nostrTrackService.SetPendingCompression(ctx, trackID, true); err != nil {
		return fmt.Errorf("failed to mark track as pending compression: %w", err)
	}

	// Process each compression option asynchronously
	for _, option := range compressionOptions {
		p.ProcessCompressionAsync(ctx, trackID, option)
	}

	return nil
}

// ProcessCompressionAsync processes a single compression option in background
func (p *ProcessingService) ProcessCompressionAsync(ctx context.Context, trackID string, option models.CompressionOption) {
	go func() {
		// Create a background context with timeout
		processCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := p.ProcessCompression(processCtx, trackID, option); err != nil {
			log.Printf("Async compression failed for track %s (option: %+v): %v", trackID, option, err)
		}
	}()
}

// ProcessCompression creates a single compressed version of a track
func (p *ProcessingService) ProcessCompression(ctx context.Context, trackID string, option models.CompressionOption) error {
	versionID := uuid.New().String()
	log.Printf("Starting compression for track %s, version %s (bitrate: %d, format: %s)", trackID, versionID, option.Bitrate, option.Format)

	// Get track info
	track, err := p.nostrTrackService.GetTrack(ctx, trackID)
	if err != nil {
		return fmt.Errorf("failed to get track: %w", err)
	}

	// Create temp files
	originalPath := filepath.Join(p.tempDir, fmt.Sprintf("%s_original.%s", trackID, track.Extension))
	compressedPath := filepath.Join(p.tempDir, fmt.Sprintf("%s_%s_compressed.%s", trackID, versionID, option.Format))

	defer func() {
		os.Remove(originalPath)
		os.Remove(compressedPath)
	}()

	// Download original file from GCS
	if err := p.downloadFile(ctx, track.OriginalURL, originalPath); err != nil {
		return fmt.Errorf("download failed: %v", err)
	}

	// Validate it's a valid audio file
	if err := p.audioProcessor.ValidateAudioFile(ctx, originalPath); err != nil {
		return fmt.Errorf("invalid audio file: %v", err)
	}

	// Compress with specific options
	if err := p.audioProcessor.CompressAudioWithOptions(ctx, originalPath, compressedPath, option); err != nil {
		return fmt.Errorf("compression failed: %v", err)
	}

	// Get compressed file info
	compressedInfo, err := os.Stat(compressedPath)
	if err != nil {
		return fmt.Errorf("failed to get compressed file info: %v", err)
	}

	// Upload compressed file to GCS
	compressedObjectName := fmt.Sprintf("tracks/compressed/%s_%s.%s", trackID, versionID, option.Format)
	compressedFile, err := os.Open(compressedPath)
	if err != nil {
		return fmt.Errorf("failed to open compressed file: %v", err)
	}
	defer compressedFile.Close()

	contentType := getContentTypeForFormat(option.Format)
	if err := p.storageService.UploadObject(ctx, compressedObjectName, compressedFile, contentType); err != nil {
		return fmt.Errorf("failed to upload compressed file: %v", err)
	}

	compressedURL := p.storageService.GetPublicURL(compressedObjectName)

	// Get actual audio info from compressed file
	actualInfo, err := p.audioProcessor.GetAudioInfo(ctx, compressedPath)
	actualBitrate := option.Bitrate
	actualSampleRate := option.SampleRate
	if actualInfo != nil {
		actualBitrate = actualInfo.Bitrate
		actualSampleRate = actualInfo.SampleRate
	}

	// Create compression version record
	version := models.CompressionVersion{
		ID:         versionID,
		URL:        compressedURL,
		Bitrate:    actualBitrate,
		Format:     option.Format,
		Quality:    option.Quality,
		SampleRate: actualSampleRate,
		Size:       compressedInfo.Size(),
		IsPublic:   false, // Default to private, user can make public later
		CreatedAt:  time.Now(),
		Options:    option,
	}

	// Add to track
	if err := p.nostrTrackService.AddCompressionVersion(ctx, trackID, version); err != nil {
		return fmt.Errorf("failed to save compression version: %v", err)
	}

	log.Printf("Successfully created compression version %s for track %s", versionID, trackID)
	return nil
}

// getContentTypeForFormat returns the appropriate MIME type for audio formats
func getContentTypeForFormat(format string) string {
	switch format {
	case "mp3":
		return "audio/mpeg"
	case "aac":
		return "audio/aac"
	case "ogg":
		return "audio/ogg"
	default:
		return "audio/mpeg"
	}
}
