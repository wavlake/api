package auth

import (
	"context"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

type FirebaseMiddleware struct {
	authClient *auth.Client
}

func NewFirebaseMiddleware(authClient *auth.Client) *FirebaseMiddleware {
	return &FirebaseMiddleware{
		authClient: authClient,
	}
}

func (m *FirebaseMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c.GetHeader("Authorization"))
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization token"})
			c.Abort()
			return
		}

		firebaseToken, err := m.authClient.VerifyIDToken(context.Background(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Firebase token"})
			c.Abort()
			return
		}

		// Store Firebase user info in context
		c.Set("firebase_uid", firebaseToken.UID)
		if email, ok := firebaseToken.Claims["email"].(string); ok {
			c.Set("firebase_email", email)
		}
		c.Next()
	}
}

func extractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}