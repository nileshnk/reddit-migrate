package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/nileshnk/reddit-migrate/internal/config"
	"github.com/nileshnk/reddit-migrate/internal/migration"
	"github.com/nileshnk/reddit-migrate/internal/reddit"
	"github.com/nileshnk/reddit-migrate/internal/types"

	"github.com/go-chi/chi/v5"
)

// Router sets up the API routes for the application.
// It defines endpoints for testing, cookie verification, and data migration.
func Router(router chi.Router) {

	// Health check endpoint to verify API is running
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
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
	})

	// Endpoint to verify the validity of a Reddit account cookie.
	router.Post("/verify-cookie", migration.VerifyTokenResponse) // TODO: verifyTokenResponse needs to be defined or imported
	config.InfoLogger.Println("Registered /api/verify-cookie POST endpoint")

	// Endpoint to handle the data migration process between Reddit accounts.
	router.Post("/migrate", migration.MigrationHandler) // TODO: MigrationHandler needs to be defined or imported
	config.InfoLogger.Println("Registered /api/migrate POST endpoint")

	// Endpoint to fetch detailed subreddit information for selection UI
	router.Post("/subreddits", func(w http.ResponseWriter, r *http.Request) {
		config.DebugLogger.Printf("Received request for /api/subreddits from %s", r.RemoteAddr)

		if r.Header.Get("Content-Type") != "application/json" {
			config.ErrorLogger.Printf("Invalid content type for /api/subreddits from %s: %s", r.RemoteAddr, r.Header.Get("Content-Type"))
			http.Error(w, "Content Type must be application/json", http.StatusUnsupportedMediaType)
			return
		}

		var requestBody types.GetSubredditsRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&requestBody); err != nil {
			config.ErrorLogger.Printf("Error decoding /api/subreddits request from %s: %v", r.RemoteAddr, err)
			http.Error(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Extract OAuth token from cookie
		token := parseTokenFromCookie(requestBody.Cookie)
		if token == "" {
			config.ErrorLogger.Printf("Failed to parse OAuth token from cookie for /api/subreddits from %s", r.RemoteAddr)
			http.Error(w, "Invalid cookie: token_v2 not found", http.StatusBadRequest)
			return
		}

		// Fetch detailed subreddit information
		subreddits, err := reddit.FetchSubredditsWithDetails(token)
		if err != nil {
			config.ErrorLogger.Printf("Error fetching subreddits for %s: %v", r.RemoteAddr, err)
			http.Error(w, "Failed to fetch subreddits: "+err.Error(), http.StatusInternalServerError)
			return
		}

		response := types.GetSubredditsResponse{
			Success:    true,
			Message:    "Subreddits fetched successfully",
			Subreddits: subreddits,
			Count:      len(subreddits),
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			config.ErrorLogger.Printf("Error encoding subreddits response for %s: %v", r.RemoteAddr, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		config.InfoLogger.Printf("Successfully sent %d subreddits to %s", len(subreddits), r.RemoteAddr)
	})
	config.InfoLogger.Println("Registered /api/subreddits POST endpoint")

	// Endpoint to fetch detailed saved posts information for selection UI
	router.Post("/saved-posts", func(w http.ResponseWriter, r *http.Request) {
		config.DebugLogger.Printf("Received request for /api/saved-posts from %s", r.RemoteAddr)

		if r.Header.Get("Content-Type") != "application/json" {
			config.ErrorLogger.Printf("Invalid content type for /api/saved-posts from %s: %s", r.RemoteAddr, r.Header.Get("Content-Type"))
			http.Error(w, "Content Type must be application/json", http.StatusUnsupportedMediaType)
			return
		}

		var requestBody types.GetSavedPostsRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&requestBody); err != nil {
			config.ErrorLogger.Printf("Error decoding /api/saved-posts request from %s: %v", r.RemoteAddr, err)
			http.Error(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
			return
		}

		// First verify cookie and get username
		username, err := getUsernameFromCookie(requestBody.Cookie)
		if err != nil {
			config.ErrorLogger.Printf("Failed to verify cookie for /api/saved-posts from %s: %v", r.RemoteAddr, err)
			http.Error(w, "Invalid cookie: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Extract OAuth token from cookie
		token := parseTokenFromCookie(requestBody.Cookie)
		if token == "" {
			config.ErrorLogger.Printf("Failed to parse OAuth token from cookie for /api/saved-posts from %s", r.RemoteAddr)
			http.Error(w, "Invalid cookie: token_v2 not found", http.StatusBadRequest)
			return
		}

		// Fetch detailed saved posts information
		posts, err := reddit.FetchSavedPostsWithDetails(token, username)
		if err != nil {
			config.ErrorLogger.Printf("Error fetching saved posts for %s: %v", r.RemoteAddr, err)
			http.Error(w, "Failed to fetch saved posts: "+err.Error(), http.StatusInternalServerError)
			return
		}

		response := types.GetSavedPostsResponse{
			Success: true,
			Message: "Saved posts fetched successfully",
			Posts:   posts,
			Count:   len(posts),
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			config.ErrorLogger.Printf("Error encoding saved posts response for %s: %v", r.RemoteAddr, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		config.InfoLogger.Printf("Successfully sent %d saved posts to %s", len(posts), r.RemoteAddr)
	})
	config.InfoLogger.Println("Registered /api/saved-posts POST endpoint")

	// Endpoint to handle custom selection migration
	router.Post("/migrate-custom", func(w http.ResponseWriter, r *http.Request) {
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
	})
	config.InfoLogger.Println("Registered /api/migrate-custom POST endpoint")

	// Endpoint to get account counts (subreddits and saved posts)
	router.Post("/account-counts", func(w http.ResponseWriter, r *http.Request) {
		config.DebugLogger.Printf("Received request for /api/account-counts from %s", r.RemoteAddr)

		if r.Header.Get("Content-Type") != "application/json" {
			config.ErrorLogger.Printf("Invalid content type for /api/account-counts from %s: %s", r.RemoteAddr, r.Header.Get("Content-Type"))
			http.Error(w, "Content Type must be application/json", http.StatusUnsupportedMediaType)
			return
		}

		var requestBody types.AccountCountsRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&requestBody); err != nil {
			config.ErrorLogger.Printf("Error decoding /api/account-counts request from %s: %v", r.RemoteAddr, err)
			http.Error(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Get username from cookie
		username, err := getUsernameFromCookie(requestBody.Cookie)
		if err != nil {
			config.ErrorLogger.Printf("Failed to verify cookie for /api/account-counts from %s: %v", r.RemoteAddr, err)
			http.Error(w, "Invalid cookie: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Extract OAuth token from cookie
		token := parseTokenFromCookie(requestBody.Cookie)
		if token == "" {
			config.ErrorLogger.Printf("Failed to parse OAuth token from cookie for /api/account-counts from %s", r.RemoteAddr)
			http.Error(w, "Invalid cookie: token_v2 not found", http.StatusBadRequest)
			return
		}

		// Get counts
		subredditCount, err := reddit.GetSubredditCount(token)
		if err != nil {
			config.ErrorLogger.Printf("Error getting subreddit count for %s: %v", r.RemoteAddr, err)
			subredditCount = -1 // Indicate error
		}

		postsCount, err := reddit.GetSavedPostsCount(token, username)
		if err != nil {
			config.ErrorLogger.Printf("Error getting saved posts count for %s: %v", r.RemoteAddr, err)
			postsCount = -1 // Indicate error
		}

		response := types.AccountCountsResponse{
			Success:         true,
			Message:         "Account counts retrieved successfully",
			Username:        username,
			SubredditCount:  subredditCount,
			SavedPostsCount: postsCount,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			config.ErrorLogger.Printf("Error encoding account counts response for %s: %v", r.RemoteAddr, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		config.InfoLogger.Printf("Successfully sent account counts to %s: %d subreddits, %d posts", r.RemoteAddr, subredditCount, postsCount)
	})
	config.InfoLogger.Println("Registered /api/account-counts POST endpoint")
}

// Helper functions

// parseTokenFromCookie extracts the 'token_v2' value from a full cookie string.
func parseTokenFromCookie(cookie string) string {

	parts := strings.Split(cookie, ";")
	for _, part := range parts {
		trimmedPart := strings.TrimSpace(part)
		if strings.HasPrefix(trimmedPart, "token_v2=") {
			tokenPair := strings.SplitN(trimmedPart, "=", 2)
			if len(tokenPair) == 2 && tokenPair[1] != "" {
				return tokenPair[1]
			}
		}
	}
	return ""
}

// getUsernameFromCookie verifies a cookie and returns the username
func getUsernameFromCookie(cookieStr string) (string, error) {

	req, err := http.NewRequest(http.MethodGet, "https://www.reddit.com/api/me.json", nil)
	if err != nil {
		return "", err
	}

	req.Header = http.Header{
		"Cookie":     {cookieStr},
		"User-Agent": {config.UserAgent},
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cookie verification failed with status %d", resp.StatusCode)
	}

	var profile types.ProfileResponseType
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return "", err
	}

	if profile.Data.Name == "" {
		return "", fmt.Errorf("username not found in response")
	}

	return profile.Data.Name, nil
}
