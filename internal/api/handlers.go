package api

import (
	"encoding/json"
	"net/http"

	"github.com/nileshnk/reddit-migrate/internal/config"
)

// HealthCheckHandler handles the health check endpoint
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	config.DebugLogger.Printf("Received health check request from %s", r.RemoteAddr)

	response := struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}{
		Status:  "healthy",
		Message: "API is running",
	}

	w.Header().Set("Content-Type", "application/json")
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		config.ErrorLogger.Printf("Error marshalling health check response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_, err = w.Write(jsonResponse)
	if err != nil {
		config.ErrorLogger.Printf("Error writing health check response: %v", err)
	}

	config.DebugLogger.Printf("Successfully responded to health check from %s", r.RemoteAddr)
}
