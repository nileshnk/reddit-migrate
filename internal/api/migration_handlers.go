package api

import (
	"encoding/json"
	"net/http"

	"github.com/nileshnk/reddit-migrate/internal/config"
	"github.com/nileshnk/reddit-migrate/internal/migration"
	"github.com/nileshnk/reddit-migrate/internal/types"
)

// CustomMigrationHandler handles the /api/migrate-custom endpoint
func CustomMigrationHandler(w http.ResponseWriter, r *http.Request) {
	config.DebugLogger.Printf("Received custom migration request from %s", r.RemoteAddr)

	if r.Header.Get("Content-Type") != "application/json" {
		config.ErrorLogger.Printf("Invalid content type for /api/migrate-custom from %s: %s", r.RemoteAddr, r.Header.Get("Content-Type"))
		http.Error(w, "Content Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	var requestBody types.CustomMigrationRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&requestBody); err != nil {
		config.ErrorLogger.Printf("Error decoding /api/migrate-custom request from %s: %v", r.RemoteAddr, err)
		http.Error(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
		return
	}

	config.InfoLogger.Printf("Custom migration request for %s: %d subreddits, %d posts",
		r.RemoteAddr, len(requestBody.SelectedSubreddits), len(requestBody.SelectedPosts))

	finalResponse := migration.HandleCustomMigration(requestBody)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(finalResponse); err != nil {
		config.ErrorLogger.Printf("Error encoding custom migration response for %s: %v", r.RemoteAddr, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	config.InfoLogger.Printf("Successfully processed custom migration for %s. Success: %t", r.RemoteAddr, finalResponse.Success)
}
