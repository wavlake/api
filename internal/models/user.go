package models

import "time"

type NostrAuth struct {
	Pubkey        string    `firestore:"pubkey"`
	FirebaseUID   string    `firestore:"firebase_uid"`
	Active        bool      `firestore:"active"`
	CreatedAt     time.Time `firestore:"created_at"`
	LastUsedAt    time.Time `firestore:"last_used_at"`
}