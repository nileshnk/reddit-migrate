package migration

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/nileshnk/reddit-migrate/internal/auth"
	"github.com/nileshnk/reddit-migrate/internal/config"
	"github.com/nileshnk/reddit-migrate/internal/reddit"
	"github.com/nileshnk/reddit-migrate/internal/types"
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
		auth.SafeSuffix(requestBody.OldAccountCookie, 6), // Adjusted field name
		auth.SafeSuffix(requestBody.NewAccountCookie, 6)) // Adjusted field name
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
	oldAccountUsername, err := auth.GetUsernameFromCookie(oldAccountCookie)
	if err != nil {
		config.ErrorLogger.Printf("Failed to verify old account cookie: %v", err)
		finalResponse.Message = fmt.Sprintf("Failed to verify old account cookie: %v", err)
		return finalResponse
	}
	newAccountUsername, err := auth.GetUsernameFromCookie(newAccountCookie)
	if err != nil {
		config.ErrorLogger.Printf("Failed to verify new account cookie: %v", err)
		finalResponse.Message = fmt.Sprintf("Failed to verify new account cookie: %v", err)
		return finalResponse
	}
	config.InfoLogger.Printf("Verified old account: %s, new account: %s", oldAccountUsername, newAccountUsername)

	// Parse OAuth tokens from cookies.
	oldAccountToken := auth.ParseTokenFromCookie(oldAccountCookie)
	newAccountToken := auth.ParseTokenFromCookie(newAccountCookie)
	if oldAccountToken == "" || newAccountToken == "" {
		config.ErrorLogger.Println("Failed to parse OAuth tokens from one or both cookies.")
		finalResponse.Message = "Failed to parse OAuth tokens from cookies. Ensure 'token_v2' is present."
		return finalResponse
	}
	config.DebugLogger.Printf("Old account token (suffix): ...%s", auth.SafeSuffix(oldAccountToken, 6))
	config.DebugLogger.Printf("New account token (suffix): ...%s", auth.SafeSuffix(newAccountToken, 6))

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
	subredditChunkSize := config.DefaultSubredditChunkSize // Initial chunk size for subscribing.
	maxRetryAttempts := config.MaxSubredditRetryAttempts   // Maximum number of retry attempts.

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

// HandleCustomMigration processes a custom selection migration request
// It migrates only the selected subreddits and posts instead of all items
func HandleCustomMigration(req types.CustomMigrationRequest) types.MigrationResponseType {
	var finalResponse types.MigrationResponseType
	finalResponse.Success = false // Default to false

	config.InfoLogger.Printf("Starting custom migration process with %d subreddits and %d posts",
		len(req.SelectedSubreddits), len(req.SelectedPosts))

	// Verify cookies and get usernames
	oldAccountUsername, err := auth.GetUsernameFromCookie(req.OldAccountCookie)
	if err != nil {
		config.ErrorLogger.Printf("Failed to verify old account cookie: %v", err)
		finalResponse.Message = fmt.Sprintf("Failed to verify old account cookie: %v", err)
		return finalResponse
	}

	newAccountUsername, err := auth.GetUsernameFromCookie(req.NewAccountCookie)
	if err != nil {
		config.ErrorLogger.Printf("Failed to verify new account cookie: %v", err)
		finalResponse.Message = fmt.Sprintf("Failed to verify new account cookie: %v", err)
		return finalResponse
	}

	config.InfoLogger.Printf("Verified accounts for custom migration: %s -> %s", oldAccountUsername, newAccountUsername)

	// Parse OAuth tokens from cookies
	oldAccountToken := auth.ParseTokenFromCookie(req.OldAccountCookie)
	newAccountToken := auth.ParseTokenFromCookie(req.NewAccountCookie)
	if oldAccountToken == "" || newAccountToken == "" {
		config.ErrorLogger.Println("Failed to parse OAuth tokens from one or both cookies.")
		finalResponse.Message = "Failed to parse OAuth tokens from cookies. Ensure 'token_v2' is present."
		return finalResponse
	}

	// Handle selected subreddits migration
	if len(req.SelectedSubreddits) > 0 {
		config.InfoLogger.Printf("Migrating %d selected subreddits", len(req.SelectedSubreddits))
		subscribeResult := reddit.ManageSubreddits(newAccountToken, req.SelectedSubreddits, types.SubscribeAction, 100)
		finalResponse.Data.SubscribeSubreddit = subscribeResult

		// Handle deletion if requested
		if req.DeleteOldSubreddits {
			config.InfoLogger.Printf("Deleting %d selected subreddits from old account", len(req.SelectedSubreddits))
			unsubscribeResult := reddit.ManageSubreddits(oldAccountToken, req.SelectedSubreddits, types.UnsubscribeAction, 100)
			finalResponse.Data.UnsubscribeSubreddit = unsubscribeResult
		}
	} else {
		config.InfoLogger.Println("No subreddits selected for migration")
	}

	// Handle selected posts migration
	if len(req.SelectedPosts) > 0 {
		config.InfoLogger.Printf("Migrating %d selected posts", len(req.SelectedPosts))

		concurrencyForPosts := config.DefaultPostConcurrency
		saveResult := reddit.ManageSavedPosts(newAccountToken, req.SelectedPosts, types.SaveAction, concurrencyForPosts)
		finalResponse.Data.SavePost = saveResult

		// Handle deletion if requested
		if req.DeleteOldPosts {
			config.InfoLogger.Printf("Deleting %d selected posts from old account", len(req.SelectedPosts))
			unsaveResult := reddit.ManageSavedPosts(oldAccountToken, req.SelectedPosts, types.UnsaveAction, concurrencyForPosts)
			finalResponse.Data.UnsavePost = unsaveResult
		}
	} else {
		config.InfoLogger.Println("No posts selected for migration")
	}

	// Determine overall success and message
	hasErrors := finalResponse.Data.SubscribeSubreddit.Error ||
		finalResponse.Data.UnsubscribeSubreddit.Error ||
		finalResponse.Data.SavePost.FailedCount > 0 ||
		finalResponse.Data.UnsavePost.FailedCount > 0

	if hasErrors {
		finalResponse.Success = false
		finalResponse.Message = "Custom migration completed with some errors. Check individual operation statuses."
		config.InfoLogger.Println("Custom migration process completed with some errors.")
	} else {
		finalResponse.Success = true
		finalResponse.Message = "Custom migration completed successfully."
		config.InfoLogger.Println("Custom migration process completed successfully.")
	}

	config.InfoLogger.Printf("Custom migration summary - Subreddits subscribed: %d, Posts saved: %d",
		finalResponse.Data.SubscribeSubreddit.SuccessCount, finalResponse.Data.SavePost.SuccessCount)

	return finalResponse
}
