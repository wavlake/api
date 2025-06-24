package handlers

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wavlake/api/internal/models"
	"github.com/wavlake/api/internal/services"
	"github.com/wavlake/api/internal/utils"
)

type TracksHandler struct {
	nostrTrackService *services.NostrTrackService
	processingService *services.ProcessingService
	audioProcessor    *utils.AudioProcessor
}

func NewTracksHandler(nostrTrackService *services.NostrTrackService, processingService *services.ProcessingService, audioProcessor *utils.AudioProcessor) *TracksHandler {
	return &TracksHandler{
		nostrTrackService: nostrTrackService,
		processingService: processingService,
		audioProcessor:    audioProcessor,
	}
}

type CreateTrackRequest struct {
	Extension string `json:"extension" binding:"required"`
}

type CreateTrackResponse struct {
	Success bool               `json:"success"`
	Data    *models.NostrTrack `json:"data,omitempty"`
	Error   string             `json:"error,omitempty"`
}

// CreateTrackNostr creates a new track via NIP-98 authentication
func (h *TracksHandler) CreateTrackNostr(c *gin.Context) {
	var req CreateTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, CreateTrackResponse{
			Success: false,
			Error:   "extension field is required",
		})
		return
	}

	// Validate file extension
	if !h.audioProcessor.IsFormatSupported(req.Extension) {
		c.JSON(http.StatusBadRequest, CreateTrackResponse{
			Success: false,
			Error:   "unsupported audio format",
		})
		return
	}

	// Get authenticated user info from NIP-98 middleware context
	pubkey, exists := c.Get("pubkey")
	if !exists {
		c.JSON(http.StatusUnauthorized, CreateTrackResponse{
			Success: false,
			Error:   "authentication required",
		})
		return
	}

	firebaseUID, exists := c.Get("firebase_uid")
	if !exists {
		c.JSON(http.StatusUnauthorized, CreateTrackResponse{
			Success: false,
			Error:   "user account not found",
		})
		return
	}

	pubkeyStr, ok := pubkey.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, CreateTrackResponse{
			Success: false,
			Error:   "invalid pubkey format",
		})
		return
	}

	firebaseUIDStr, ok := firebaseUID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, CreateTrackResponse{
			Success: false,
			Error:   "invalid user ID format",
		})
		return
	}

	// Create the track
	track, err := h.nostrTrackService.CreateTrack(
		c.Request.Context(),
		pubkeyStr,
		firebaseUIDStr,
		strings.TrimPrefix(req.Extension, "."),
	)
	if err != nil {
		log.Printf("Failed to create track: %v", err)
		c.JSON(http.StatusInternalServerError, CreateTrackResponse{
			Success: false,
			Error:   "failed to create track",
		})
		return
	}

	c.JSON(http.StatusOK, CreateTrackResponse{
		Success: true,
		Data:    track,
	})
}

type GetTracksResponse struct {
	Success bool                `json:"success"`
	Data    []*models.NostrTrack `json:"data,omitempty"`
	Error   string              `json:"error,omitempty"`
}

// GetMyTracks returns tracks for the authenticated user
func (h *TracksHandler) GetMyTracks(c *gin.Context) {
	// Get authenticated user info from NIP-98 middleware context
	pubkey, exists := c.Get("pubkey")
	if !exists {
		c.JSON(http.StatusUnauthorized, GetTracksResponse{
			Success: false,
			Error:   "authentication required",
		})
		return
	}

	pubkeyStr, ok := pubkey.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, GetTracksResponse{
			Success: false,
			Error:   "invalid pubkey format",
		})
		return
	}

	// Get tracks for this pubkey
	tracks, err := h.nostrTrackService.GetTracksByPubkey(c.Request.Context(), pubkeyStr)
	if err != nil {
		log.Printf("Failed to get tracks for pubkey %s: %v", pubkeyStr, err)
		c.JSON(http.StatusInternalServerError, GetTracksResponse{
			Success: false,
			Error:   "failed to retrieve tracks",
		})
		return
	}

	c.JSON(http.StatusOK, GetTracksResponse{
		Success: true,
		Data:    tracks,
	})
}

type GetTrackResponse struct {
	Success bool               `json:"success"`
	Data    *models.NostrTrack `json:"data,omitempty"`
	Error   string             `json:"error,omitempty"`
}

// GetTrack returns a specific track by ID
func (h *TracksHandler) GetTrack(c *gin.Context) {
	trackID := c.Param("id")
	if trackID == "" {
		c.JSON(http.StatusBadRequest, GetTrackResponse{
			Success: false,
			Error:   "track ID is required",
		})
		return
	}

	track, err := h.nostrTrackService.GetTrack(c.Request.Context(), trackID)
	if err != nil {
		log.Printf("Failed to get track %s: %v", trackID, err)
		c.JSON(http.StatusNotFound, GetTrackResponse{
			Success: false,
			Error:   "track not found",
		})
		return
	}

	// Check if user has access to this track
	pubkey, exists := c.Get("pubkey")
	if exists {
		pubkeyStr, ok := pubkey.(string)
		if ok && track.Pubkey == pubkeyStr {
			// User owns this track, return full details
			c.JSON(http.StatusOK, GetTrackResponse{
				Success: true,
				Data:    track,
			})
			return
		}
	}

	// Return limited public information
	publicTrack := &models.NostrTrack{
		ID:            track.ID,
		OriginalURL:   track.OriginalURL,
		CompressedURL: track.CompressedURL,
		Duration:      track.Duration,
		IsProcessing:  track.IsProcessing,
		IsCompressed:  track.IsCompressed,
		CreatedAt:     track.CreatedAt,
	}

	c.JSON(http.StatusOK, GetTrackResponse{
		Success: true,
		Data:    publicTrack,
	})
}

// DeleteTrack soft deletes a track
func (h *TracksHandler) DeleteTrack(c *gin.Context) {
	trackID := c.Param("id")
	if trackID == "" {
		c.JSON(http.StatusBadRequest, CreateTrackResponse{
			Success: false,
			Error:   "track ID is required",
		})
		return
	}

	// Get track to verify ownership
	track, err := h.nostrTrackService.GetTrack(c.Request.Context(), trackID)
	if err != nil {
		c.JSON(http.StatusNotFound, CreateTrackResponse{
			Success: false,
			Error:   "track not found",
		})
		return
	}

	// Check ownership
	pubkey, exists := c.Get("pubkey")
	if !exists {
		c.JSON(http.StatusUnauthorized, CreateTrackResponse{
			Success: false,
			Error:   "authentication required",
		})
		return
	}

	pubkeyStr, ok := pubkey.(string)
	if !ok || track.Pubkey != pubkeyStr {
		c.JSON(http.StatusForbidden, CreateTrackResponse{
			Success: false,
			Error:   "not authorized to delete this track",
		})
		return
	}

	// Delete the track
	if err := h.nostrTrackService.DeleteTrack(c.Request.Context(), trackID); err != nil {
		log.Printf("Failed to delete track %s: %v", trackID, err)
		c.JSON(http.StatusInternalServerError, CreateTrackResponse{
			Success: false,
			Error:   "failed to delete track",
		})
		return
	}

	c.JSON(http.StatusOK, CreateTrackResponse{
		Success: true,
	})
}

// GetTrackStatus returns the current processing status of a track
func (h *TracksHandler) GetTrackStatus(c *gin.Context) {
	trackID := c.Param("id")
	if trackID == "" {
		c.JSON(http.StatusBadRequest, GetTrackResponse{
			Success: false,
			Error:   "track ID is required",
		})
		return
	}

	track, err := h.nostrTrackService.GetTrack(c.Request.Context(), trackID)
	if err != nil {
		c.JSON(http.StatusNotFound, GetTrackResponse{
			Success: false,
			Error:   "track not found",
		})
		return
	}

	// Check ownership for detailed status
	pubkey, exists := c.Get("pubkey")
	if !exists {
		c.JSON(http.StatusUnauthorized, GetTrackResponse{
			Success: false,
			Error:   "authentication required",
		})
		return
	}

	pubkeyStr, ok := pubkey.(string)
	if !ok || track.Pubkey != pubkeyStr {
		c.JSON(http.StatusForbidden, GetTrackResponse{
			Success: false,
			Error:   "not authorized to view this track status",
		})
		return
	}

	// Return full track details including processing status
	c.JSON(http.StatusOK, GetTrackResponse{
		Success: true,
		Data:    track,
	})
}

// TriggerProcessing manually triggers processing for a track
func (h *TracksHandler) TriggerProcessing(c *gin.Context) {
	trackID := c.Param("id")
	if trackID == "" {
		c.JSON(http.StatusBadRequest, CreateTrackResponse{
			Success: false,
			Error:   "track ID is required",
		})
		return
	}

	// Get track to verify ownership and status
	track, err := h.nostrTrackService.GetTrack(c.Request.Context(), trackID)
	if err != nil {
		c.JSON(http.StatusNotFound, CreateTrackResponse{
			Success: false,
			Error:   "track not found",
		})
		return
	}

	// Check ownership
	pubkey, exists := c.Get("pubkey")
	if !exists {
		c.JSON(http.StatusUnauthorized, CreateTrackResponse{
			Success: false,
			Error:   "authentication required",
		})
		return
	}

	pubkeyStr, ok := pubkey.(string)
	if !ok || track.Pubkey != pubkeyStr {
		c.JSON(http.StatusForbidden, CreateTrackResponse{
			Success: false,
			Error:   "not authorized to process this track",
		})
		return
	}

	// Don't re-process already processed tracks
	if !track.IsProcessing && track.CompressedURL != "" {
		c.JSON(http.StatusBadRequest, CreateTrackResponse{
			Success: false,
			Error:   "track already processed",
		})
		return
	}

	// Mark as processing and start async processing
	updates := map[string]interface{}{
		"is_processing": true,
	}
	if err := h.nostrTrackService.UpdateTrack(c.Request.Context(), trackID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, CreateTrackResponse{
			Success: false,
			Error:   "failed to update track status",
		})
		return
	}

	// Start processing
	h.processingService.ProcessTrackAsync(c.Request.Context(), trackID)

	c.JSON(http.StatusOK, CreateTrackResponse{
		Success: true,
	})
}

// ProcessTrackWebhook handles file processing webhooks (e.g., from Cloud Functions)
func (h *TracksHandler) ProcessTrackWebhook(c *gin.Context) {
	// Optional webhook authentication
	if expectedSecret := os.Getenv("WEBHOOK_SECRET"); expectedSecret != "" {
		providedSecret := c.GetHeader("X-Webhook-Secret")
		if providedSecret != expectedSecret {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "invalid webhook secret",
			})
			return
		}
	}

	type WebhookPayload struct {
		TrackID       string `json:"track_id"`
		Status        string `json:"status"` // "uploaded", "processed", or "failed"
		Size          int64  `json:"size,omitempty"`
		Duration      int    `json:"duration,omitempty"`
		CompressedURL string `json:"compressed_url,omitempty"`
		Error         string `json:"error,omitempty"`
		Source        string `json:"source,omitempty"` // "gcs_trigger", "manual", etc.
	}

	var payload WebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid payload",
		})
		return
	}

	ctx := c.Request.Context()

	switch payload.Status {
	case "uploaded":
		// File was uploaded to GCS, start processing
		log.Printf("Starting processing for uploaded track %s (source: %s)", payload.TrackID, payload.Source)
		
		// Start async processing
		h.processingService.ProcessTrackAsync(ctx, payload.TrackID)
		
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "processing started",
		})
		return

	case "processed":
		// Update track as processed
		if err := h.nostrTrackService.MarkTrackAsProcessed(ctx, payload.TrackID, payload.Size, payload.Duration); err != nil {
			log.Printf("Failed to mark track as processed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "failed to update track status",
			})
			return
		}

		// If compressed file is available, update that too
		if payload.CompressedURL != "" {
			if err := h.nostrTrackService.MarkTrackAsCompressed(ctx, payload.TrackID, payload.CompressedURL); err != nil {
				log.Printf("Failed to mark track as compressed: %v", err)
				// Don't fail the request for this
			}
		}

	case "failed":
		// Mark track as failed processing
		updates := map[string]interface{}{
			"is_processing": false,
			"error":        payload.Error,
		}
		if err := h.nostrTrackService.UpdateTrack(ctx, payload.TrackID, updates); err != nil {
			log.Printf("Failed to mark track as failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "failed to update track status",
			})
			return
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid status",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}