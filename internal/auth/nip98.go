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
	gonostr "github.com/nbd-wtf/go-nostr"
	"github.com/wavlake/api/internal/models"
	"github.com/wavlake/api/pkg/nostr"
	"google.golang.org/api/iterator"
)

type NIP98Middleware struct {
	firestoreClient *firestore.Client
}

func NewNIP98Middleware(ctx context.Context, projectID string) (*NIP98Middleware, error) {
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create firestore client: %w", err)
	}

	return &NIP98Middleware{
		firestoreClient: client,
	}, nil
}

func (m *NIP98Middleware) Close() error {
	return m.firestoreClient.Close()
}

// SignatureValidationMiddleware validates NIP-98 signatures without database lookup
func (m *NIP98Middleware) SignatureValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/heartbeat" {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		if !strings.HasPrefix(authHeader, "Nostr ") {
			http.Error(w, "Invalid Authorization scheme", http.StatusUnauthorized)
			return
		}

		encodedEvent := strings.TrimPrefix(authHeader, "Nostr ")
		eventData, err := base64.StdEncoding.DecodeString(encodedEvent)
		if err != nil {
			http.Error(w, "Invalid base64 encoding", http.StatusUnauthorized)
			return
		}

		var gonostrEvent gonostr.Event
		if err := json.Unmarshal(eventData, &gonostrEvent); err != nil {
			http.Error(w, "Invalid event JSON", http.StatusUnauthorized)
			return
		}

		event := &nostr.Event{Event: &gonostrEvent}

		if event.Kind != 27235 {
			http.Error(w, "Invalid event kind", http.StatusUnauthorized)
			return
		}

		now := time.Now().Unix()
		createdAt := int64(event.CreatedAt)
		if now-createdAt > 60 || createdAt > now+60 {
			http.Error(w, "Event timestamp out of range", http.StatusUnauthorized)
			return
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

		if urlTag != fullURL {
			log.Printf("URL mismatch: expected %s, got %s", fullURL, urlTag)
			http.Error(w, "URL mismatch", http.StatusUnauthorized)
			return
		}

		if methodTag != r.Method {
			http.Error(w, "Method mismatch", http.StatusUnauthorized)
			return
		}

		if !event.Verify() {
			http.Error(w, "Invalid event signature", http.StatusUnauthorized)
			return
		}

		// Only set the pubkey in context, no database lookup
		ctx := context.WithValue(r.Context(), "pubkey", event.PubKey)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// DatabaseLookupMiddleware performs database lookup for authenticated pubkey
func (m *NIP98Middleware) DatabaseLookupMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the pubkey from context (should be set by SignatureValidationMiddleware)
		pubkey, exists := r.Context().Value("pubkey").(string)
		if !exists || pubkey == "" {
			http.Error(w, "Missing pubkey in context", http.StatusUnauthorized)
			return
		}

		ctx := context.Background()
		auth, err := m.getNostrAuth(ctx, pubkey)
		if err != nil {
			log.Printf("Failed to get auth: %v", err)
			http.Error(w, "Authentication failed", http.StatusUnauthorized)
			return
		}

		if !auth.Active {
			http.Error(w, "Account inactive", http.StatusUnauthorized)
			return
		}

		go m.updateLastUsed(context.Background(), pubkey)

		// Add firebase_uid to context
		ctx = context.WithValue(r.Context(), "firebase_uid", auth.FirebaseUID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Middleware provides the full NIP-98 authentication (signature + database lookup)
func (m *NIP98Middleware) Middleware(next http.Handler) http.Handler {
	return m.SignatureValidationMiddleware(m.DatabaseLookupMiddleware(next))
}

func (m *NIP98Middleware) getNostrAuth(ctx context.Context, pubkey string) (*models.NostrAuth, error) {
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

func (m *NIP98Middleware) updateLastUsed(ctx context.Context, pubkey string) {
	query := m.firestoreClient.Collection("nostr_auth").Where("pubkey", "==", pubkey).Limit(1)
	iter := query.Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err != nil {
		return
	}

	_, err = doc.Ref.Update(ctx, []firestore.Update{
		{Path: "last_used_at", Value: time.Now()},
	})
	if err != nil {
		log.Printf("Failed to update last_used_at: %v", err)
	}
}

func (m *NIP98Middleware) operator(handler http.Handler) http.Handler {
	return m.Middleware(handler)
}
