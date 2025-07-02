package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3StorageService implements StorageServiceInterface using AWS S3
type S3StorageService struct {
	client     *s3.Client
	bucketName string
	region     string
	cdnDomain  string
}

// NewS3StorageService creates a new S3 storage service
func NewS3StorageService(ctx context.Context, bucketName string) (*S3StorageService, error) {
	// Load AWS configuration from environment variables or default chain
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(getEnvOrDefault("AWS_REGION", "us-east-2")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &S3StorageService{
		client:     s3.NewFromConfig(cfg),
		bucketName: bucketName,
		region:     getEnvOrDefault("AWS_REGION", "us-east-2"),
		cdnDomain:  os.Getenv("AWS_CDN_DOMAIN"),
	}, nil
}

// GeneratePresignedURL creates a presigned URL for uploading files
func (s *S3StorageService) GeneratePresignedURL(ctx context.Context, objectName string, expiration time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

	request, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(objectName),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiration
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return request.URL, nil
}

// GetPublicURL returns the public URL for a storage object
func (s *S3StorageService) GetPublicURL(objectName string) string {
	// Use CloudFront CDN URL if configured, otherwise direct S3 URL
	if s.cdnDomain != "" {
		return fmt.Sprintf("https://%s/%s", s.cdnDomain, objectName)
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucketName, s.region, objectName)
}

// UploadObject uploads data to S3
func (s *S3StorageService) UploadObject(ctx context.Context, objectName string, data io.Reader, contentType string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(objectName),
		Body:        data,
		ContentType: aws.String(contentType),
	})

	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}

	return nil
}

// CopyObject copies an object within the same bucket
func (s *S3StorageService) CopyObject(ctx context.Context, srcObject, dstObject string) error {
	copySource := fmt.Sprintf("%s/%s", s.bucketName, srcObject)

	_, err := s.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(s.bucketName),
		CopySource: aws.String(copySource),
		Key:        aws.String(dstObject),
	})

	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	return nil
}

// DeleteObject deletes an object from S3
func (s *S3StorageService) DeleteObject(ctx context.Context, objectName string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(objectName),
	})

	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// GetObjectMetadata returns metadata for an S3 object
func (s *S3StorageService) GetObjectMetadata(ctx context.Context, objectName string) (interface{}, error) {
	result, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(objectName),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get object metadata: %w", err)
	}

	// Convert S3 HeadObjectOutput to a generic metadata structure
	metadata := map[string]interface{}{
		"ContentLength": result.ContentLength,
		"ContentType":   aws.ToString(result.ContentType),
		"LastModified":  result.LastModified,
		"ETag":          aws.ToString(result.ETag),
		"Metadata":      result.Metadata,
	}

	return metadata, nil
}

// GetBucketName returns the bucket name
func (s *S3StorageService) GetBucketName() string {
	return s.bucketName
}

// Close closes the S3 client (S3 client doesn't require explicit closing)
func (s *S3StorageService) Close() error {
	// AWS SDK v2 S3 client doesn't require explicit closing
	return nil
}

// DeleteObjects deletes multiple objects from S3 (batch operation)
func (s *S3StorageService) DeleteObjects(ctx context.Context, objectNames []string) error {
	if len(objectNames) == 0 {
		return nil
	}

	// Build delete request
	var objectsToDelete []types.ObjectIdentifier
	for _, name := range objectNames {
		objectsToDelete = append(objectsToDelete, types.ObjectIdentifier{
			Key: aws.String(name),
		})
	}

	_, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(s.bucketName),
		Delete: &types.Delete{
			Objects: objectsToDelete,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to delete objects: %w", err)
	}

	return nil
}

// GetObjectReader returns a reader for an S3 object
func (s *S3StorageService) GetObjectReader(ctx context.Context, objectName string) (io.ReadCloser, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(objectName),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	return result.Body, nil
}

// getEnvOrDefault returns the environment variable value or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Ensure S3StorageService implements StorageServiceInterface
var _ StorageServiceInterface = (*S3StorageService)(nil)
