package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStoragePathConfigGCS(t *testing.T) {
	// Test GCS configuration (now the only configuration)
	config := GetStoragePathConfig()

	assert.Equal(t, "tracks/original", config.OriginalPrefix)
	assert.Equal(t, "tracks/compressed", config.CompressedPrefix)
	assert.False(t, config.UseLegacyPaths)
}

func TestStoragePathMethods(t *testing.T) {
	config := &StoragePathConfig{
		OriginalPrefix:   "tracks/original",
		CompressedPrefix: "tracks/compressed",
		UseLegacyPaths:   false,
	}

	trackID := "12345678-1234-5678-9012-123456789012"
	extension := "mp3"
	versionID := "v1"
	format := "aac"

	// Test path generation methods
	originalPath := config.GetOriginalPath(trackID, extension)
	expectedOriginal := "tracks/original/12345678-1234-5678-9012-123456789012.mp3"
	assert.Equal(t, expectedOriginal, originalPath)

	compressedPath := config.GetCompressedPath(trackID)
	expectedCompressed := "tracks/compressed/12345678-1234-5678-9012-123456789012.mp3"
	assert.Equal(t, expectedCompressed, compressedPath)

	versionPath := config.GetCompressedVersionPath(trackID, versionID, format)
	expectedVersion := "tracks/compressed/12345678-1234-5678-9012-123456789012_v1.aac"
	assert.Equal(t, expectedVersion, versionPath)
}

func TestStoragePathValidation(t *testing.T) {
	config := &StoragePathConfig{
		OriginalPrefix:   "tracks/original",
		CompressedPrefix: "tracks/compressed",
		UseLegacyPaths:   false,
	}

	// Test path validation methods
	assert.True(t, config.IsOriginalPath("tracks/original/test-file.mp3"))
	assert.False(t, config.IsOriginalPath("tracks/compressed/test-file.mp3"))
	assert.False(t, config.IsOriginalPath("other/test-file.mp3"))

	assert.True(t, config.IsCompressedPath("tracks/compressed/test-file.mp3"))
	assert.False(t, config.IsCompressedPath("tracks/original/test-file.mp3"))
	assert.False(t, config.IsCompressedPath("other/test-file.mp3"))
}

func TestTrackIDExtraction(t *testing.T) {
	config := &StoragePathConfig{
		OriginalPrefix:   "tracks/original",
		CompressedPrefix: "tracks/compressed",
		UseLegacyPaths:   false,
	}

	// Test track ID extraction from various path formats
	testCases := []struct {
		path     string
		expected string
	}{
		{"tracks/original/12345678-1234-5678-9012-123456789012.mp3", "12345678-1234-5678-9012-123456789012"},
		{"tracks/compressed/12345678-1234-5678-9012-123456789012.mp3", "12345678-1234-5678-9012-123456789012"},
		{"tracks/compressed/12345678-1234-5678-9012-123456789012_v1.aac", "12345678-1234-5678-9012-123456789012"},
		{"other/file.mp3", ""},
		{"invalid", ""},
	}

	for _, tc := range testCases {
		result := config.GetTrackIDFromPath(tc.path)
		assert.Equal(t, tc.expected, result, "Failed for path: %s", tc.path)
	}
}
