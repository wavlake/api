package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wavlake/api/internal/services"
)

type AuthHandlers struct {
	userService services.UserServiceInterface
}

func NewAuthHandlers(userService services.UserServiceInterface) *AuthHandlers {
	return &AuthHandlers{
		userService: userService,
	}
}

// LinkPubkeyRequest represents the request body for linking a pubkey
type LinkPubkeyRequest struct {
	PubKey string `json:"pubkey,omitempty"`
}

// LinkPubkeyResponse represents the response for linking a pubkey
type LinkPubkeyResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	FirebaseUID string `json:"firebase_uid"`
	PubKey      string `json:"pubkey"`
	LinkedAt    string `json:"linked_at"`
}

// LinkPubkey handles POST /v1/auth/link-pubkey
// Requires dual authentication (Firebase + NIP-98)
func (h *AuthHandlers) LinkPubkey(c *gin.Context) {
	// Get auth info from context (set by DualAuthMiddleware)
	firebaseUID, exists := c.Get("firebase_uid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing Firebase authentication"})
		return
	}

	nostrPubkey, exists := c.Get("nostr_pubkey")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing Nostr authentication"})
		return
	}

	pubkey := nostrPubkey.(string)
	uid := firebaseUID.(string)

	// Optional: validate request body pubkey matches auth pubkey
	var req LinkPubkeyRequest
	if err := c.ShouldBindJSON(&req); err == nil && req.PubKey != "" {
		if req.PubKey != pubkey {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Request pubkey does not match authenticated pubkey"})
			return
		}
	}

	// Link the pubkey to the Firebase user
	err := h.userService.LinkPubkeyToUser(c.Request.Context(), pubkey, uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response := LinkPubkeyResponse{
		Success:     true,
		Message:     "Pubkey linked successfully to Firebase account",
		FirebaseUID: uid,
		PubKey:      pubkey,
		LinkedAt:    time.Now().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}

// UnlinkPubkeyRequest represents the request body for unlinking a pubkey
type UnlinkPubkeyRequest struct {
	PubKey string `json:"pubkey" binding:"required"`
}

// UnlinkPubkeyResponse represents the response for unlinking a pubkey
type UnlinkPubkeyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	PubKey  string `json:"pubkey"`
}

// UnlinkPubkey handles POST /v1/auth/unlink-pubkey
// Requires Firebase authentication only
func (h *AuthHandlers) UnlinkPubkey(c *gin.Context) {
	// Get Firebase UID from context (set by FirebaseMiddleware)
	firebaseUID, exists := c.Get("firebase_uid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing Firebase authentication"})
		return
	}

	var req UnlinkPubkeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	uid := firebaseUID.(string)

	// Unlink the pubkey from the Firebase user
	err := h.userService.UnlinkPubkeyFromUser(c.Request.Context(), req.PubKey, uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response := UnlinkPubkeyResponse{
		Success: true,
		Message: "Pubkey unlinked successfully from Firebase account",
		PubKey:  req.PubKey,
	}

	c.JSON(http.StatusOK, response)
}

// LinkedPubkeyInfo represents pubkey information in the response
type LinkedPubkeyInfo struct {
	PubKey        string `json:"pubkey"`
	DisplayPubkey string `json:"display_pubkey"`
	LinkedAt      string `json:"linked_at"`
	LastUsedAt    string `json:"last_used_at,omitempty"`
}

// GetLinkedPubkeysResponse represents the response for getting linked pubkeys
type GetLinkedPubkeysResponse struct {
	Success       bool               `json:"success"`
	FirebaseUID   string             `json:"firebase_uid"`
	LinkedPubkeys []LinkedPubkeyInfo `json:"linked_pubkeys"`
}

// GetLinkedPubkeys handles GET /v1/auth/get-linked-pubkeys
// Requires Firebase authentication only
func (h *AuthHandlers) GetLinkedPubkeys(c *gin.Context) {
	// Get Firebase UID from context (set by FirebaseMiddleware)
	firebaseUID, exists := c.Get("firebase_uid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing Firebase authentication"})
		return
	}

	uid := firebaseUID.(string)

	// Get linked pubkeys for the user
	pubkeys, err := h.userService.GetLinkedPubkeys(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve linked pubkeys"})
		return
	}

	// Convert to response format
	var linkedPubkeys []LinkedPubkeyInfo
	for _, p := range pubkeys {
		info := LinkedPubkeyInfo{
			PubKey:        p.Pubkey,
			DisplayPubkey: p.DisplayPubkey,
			LinkedAt:      p.LinkedAt.Format(time.RFC3339),
		}
		
		if !p.LastUsedAt.IsZero() {
			info.LastUsedAt = p.LastUsedAt.Format(time.RFC3339)
		}
		
		linkedPubkeys = append(linkedPubkeys, info)
	}

	response := GetLinkedPubkeysResponse{
		Success:       true,
		FirebaseUID:   uid,
		LinkedPubkeys: linkedPubkeys,
	}

	c.JSON(http.StatusOK, response)
}