package utils

import (
	"fmt"
)

// StoragePathConfig holds path configuration for different storage providers
type StoragePathConfig struct {
	OriginalPrefix   string
	CompressedPrefix string
	UseLegacyPaths   bool
}

// GetStoragePathConfig returns a fixed path configuration for GCS storage.
// The paths are set to standard prefixes: 'tracks/original' and 'tracks/compressed'.

func GetStoragePathConfig() *StoragePathConfig {
	config := &StoragePathConfig{
		OriginalPrefix:   "tracks/original",
		CompressedPrefix: "tracks/compressed",
		UseLegacyPaths:   false,
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
