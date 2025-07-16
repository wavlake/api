package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	gonostr "github.com/nbd-wtf/go-nostr"
	"github.com/wavlake/api/internal/models"
	"github.com/wavlake/api/pkg/nostr"
	"google.golang.org/api/iterator"
)

// NIP98AuthResult represents the result of NIP-98 authentication attempt
type NIP98AuthResult struct {
	Success     bool
	FirebaseUID string
	ErrorType   string
	ErrorMsg    string
}

// FlexibleAuthMiddleware provides authentication via Firebase Bearer token or NIP-98 signature
// with graceful fallback between the two methods
type FlexibleAuthMiddleware struct {
	firebaseAuth    *auth.Client
	firestoreClient *firestore.Client
}

// NewFlexibleAuthMiddleware creates a new flexible authentication middleware
func NewFlexibleAuthMiddleware(firebaseAuth *auth.Client, firestoreClient *firestore.Client) *FlexibleAuthMiddleware {
	return &FlexibleAuthMiddleware{
		firebaseAuth:    firebaseAuth,
		firestoreClient: firestoreClient,
	}
}

// Middleware returns the Gin middleware handler that tries Firebase auth first, then NIP-98
func (m *FlexibleAuthMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip heartbeat endpoint
		if c.Request.URL.Path == "/heartbeat" {
			c.Next()
			return
		}

		// First try Firebase Bearer token authentication
		if firebaseUID := m.tryFirebaseAuth(c); firebaseUID != "" {
			// Firebase auth successful
			c.Set("firebase_uid", firebaseUID)
			c.Set("auth_method", "firebase")
			c.Next()
			return
		}

		// Firebase failed, try NIP-98 signature authentication
		nip98Result := m.tryNIP98Auth(c)
		if nip98Result.Success {
			// NIP-98 auth successful
			c.Set("firebase_uid", nip98Result.FirebaseUID)
			c.Set("auth_method", "nip98")
			c.Next()
			return
		}

		// Both authentication methods failed - provide specific error message
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": nip98Result.ErrorMsg,
		})
		c.Abort()
	}
}

// tryFirebaseAuth attempts to authenticate using Firebase Bearer token
// Returns firebase_uid on success, empty string on failure
func (m *FlexibleAuthMiddleware) tryFirebaseAuth(c *gin.Context) string {
	// Extract Bearer token from Authorization header
	token := extractBearerToken(c.GetHeader("Authorization"))
	if token == "" {
		// Also check X-Firebase-Token header
		token = c.GetHeader("X-Firebase-Token")
	}
	if token == "" {
		return ""
	}

	// Verify Firebase token
	firebaseToken, err := m.firebaseAuth.VerifyIDToken(context.Background(), token)
	if err != nil {
		log.Printf("Firebase token verification failed: %v", err)
		return ""
	}

	// Store additional Firebase user info in context
	if email, ok := firebaseToken.Claims["email"].(string); ok {
		c.Set("firebase_email", email)
	}

	return firebaseToken.UID
}

// tryNIP98Auth attempts to authenticate using NIP-98 signature
// Returns detailed result with success status and specific error information
func (m *FlexibleAuthMiddleware) tryNIP98Auth(c *gin.Context) NIP98AuthResult {
	// First validate the NIP-98 signature
	pubkey := m.validateNIP98Signature(c.Request)
	if pubkey == "" {
		return NIP98AuthResult{
			Success:   false,
			ErrorType: "invalid_signature",
			ErrorMsg:  "Invalid or missing NIP-98 signature",
		}
	}

	// Look up the linked Firebase UID for this pubkey
	ctx := context.Background()
	auth, err := m.getNostrAuth(ctx, pubkey)
	if err != nil {
		log.Printf("Failed to get auth for pubkey %s: %v", pubkey, err)
		if err.Error() == "pubkey not found" {
			return NIP98AuthResult{
				Success:   false,
				ErrorType: "pubkey_not_linked",
				ErrorMsg:  "Nostr pubkey not linked to Firebase account. Please link your pubkey first.",
			}
		}
		return NIP98AuthResult{
			Success:   false,
			ErrorType: "database_error",
			ErrorMsg:  "Failed to verify account linking",
		}
	}

	if !auth.Active {
		log.Printf("Account inactive for pubkey %s", pubkey)
		return NIP98AuthResult{
			Success:   false,
			ErrorType: "account_inactive",
			ErrorMsg:  "Account is inactive",
		}
	}

	// Store NIP-98 specific context
	c.Set("nostr_pubkey", pubkey)

	// Update last used timestamp in background
	go m.updateLastUsed(context.Background(), pubkey)

	return NIP98AuthResult{
		Success:     true,
		FirebaseUID: auth.FirebaseUID,
	}
}

// validateNIP98Signature validates the NIP-98 signature and returns the pubkey
// Returns empty string if validation fails
func (m *FlexibleAuthMiddleware) validateNIP98Signature(r *http.Request) string {
	// Check for Nostr authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	// Must be "Nostr " scheme
	if !strings.HasPrefix(authHeader, "Nostr ") {
		return ""
	}

	// Decode the base64 event
	encodedEvent := strings.TrimPrefix(authHeader, "Nostr ")
	eventData, err := base64.StdEncoding.DecodeString(encodedEvent)
	if err != nil {
		log.Printf("Invalid base64 encoding in NIP-98 auth: %v", err)
		return ""
	}

	// Parse the event
	var gonostrEvent gonostr.Event
	if err := json.Unmarshal(eventData, &gonostrEvent); err != nil {
		log.Printf("Invalid event JSON in NIP-98 auth: %v", err)
		return ""
	}

	event := &nostr.Event{Event: &gonostrEvent}

	// Validate NIP-98 requirements
	if event.Kind != 27235 {
		log.Printf("Invalid event kind in NIP-98 auth: expected 27235, got %d", event.Kind)
		return ""
	}

	// Check timestamp (must be within 60 seconds)
	now := time.Now().Unix()
	createdAt := int64(event.CreatedAt)
	if now-createdAt > 60 || createdAt > now+60 {
		log.Printf("Event timestamp out of range in NIP-98 auth: now=%d, created_at=%d", now, createdAt)
		return ""
	}

	// Validate URL and method tags
	var urlTag, methodTag string
	for _, tag := range event.Tags {
		if len(tag) >= 2 {
			switch tag[0] {
			case "u":
				urlTag = tag[1]
			case "method":
				methodTag = tag[1]
			}
		}
	}

	// Construct the expected URL
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	// Check X-Forwarded-Proto header for proxy/load balancer setups (like Cloud Run)
	if proto := r.Header.Get("X-Forwarded-Proto"); proto == "https" {
		scheme = "https"
	}
	fullURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI)

	if urlTag != fullURL {
		log.Printf("URL mismatch in NIP-98 auth: expected %s, got %s", fullURL, urlTag)
		return ""
	}

	if methodTag != r.Method {
		log.Printf("Method mismatch in NIP-98 auth: expected %s, got %s", r.Method, methodTag)
		return ""
	}

	// Verify the signature
	if !event.Verify() {
		log.Printf("Invalid event signature in NIP-98 auth")
		return ""
	}

	return event.PubKey
}

// getNostrAuth retrieves the NostrAuth record for a given pubkey
func (m *FlexibleAuthMiddleware) getNostrAuth(ctx context.Context, pubkey string) (*models.NostrAuth, error) {
	query := m.firestoreClient.Collection("nostr_auth").Where("pubkey", "==", pubkey).Where("active", "==", true).Limit(1)
	iter := query.Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("pubkey not found")
	}
	if err != nil {
		return nil, err
	}

	var auth models.NostrAuth
	if err := doc.DataTo(&auth); err != nil {
		return nil, err
	}

	return &auth, nil
}

// updateLastUsed updates the last_used_at timestamp for a pubkey
func (m *FlexibleAuthMiddleware) updateLastUsed(ctx context.Context, pubkey string) {
	query := m.firestoreClient.Collection("nostr_auth").Where("pubkey", "==", pubkey).Limit(1)
	iter := query.Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err != nil {
		log.Printf("Failed to find document for last_used_at update: %v", err)
		return
	}

	_, err = doc.Ref.Update(ctx, []firestore.Update{
		{Path: "last_used_at", Value: time.Now()},
	})
	if err != nil {
		log.Printf("Failed to update last_used_at for pubkey %s: %v", pubkey, err)
	}
}

// GetAuthMethod returns the authentication method used for the current request
func GetAuthMethod(c *gin.Context) string {
	if method, exists := c.Get("auth_method"); exists {
		return method.(string)
	}
	return ""
}

// IsFirebaseAuth returns true if the request was authenticated via Firebase
func IsFirebaseAuth(c *gin.Context) bool {
	return GetAuthMethod(c) == "firebase"
}

// IsNIP98Auth returns true if the request was authenticated via NIP-98
func IsNIP98Auth(c *gin.Context) bool {
	return GetAuthMethod(c) == "nip98"
}

// GetNostrPubkey returns the Nostr pubkey if authenticated via NIP-98
func GetNostrPubkey(c *gin.Context) string {
	if pubkey, exists := c.Get("nostr_pubkey"); exists {
		return pubkey.(string)
	}
	return ""
}

// GetFirebaseUID returns the Firebase UID (available for both auth methods)
func GetFirebaseUID(c *gin.Context) string {
	if uid, exists := c.Get("firebase_uid"); exists {
		return uid.(string)
	}
	return ""
}

// GetFirebaseEmail returns the Firebase email (only available for Firebase auth)
func GetFirebaseEmail(c *gin.Context) string {
	if email, exists := c.Get("firebase_email"); exists {
		return email.(string)
	}
	return ""
}
