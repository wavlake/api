package nostr

import (
	"log"

	gonostr "github.com/nbd-wtf/go-nostr"
)

// Event wraps the go-nostr Event to maintain API compatibility
type Event struct {
	*gonostr.Event
}

func (e *Event) Verify() bool {
	log.Printf("Event Verify Debug - Using go-nostr CheckSignature for event ID: %s", e.ID)
	log.Printf("Event Verify Debug - PubKey: %s", e.PubKey)
	log.Printf("Event Verify Debug - Signature: %s", e.Sig)
	
	isValid, err := e.Event.CheckSignature()
	if err != nil {
		log.Printf("Event Verify Debug - CheckSignature error: %v", err)
		return false
	}
	
	log.Printf("Event Verify Debug - Signature verification result: %t", isValid)
	return isValid
}
