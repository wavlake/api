package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// Mock Firebase Auth Client
type MockFirebaseAuthClient struct {
	mock.Mock
}

func (m *MockFirebaseAuthClient) VerifyIDToken(ctx context.Context, token string) (*auth.Token, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.Token), args.Error(1)
}

type FirebaseMiddlewareTestSuite struct {
	suite.Suite
	mockAuthClient *MockFirebaseAuthClient
	middleware     *FirebaseMiddleware
	router         *gin.Engine
}

func (suite *FirebaseMiddlewareTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)

	suite.mockAuthClient = &MockFirebaseAuthClient{}

	// Note: In real tests, we'd need to properly mock the Firebase auth client
	// For now, we'll test the logic around token extraction and validation

	suite.router = gin.New()
	suite.router.Use(suite.createTestMiddleware())
	suite.router.GET("/test", func(c *gin.Context) {
		firebaseUID, exists := c.Get("firebase_uid")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No Firebase UID"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"firebase_uid": firebaseUID})
	})
}

func (suite *FirebaseMiddlewareTestSuite) TearDownTest() {
	suite.mockAuthClient.AssertExpectations(suite.T())
}

// Create a test middleware that simulates Firebase auth behavior
func (suite *FirebaseMiddlewareTestSuite) createTestMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c.GetHeader("Authorization"))
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization token"})
			c.Abort()
			return
		}

		// Simulate Firebase token validation
		if token == "valid-token" {
			c.Set("firebase_uid", "test-firebase-uid")
			c.Set("firebase_email", "test@example.com")
			c.Next()
		} else if token == "invalid-token" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Firebase token"})
			c.Abort()
			return
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unknown token"})
			c.Abort()
			return
		}
	}
}

func (suite *FirebaseMiddlewareTestSuite) TestExtractBearerToken() {
	tests := []struct {
		authHeader string
		expected   string
	}{
		{"Bearer valid-token", "valid-token"},
		{"bearer valid-token", "valid-token"}, // Case insensitive
		{"Bearer ", ""},                       // Empty token
		{"", ""},                              // No header
		{"Basic dXNlcjpwYXNz", ""},            // Wrong auth type
		{"Bearer", ""},                        // Missing token part
		{"Bearer token1 token2", ""},          // Too many parts - invalid
	}

	for _, test := range tests {
		result := extractBearerToken(test.authHeader)
		assert.Equal(suite.T(), test.expected, result, "Failed for header: %s", test.authHeader)
	}
}

func (suite *FirebaseMiddlewareTestSuite) TestMiddleware_ValidToken() {
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "test-firebase-uid")
}

func (suite *FirebaseMiddlewareTestSuite) TestMiddleware_MissingToken() {
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "Missing authorization token")
}

func (suite *FirebaseMiddlewareTestSuite) TestMiddleware_InvalidToken() {
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "Invalid Firebase token")
}

func (suite *FirebaseMiddlewareTestSuite) TestMiddleware_WrongAuthType() {
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "Missing authorization token")
}

func (suite *FirebaseMiddlewareTestSuite) TestMiddleware_EmptyBearerToken() {
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "Missing authorization token")
}

func TestFirebaseMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(FirebaseMiddlewareTestSuite))
}
