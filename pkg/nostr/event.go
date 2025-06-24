package nostr

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"

	"github.com/decred/dcrd/dcrec/secp256k1/v4/schnorr"
)

type Event struct {
	ID        string     `json:"id"`
	PubKey    string     `json:"pubkey"`
	CreatedAt int64      `json:"created_at"`
	Kind      int        `json:"kind"`
	Tags      [][]string `json:"tags"`
	Content   string     `json:"content"`
	Sig       string     `json:"sig"`
}

func (e *Event) Verify() bool {
	serialized := e.serialize()
	hash := sha256.Sum256([]byte(serialized))
	computedID := hex.EncodeToString(hash[:])

	log.Printf("Event Verify Debug - Serialized: %s", serialized)
	log.Printf("Event Verify Debug - Computed ID: %s", computedID)
	log.Printf("Event Verify Debug - Event ID: %s", e.ID)

	if computedID != e.ID {
		log.Printf("Event Verify Debug - ID mismatch!")
		return false
	}

	pubKeyBytes, err := hex.DecodeString(e.PubKey)
	if err != nil || len(pubKeyBytes) != 32 {
		log.Printf("Event Verify Debug - PubKey decode error: %v, len: %d", err, len(pubKeyBytes))
		return false
	}

	sigBytes, err := hex.DecodeString(e.Sig)
	if err != nil || len(sigBytes) != 64 {
		log.Printf("Event Verify Debug - Signature decode error: %v, len: %d", err, len(sigBytes))
		return false
	}

	publicKey, err := schnorr.ParsePubKey(pubKeyBytes)
	if err != nil {
		log.Printf("Event Verify Debug - PubKey parse error: %v", err)
		return false
	}

	signature, err := schnorr.ParseSignature(sigBytes)
	if err != nil {
		log.Printf("Event Verify Debug - Signature parse error: %v", err)
		return false
	}

	isValid := signature.Verify(hash[:], publicKey)
	log.Printf("Event Verify Debug - Signature verification result: %t", isValid)
	return isValid
}

func (e *Event) serialize() string {
	arr := []interface{}{
		0,
		e.PubKey,
		e.CreatedAt,
		e.Kind,
		e.Tags,
		e.Content,
	}

	serialized, _ := json.Marshal(arr)
	return string(serialized)
}
