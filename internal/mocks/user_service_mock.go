package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/wavlake/api/internal/models"
	"github.com/wavlake/api/internal/services"
)

type MockUserService struct {
	mock.Mock
}

// Ensure MockUserService implements UserServiceInterface
var _ services.UserServiceInterface = (*MockUserService)(nil)

func (m *MockUserService) LinkPubkeyToUser(ctx context.Context, pubkey, firebaseUID string) error {
	args := m.Called(ctx, pubkey, firebaseUID)
	return args.Error(0)
}

func (m *MockUserService) UnlinkPubkeyFromUser(ctx context.Context, pubkey, firebaseUID string) error {
	args := m.Called(ctx, pubkey, firebaseUID)
	return args.Error(0)
}

func (m *MockUserService) GetLinkedPubkeys(ctx context.Context, firebaseUID string) ([]models.NostrAuth, error) {
	args := m.Called(ctx, firebaseUID)
	return args.Get(0).([]models.NostrAuth), args.Error(1)
}

func (m *MockUserService) GetFirebaseUIDByPubkey(ctx context.Context, pubkey string) (string, error) {
	args := m.Called(ctx, pubkey)
	return args.String(0), args.Error(1)
}

func (m *MockUserService) GetUserEmail(ctx context.Context, firebaseUID string) (string, error) {
	args := m.Called(ctx, firebaseUID)
	return args.String(0), args.Error(1)
}
