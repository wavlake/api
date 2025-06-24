package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/wavlake/api/internal/mocks"
	"github.com/wavlake/api/internal/models"
)

type AuthHandlerTestSuite struct {
	suite.Suite
	router      *gin.Engine
	userService *mocks.MockUserService
	handlers    *AuthHandlers
}

func (suite *AuthHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)

	suite.userService = &mocks.MockUserService{}
	suite.handlers = NewAuthHandlers(suite.userService)

	suite.router = gin.New()

	// Setup routes with mock middleware that sets auth context
	auth := suite.router.Group("/v1/auth")
	{
		auth.GET("/get-linked-pubkeys", suite.mockFirebaseAuth(), suite.handlers.GetLinkedPubkeys)
		auth.POST("/unlink-pubkey", suite.mockFirebaseAuth(), suite.handlers.UnlinkPubkey)
		auth.POST("/link-pubkey", suite.mockDualAuth(), suite.handlers.LinkPubkey)
	}
}

func (suite *AuthHandlerTestSuite) TearDownTest() {
	suite.userService.AssertExpectations(suite.T())
}

// Mock middleware that sets Firebase auth context
func (suite *AuthHandlerTestSuite) mockFirebaseAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("firebase_uid", "test-firebase-uid")
		c.Set("firebase_email", "test@example.com")
		c.Next()
	}
}

// Mock middleware that sets dual auth context
func (suite *AuthHandlerTestSuite) mockDualAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("firebase_uid", "test-firebase-uid")
		c.Set("firebase_email", "test@example.com")
		c.Set("nostr_pubkey", "test-pubkey-123")
		c.Next()
	}
}

// Test GetLinkedPubkeys endpoint
func (suite *AuthHandlerTestSuite) TestGetLinkedPubkeys_Success() {
	// Setup mock response
	mockPubkeys := []models.NostrAuth{
		{
			Pubkey:        "pubkey1",
			FirebaseUID:   "test-firebase-uid",
			Active:        true,
			LinkedAt:      time.Now(),
			LastUsedAt:    time.Now(),
			DisplayPubkey: "pubkey1...123",
		},
		{
			Pubkey:        "pubkey2",
			FirebaseUID:   "test-firebase-uid",
			Active:        true,
			LinkedAt:      time.Now(),
			DisplayPubkey: "pubkey2...456",
		},
	}

	suite.userService.On("GetLinkedPubkeys", mock.Anything, "test-firebase-uid").Return(mockPubkeys, nil)

	// Make request
	req, _ := http.NewRequest("GET", "/v1/auth/get-linked-pubkeys", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response GetLinkedPubkeysResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)
	assert.Equal(suite.T(), "test-firebase-uid", response.FirebaseUID)
	assert.Len(suite.T(), response.LinkedPubkeys, 2)
	assert.Equal(suite.T(), "pubkey1", response.LinkedPubkeys[0].PubKey)
	assert.Equal(suite.T(), "pubkey1...123", response.LinkedPubkeys[0].DisplayPubkey)
}

func (suite *AuthHandlerTestSuite) TestGetLinkedPubkeys_ServiceError() {
	suite.userService.On("GetLinkedPubkeys", mock.Anything, "test-firebase-uid").Return([]models.NostrAuth{}, errors.New("database error"))

	req, _ := http.NewRequest("GET", "/v1/auth/get-linked-pubkeys", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(suite.T(), "Failed to retrieve linked pubkeys", response["error"])
}

// Test UnlinkPubkey endpoint
func (suite *AuthHandlerTestSuite) TestUnlinkPubkey_Success() {
	requestBody := UnlinkPubkeyRequest{
		PubKey: "test-pubkey-to-unlink",
	}

	suite.userService.On("UnlinkPubkeyFromUser", mock.Anything, "test-pubkey-to-unlink", "test-firebase-uid").Return(nil)

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/v1/auth/unlink-pubkey", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response UnlinkPubkeyResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)
	assert.Equal(suite.T(), "test-pubkey-to-unlink", response.PubKey)
	assert.Contains(suite.T(), response.Message, "unlinked successfully")
}

func (suite *AuthHandlerTestSuite) TestUnlinkPubkey_InvalidRequest() {
	req, _ := http.NewRequest("POST", "/v1/auth/unlink-pubkey", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(suite.T(), "Invalid request body", response["error"])
}

func (suite *AuthHandlerTestSuite) TestUnlinkPubkey_ServiceError() {
	requestBody := UnlinkPubkeyRequest{
		PubKey: "test-pubkey",
	}

	suite.userService.On("UnlinkPubkeyFromUser", mock.Anything, "test-pubkey", "test-firebase-uid").Return(errors.New("pubkey not found"))

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/v1/auth/unlink-pubkey", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(suite.T(), "pubkey not found", response["error"])
}

// Test LinkPubkey endpoint
func (suite *AuthHandlerTestSuite) TestLinkPubkey_Success() {
	suite.userService.On("LinkPubkeyToUser", mock.Anything, "test-pubkey-123", "test-firebase-uid").Return(nil)

	req, _ := http.NewRequest("POST", "/v1/auth/link-pubkey", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response LinkPubkeyResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)
	assert.Equal(suite.T(), "test-firebase-uid", response.FirebaseUID)
	assert.Equal(suite.T(), "test-pubkey-123", response.PubKey)
	assert.Contains(suite.T(), response.Message, "linked successfully")
}

func (suite *AuthHandlerTestSuite) TestLinkPubkey_WithValidationSuccess() {
	requestBody := LinkPubkeyRequest{
		PubKey: "test-pubkey-123", // Should match the one from dual auth middleware
	}

	suite.userService.On("LinkPubkeyToUser", mock.Anything, "test-pubkey-123", "test-firebase-uid").Return(nil)

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/v1/auth/link-pubkey", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response LinkPubkeyResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)
}

func (suite *AuthHandlerTestSuite) TestLinkPubkey_PubkeyMismatch() {
	requestBody := LinkPubkeyRequest{
		PubKey: "different-pubkey", // Different from the one in dual auth middleware
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/v1/auth/link-pubkey", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(suite.T(), "Request pubkey does not match authenticated pubkey", response["error"])
}

func (suite *AuthHandlerTestSuite) TestLinkPubkey_ServiceError() {
	suite.userService.On("LinkPubkeyToUser", mock.Anything, "test-pubkey-123", "test-firebase-uid").Return(errors.New("pubkey already linked to different user"))

	req, _ := http.NewRequest("POST", "/v1/auth/link-pubkey", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(suite.T(), "pubkey already linked to different user", response["error"])
}

// Test missing auth context scenarios
func (suite *AuthHandlerTestSuite) TestEndpoints_MissingAuth() {
	// Create router without auth middleware
	router := gin.New()
	auth := router.Group("/v1/auth")
	{
		auth.GET("/get-linked-pubkeys", suite.handlers.GetLinkedPubkeys)
		auth.POST("/unlink-pubkey", suite.handlers.UnlinkPubkey)
		auth.POST("/link-pubkey", suite.handlers.LinkPubkey)
	}

	tests := []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/v1/auth/get-linked-pubkeys", ""},
		{"POST", "/v1/auth/unlink-pubkey", `{"pubkey":"test"}`},
		{"POST", "/v1/auth/link-pubkey", "{}"},
	}

	for _, test := range tests {
		req, _ := http.NewRequest(test.method, test.path, bytes.NewBufferString(test.body))
		if test.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Contains(suite.T(), response["error"].(string), "authentication")
	}
}

func TestAuthHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(AuthHandlerTestSuite))
}
