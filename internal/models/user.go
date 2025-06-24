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
