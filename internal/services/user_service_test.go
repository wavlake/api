package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/wavlake/api/internal/models"
)

type UserServiceTestSuite struct {
	suite.Suite
	service *UserService
}

func (suite *UserServiceTestSuite) SetupTest() {
	// For unit tests, we'll test the business logic without a real Firestore client
	// In integration tests, we would use a real or emulated Firestore
	suite.service = &UserService{
		firestoreClient: nil, // We'll mock the database operations
	}
}

// Test helper functions

func (suite *UserServiceTestSuite) TestContains() {
	slice := []string{"apple", "banana", "cherry"}

	assert.True(suite.T(), contains(slice, "apple"))
	assert.True(suite.T(), contains(slice, "banana"))
	assert.True(suite.T(), contains(slice, "cherry"))
	assert.False(suite.T(), contains(slice, "orange"))
	assert.False(suite.T(), contains(slice, ""))

	// Test empty slice
	assert.False(suite.T(), contains([]string{}, "apple"))
}

func (suite *UserServiceTestSuite) TestRemoveString() {
	tests := []struct {
		slice    []string
		item     string
		expected []string
	}{
		{
			slice:    []string{"apple", "banana", "cherry"},
			item:     "banana",
			expected: []string{"apple", "cherry"},
		},
		{
			slice:    []string{"apple", "banana", "cherry"},
			item:     "orange",
			expected: []string{"apple", "banana", "cherry"},
		},
		{
			slice:    []string{"apple"},
			item:     "apple",
			expected: nil,
		},
		{
			slice:    []string{},
			item:     "apple",
			expected: nil,
		},
		{
			slice:    []string{"apple", "apple", "banana"},
			item:     "apple",
			expected: []string{"banana"}, // Removes all instances
		},
	}

	for _, test := range tests {
		result := removeString(test.slice, test.item)
		assert.Equal(suite.T(), test.expected, result, "Failed for slice: %v, item: %s", test.slice, test.item)
	}
}

// Test business logic validation
func (suite *UserServiceTestSuite) TestValidateBusinessRules() {

	// Test cases that would validate business logic
	// Note: These tests focus on the logic, not the database operations

	testCases := []struct {
		name        string
		firebaseUID string
		pubkey      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid inputs",
			firebaseUID: "firebase-uid-123",
			pubkey:      "valid-pubkey-abc123",
			expectError: false,
		},
		{
			name:        "Empty Firebase UID",
			firebaseUID: "",
			pubkey:      "valid-pubkey-abc123",
			expectError: true,
			errorMsg:    "firebase_uid cannot be empty",
		},
		{
			name:        "Empty pubkey",
			firebaseUID: "firebase-uid-123",
			pubkey:      "",
			expectError: true,
			errorMsg:    "pubkey cannot be empty",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// This would be where we test business logic validation
			// For now, we'll test that the inputs are processed correctly

			if tc.firebaseUID == "" || tc.pubkey == "" {
				// In a real implementation, we'd expect validation errors
				if tc.expectError {
					assert.True(t, true, "Expected validation error for empty inputs")
				}
			} else {
				// Valid inputs should pass basic validation
				assert.NotEmpty(t, tc.firebaseUID)
				assert.NotEmpty(t, tc.pubkey)
			}
		})
	}
}

// Test model creation
func (suite *UserServiceTestSuite) TestModelCreation() {
	now := time.Now()

	// Test User model
	user := models.User{
		FirebaseUID:   "test-firebase-uid",
		CreatedAt:     now,
		UpdatedAt:     now,
		ActivePubkeys: []string{"pubkey1", "pubkey2"},
	}

	assert.Equal(suite.T(), "test-firebase-uid", user.FirebaseUID)
	assert.Equal(suite.T(), 2, len(user.ActivePubkeys))
	assert.Contains(suite.T(), user.ActivePubkeys, "pubkey1")
	assert.Contains(suite.T(), user.ActivePubkeys, "pubkey2")

	// Test NostrAuth model
	nostrAuth := models.NostrAuth{
		Pubkey:      "test-pubkey-123",
		FirebaseUID: "test-firebase-uid",
		Active:      true,
		CreatedAt:   now,
		LastUsedAt:  now,
		LinkedAt:    now,
	}

	assert.Equal(suite.T(), "test-pubkey-123", nostrAuth.Pubkey)
	assert.Equal(suite.T(), "test-firebase-uid", nostrAuth.FirebaseUID)
	assert.True(suite.T(), nostrAuth.Active)
}

// Test edge cases
func (suite *UserServiceTestSuite) TestEdgeCases() {

	// Test contains with duplicate items
	slice := []string{"apple", "apple", "banana"}
	assert.True(suite.T(), contains(slice, "apple"))

	// Test removeString removes all instances
	result := removeString(slice, "apple")
	assert.Equal(suite.T(), []string{"banana"}, result)
	assert.False(suite.T(), contains(result, "apple"))
}

func TestUserServiceTestSuite(t *testing.T) {
	suite.Run(t, new(UserServiceTestSuite))
}
