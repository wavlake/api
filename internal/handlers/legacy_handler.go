package handlers

import (
	"log"
	"net/http"

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
	pubkey := c.GetString("pubkey")
	log.Printf("Legacy Debug - pubkey: %s, firebase_uid: %s", pubkey, firebaseUID)

	if firebaseUID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to find an associated Firebase UID"})
		return
	}

	ctx := c.Request.Context()

	// Get user data
	log.Printf("Legacy Debug - Querying PostgreSQL for Firebase UID: %s", firebaseUID)
	user, err := h.postgresService.GetUserByFirebaseUID(ctx, firebaseUID)
	if err != nil {
		log.Printf("Legacy Debug - GetUserByFirebaseUID error: %v", err)
		// Return empty response for user not found
		response := UserMetadataResponse{
			User:    nil,
			Artists: []models.LegacyArtist{},
			Albums:  []models.LegacyAlbum{},
			Tracks:  []models.LegacyTrack{},
		}
		c.JSON(http.StatusOK, response)
		return
	}

	// Get associated data (continue even if some fail)
	artists, err := h.postgresService.GetUserArtists(ctx, firebaseUID)
	if err != nil {
		log.Printf("Legacy Debug - GetUserArtists error: %v", err)
		artists = []models.LegacyArtist{}
	} else {
		log.Printf("Legacy Debug - Found %d artists", len(artists))
	}

	albums, err := h.postgresService.GetUserAlbums(ctx, firebaseUID)
	if err != nil {
		log.Printf("Legacy Debug - GetUserAlbums error: %v", err)
		albums = []models.LegacyAlbum{}
	} else {
		log.Printf("Legacy Debug - Found %d albums", len(albums))
	}

	tracks, err := h.postgresService.GetUserTracks(ctx, firebaseUID)
	if err != nil {
		log.Printf("Legacy Debug - GetUserTracks error: %v", err)
		tracks = []models.LegacyTrack{}
	} else {
		log.Printf("Legacy Debug - Found %d tracks", len(tracks))
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
