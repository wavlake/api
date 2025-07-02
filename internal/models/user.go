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
	Bitrate    int    `json:"bitrate"`               // e.g., 128, 256, 320
	Format     string `json:"format"`                // e.g., "mp3", "aac", "ogg"
	Quality    string `json:"quality"`               // e.g., "low", "medium", "high"
	SampleRate int    `json:"sample_rate,omitempty"` // e.g., 44100, 48000
}

// CompressionVersion represents a generated compressed version
type CompressionVersion struct {
	ID         string            `firestore:"id" json:"id"`                   // Unique ID for this version
	URL        string            `firestore:"url" json:"url"`                 // GCS URL
	Bitrate    int               `firestore:"bitrate" json:"bitrate"`         // Actual bitrate
	Format     string            `firestore:"format" json:"format"`           // File format
	Quality    string            `firestore:"quality" json:"quality"`         // Quality level
	SampleRate int               `firestore:"sample_rate" json:"sample_rate"` // Sample rate
	Size       int64             `firestore:"size" json:"size"`               // File size in bytes
	IsPublic   bool              `firestore:"is_public" json:"is_public"`     // Whether to include in Nostr event
	CreatedAt  time.Time         `firestore:"created_at" json:"created_at"`
	Options    CompressionOption `firestore:"options" json:"options"` // Original compression request
}

type NostrTrack struct {
	ID                    string               `firestore:"id" json:"id"`                                                         // UUID
	FirebaseUID           string               `firestore:"firebase_uid" json:"firebase_uid"`                                     // User who uploaded
	Pubkey                string               `firestore:"pubkey" json:"pubkey"`                                                 // Nostr pubkey
	OriginalURL           string               `firestore:"original_url" json:"original_url"`                                     // GCS URL for original file
	PresignedURL          string               `firestore:"-" json:"presigned_url,omitempty"`                                     // Temporary upload URL (not stored)
	Extension             string               `firestore:"extension" json:"extension"`                                           // File extension
	Size                  int64                `firestore:"size,omitempty" json:"size,omitempty"`                                 // Original file size in bytes
	Duration              int                  `firestore:"duration,omitempty" json:"duration,omitempty"`                         // Duration in seconds
	IsProcessing          bool                 `firestore:"is_processing" json:"is_processing"`                                   // Processing status
	CompressionVersions   []CompressionVersion `firestore:"compression_versions,omitempty" json:"compression_versions,omitempty"` // All compressed versions
	HasPendingCompression bool                 `firestore:"has_pending_compression" json:"has_pending_compression"`               // Whether compression is queued
	Deleted               bool                 `firestore:"deleted" json:"deleted"`                                               // Soft delete flag
	NostrKind             int                  `firestore:"nostr_kind,omitempty" json:"nostr_kind,omitempty"`                     // Nostr event kind
	NostrDTag             string               `firestore:"nostr_d_tag,omitempty" json:"nostr_d_tag,omitempty"`                   // Nostr d tag
	CreatedAt             time.Time            `firestore:"created_at" json:"created_at"`
	UpdatedAt             time.Time            `firestore:"updated_at" json:"updated_at"`

	// Deprecated fields - kept for backward compatibility
	CompressedURL string `firestore:"compressed_url,omitempty" json:"compressed_url,omitempty"` // Legacy compressed file
	IsCompressed  bool   `firestore:"is_compressed" json:"is_compressed"`                       // Legacy compression status
}

// VersionUpdate represents a request to update compression version visibility
type VersionUpdate struct {
	VersionID string `json:"version_id"`
	IsPublic  bool   `json:"is_public"`
}

// Legacy PostgreSQL Models
// These models map to the legacy catalog API's PostgreSQL database

type LegacyUser struct {
	ID               string    `db:"id" json:"id"`
	Name             string    `db:"name" json:"name"`
	LightningAddress string    `db:"lightning_address" json:"lightning_address"`
	MSatBalance      int64     `db:"msat_balance" json:"msat_balance"`
	AmpMsat          int       `db:"amp_msat" json:"amp_msat"`
	ArtworkURL       string    `db:"artwork_url" json:"artwork_url"`
	ProfileURL       string    `db:"profile_url" json:"profile_url"`
	IsLocked         bool      `db:"is_locked" json:"is_locked"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time `db:"updated_at" json:"updated_at"`
}

type LegacyTrack struct {
	ID              string    `db:"id" json:"id"`
	ArtistID        string    `db:"artist_id" json:"artist_id"`
	AlbumID         string    `db:"album_id" json:"album_id"`
	Title           string    `db:"title" json:"title"`
	Order           int       `db:"order" json:"order"`
	PlayCount       int       `db:"play_count" json:"play_count"`
	MSatTotal       int64     `db:"msat_total" json:"msat_total"`
	LiveURL         string    `db:"live_url" json:"live_url"`
	RawURL          string    `db:"raw_url" json:"raw_url"`
	Size            int       `db:"size" json:"size"`
	Duration        int       `db:"duration" json:"duration"`
	IsProcessing    bool      `db:"is_processing" json:"is_processing"`
	IsDraft         bool      `db:"is_draft" json:"is_draft"`
	IsExplicit      bool      `db:"is_explicit" json:"is_explicit"`
	CompressorError bool      `db:"compressor_error" json:"compressor_error"`
	Deleted         bool      `db:"deleted" json:"deleted"`
	Lyrics          string    `db:"lyrics" json:"lyrics"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
	PublishedAt     time.Time `db:"published_at" json:"published_at"`
}

type LegacyArtist struct {
	ID         string    `db:"id" json:"id"`
	UserID     string    `db:"user_id" json:"user_id"`
	Name       string    `db:"name" json:"name"`
	ArtworkURL string    `db:"artwork_url" json:"artwork_url"`
	ArtistURL  string    `db:"artist_url" json:"artist_url"`
	Bio        string    `db:"bio" json:"bio"`
	Twitter    string    `db:"twitter" json:"twitter"`
	Instagram  string    `db:"instagram" json:"instagram"`
	Youtube    string    `db:"youtube" json:"youtube"`
	Website    string    `db:"website" json:"website"`
	Npub       string    `db:"npub" json:"npub"`
	Verified   bool      `db:"verified" json:"verified"`
	Deleted    bool      `db:"deleted" json:"deleted"`
	MSatTotal  int64     `db:"msat_total" json:"msat_total"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

type LegacyAlbum struct {
	ID              string    `db:"id" json:"id"`
	ArtistID        string    `db:"artist_id" json:"artist_id"`
	Title           string    `db:"title" json:"title"`
	ArtworkURL      string    `db:"artwork_url" json:"artwork_url"`
	Description     string    `db:"description" json:"description"`
	GenreID         int       `db:"genre_id" json:"genre_id"`
	SubgenreID      int       `db:"subgenre_id" json:"subgenre_id"`
	IsDraft         bool      `db:"is_draft" json:"is_draft"`
	IsSingle        bool      `db:"is_single" json:"is_single"`
	Deleted         bool      `db:"deleted" json:"deleted"`
	MSatTotal       int64     `db:"msat_total" json:"msat_total"`
	IsFeedPublished bool      `db:"is_feed_published" json:"is_feed_published"`
	PublishedAt     time.Time `db:"published_at" json:"published_at"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}
