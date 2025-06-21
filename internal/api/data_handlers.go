package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nileshnk/reddit-migrate/internal/auth"
	"github.com/nileshnk/reddit-migrate/internal/config"
	"github.com/nileshnk/reddit-migrate/internal/reddit"
	"github.com/nileshnk/reddit-migrate/internal/types"
)

// extractAuthData extracts authentication information from the request
// Returns (token, username, error)
func extractAuthData(authMethod, cookie, accessToken, providedUsername string) (string, string, error) {
	var token, username string
	var err error

	config.DebugLogger.Printf("extractAuthData called: authMethod=%s, providedUsername='%s', hasAccessToken=%t",
		authMethod, providedUsername, accessToken != "")

	if authMethod == "oauth" {
		// Direct OAuth token
		token = accessToken

		// Use provided username if available, otherwise get from Reddit API
		if providedUsername != "" {
			username = providedUsername
			config.DebugLogger.Printf("Using provided username: %s", username)
		} else {
			config.DebugLogger.Printf("No username provided, fetching from Reddit API")
			// For OAuth, we need to get the username from the Reddit API
			userInfo, err := auth.GetUserInfoWithToken(token)
			if err != nil {
				config.ErrorLogger.Printf("Failed to get user info from token: %v", err)
				return "", "", fmt.Errorf("failed to get user info: %w", err)
			}
			username = userInfo.Data.Name
			config.DebugLogger.Printf("Fetched username from Reddit API: %s", username)
		}
	} else {
		// Cookie-based authentication (default/backward compatibility)
		username, err = auth.GetUsernameFromCookie(cookie)
		if err != nil {
			return "", "", err
		}

		token = auth.ParseTokenFromCookie(cookie)
		if token == "" {
			return "", "", fmt.Errorf("failed to parse OAuth token from cookie")
		}
	}

	config.DebugLogger.Printf("extractAuthData returning: username='%s', hasToken=%t", username, token != "")
	return token, username, nil
}

// SubredditsHandler handles the /api/subreddits endpoint
func SubredditsHandler(w http.ResponseWriter, r *http.Request) {
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

	// Extract authentication data
	token, _, err := extractAuthData(requestBody.AuthMethod, requestBody.Cookie, requestBody.AccessToken, requestBody.Username)
	if err != nil {
		config.ErrorLogger.Printf("Failed to extract auth data for /api/subreddits from %s: %v", r.RemoteAddr, err)
		http.Error(w, "Authentication failed: "+err.Error(), http.StatusBadRequest)
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
}

// SavedPostsHandler handles the /api/saved-posts endpoint
func SavedPostsHandler(w http.ResponseWriter, r *http.Request) {
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

	// Extract authentication data
	token, username, err := extractAuthData(requestBody.AuthMethod, requestBody.Cookie, requestBody.AccessToken, requestBody.Username)
	if err != nil {
		config.ErrorLogger.Printf("Failed to extract auth data for /api/saved-posts from %s: %v", r.RemoteAddr, err)
		http.Error(w, "Authentication failed: "+err.Error(), http.StatusBadRequest)
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
}

// AccountCountsHandler handles the /api/account-counts endpoint
func AccountCountsHandler(w http.ResponseWriter, r *http.Request) {
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

	// Extract authentication data
	token, username, err := extractAuthData(requestBody.AuthMethod, requestBody.Cookie, requestBody.AccessToken, requestBody.Username)
	if err != nil {
		config.ErrorLogger.Printf("Failed to extract auth data for /api/account-counts from %s: %v", r.RemoteAddr, err)
		http.Error(w, "Authentication failed: "+err.Error(), http.StatusBadRequest)
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
}
