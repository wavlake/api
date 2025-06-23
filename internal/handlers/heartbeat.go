package handlers

import (
	"encoding/json"
	"net/http"
	"os"
)

type HeartbeatResponse struct {
	Status    string `json:"status"`
	CommitSHA string `json:"commit_sha"`
}

func Heartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	commitSHA := os.Getenv("COMMIT_SHA")
	if commitSHA == "" {
		commitSHA = "unknown"
	}

	response := HeartbeatResponse{
		Status:    "ok",
		CommitSHA: commitSHA,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func NotFound(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not found", http.StatusNotFound)
}