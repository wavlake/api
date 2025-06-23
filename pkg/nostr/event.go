package nostr

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

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
	
	if hex.EncodeToString(hash[:]) != e.ID {
		return false
	}

	pubKeyBytes, err := hex.DecodeString(e.PubKey)
	if err != nil || len(pubKeyBytes) != 32 {
		return false
	}

	sigBytes, err := hex.DecodeString(e.Sig)
	if err != nil || len(sigBytes) != 64 {
		return false
	}

	publicKey, err := schnorr.ParsePubKey(pubKeyBytes)
	if err != nil {
		return false
	}

	signature, err := schnorr.ParseSignature(sigBytes)
	if err != nil {
		return false
	}

	return signature.Verify(hash[:], publicKey)
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