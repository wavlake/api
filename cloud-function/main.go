package cloudfunction

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// GCSEvent represents a Cloud Storage event
type GCSEvent struct {
	Bucket         string    `json:"bucket"`
	Name           string    `json:"name"`
	Metageneration string    `json:"metageneration"`
	TimeCreated    time.Time `json:"timeCreated"`
	Updated        time.Time `json:"updated"`
}

// CloudEvent represents the Cloud Function event wrapper
type CloudEvent struct {
	Data GCSEvent `json:"data"`
}

// ProcessAudioUpload is triggered when a file is uploaded to GCS
func ProcessAudioUpload(w http.ResponseWriter, r *http.Request) {
	var e CloudEvent
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		log.Printf("Failed to decode event: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Log the full event for debugging
	log.Printf("Received GCS event - Bucket: %s, Name: %s", e.Data.Bucket, e.Data.Name)

	// Only process files in the tracks/original/ path
	if !strings.HasPrefix(e.Data.Name, "tracks/original/") {
		log.Printf("Ignoring file outside tracks/original/: '%s'", e.Data.Name)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extract track ID from filename
	// Format: tracks/original/uuid.extension
	parts := strings.Split(e.Data.Name, "/")
	if len(parts) != 3 {
		log.Printf("Invalid file path format: %s", e.Data.Name)
		w.WriteHeader(http.StatusOK)
		return
	}

	filename := parts[2]
	trackID := strings.TrimSuffix(filename, "."+getFileExtension(filename))

	log.Printf("Processing track upload: %s (file: %s)", trackID, e.Data.Name)

	// Call the API to trigger processing
	if err := triggerProcessing(trackID); err != nil {
		log.Printf("Failed to trigger processing for track %s: %v", trackID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully triggered processing for track %s", trackID)
	w.WriteHeader(http.StatusOK)
}

// getFileExtension extracts file extension from filename
func getFileExtension(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-1]
}

// triggerProcessing calls the API to start track processing
func triggerProcessing(trackID string) error {
	apiURL := os.Getenv("API_BASE_URL")
	if apiURL == "" {
		return fmt.Errorf("API_BASE_URL environment variable not set")
	}

	webhookURL := fmt.Sprintf("%s/v1/tracks/webhook/process", apiURL)

	payload := map[string]interface{}{
		"track_id": trackID,
		"status":   "uploaded",
		"source":   "gcs_trigger",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	
	// Add webhook authentication if configured
	if webhookSecret := os.Getenv("WEBHOOK_SECRET"); webhookSecret != "" {
		req.Header.Set("X-Webhook-Secret", webhookSecret)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}