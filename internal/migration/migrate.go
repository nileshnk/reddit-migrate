package migration

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/nileshnk/reddit-migrate/internal/config"
	"github.com/nileshnk/reddit-migrate/internal/reddit"
	"github.com/nileshnk/reddit-migrate/internal/types"
	"io"
	"net/http"
	"strings"
)

// MigrationHandler is the HTTP handler for the /migrate endpoint.
// It orchestrates the entire migration process based on the provided old and new account cookies and user preferences.
func MigrationHandler(w http.ResponseWriter, r *http.Request) {
	config.DebugLogger.Printf("Received migration request from %s", r.RemoteAddr)

	// Validate request content type.
	if r.Header.Get("Content-Type") != "application/json" {
		config.ErrorLogger.Printf("Invalid content type from %s: %s", r.RemoteAddr, r.Header.Get("Content-Type"))
		errorResponse(w, "Content Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	// Decode request body.
	var requestBody types.MigrationRequestType // Changed from migration_request_type
	if err := decodeMigrationRequest(r, &requestBody); err != nil {
		config.ErrorLogger.Printf("Error decoding migration request from %s: %v", r.RemoteAddr, err)
		// errorResponse is called within decodeMigrationRequest for specific errors
		if !strings.Contains(err.Error(), "Bad Request") { // Avoid double response if errorResponse was already called
			errorResponse(w, fmt.Sprintf("Bad Request: %v", err), http.StatusBadRequest)
		}
		return
	}

	config.InfoLogger.Printf("Migration request validated for %s. Old cookie ends: ...%s, New cookie ends: ...%s",
		r.RemoteAddr,
		safeSuffix(requestBody.OldAccountCookie, 6), // Adjusted field name
		safeSuffix(requestBody.NewAccountCookie, 6)) // Adjusted field name
	config.DebugLogger.Printf("Migration preferences: %+v", requestBody.Preferences)

	// Perform the migration.
	finalResponse := initializeMigration(requestBody.OldAccountCookie, requestBody.NewAccountCookie, requestBody.Preferences)

	// Send response.
	w.Header().Set("Content-Type", "application/json")
	jsonResp, err := json.Marshal(finalResponse)
	if err != nil {
		config.ErrorLogger.Printf("Error marshalling migration response for %s: %v", r.RemoteAddr, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(jsonResp); err != nil {
		config.ErrorLogger.Printf("Error writing migration response for %s: %v", r.RemoteAddr, err)
	} else {
		config.InfoLogger.Printf("Successfully sent migration response to %s. Success: %t", r.RemoteAddr, finalResponse.Success)
	}
}

// decodeMigrationRequest decodes the JSON request body into the types.MigrationRequestType struct.
// It handles potential unmarshalling errors and unknown fields.
func decodeMigrationRequest(r *http.Request, requestBody *types.MigrationRequestType) error { // Adjusted type
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // Consider enabling this for stricter validation.

	err := decoder.Decode(requestBody)
	if err != nil {
		var unmarshalErr *json.UnmarshalTypeError
		if errors.As(err, &unmarshalErr) {
			return fmt.Errorf("Bad Request. Wrong Type provided for field '%s'", unmarshalErr.Field)
		}
		return fmt.Errorf("Bad Request: %w", err)
	}
	return nil
}

// initializeMigration orchestrates the entire migration process based on cookies and preferences.
// It verifies accounts, fetches data, and performs migration actions like subscribing/unsubscribing subreddits and saving/unsaving posts.
func initializeMigration(oldAccountCookie, newAccountCookie string, preferences types.PreferencesType) types.MigrationResponseType { // Adjusted types
	var finalResponse types.MigrationResponseType // Adjusted type
	finalResponse.Success = false                 // Default to false

	config.InfoLogger.Println("Starting migration process...")

	// Verify cookies and get usernames.
	oldAccountUsername, err := getUsernameFromCookie(oldAccountCookie)
	if err != nil {
		config.ErrorLogger.Printf("Failed to verify old account cookie: %v", err)
		finalResponse.Message = fmt.Sprintf("Failed to verify old account cookie: %v", err)
		return finalResponse
	}
	newAccountUsername, err := getUsernameFromCookie(newAccountCookie)
	if err != nil {
		config.ErrorLogger.Printf("Failed to verify new account cookie: %v", err)
		finalResponse.Message = fmt.Sprintf("Failed to verify new account cookie: %v", err)
		return finalResponse
	}
	config.InfoLogger.Printf("Verified old account: %s, new account: %s", oldAccountUsername, newAccountUsername)

	// Parse OAuth tokens from cookies.
	oldAccountToken := parseTokenFromCookie(oldAccountCookie)
	newAccountToken := parseTokenFromCookie(newAccountCookie)
	if oldAccountToken == "" || newAccountToken == "" {
		config.ErrorLogger.Println("Failed to parse OAuth tokens from one or both cookies.")
		finalResponse.Message = "Failed to parse OAuth tokens from cookies. Ensure 'token_v2' is present."
		return finalResponse
	}
	config.DebugLogger.Printf("Old account token (suffix): ...%s", safeSuffix(oldAccountToken, 6))
	config.DebugLogger.Printf("New account token (suffix): ...%s", safeSuffix(newAccountToken, 6))

	// Handle subreddit migration/deletion.
	if preferences.MigrateSubredditBool || preferences.DeleteSubredditBool { // Adjusted field names
		if err := processSubreddits(oldAccountToken, newAccountToken, oldAccountUsername, newAccountUsername, preferences, &finalResponse.Data); err != nil {
			config.ErrorLogger.Printf("Error processing subreddits: %v", err)
			// Message is set within processSubreddits or its sub-functions for partial success.
			// If a critical error occurs, it might stop here.
		}
	}

	// Handle post migration/deletion.
	if preferences.MigratePostBool || preferences.DeletePostBool { // Adjusted field names
		if err := processPosts(oldAccountToken, newAccountToken, oldAccountUsername, newAccountUsername, preferences, &finalResponse.Data); err != nil {
			config.ErrorLogger.Printf("Error processing posts: %v", err)
			// Similar to subreddits, messages handled internally for partial success.
		}
	}

	// Determine overall success and message.
	// A more sophisticated check might be needed if partial successes are not considered overall success.
	if finalResponse.Data.SubscribeSubreddit.Error || finalResponse.Data.UnsubscribeSubreddit.Error ||
		finalResponse.Data.SavePost.FailedCount > 0 || finalResponse.Data.UnsavePost.FailedCount > 0 {
		finalResponse.Success = false
		finalResponse.Message = "Migration completed with some errors. Check individual operation statuses."
		config.InfoLogger.Println("Migration process completed with some errors.")
	} else {
		finalResponse.Success = true
		finalResponse.Message = "Migration completed successfully."
		config.InfoLogger.Println("Migration process completed successfully.")
	}

	return finalResponse
}

// processSubreddits handles the migration and/or deletion of subreddits.
func processSubreddits(oldToken, newToken, oldUser, newUser string, prefs types.PreferencesType, responseData *types.MigrationDetails) error { // Adjusted types
	config.InfoLogger.Println("Fetching all subreddit and followed user names from old account...")
	// Use reddit.FetchSubredditFullNames
	subredditNameList, err := reddit.FetchSubredditFullNames(oldToken)
	if err != nil {
		return fmt.Errorf("failed to fetch subreddit names from old account: %w", err)
	}
	config.InfoLogger.Printf("Fetched %d subreddits and %d followed users from %s.",
		len(subredditNameList.DisplayNamesList),
		len(subredditNameList.UserDisplayNameList), oldUser)

	// Migrate (subscribe) subreddits to the new account.
	if prefs.MigrateSubredditBool {
		config.InfoLogger.Printf("Starting subreddit migration for %s -> %s.", oldUser, newUser)
		responseData.SubscribeSubreddit = migrateSubredditsWithRetry(newToken, subredditNameList.DisplayNamesList, newUser)

		if len(subredditNameList.UserDisplayNameList) > 0 {
			config.InfoLogger.Printf("Starting followed user migration for %s -> %s.", oldUser, newUser)

			followedUsersResult := reddit.ManageFollowedUsers(newToken, subredditNameList.UserDisplayNameList, types.SubscribeAction)
			config.InfoLogger.Printf("Followed %d users for %s (failed: %d).", followedUsersResult.SuccessCount, newUser, followedUsersResult.FailedCount)
		} else {
			config.InfoLogger.Printf("No followed users to migrate for %s.", oldUser)
		}
	}

	// Delete (unsubscribe) subreddits from the old account.
	if prefs.DeleteSubredditBool {
		config.InfoLogger.Printf("Starting subreddit deletion (unsubscribing) from %s.", oldUser)
		// Use reddit.ManageSubreddits
		unsubscribeData := reddit.ManageSubreddits(oldToken, subredditNameList.DisplayNamesList, types.UnsubscribeAction, 500)
		config.InfoLogger.Printf("Unsubscribed %d subreddits from %s (failed: %d).", unsubscribeData.SuccessCount, oldUser, unsubscribeData.FailedCount)
		responseData.UnsubscribeSubreddit = unsubscribeData
	}
	return nil
}

// migrateSubredditsWithRetry attempts to subscribe to subreddits with a retry mechanism.
func migrateSubredditsWithRetry(token string, displayNames []string, username string) types.ManageSubredditResponseType { // Adjusted type
	// TODO: These should come from config
	subredditChunkSize := 100 // Initial chunk size for subscribing.
	maxRetryAttempts := 5     // Maximum number of retry attempts.

	config.InfoLogger.Printf("Migrating %d subreddits to account %s.", len(displayNames), username)

	subscribeData := reddit.ManageSubreddits(token, displayNames, types.SubscribeAction, subredditChunkSize)
	config.InfoLogger.Printf("Initial subscription attempt for %s: %d successful, %d failed.", username, subscribeData.SuccessCount, subscribeData.FailedCount)

	retryAttempts := 1
	for subscribeData.FailedCount > 0 && retryAttempts <= maxRetryAttempts {
		config.InfoLogger.Printf("Retrying %d failed subreddits for %s (attempt %d/%d). Chunk size: %d",
			subscribeData.FailedCount, username, retryAttempts, maxRetryAttempts, subredditChunkSize/retryAttempts)

		failedToRetry := subscribeData.FailedSubreddits
		subscribeData.FailedSubreddits = nil
		subscribeData.FailedCount = 0

		retryResult := reddit.ManageSubreddits(token, failedToRetry, types.SubscribeAction, subredditChunkSize/retryAttempts)

		subscribeData.SuccessCount += retryResult.SuccessCount
		subscribeData.FailedCount = retryResult.FailedCount
		subscribeData.FailedSubreddits = retryResult.FailedSubreddits

		config.InfoLogger.Printf("Retry attempt %d for %s: %d successful, %d still failed.",
			retryAttempts, username, retryResult.SuccessCount, retryResult.FailedCount)
		retryAttempts++
	}

	if subscribeData.FailedCount > 0 {
		config.ErrorLogger.Printf("Failed to migrate %d subreddits for %s after %d attempts. Failures: %v",
			subscribeData.FailedCount, username, maxRetryAttempts, subscribeData.FailedSubreddits)
	} else {
		config.InfoLogger.Printf("Successfully migrated all %d initially targeted subreddits for %s.", len(displayNames), username)
	}
	return subscribeData
}

// processPosts handles the migration and/or deletion of saved posts.
func processPosts(oldToken, newToken, oldUser, newUser string, prefs types.PreferencesType, responseData *types.MigrationDetails) error { // Adjusted types
	config.InfoLogger.Printf("Fetching saved post full names from old account %s...", oldUser)

	savedPostsFullNamesList, err := reddit.FetchSavedPostsFullNames(oldToken, oldUser)
	if err != nil {
		return fmt.Errorf("failed to fetch saved post names from %s: %w", oldUser, err)
	}
	config.InfoLogger.Printf("Fetched %d saved posts from %s.", len(savedPostsFullNamesList), oldUser)

	concurrencyForPosts := config.DefaultPostConcurrency // Concurrency level for post operations.

	if prefs.MigratePostBool { // Adjusted field name
		config.InfoLogger.Printf("Starting saved post migration for %s -> %s (%d posts).", oldUser, newUser, len(savedPostsFullNamesList))
		savePostsResponse := reddit.ManageSavedPosts(newToken, savedPostsFullNamesList, types.SaveAction, concurrencyForPosts)
		config.InfoLogger.Printf("Saved %d posts to %s (failed: %d).", savePostsResponse.SuccessCount, newUser, savePostsResponse.FailedCount)
		responseData.SavePost = savePostsResponse
	}

	if prefs.DeletePostBool { // Adjusted field name
		config.InfoLogger.Printf("Starting saved post deletion (unsaving) from %s (%d posts).", oldUser, len(savedPostsFullNamesList))
		unsavePostsResponse := reddit.ManageSavedPosts(oldToken, savedPostsFullNamesList, types.UnsaveAction, concurrencyForPosts)
		config.InfoLogger.Printf("Unsaved %d posts from %s (failed: %d).", unsavePostsResponse.SuccessCount, oldUser, unsavePostsResponse.FailedCount)
		responseData.UnsavePost = unsavePostsResponse
	}
	return nil
}

// verifyTokenResponse is the HTTP handler for the /verify-cookie endpoint.
// It validates a Reddit cookie and returns the associated username if valid.
func VerifyTokenResponse(w http.ResponseWriter, r *http.Request) {
	config.DebugLogger.Printf("Received /verify-cookie request from %s", r.RemoteAddr)

	if r.Header.Get("Content-Type") != "application/json" {
		config.ErrorLogger.Printf("Invalid content type for /verify-cookie from %s: %s", r.RemoteAddr, r.Header.Get("Content-Type"))
		errorResponse(w, "Content Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	var requestBody types.VerifyCookieType
	decoder := json.NewDecoder(r.Body)
	// Consider DisallowUnknownFields for stricter parsing.
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&requestBody)
	if err != nil {
		config.ErrorLogger.Printf("Error decoding /verify-cookie request from %s: %v", r.RemoteAddr, err)
		var unmarshalErr *json.UnmarshalTypeError
		if errors.As(err, &unmarshalErr) {
			errorResponse(w, "Bad Request. Wrong Type provided for field "+unmarshalErr.Field, http.StatusBadRequest)
		} else {
			errorResponse(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
		}
		return
	}
	config.InfoLogger.Printf("Verifying cookie for %s (ends with ...%s)", r.RemoteAddr, safeSuffix(requestBody.Cookie, 6))

	finalResponse := verifyCookieAndGetResponse(requestBody.Cookie)

	w.Header().Set("Content-Type", "application/json")
	jsonResp, err := json.Marshal(finalResponse)
	if err != nil {
		config.ErrorLogger.Printf("Error marshalling /verify-cookie response for %s: %v", r.RemoteAddr, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(jsonResp); err != nil {
		config.ErrorLogger.Printf("Error writing /verify-cookie response for %s: %v", r.RemoteAddr, err)
	} else {
		config.DebugLogger.Printf("Successfully sent /verify-cookie response to %s. Success: %t, User: %s",
			r.RemoteAddr, finalResponse.Success, finalResponse.Data.Username)
	}
}

// verifyCookieAndGetResponse takes a cookie string, calls Reddit's API to verify it, and returns a structured response.
func verifyCookieAndGetResponse(cookieStr string) types.TokenResponseType {
	var finalResponse types.TokenResponseType

	// Make request to Reddit's /api/me.json
	req, err := http.NewRequest(http.MethodGet, "https://www.reddit.com/api/me.json", nil)
	if err != nil {
		config.ErrorLogger.Printf("Error creating request for /api/me.json: %v", err)
		finalResponse.Success = false
		finalResponse.Message = "Internal error creating request to verify cookie."
		return finalResponse
	}

	req.Header = http.Header{
		"Cookie":     {cookieStr},
		"User-Agent": {config.UserAgent}, // Use a global or configured user agent.
	}

	config.DebugLogger.Printf("Sending request to /api/me.json to verify cookie (ends ...%s)", safeSuffix(cookieStr, 6))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		config.ErrorLogger.Printf("Error sending request to /api/me.json: %v", err)
		finalResponse.Success = false
		finalResponse.Message = "Error contacting Reddit to verify cookie."
		return finalResponse
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		config.ErrorLogger.Printf("Error reading response body from /api/me.json (status %d): %v", resp.StatusCode, err)
		finalResponse.Success = false
		finalResponse.Message = "Error reading Reddit's response."
		return finalResponse
	}
	config.DebugLogger.Printf("/api/me.json response (Status: %d): %s", resp.StatusCode, string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		config.ErrorLogger.Printf("Cookie verification failed. /api/me.json status: %d. Body: %s", resp.StatusCode, string(bodyBytes))
		var errorRespData types.ErrorResponseType
		if err := json.Unmarshal(bodyBytes, &errorRespData); err == nil && errorRespData.Message != "" {
			finalResponse.Message = fmt.Sprintf("Invalid Token/Cookie: %s", errorRespData.Message)
			finalResponse.Data.Username = errorRespData.Message // Often contains "invalid_token" or similar.
		} else {
			finalResponse.Message = fmt.Sprintf("Invalid Token/Cookie (status %d)", resp.StatusCode)
			finalResponse.Data.Username = "Unknown error"
		}
		finalResponse.Success = false
		return finalResponse
	}

	var profile types.ProfileResponseType
	if err := json.Unmarshal(bodyBytes, &profile); err != nil {
		config.ErrorLogger.Printf("Error unmarshalling /api/me.json response: %v. Body: %s", err, string(bodyBytes))
		finalResponse.Success = false
		finalResponse.Message = "Error parsing Reddit's response."
		return finalResponse
	}

	if profile.Data.Name == "" {
		config.ErrorLogger.Printf("Cookie verified (status 200) but no username found in /api/me.json response. Body: %s", string(bodyBytes))
		finalResponse.Success = false
		finalResponse.Message = "Cookie seems valid, but username could not be retrieved."
		return finalResponse
	}

	config.InfoLogger.Printf("Cookie successfully verified for username: %s", profile.Data.Name)
	finalResponse.Success = true
	finalResponse.Message = "Valid Token/Cookie"
	finalResponse.Data.Username = profile.Data.Name
	return finalResponse
}

// getUsernameFromCookie verifies a cookie and returns the username or an error.
// This is a helper for internal use within the migration logic.
func getUsernameFromCookie(cookieStr string) (string, error) {
	response := verifyCookieAndGetResponse(cookieStr)
	if !response.Success {
		return "", fmt.Errorf("cookie verification failed: %s", response.Message)
	}
	if response.Data.Username == "" {
		return "", errors.New("cookie verified but username is empty")
	}
	return response.Data.Username, nil
}

// parseTokenFromCookie extracts the 'token_v2' value from a full cookie string.
// Returns an empty string if 'token_v2' is not found or the cookie format is unexpected.
func parseTokenFromCookie(cookie string) string {
	parts := strings.Split(cookie, ";")
	for _, part := range parts {
		trimmedPart := strings.TrimSpace(part)
		if strings.HasPrefix(trimmedPart, "token_v2=") {
			tokenPair := strings.SplitN(trimmedPart, "=", 2)
			if len(tokenPair) == 2 && tokenPair[1] != "" {
				config.DebugLogger.Printf("Successfully parsed token_v2 from cookie (value ends ...%s)", safeSuffix(tokenPair[1], 6))
				return tokenPair[1]
			}
			config.ErrorLogger.Printf("Found 'token_v2=' but failed to parse value from part: '%s'", trimmedPart)
			return "" // Found prefix but value is malformed
		}
	}
	config.DebugLogger.Printf("Could not find 'token_v2=' in cookie string: ...%s", safeSuffix(cookie, 20))
	return "" // Token not found
}

// errorResponse sends a JSON error message to the client with a given HTTP status code.
func errorResponse(w http.ResponseWriter, message string, httpStatusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusCode) // Must be called before Write
	resp := make(map[string]string)
	resp["message"] = message
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		// If marshalling the error response itself fails, log it and send a plain text error.
		config.ErrorLogger.Printf("Critical: Failed to marshal error response object: %v. Original message: %s", err, message)
		http.Error(w, `{"message":"Error generating error response"}`, http.StatusInternalServerError)
		return
	}
	if _, writeErr := w.Write(jsonResp); writeErr != nil {
		config.ErrorLogger.Printf("Failed to write error response to client: %v. Original message: %s, Status: %d", writeErr, message, httpStatusCode)
	}
}

// safeSuffix returns the last N characters of a string, or the whole string if shorter than N.
// Useful for logging sensitive data like tokens without exposing the full value.
func safeSuffix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
