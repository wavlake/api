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

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	gonostr "github.com/nbd-wtf/go-nostr"
	"github.com/wavlake/api/pkg/nostr"
)

type DualAuthMiddleware struct {
	firebaseAuth *auth.Client
}

func NewDualAuthMiddleware(firebaseAuth *auth.Client) *DualAuthMiddleware {
	return &DualAuthMiddleware{
		firebaseAuth: firebaseAuth,
	}
}

func (m *DualAuthMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Validate Firebase token
		firebaseToken := extractBearerToken(c.GetHeader("Authorization"))
		if firebaseToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing Firebase authorization token"})
			c.Abort()
			return
		}

		firebaseUser, err := m.firebaseAuth.VerifyIDToken(context.Background(), firebaseToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Firebase token"})
			c.Abort()
			return
		}

		// 2. Validate NIP-98 signature
		nip98Event, err := m.validateNIP98(c.Request)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Invalid NIP-98 signature: %v", err)})
			c.Abort()
			return
		}

		// 3. Store both auth contexts
		c.Set("firebase_uid", firebaseUser.UID)
		if email, ok := firebaseUser.Claims["email"].(string); ok {
			c.Set("firebase_email", email)
		}
		c.Set("nostr_pubkey", nip98Event.PubKey)
		c.Next()
	}
}

func (m *DualAuthMiddleware) validateNIP98(r *http.Request) (*nostr.Event, error) {
	// Check for Nostr authorization header
	nostrHeader := r.Header.Get("X-Nostr-Authorization")
	if nostrHeader == "" {
		return nil, fmt.Errorf("missing X-Nostr-Authorization header")
	}

	if !strings.HasPrefix(nostrHeader, "Nostr ") {
		return nil, fmt.Errorf("invalid Nostr authorization scheme")
	}

	encodedEvent := strings.TrimPrefix(nostrHeader, "Nostr ")
	eventData, err := base64.StdEncoding.DecodeString(encodedEvent)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 encoding: %w", err)
	}

	var gonostrEvent gonostr.Event
	if err := json.Unmarshal(eventData, &gonostrEvent); err != nil {
		return nil, fmt.Errorf("invalid event JSON: %w", err)
	}

	event := &nostr.Event{Event: &gonostrEvent}

	// Validate NIP-98 requirements
	if event.Kind != 27235 {
		return nil, fmt.Errorf("invalid event kind: expected 27235, got %d", event.Kind)
	}

	now := time.Now().Unix()
	createdAt := int64(event.CreatedAt)
	if now-createdAt > 60 || createdAt > now+60 {
		return nil, fmt.Errorf("event timestamp out of range")
	}

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

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	// Check X-Forwarded-Proto header for proxy/load balancer setups (like Cloud Run)
	if proto := r.Header.Get("X-Forwarded-Proto"); proto == "https" {
		scheme = "https"
	}
	fullURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI)

	log.Printf("NIP-98 Debug - URL check: fullURL='%s', urlTag='%s'", fullURL, urlTag)
	if urlTag != fullURL {
		return nil, fmt.Errorf("URL mismatch: expected %s, got %s", fullURL, urlTag)
	}

	log.Printf("NIP-98 Debug - Method check: method='%s', methodTag='%s'", r.Method, methodTag)
	if methodTag != r.Method {
		return nil, fmt.Errorf("method mismatch: expected %s, got %s", r.Method, methodTag)
	}

	log.Printf("NIP-98 Debug - About to verify signature for event ID: %s", event.ID)
	if !event.Verify() {
		log.Printf("NIP-98 Debug - Signature verification failed for event: %+v", event)
		return nil, fmt.Errorf("invalid event signature")
	}

	return event, nil
}
