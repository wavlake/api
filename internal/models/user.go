package models

import "time"

type User struct {
	FirebaseUID   string    `firestore:"firebase_uid"` // Primary key
	CreatedAt     time.Time `firestore:"created_at"`
	UpdatedAt     time.Time `firestore:"updated_at"`
	ActivePubkeys []string  `firestore:"active_pubkeys"` // Denormalized for quick lookup
}

type NostrAuth struct {
	Pubkey      string    `firestore:"pubkey"`       // Primary key
	FirebaseUID string    `firestore:"firebase_uid"` // Foreign key to User
	Active      bool      `firestore:"active"`
	CreatedAt   time.Time `firestore:"created_at"`
	LastUsedAt  time.Time `firestore:"last_used_at"`
	LinkedAt    time.Time `firestore:"linked_at"` // When linked to Firebase user
}

// CompressionOption represents a user's choice for audio compression
type CompressionOption struct {
	Bitrate    int    `json:"bitrate"`    // e.g., 128, 256, 320
	Format     string `json:"format"`     // e.g., "mp3", "aac", "ogg"
	Quality    string `json:"quality"`    // e.g., "low", "medium", "high"
	SampleRate int    `json:"sample_rate,omitempty"` // e.g., 44100, 48000
}

// CompressionVersion represents a generated compressed version
type CompressionVersion struct {
	ID           string             `firestore:"id" json:"id"`                     // Unique ID for this version
	URL          string             `firestore:"url" json:"url"`                   // GCS URL
	Bitrate      int                `firestore:"bitrate" json:"bitrate"`           // Actual bitrate
	Format       string             `firestore:"format" json:"format"`             // File format
	Quality      string             `firestore:"quality" json:"quality"`           // Quality level
	SampleRate   int                `firestore:"sample_rate" json:"sample_rate"`   // Sample rate
	Size         int64              `firestore:"size" json:"size"`                 // File size in bytes
	IsPublic     bool               `firestore:"is_public" json:"is_public"`       // Whether to include in Nostr event
	CreatedAt    time.Time          `firestore:"created_at" json:"created_at"`
	Options      CompressionOption  `firestore:"options" json:"options"`           // Original compression request
}

type NostrTrack struct {
	ID                    string                `firestore:"id" json:"id"`                               // UUID
	FirebaseUID           string                `firestore:"firebase_uid" json:"firebase_uid"`           // User who uploaded
	Pubkey                string                `firestore:"pubkey" json:"pubkey"`                       // Nostr pubkey
	OriginalURL           string                `firestore:"original_url" json:"original_url"`           // GCS URL for original file
	PresignedURL          string                `firestore:"-" json:"presigned_url,omitempty"`          // Temporary upload URL (not stored)
	Extension             string                `firestore:"extension" json:"extension"`                 // File extension
	Size                  int64                 `firestore:"size,omitempty" json:"size,omitempty"`      // Original file size in bytes
	Duration              int                   `firestore:"duration,omitempty" json:"duration,omitempty"` // Duration in seconds
	IsProcessing          bool                  `firestore:"is_processing" json:"is_processing"`         // Processing status
	CompressionVersions   []CompressionVersion  `firestore:"compression_versions,omitempty" json:"compression_versions,omitempty"` // All compressed versions
	HasPendingCompression bool                  `firestore:"has_pending_compression" json:"has_pending_compression"` // Whether compression is queued
	Deleted               bool                  `firestore:"deleted" json:"deleted"`                     // Soft delete flag
	NostrKind             int                   `firestore:"nostr_kind,omitempty" json:"nostr_kind,omitempty"`     // Nostr event kind
	NostrDTag             string                `firestore:"nostr_d_tag,omitempty" json:"nostr_d_tag,omitempty"`   // Nostr d tag
	CreatedAt             time.Time             `firestore:"created_at" json:"created_at"`
	UpdatedAt             time.Time             `firestore:"updated_at" json:"updated_at"`

	// Deprecated fields - kept for backward compatibility
	CompressedURL string `firestore:"compressed_url,omitempty" json:"compressed_url,omitempty"` // Legacy compressed file
	IsCompressed  bool   `firestore:"is_compressed" json:"is_compressed"`                       // Legacy compression status
}
