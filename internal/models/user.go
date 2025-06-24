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

type NostrTrack struct {
	ID               string    `firestore:"id" json:"id"`                               // UUID
	FirebaseUID      string    `firestore:"firebase_uid" json:"firebase_uid"`           // User who uploaded
	Pubkey           string    `firestore:"pubkey" json:"pubkey"`                       // Nostr pubkey
	OriginalURL      string    `firestore:"original_url" json:"original_url"`           // GCS URL for original file
	CompressedURL    string    `firestore:"compressed_url,omitempty" json:"compressed_url,omitempty"` // GCS URL for compressed file
	PresignedURL     string    `firestore:"-" json:"presigned_url,omitempty"`          // Temporary upload URL (not stored)
	Extension        string    `firestore:"extension" json:"extension"`                 // File extension
	Size             int64     `firestore:"size,omitempty" json:"size,omitempty"`      // File size in bytes
	Duration         int       `firestore:"duration,omitempty" json:"duration,omitempty"` // Duration in seconds
	IsProcessing     bool      `firestore:"is_processing" json:"is_processing"`         // Processing status
	IsCompressed     bool      `firestore:"is_compressed" json:"is_compressed"`         // Whether compression completed
	Deleted          bool      `firestore:"deleted" json:"deleted"`                     // Soft delete flag
	NostrKind        int       `firestore:"nostr_kind,omitempty" json:"nostr_kind,omitempty"`     // Nostr event kind
	NostrDTag        string    `firestore:"nostr_d_tag,omitempty" json:"nostr_d_tag,omitempty"`   // Nostr d tag
	CreatedAt        time.Time `firestore:"created_at" json:"created_at"`
	UpdatedAt        time.Time `firestore:"updated_at" json:"updated_at"`
}
