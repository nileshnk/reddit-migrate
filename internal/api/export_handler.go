package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nileshnk/reddit-migrate/internal/config"
	"github.com/nileshnk/reddit-migrate/internal/reddit"
	"github.com/nileshnk/reddit-migrate/internal/types"
)

// ExportHandler handles the /api/export endpoint.
// It aggregates account data for the requested sections and returns it as a downloadable JSON file.
func ExportHandler(w http.ResponseWriter, r *http.Request) {
	config.DebugLogger.Printf("Received export request from %s", r.RemoteAddr)

	if r.Header.Get("Content-Type") != "application/json" {
		config.ErrorLogger.Printf("Invalid content type for /api/export from %s: %s", r.RemoteAddr, r.Header.Get("Content-Type"))
		http.Error(w, "Content Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	var requestBody types.ExportRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&requestBody); err != nil {
		config.ErrorLogger.Printf("Error decoding /api/export request from %s: %v", r.RemoteAddr, err)
		http.Error(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Extract authentication data
	token, username, err := extractAuthData(requestBody.AuthMethod, requestBody.Cookie, requestBody.AccessToken, requestBody.Username)
	if err != nil {
		config.ErrorLogger.Printf("Failed to extract auth data for /api/export from %s: %v", r.RemoteAddr, err)
		http.Error(w, "Authentication failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Build the set of requested sections for fast lookup
	sectionSet := make(map[string]bool)
	for _, s := range requestBody.Sections {
		sectionSet[s] = true
	}

	// If no sections specified, export everything
	if len(sectionSet) == 0 {
		sectionSet["subreddits"] = true
		sectionSet["saved_posts"] = true
		sectionSet["saved_comments"] = true
	}

	exportData := types.ExportData{
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Username:   username,
	}

	var exportErrors []string

	// Fetch each requested section
	if sectionSet["subreddits"] {
		subreddits, err := reddit.FetchSubredditsWithDetails(token)
		if err != nil {
			config.ErrorLogger.Printf("Error fetching subreddits for export: %v", err)
			exportErrors = append(exportErrors, fmt.Sprintf("subreddits: %v", err))
		} else {
			exportData.Subreddits = subreddits
		}
	}

	if sectionSet["saved_posts"] {
		posts, err := reddit.FetchSavedPostsWithDetails(token, username)
		if err != nil {
			config.ErrorLogger.Printf("Error fetching saved posts for export: %v", err)
			exportErrors = append(exportErrors, fmt.Sprintf("saved_posts: %v", err))
		} else {
			exportData.SavedPosts = posts
		}
	}

	if sectionSet["saved_comments"] {
		comments, err := reddit.FetchSavedCommentsWithDetails(token, username)
		if err != nil {
			config.ErrorLogger.Printf("Error fetching saved comments for export: %v", err)
			exportErrors = append(exportErrors, fmt.Sprintf("saved_comments: %v", err))
		} else {
			exportData.SavedComments = comments
		}
	}

	if len(exportErrors) > 0 {
		exportData.Errors = exportErrors
	}

	// Set headers for file download
	filename := fmt.Sprintf("reddit-export-%s-%s.json", username, time.Now().UTC().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(exportData); err != nil {
		config.ErrorLogger.Printf("Error encoding export response for %s: %v", r.RemoteAddr, err)
		return
	}

	config.InfoLogger.Printf("Successfully exported data for %s to %s (sections: %v)", username, r.RemoteAddr, requestBody.Sections)
}
