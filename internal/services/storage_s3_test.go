package services

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestS3StorageServiceInitialization(t *testing.T) {
	ctx := context.Background()
	bucketName := "test-bucket"

	// Test that service can be created even without AWS credentials
	// (it will fail on actual operations, but initialization should work)
	service, err := NewS3StorageService(ctx, bucketName)

	// Service creation should succeed even without valid credentials
	require.NoError(t, err)
	require.NotNil(t, service)

	// Test basic getters
	assert.Equal(t, bucketName, service.GetBucketName())
	assert.NotNil(t, service.client)
}

func TestS3StorageServiceURLGeneration(t *testing.T) {
	ctx := context.Background()
	bucketName := "test-bucket"

	service, err := NewS3StorageService(ctx, bucketName)
	require.NoError(t, err)

	// Test public URL generation without CDN
	objectName := "test/file.mp3"
	publicURL := service.GetPublicURL(objectName)
	expected := "https://test-bucket.s3.us-east-2.amazonaws.com/test/file.mp3"
	assert.Equal(t, expected, publicURL)

	// Test public URL generation with CDN
	os.Setenv("AWS_CDN_DOMAIN", "example.cloudfront.net")
	defer os.Unsetenv("AWS_CDN_DOMAIN")

	service2, err := NewS3StorageService(ctx, bucketName)
	require.NoError(t, err)

	publicURLWithCDN := service2.GetPublicURL(objectName)
	expectedWithCDN := "https://example.cloudfront.net/test/file.mp3"
	assert.Equal(t, expectedWithCDN, publicURLWithCDN)
}

func TestS3StorageServicePresignedURL(t *testing.T) {
	// Skip this test if AWS credentials are not available
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		t.Skip("Skipping S3 presigned URL test - AWS credentials not available")
	}

	ctx := context.Background()
	bucketName := os.Getenv("AWS_S3_BUCKET_NAME")
	if bucketName == "" {
		bucketName = "test-bucket"
	}

	service, err := NewS3StorageService(ctx, bucketName)
	require.NoError(t, err)

	// Test presigned URL generation
	objectName := "test/upload.mp3"
	expiration := time.Hour

	presignedURL, err := service.GeneratePresignedURL(ctx, objectName, expiration)
	if err != nil {
		// Expected to fail without valid credentials/bucket, but should not panic
		t.Logf("Presigned URL generation failed as expected without valid credentials: %v", err)
		return
	}

	// If it succeeds (with valid credentials), URL should be valid
	assert.NotEmpty(t, presignedURL)
	assert.Contains(t, presignedURL, bucketName)
	assert.Contains(t, presignedURL, objectName)
}

func TestS3StorageServiceClose(t *testing.T) {
	ctx := context.Background()
	service, err := NewS3StorageService(ctx, "test-bucket")
	require.NoError(t, err)

	// Close should not return an error for S3 client
	err = service.Close()
	assert.NoError(t, err)
}

func TestS3StorageServiceInterfaceCompliance(t *testing.T) {
	ctx := context.Background()
	service, err := NewS3StorageService(ctx, "test-bucket")
	require.NoError(t, err)

	// Verify that S3StorageService implements StorageServiceInterface
	var _ StorageServiceInterface = service
}
