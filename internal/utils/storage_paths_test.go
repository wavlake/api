package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStoragePathConfigGCS(t *testing.T) {
	// Test GCS configuration (default)
	os.Setenv("STORAGE_PROVIDER", "gcs")
	defer os.Unsetenv("STORAGE_PROVIDER")

	config := GetStoragePathConfig()

	assert.Equal(t, "tracks/original", config.OriginalPrefix)
	assert.Equal(t, "tracks/compressed", config.CompressedPrefix)
	assert.False(t, config.UseLegacyPaths)
}

func TestStoragePathConfigS3(t *testing.T) {
	// Test S3 configuration with defaults
	os.Setenv("STORAGE_PROVIDER", "s3")
	defer os.Unsetenv("STORAGE_PROVIDER")

	config := GetStoragePathConfig()

	assert.Equal(t, "raw", config.OriginalPrefix)
	assert.Equal(t, "track", config.CompressedPrefix)
	assert.True(t, config.UseLegacyPaths)
}

func TestStoragePathConfigS3CustomPrefixes(t *testing.T) {
	// Test S3 configuration with custom prefixes
	os.Setenv("STORAGE_PROVIDER", "s3")
	os.Setenv("AWS_S3_RAW_PREFIX", "uploads")
	os.Setenv("AWS_S3_TRACK_PREFIX", "processed")
	defer func() {
		os.Unsetenv("STORAGE_PROVIDER")
		os.Unsetenv("AWS_S3_RAW_PREFIX")
		os.Unsetenv("AWS_S3_TRACK_PREFIX")
	}()

	config := GetStoragePathConfig()

	assert.Equal(t, "uploads", config.OriginalPrefix)
	assert.Equal(t, "processed", config.CompressedPrefix)
	assert.True(t, config.UseLegacyPaths)
}

func TestStoragePathMethods(t *testing.T) {
	config := &StoragePathConfig{
		OriginalPrefix:   "raw",
		CompressedPrefix: "track",
		UseLegacyPaths:   true,
	}

	trackID := "12345678-1234-5678-9012-123456789012"
	extension := "mp3"
	versionID := "v1"
	format := "aac"

	// Test path generation methods
	originalPath := config.GetOriginalPath(trackID, extension)
	expectedOriginal := "raw/12345678-1234-5678-9012-123456789012.mp3"
	assert.Equal(t, expectedOriginal, originalPath)

	compressedPath := config.GetCompressedPath(trackID)
	expectedCompressed := "track/12345678-1234-5678-9012-123456789012.mp3"
	assert.Equal(t, expectedCompressed, compressedPath)

	versionPath := config.GetCompressedVersionPath(trackID, versionID, format)
	expectedVersion := "track/12345678-1234-5678-9012-123456789012_v1.aac"
	assert.Equal(t, expectedVersion, versionPath)
}

func TestStoragePathValidation(t *testing.T) {
	config := &StoragePathConfig{
		OriginalPrefix:   "raw",
		CompressedPrefix: "track",
		UseLegacyPaths:   true,
	}

	// Test path validation methods
	assert.True(t, config.IsOriginalPath("raw/test-file.mp3"))
	assert.False(t, config.IsOriginalPath("track/test-file.mp3"))
	assert.False(t, config.IsOriginalPath("other/test-file.mp3"))

	assert.True(t, config.IsCompressedPath("track/test-file.mp3"))
	assert.False(t, config.IsCompressedPath("raw/test-file.mp3"))
	assert.False(t, config.IsCompressedPath("other/test-file.mp3"))
}

func TestTrackIDExtraction(t *testing.T) {
	config := &StoragePathConfig{
		OriginalPrefix:   "raw",
		CompressedPrefix: "track",
		UseLegacyPaths:   true,
	}

	// Test track ID extraction from various path formats
	testCases := []struct {
		path     string
		expected string
	}{
		{"raw/12345678-1234-5678-9012-123456789012.mp3", "12345678-1234-5678-9012-123456789012"},
		{"track/12345678-1234-5678-9012-123456789012.mp3", "12345678-1234-5678-9012-123456789012"},
		{"track/12345678-1234-5678-9012-123456789012_v1.aac", "12345678-1234-5678-9012-123456789012"},
		{"other/file.mp3", ""},
		{"invalid", ""},
	}

	for _, tc := range testCases {
		result := config.GetTrackIDFromPath(tc.path)
		assert.Equal(t, tc.expected, result, "Failed for path: %s", tc.path)
	}
}

func TestStoragePathConfigWithGCSPaths(t *testing.T) {
	// Test with GCS-style paths
	config := &StoragePathConfig{
		OriginalPrefix:   "tracks/original",
		CompressedPrefix: "tracks/compressed",
		UseLegacyPaths:   false,
	}

	trackID := "test-track-id"
	extension := "wav"

	originalPath := config.GetOriginalPath(trackID, extension)
	assert.Equal(t, "tracks/original/test-track-id.wav", originalPath)

	compressedPath := config.GetCompressedPath(trackID)
	assert.Equal(t, "tracks/compressed/test-track-id.mp3", compressedPath)

	// Test path validation with nested paths
	assert.True(t, config.IsOriginalPath("tracks/original/file.mp3"))
	assert.True(t, config.IsCompressedPath("tracks/compressed/file.mp3"))
	assert.False(t, config.IsOriginalPath("tracks/compressed/file.mp3"))
}
