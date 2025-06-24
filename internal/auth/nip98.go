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

func (m *NIP98Middleware) Middleware(next http.Handler) http.Handler {
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

		var event nostr.Event
		if err := json.Unmarshal(eventData, &event); err != nil {
			http.Error(w, "Invalid event JSON", http.StatusUnauthorized)
			return
		}

		if event.Kind != 27235 {
			http.Error(w, "Invalid event kind", http.StatusUnauthorized)
			return
		}

		now := time.Now().Unix()
		if now-event.CreatedAt > 60 || event.CreatedAt > now+60 {
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

		ctx := context.Background()
		auth, err := m.getNostrAuth(ctx, event.PubKey)
		if err != nil {
			log.Printf("Failed to get auth: %v", err)
			http.Error(w, "Authentication failed", http.StatusUnauthorized)
			return
		}

		if !auth.Active {
			http.Error(w, "Account inactive", http.StatusUnauthorized)
			return
		}

		go m.updateLastUsed(context.Background(), event.PubKey)

		ctx = context.WithValue(r.Context(), "pubkey", event.PubKey)
		ctx = context.WithValue(ctx, "firebase_uid", auth.FirebaseUID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
