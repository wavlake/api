package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/wavlake/api/internal/models"
	"google.golang.org/api/iterator"
)

// FirebaseLinkGuard ensures that a pubkey is linked to a Firebase UID
type FirebaseLinkGuard struct {
	firestoreClient *firestore.Client
}

// NewFirebaseLinkGuard creates a new Firebase link guard middleware
func NewFirebaseLinkGuard(firestoreClient *firestore.Client) *FirebaseLinkGuard {
	return &FirebaseLinkGuard{
		firestoreClient: firestoreClient,
	}
}

// Middleware checks if the authenticated pubkey is linked to a Firebase UID
// This middleware should be used after NIP-98 signature validation middleware
func (g *FirebaseLinkGuard) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the pubkey from context (should be set by NIP-98 middleware)
		pubkey, exists := c.Get("pubkey")
		if !exists || pubkey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing pubkey in context"})
			c.Abort()
			return
		}

		pubkeyStr, ok := pubkey.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid pubkey format"})
			c.Abort()
			return
		}

		// Check if pubkey is linked to a Firebase UID
		ctx := context.Background()
		auth, err := g.getNostrAuth(ctx, pubkeyStr)
		if err != nil {
			log.Printf("Firebase link check failed for pubkey %s: %v", pubkeyStr, err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User is not authorized. Please link your Nostr identity to your Firebase account to access this feature.",
			})
			c.Abort()
			return
		}

		if !auth.Active {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User is not authorized. Account is inactive.",
			})
			c.Abort()
			return
		}

		// Set firebase_uid in context for downstream handlers
		c.Set("firebase_uid", auth.FirebaseUID)
		c.Next()
	}
}

// getNostrAuth retrieves NostrAuth record for the given pubkey
func (g *FirebaseLinkGuard) getNostrAuth(ctx context.Context, pubkey string) (*models.NostrAuth, error) {
	query := g.firestoreClient.Collection("nostr_auth").Where("pubkey", "==", pubkey).Where("active", "==", true).Limit(1)
	iter := query.Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("pubkey not linked to Firebase UID")
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
