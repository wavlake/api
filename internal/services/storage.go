package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iamcredentials/v1"
	"google.golang.org/api/option"
)

type StorageService struct {
	client     *storage.Client
	bucketName string
}

// Make client accessible for direct operations
func (s *StorageService) GetClient() *storage.Client {
	return s.client
}

func (s *StorageService) GetBucketName() string {
	return s.bucketName
}

func NewStorageService(ctx context.Context, bucketName string) (*StorageService, error) {
	// Try to use service account key if available, otherwise use default credentials
	var client *storage.Client
	var err error

	if keyPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); keyPath != "" {
		client, err = storage.NewClient(ctx, option.WithCredentialsFile(keyPath))
	} else {
		client, err = storage.NewClient(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	return &StorageService{
		client:     client,
		bucketName: bucketName,
	}, nil
}

func (s *StorageService) Close() error {
	return s.client.Close()
}

// GeneratePresignedURL creates a presigned URL for uploading files
func (s *StorageService) GeneratePresignedURL(ctx context.Context, objectName string, expiration time.Duration) (string, error) {
	// For Cloud Run with default credentials, we need to specify the service account email
	serviceAccountEmail := "api-service@wavlake-alpha.iam.gserviceaccount.com"

	// Generate a presigned URL for PUT operations
	opts := &storage.SignedURLOptions{
		Scheme:         storage.SigningSchemeV4,
		Method:         "PUT",
		Headers:        []string{"Content-Type"},
		Expires:        time.Now().Add(expiration),
		GoogleAccessID: serviceAccountEmail,
		SignBytes: func(b []byte) ([]byte, error) {
			// Use the IAM service to sign the bytes
			return signBytes(ctx, serviceAccountEmail, b)
		},
	}

	url, err := s.client.Bucket(s.bucketName).SignedURL(objectName, opts)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return url, nil
}

// GetPublicURL returns the public URL for a storage object
func (s *StorageService) GetPublicURL(objectName string) string {
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", s.bucketName, objectName)
}

// CopyObject copies an object within the same bucket
func (s *StorageService) CopyObject(ctx context.Context, srcObject, dstObject string) error {
	src := s.client.Bucket(s.bucketName).Object(srcObject)
	dst := s.client.Bucket(s.bucketName).Object(dstObject)

	_, err := dst.CopierFrom(src).Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	return nil
}

// DeleteObject deletes an object from storage
func (s *StorageService) DeleteObject(ctx context.Context, objectName string) error {
	obj := s.client.Bucket(s.bucketName).Object(objectName)
	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

// UploadObject uploads data to storage
func (s *StorageService) UploadObject(ctx context.Context, objectName string, data io.Reader, contentType string) error {
	obj := s.client.Bucket(s.bucketName).Object(objectName)
	writer := obj.NewWriter(ctx)
	writer.ContentType = contentType

	if _, err := io.Copy(writer, data); err != nil {
		_ = writer.Close() // #nosec G104 -- Error in cleanup, primary error is more important
		return fmt.Errorf("failed to upload object: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return nil
}

// GetObjectMetadata returns metadata for an object
func (s *StorageService) GetObjectMetadata(ctx context.Context, objectName string) (interface{}, error) {
	obj := s.client.Bucket(s.bucketName).Object(objectName)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get object metadata: %w", err)
	}
	return attrs, nil
}

// GetObjectReader returns a reader for an object
func (s *StorageService) GetObjectReader(ctx context.Context, objectName string) (io.ReadCloser, error) {
	obj := s.client.Bucket(s.bucketName).Object(objectName)
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create object reader: %w", err)
	}
	return reader, nil
}

// signBytes uses the Service Account Credentials API to sign bytes with the service account
func signBytes(ctx context.Context, serviceAccountEmail string, bytesToSign []byte) ([]byte, error) {
	// Create IAM Credentials service client
	credentialsService, err := iamcredentials.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create IAM credentials service: %w", err)
	}

	// Prepare the sign request
	resourceName := fmt.Sprintf("projects/-/serviceAccounts/%s", serviceAccountEmail)
	request := &iamcredentials.SignBlobRequest{
		Payload: base64.StdEncoding.EncodeToString(bytesToSign),
	}

	// Sign the bytes
	response, err := credentialsService.Projects.ServiceAccounts.SignBlob(resourceName, request).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to sign bytes: %w", err)
	}

	// Decode the signature
	signature, err := base64.StdEncoding.DecodeString(response.SignedBlob)
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %w", err)
	}

	return signature, nil
}
