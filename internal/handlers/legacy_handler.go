package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wavlake/api/internal/models"
	"github.com/wavlake/api/internal/services"
)

type LegacyHandler struct {
	postgresService services.PostgresServiceInterface
}

// NewLegacyHandler creates a new legacy handler
func NewLegacyHandler(postgresService services.PostgresServiceInterface) *LegacyHandler {
	return &LegacyHandler{
		postgresService: postgresService,
	}
}

// isDatabaseError checks if the error is a database/SQL error vs user-not-found
func isDatabaseError(err error) bool {
	if err == nil {
		return false
	}

	// If it's sql.ErrNoRows, it's a legitimate "not found" case
	if err == sql.ErrNoRows {
		return false
	}

	errMsg := err.Error()
	// Check for common database/SQL errors
	databaseErrors := []string{
		"relation", "does not exist",
		"syntax error", "column", "unknown",
		"connection", "timeout", "network",
		"permission denied", "access denied",
		"invalid", "constraint",
	}

	for _, dbErr := range databaseErrors {
		if strings.Contains(strings.ToLower(errMsg), dbErr) {
			return true
		}
	}

	return false
}

// UserMetadataResponse represents the complete user metadata response
type UserMetadataResponse struct {
	User    *models.LegacyUser    `json:"user"`
	Artists []models.LegacyArtist `json:"artists"`
	Albums  []models.LegacyAlbum  `json:"albums"`
	Tracks  []models.LegacyTrack  `json:"tracks"`
}

// GetUserMetadata handles GET /v1/legacy/metadata
// Returns all user metadata from the legacy PostgreSQL system
func (h *LegacyHandler) GetUserMetadata(c *gin.Context) {
	firebaseUID := c.GetString("firebase_uid")

	if firebaseUID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to find an associated Firebase UID"})
		return
	}

	ctx := c.Request.Context()

	// Get user data
	user, err := h.postgresService.GetUserByFirebaseUID(ctx, firebaseUID)
	if err != nil {
		// Check if this is a database error vs user not found
		if isDatabaseError(err) {
			log.Printf("PostgreSQL error getting user %s: %v", firebaseUID, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Database error occurred",
				"details": err.Error(),
			})
			return
		}

		// User not found - return empty response
		response := UserMetadataResponse{
			User:    nil,
			Artists: []models.LegacyArtist{},
			Albums:  []models.LegacyAlbum{},
			Tracks:  []models.LegacyTrack{},
		}
		c.JSON(http.StatusOK, response)
		return
	}

	// Get associated data (return error for database issues, empty arrays for no data)
	artists, err := h.postgresService.GetUserArtists(ctx, firebaseUID)
	if err != nil {
		if isDatabaseError(err) {
			log.Printf("PostgreSQL error getting artists for user %s: %v", firebaseUID, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Database error while fetching artists",
				"details": err.Error(),
			})
			return
		}
		artists = []models.LegacyArtist{}
	}

	albums, err := h.postgresService.GetUserAlbums(ctx, firebaseUID)
	if err != nil {
		if isDatabaseError(err) {
			log.Printf("PostgreSQL error getting albums for user %s: %v", firebaseUID, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Database error while fetching albums",
				"details": err.Error(),
			})
			return
		}
		albums = []models.LegacyAlbum{}
	}

	tracks, err := h.postgresService.GetUserTracks(ctx, firebaseUID)
	if err != nil {
		if isDatabaseError(err) {
			log.Printf("PostgreSQL error getting tracks for user %s: %v", firebaseUID, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Database error while fetching tracks",
				"details": err.Error(),
			})
			return
		}
		tracks = []models.LegacyTrack{}
	}

	response := UserMetadataResponse{
		User:    user,
		Artists: artists,
		Albums:  albums,
		Tracks:  tracks,
	}

	c.JSON(http.StatusOK, response)
}

// GetUserTracks handles GET /v1/legacy/tracks
// Returns user's tracks from the legacy system
func (h *LegacyHandler) GetUserTracks(c *gin.Context) {
	firebaseUID := c.GetString("firebase_uid")
	if firebaseUID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to find an associated Firebase UID"})
		return
	}

	ctx := c.Request.Context()

	tracks, err := h.postgresService.GetUserTracks(ctx, firebaseUID)
	if err != nil {
		// Return empty array instead of error
		tracks = []models.LegacyTrack{}
	}

	c.JSON(http.StatusOK, gin.H{"tracks": tracks})
}

// GetUserArtists handles GET /v1/legacy/artists
// Returns user's artists from the legacy system
func (h *LegacyHandler) GetUserArtists(c *gin.Context) {
	firebaseUID := c.GetString("firebase_uid")
	if firebaseUID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to find an associated Firebase UID"})
		return
	}

	ctx := c.Request.Context()

	artists, err := h.postgresService.GetUserArtists(ctx, firebaseUID)
	if err != nil {
		// Return empty array instead of error
		artists = []models.LegacyArtist{}
	}

	c.JSON(http.StatusOK, gin.H{"artists": artists})
}

// GetUserAlbums handles GET /v1/legacy/albums
// Returns user's albums from the legacy system
func (h *LegacyHandler) GetUserAlbums(c *gin.Context) {
	firebaseUID := c.GetString("firebase_uid")
	if firebaseUID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to find an associated Firebase UID"})
		return
	}

	ctx := c.Request.Context()

	albums, err := h.postgresService.GetUserAlbums(ctx, firebaseUID)
	if err != nil {
		// Return empty array instead of error
		albums = []models.LegacyAlbum{}
	}

	c.JSON(http.StatusOK, gin.H{"albums": albums})
}

// GetTracksByArtist handles GET /v1/legacy/artists/:artist_id/tracks
// Returns tracks for a specific artist
func (h *LegacyHandler) GetTracksByArtist(c *gin.Context) {
	artistID := c.Param("artist_id")
	if artistID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Artist ID is required"})
		return
	}

	ctx := c.Request.Context()

	tracks, err := h.postgresService.GetTracksByArtist(ctx, artistID)
	if err != nil {
		// Return empty array instead of error
		tracks = []models.LegacyTrack{}
	}

	c.JSON(http.StatusOK, gin.H{"tracks": tracks})
}

// GetTracksByAlbum handles GET /v1/legacy/albums/:album_id/tracks
// Returns tracks for a specific album
func (h *LegacyHandler) GetTracksByAlbum(c *gin.Context) {
	albumID := c.Param("album_id")
	if albumID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Album ID is required"})
		return
	}

	ctx := c.Request.Context()

	tracks, err := h.postgresService.GetTracksByAlbum(ctx, albumID)
	if err != nil {
		// Return empty array instead of error
		tracks = []models.LegacyTrack{}
	}

	c.JSON(http.StatusOK, gin.H{"tracks": tracks})
}
