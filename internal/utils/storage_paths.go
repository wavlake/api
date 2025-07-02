package utils

import (
	"fmt"
	"os"
)

// StoragePathConfig holds path configuration for different storage providers
type StoragePathConfig struct {
	OriginalPrefix   string
	CompressedPrefix string
	UseLegacyPaths   bool
}

// GetStoragePathConfig returns path configuration based on storage provider
func GetStoragePathConfig() *StoragePathConfig {
	storageProvider := getEnvOrDefault("STORAGE_PROVIDER", "gcs")

	config := &StoragePathConfig{}

	if storageProvider == "s3" {
		// For S3, use legacy catalog API path structure by default for compatibility
		config.OriginalPrefix = getEnvOrDefault("AWS_S3_RAW_PREFIX", "raw")
		config.CompressedPrefix = getEnvOrDefault("AWS_S3_TRACK_PREFIX", "track")
		config.UseLegacyPaths = true
	} else {
		// Use current path structure for GCS
		config.OriginalPrefix = "tracks/original"
		config.CompressedPrefix = "tracks/compressed"
		config.UseLegacyPaths = false
	}

	return config
}

// GetOriginalPath returns the storage path for original uploaded files
func (c *StoragePathConfig) GetOriginalPath(trackID, extension string) string {
	return fmt.Sprintf("%s/%s.%s", c.OriginalPrefix, trackID, extension)
}

// GetCompressedPath returns the storage path for compressed files
func (c *StoragePathConfig) GetCompressedPath(trackID string) string {
	return fmt.Sprintf("%s/%s.mp3", c.CompressedPrefix, trackID)
}

// GetCompressedVersionPath returns the storage path for specific compression versions
func (c *StoragePathConfig) GetCompressedVersionPath(trackID, versionID, format string) string {
	return fmt.Sprintf("%s/%s_%s.%s", c.CompressedPrefix, trackID, versionID, format)
}

// IsOriginalPath checks if a given path is in the original files directory
func (c *StoragePathConfig) IsOriginalPath(objectPath string) bool {
	expectedPrefix := c.OriginalPrefix + "/"
	return len(objectPath) > len(expectedPrefix) && objectPath[:len(expectedPrefix)] == expectedPrefix
}

// IsCompressedPath checks if a given path is in the compressed files directory
func (c *StoragePathConfig) IsCompressedPath(objectPath string) bool {
	expectedPrefix := c.CompressedPrefix + "/"
	return len(objectPath) > len(expectedPrefix) && objectPath[:len(expectedPrefix)] == expectedPrefix
}

// GetTrackIDFromPath extracts track ID from a storage path
func (c *StoragePathConfig) GetTrackIDFromPath(objectPath string) string {
	var prefix string
	if c.IsOriginalPath(objectPath) {
		prefix = c.OriginalPrefix + "/"
	} else if c.IsCompressedPath(objectPath) {
		prefix = c.CompressedPrefix + "/"
	} else {
		return ""
	}

	// Extract filename without path
	filename := objectPath[len(prefix):]

	// Extract track ID (everything before first dot)
	for i, char := range filename {
		if char == '.' {
			return filename[:i]
		}
		if char == '_' {
			// For versioned compressed files, track ID is before underscore
			return filename[:i]
		}
	}

	return filename
}

// getEnvOrDefault returns an environment variable value or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
