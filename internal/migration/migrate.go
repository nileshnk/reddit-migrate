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
// It orchestrates the entire migration process based on the provided source and destination account credentials and user preferences.
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

	if requestBody.AuthMethod == "oauth" {
		config.InfoLogger.Printf("OAuth migration request validated for %s. Source token ends: ...%s, Dest token ends: ...%s",
			r.RemoteAddr,
			auth.SafeSuffix(requestBody.SourceAccountToken, 6),
			auth.SafeSuffix(requestBody.DestAccountToken, 6))
	} else {
		config.InfoLogger.Printf("Cookie migration request validated for %s. Source cookie ends: ...%s, Dest cookie ends: ...%s",
			r.RemoteAddr,
			auth.SafeSuffix(requestBody.SourceAccountCookie, 6),
			auth.SafeSuffix(requestBody.DestAccountCookie, 6))
	}
	config.DebugLogger.Printf("Migration preferences: %+v", requestBody.Preferences)

	// Perform the migration.
	finalResponse := initializeMigration(requestBody)

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

// initializeMigration orchestrates the entire migration process based on authentication data and preferences.
// It verifies accounts, fetches data, and performs migration actions like subscribing/unsubscribing subreddits and saving/unsaving posts.
func initializeMigration(req types.MigrationRequestType) types.MigrationResponseType {
	var finalResponse types.MigrationResponseType
	finalResponse.Success = false // Default to false

	config.InfoLogger.Println("Starting migration process...")

	// Extract authentication data for source account
	var sourceAccountToken, sourceAccountUsername string
	var err error

	if req.AuthMethod == "oauth" {
		sourceAccountToken = req.SourceAccountToken
		// Use provided username if available, otherwise get from OAuth token
		if req.SourceAccountUsername != "" {
			sourceAccountUsername = req.SourceAccountUsername
		} else {
			// Get username from OAuth token
			userInfo, err := auth.GetUserInfoWithToken(sourceAccountToken)
			if err != nil {
				config.ErrorLogger.Printf("Failed to verify source account OAuth token: %v", err)
				finalResponse.Message = fmt.Sprintf("Failed to verify source account OAuth token: %v", err)
				return finalResponse
			}
			sourceAccountUsername = userInfo.Data.Name
		}
	} else {
		// Cookie-based authentication (default/backward compatibility)
		sourceAccountUsername, err = auth.GetUsernameFromCookie(req.SourceAccountCookie)
		if err != nil {
			config.ErrorLogger.Printf("Failed to verify source account cookie: %v", err)
			finalResponse.Message = fmt.Sprintf("Failed to verify source account cookie: %v", err)
			return finalResponse
		}
		sourceAccountToken = auth.ParseTokenFromCookie(req.SourceAccountCookie)
		if sourceAccountToken == "" {
			config.ErrorLogger.Println("Failed to parse OAuth token from source account cookie.")
			finalResponse.Message = "Failed to parse OAuth token from source account cookie. Ensure 'token_v2' is present."
			return finalResponse
		}
	}

	// Extract authentication data for destination account
	var destAccountToken, destAccountUsername string

	if req.AuthMethod == "oauth" {
		destAccountToken = req.DestAccountToken
		// Use provided username if available, otherwise get from OAuth token
		if req.DestAccountUsername != "" {
			destAccountUsername = req.DestAccountUsername
		} else {
			// Get username from OAuth token
			userInfo, err := auth.GetUserInfoWithToken(destAccountToken)
			if err != nil {
				config.ErrorLogger.Printf("Failed to verify destination account OAuth token: %v", err)
				finalResponse.Message = fmt.Sprintf("Failed to verify destination account OAuth token: %v", err)
				return finalResponse
			}
			destAccountUsername = userInfo.Data.Name
		}
	} else {
		// Cookie-based authentication (default/backward compatibility)
		destAccountUsername, err = auth.GetUsernameFromCookie(req.DestAccountCookie)
		if err != nil {
			config.ErrorLogger.Printf("Failed to verify destination account cookie: %v", err)
			finalResponse.Message = fmt.Sprintf("Failed to verify destination account cookie: %v", err)
			return finalResponse
		}
		destAccountToken = auth.ParseTokenFromCookie(req.DestAccountCookie)
		if destAccountToken == "" {
			config.ErrorLogger.Println("Failed to parse OAuth token from destination account cookie.")
			finalResponse.Message = "Failed to parse OAuth token from destination account cookie. Ensure 'token_v2' is present."
			return finalResponse
		}
	}

	config.InfoLogger.Printf("Verified source account: %s, destination account: %s", sourceAccountUsername, destAccountUsername)
	config.DebugLogger.Printf("Source account token (suffix): ...%s", auth.SafeSuffix(sourceAccountToken, 6))
	config.DebugLogger.Printf("Destination account token (suffix): ...%s", auth.SafeSuffix(destAccountToken, 6))

	// Handle subreddit migration/deletion.
	if req.Preferences.MigrateSubredditBool || req.Preferences.DeleteSubredditBool {
		if err := processSubreddits(sourceAccountToken, destAccountToken, sourceAccountUsername, destAccountUsername, req.Preferences, &finalResponse.Data); err != nil {
			config.ErrorLogger.Printf("Error processing subreddits: %v", err)
			// Message is set within processSubreddits or its sub-functions for partial success.
			// If a critical error occurs, it might stop here.
		}
	}

	// Handle post migration/deletion.
	if req.Preferences.MigratePostBool || req.Preferences.DeletePostBool {
		if err := processPosts(sourceAccountToken, destAccountToken, sourceAccountUsername, destAccountUsername, req.Preferences, &finalResponse.Data); err != nil {
			config.ErrorLogger.Printf("Error processing posts: %v", err)
		}
	}

	// Handle comment migration/deletion.
	if req.Preferences.MigrateCommentBool || req.Preferences.DeleteCommentBool {
		if err := processComments(sourceAccountToken, destAccountToken, sourceAccountUsername, destAccountUsername, req.Preferences, &finalResponse.Data); err != nil {
			config.ErrorLogger.Printf("Error processing comments: %v", err)
		}
	}

	// Handle multireddit migration.
	if req.Preferences.MigrateMultiredditBool {
		if err := processMultireddits(sourceAccountToken, destAccountToken, destAccountUsername, &finalResponse.Data); err != nil {
			config.ErrorLogger.Printf("Error processing multireddits: %v", err)
		}
	}

	// Determine overall success and message.
	if finalResponse.Data.SubscribeSubreddit.Error || finalResponse.Data.UnsubscribeSubreddit.Error ||
		finalResponse.Data.SavePost.FailedCount > 0 || finalResponse.Data.UnsavePost.FailedCount > 0 ||
		finalResponse.Data.SaveComment.FailedCount > 0 || finalResponse.Data.UnsaveComment.FailedCount > 0 ||
		finalResponse.Data.CreateMultireddit.FailedCount > 0 {
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

// filterSlice removes items from a source slice that are present in a toRemoveItems slice.
func filterSlice(source []string, toRemoveItems []string) []string {
	toRemoveMap := make(map[string]bool)
	for _, item := range toRemoveItems {
		toRemoveMap[item] = true
	}

	var filteredSlice []string
	for _, item := range source {
		if !toRemoveMap[item] {
			filteredSlice = append(filteredSlice, item)
		}
	}
	return filteredSlice
}

// processSubreddits handles the migration and/or deletion of subreddits.
func processSubreddits(sourceToken, destToken, sourceUser, destUser string, prefs types.PreferencesType, responseData *types.MigrationDetails) error {
	config.InfoLogger.Println("Fetching all subreddit and followed user names from source account...")
	sourceSubredditNameList, err := reddit.FetchSubredditFullNames(sourceToken)
	if err != nil {
		return fmt.Errorf("failed to fetch subreddit names from source account: %w", err)
	}
	config.InfoLogger.Printf("Fetched %d subreddits and %d followed users from %s.",
		len(sourceSubredditNameList.DisplayNamesList),
		len(sourceSubredditNameList.UserDisplayNameList), sourceUser)

	// Migrate (subscribe) subreddits to the destination account.
	if prefs.MigrateSubredditBool {
		config.InfoLogger.Printf("Fetching subreddits from destination account %s to filter out duplicates...", destUser)
		destSubredditNameList, err := reddit.FetchSubredditFullNames(destToken)

		subredditsToMigrate := sourceSubredditNameList.DisplayNamesList
		followedToMigrate := sourceSubredditNameList.UserDisplayNameList

		if err != nil {
			config.ErrorLogger.Printf("Could not fetch subreddits from destination account. Proceeding with all subreddits and followed users. Error: %v", err)
		} else {
			subredditsToMigrate = filterSlice(sourceSubredditNameList.DisplayNamesList, destSubredditNameList.DisplayNamesList)
			config.InfoLogger.Printf("Filtered selection: %d subreddits to migrate after removing duplicates.", len(subredditsToMigrate))

			followedToMigrate = filterSlice(sourceSubredditNameList.UserDisplayNameList, destSubredditNameList.UserDisplayNameList)
			config.InfoLogger.Printf("Filtered selection: %d followed users to migrate after removing duplicates.", len(followedToMigrate))
		}

		if len(subredditsToMigrate) > 0 {
			config.InfoLogger.Printf("Starting subreddit migration for %s -> %s.", sourceUser, destUser)
			responseData.SubscribeSubreddit = migrateSubredditsWithRetry(destToken, subredditsToMigrate, destUser)
		} else {
			config.InfoLogger.Printf("No new subreddits to migrate for %s.", destUser)
		}

		if len(followedToMigrate) > 0 {
			config.InfoLogger.Printf("Starting followed user migration for %s -> %s.", sourceUser, destUser)
			followedUsersResult := reddit.ManageFollowedUsers(destToken, followedToMigrate, types.SubscribeAction)
			config.InfoLogger.Printf("Followed %d users for %s (failed: %d).", followedUsersResult.SuccessCount, destUser, followedUsersResult.FailedCount)
		} else {
			config.InfoLogger.Printf("No followed users to migrate for %s.", sourceUser)
		}
	}

	// Delete (unsubscribe) subreddits from the source account.
	if prefs.DeleteSubredditBool {
		config.InfoLogger.Printf("Starting subreddit deletion (unsubscribing) from %s.", sourceUser)
		unsubscribeData := reddit.ManageSubreddits(sourceToken, sourceSubredditNameList.DisplayNamesList, types.UnsubscribeAction, 500)
		config.InfoLogger.Printf("Unsubscribed %d subreddits from %s (failed: %d).", unsubscribeData.SuccessCount, sourceUser, unsubscribeData.FailedCount)
		responseData.UnsubscribeSubreddit = unsubscribeData
	}
	return nil
}

// migrateSubredditsWithRetry attempts to subscribe to subreddits with a retry mechanism.
func migrateSubredditsWithRetry(token string, displayNames []string, username string) types.ManageSubredditResponseType { // Adjusted type
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
func processPosts(sourceToken, destToken, sourceUser, destUser string, prefs types.PreferencesType, responseData *types.MigrationDetails) error {
	config.InfoLogger.Printf("Fetching saved post full names from source account %s...", sourceUser)

	sourceSavedPostsFullNamesList, err := reddit.FetchSavedPostsFullNames(sourceToken, sourceUser)
	if err != nil {
		return fmt.Errorf("failed to fetch saved post names from %s: %w", sourceUser, err)
	}

	config.InfoLogger.Printf("Fetching saved post full names from destination account %s...", destUser)

	destSavedPostsFullNamesList, err := reddit.FetchSavedPostsFullNames(destToken, destUser)
	if err != nil {
		return fmt.Errorf("failed to fetch saved post names from %s: %w", destUser, err)
	}

	config.InfoLogger.Printf("Analyzing and selecting only posts not in destination account %s from source account %s...", destUser, sourceUser)

	// Filter out posts that are already saved in the destination account
	savedPostsFullNamesList := filterSlice(sourceSavedPostsFullNamesList, destSavedPostsFullNamesList)

	config.InfoLogger.Printf("Found %d unique posts in source account that aren't in destination account", len(savedPostsFullNamesList))

	// Reverse the order so that oldest posts are saved first to maintain chronological order in destination account
	// Reddit API returns newest posts first, but we want oldest posts to be saved first so they appear at bottom
	for i, j := 0, len(savedPostsFullNamesList)-1; i < j; i, j = i+1, j-1 {
		savedPostsFullNamesList[i], savedPostsFullNamesList[j] = savedPostsFullNamesList[j], savedPostsFullNamesList[i]
	}

	config.InfoLogger.Printf("Fetched %d saved posts from %s.", len(savedPostsFullNamesList), sourceUser)

	concurrencyForPosts := config.DefaultPostConcurrency

	if prefs.MigratePostBool {
		config.InfoLogger.Printf("Starting saved post migration for %s -> %s (%d posts).", sourceUser, destUser, len(savedPostsFullNamesList))
		savePostsResponse := reddit.ManageSavedPosts(destToken, savedPostsFullNamesList, types.SaveAction, concurrencyForPosts)
		config.InfoLogger.Printf("Saved %d posts to %s (failed: %d).", savePostsResponse.SuccessCount, destUser, savePostsResponse.FailedCount)
		responseData.SavePost = savePostsResponse
	}

	if prefs.DeletePostBool {
		config.InfoLogger.Printf("Starting saved post deletion (unsaving) from %s (%d posts).", sourceUser, len(savedPostsFullNamesList))
		unsavePostsResponse := reddit.ManageSavedPosts(sourceToken, savedPostsFullNamesList, types.UnsaveAction, concurrencyForPosts)
		config.InfoLogger.Printf("Unsaved %d posts from %s (failed: %d).", unsavePostsResponse.SuccessCount, sourceUser, unsavePostsResponse.FailedCount)
		responseData.UnsavePost = unsavePostsResponse
	}
	return nil
}

// processComments handles the migration and/or deletion of saved comments.
func processComments(sourceToken, destToken, sourceUser, destUser string, prefs types.PreferencesType, responseData *types.MigrationDetails) error {
	config.InfoLogger.Printf("Fetching saved comment full names from source account %s...", sourceUser)

	sourceSavedCommentsFullNamesList, err := reddit.FetchSavedCommentFullNames(sourceToken, sourceUser)
	if err != nil {
		return fmt.Errorf("failed to fetch saved comment names from %s: %w", sourceUser, err)
	}

	destSavedCommentsFullNamesList, err := reddit.FetchSavedCommentFullNames(destToken, destUser)
	if err != nil {
		return fmt.Errorf("failed to fetch saved comment names from %s: %w", destUser, err)
	}

	// Filter out comments that are already saved in the destination account
	commentsToMigrate := filterSlice(sourceSavedCommentsFullNamesList, destSavedCommentsFullNamesList)

	config.InfoLogger.Printf("Found %d unique comments in source account that aren't in destination account", len(commentsToMigrate))

	// Reverse the order so that oldest comments are saved first to maintain chronological order
	for i, j := 0, len(commentsToMigrate)-1; i < j; i, j = i+1, j-1 {
		commentsToMigrate[i], commentsToMigrate[j] = commentsToMigrate[j], commentsToMigrate[i]
	}

	concurrency := config.DefaultPostConcurrency

	// ManageSavedPosts works for comments too — POST /api/save accepts both t3_ and t1_ IDs
	if prefs.MigrateCommentBool {
		config.InfoLogger.Printf("Starting saved comment migration for %s -> %s (%d comments).", sourceUser, destUser, len(commentsToMigrate))
		saveResult := reddit.ManageSavedPosts(destToken, commentsToMigrate, types.SaveAction, concurrency)
		config.InfoLogger.Printf("Saved %d comments to %s (failed: %d).", saveResult.SuccessCount, destUser, saveResult.FailedCount)
		responseData.SaveComment = saveResult
	}

	if prefs.DeleteCommentBool {
		config.InfoLogger.Printf("Starting saved comment deletion (unsaving) from %s (%d comments).", sourceUser, len(commentsToMigrate))
		unsaveResult := reddit.ManageSavedPosts(sourceToken, commentsToMigrate, types.UnsaveAction, concurrency)
		config.InfoLogger.Printf("Unsaved %d comments from %s (failed: %d).", unsaveResult.SuccessCount, sourceUser, unsaveResult.FailedCount)
		responseData.UnsaveComment = unsaveResult
	}
	return nil
}

// processMultireddits handles the migration of multireddits from source to destination account.
func processMultireddits(sourceToken, destToken, destUser string, responseData *types.MigrationDetails) error {
	config.InfoLogger.Println("Fetching multireddits from source account...")

	sourceMultis, err := reddit.FetchMultireddits(sourceToken)
	if err != nil {
		return fmt.Errorf("failed to fetch multireddits from source account: %w", err)
	}

	if len(sourceMultis) == 0 {
		config.InfoLogger.Println("No multireddits found in source account.")
		return nil
	}

	// Fetch destination multireddits to filter duplicates by name
	destMultis, err := reddit.FetchMultireddits(destToken)
	multisToCreate := sourceMultis
	if err != nil {
		config.ErrorLogger.Printf("Could not fetch multireddits from destination account. Proceeding with all %d multireddits. Error: %v", len(sourceMultis), err)
	} else {
		destMultiNames := make(map[string]bool)
		for _, m := range destMultis {
			destMultiNames[m.Name] = true
		}
		var filtered []types.MultiredditInfo
		for _, m := range sourceMultis {
			if !destMultiNames[m.Name] {
				filtered = append(filtered, m)
			}
		}
		multisToCreate = filtered
		config.InfoLogger.Printf("Filtered: %d multireddits to create after removing %d duplicates.", len(multisToCreate), len(sourceMultis)-len(multisToCreate))
	}

	if len(multisToCreate) > 0 {
		result := reddit.ManageMultireddits(destToken, destUser, multisToCreate)
		responseData.CreateMultireddit = result
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

	config.InfoLogger.Printf("Starting custom migration process with %d subreddits, %d posts, and %d comments",
		len(req.SelectedSubreddits), len(req.SelectedPosts), len(req.SelectedComments))

	// Extract authentication data for source account
	var sourceAccountToken, sourceAccountUsername string
	var err error

	if req.SourceAccountCookie == "" && req.SourceAccountToken == "" {
		// No source credentials supplied — items were resolved without a live source account
		// (e.g. imported from a Reddit data-export CSV). Skip source verification entirely.
		config.InfoLogger.Println("No source account credentials supplied; proceeding without a live source account.")
	} else if req.AuthMethod == "oauth" {
		sourceAccountToken = req.SourceAccountToken
		// Use provided username if available, otherwise get from OAuth token
		if req.SourceAccountUsername != "" {
			sourceAccountUsername = req.SourceAccountUsername
		} else {
			// Get username from OAuth token
			userInfo, err := auth.GetUserInfoWithToken(sourceAccountToken)
			if err != nil {
				config.ErrorLogger.Printf("Failed to verify source account OAuth token: %v", err)
				finalResponse.Message = fmt.Sprintf("Failed to verify source account OAuth token: %v", err)
				return finalResponse
			}
			sourceAccountUsername = userInfo.Data.Name
		}
	} else {
		// Cookie-based authentication (default/backward compatibility)
		sourceAccountUsername, err = auth.GetUsernameFromCookie(req.SourceAccountCookie)
		if err != nil {
			config.ErrorLogger.Printf("Failed to verify source account cookie: %v", err)
			finalResponse.Message = fmt.Sprintf("Failed to verify source account cookie: %v", err)
			return finalResponse
		}
		sourceAccountToken = auth.ParseTokenFromCookie(req.SourceAccountCookie)
		if sourceAccountToken == "" {
			config.ErrorLogger.Println("Failed to parse OAuth token from source account cookie.")
			finalResponse.Message = "Failed to parse OAuth token from source account cookie. Ensure 'token_v2' is present."
			return finalResponse
		}
	}

	// Extract authentication data for destination account
	var destAccountToken, destAccountUsername string

	if req.AuthMethod == "oauth" {
		destAccountToken = req.DestAccountToken
		// Use provided username if available, otherwise get from OAuth token
		if req.DestAccountUsername != "" {
			destAccountUsername = req.DestAccountUsername
		} else {
			// Get username from OAuth token
			userInfo, err := auth.GetUserInfoWithToken(destAccountToken)
			if err != nil {
				config.ErrorLogger.Printf("Failed to verify destination account OAuth token: %v", err)
				finalResponse.Message = fmt.Sprintf("Failed to verify destination account OAuth token: %v", err)
				return finalResponse
			}
			destAccountUsername = userInfo.Data.Name
		}
	} else {
		// Cookie-based authentication (default/backward compatibility)
		destAccountUsername, err = auth.GetUsernameFromCookie(req.DestAccountCookie)
		if err != nil {
			config.ErrorLogger.Printf("Failed to verify destination account cookie: %v", err)
			finalResponse.Message = fmt.Sprintf("Failed to verify destination account cookie: %v", err)
			return finalResponse
		}
		destAccountToken = auth.ParseTokenFromCookie(req.DestAccountCookie)
		if destAccountToken == "" {
			config.ErrorLogger.Println("Failed to parse OAuth token from destination account cookie.")
			finalResponse.Message = "Failed to parse OAuth token from destination account cookie. Ensure 'token_v2' is present."
			return finalResponse
		}
	}

	config.InfoLogger.Printf("Verified accounts for custom migration: %s -> %s", sourceAccountUsername, destAccountUsername)

	// Handle selected subreddits migration
	if len(req.SelectedSubreddits) > 0 {
		config.InfoLogger.Printf("Migrating %d selected subreddits", len(req.SelectedSubreddits))
		config.InfoLogger.Printf("Fetching subreddits from destination account %s to filter out duplicates...", destAccountUsername)
		destSubredditNameList, err := reddit.FetchSubredditFullNames(destAccountToken)

		subredditsToMigrate := req.SelectedSubreddits

		if err != nil {
			config.ErrorLogger.Printf("Could not fetch subreddits from destination account. Proceeding with all %d selected subreddits. Error: %v", len(req.SelectedSubreddits), err)
		} else {
			subredditsToMigrate = filterSlice(req.SelectedSubreddits, destSubredditNameList.DisplayNamesList)
			config.InfoLogger.Printf("Filtered selection: %d subreddits to migrate after removing %d duplicates.", len(subredditsToMigrate), len(req.SelectedSubreddits)-len(subredditsToMigrate))
		}

		if len(subredditsToMigrate) > 0 {
			subscribeResult := reddit.ManageSubreddits(destAccountToken, subredditsToMigrate, types.SubscribeAction, 100)
			finalResponse.Data.SubscribeSubreddit = subscribeResult

			// Handle deletion if requested
			if req.DeleteSourceSubreddits && sourceAccountToken != "" {
				config.InfoLogger.Printf("Deleting %d selected subreddits from source account", len(subredditsToMigrate))
				unsubscribeResult := reddit.ManageSubreddits(sourceAccountToken, subredditsToMigrate, types.UnsubscribeAction, 100)
				finalResponse.Data.UnsubscribeSubreddit = unsubscribeResult
			}
		} else {
			config.InfoLogger.Println("No new subreddits to migrate from selection.")
		}
	} else {
		config.InfoLogger.Println("No subreddits selected for migration")
	}

	// Handle selected posts migration
	if len(req.SelectedPosts) > 0 {
		config.InfoLogger.Printf("Migrating %d selected posts", len(req.SelectedPosts))

		// Fetch saved posts from destination account to avoid duplicates
		config.InfoLogger.Printf("Fetching saved posts from destination account %s to avoid duplicates...", destAccountUsername)
		destSavedPosts, err := reddit.FetchSavedPostsFullNames(destAccountToken, destAccountUsername)
		postsToMigrate := req.SelectedPosts
		if err != nil {
			config.ErrorLogger.Printf("Could not fetch saved posts from destination account. Proceeding with all %d selected posts. Error: %v", len(req.SelectedPosts), err)
		} else {
			postsToMigrate = filterSlice(req.SelectedPosts, destSavedPosts)
			config.InfoLogger.Printf("Filtered selection: %d posts to migrate after removing %d duplicates.", len(postsToMigrate), len(req.SelectedPosts)-len(postsToMigrate))
		}

		if len(postsToMigrate) > 0 {
			// Reverse the order so that oldest posts are saved first to maintain chronological order in destination account
			for i, j := 0, len(postsToMigrate)-1; i < j; i, j = i+1, j-1 {
				postsToMigrate[i], postsToMigrate[j] = postsToMigrate[j], postsToMigrate[i]
			}

			concurrencyForPosts := config.DefaultPostConcurrency
			saveResult := reddit.ManageSavedPosts(destAccountToken, postsToMigrate, types.SaveAction, concurrencyForPosts)
			finalResponse.Data.SavePost = saveResult

			// Handle deletion if requested
			if req.DeleteSourcePosts && sourceAccountToken != "" {
				config.InfoLogger.Printf("Deleting %d selected posts from source account", len(postsToMigrate))
				unsaveResult := reddit.ManageSavedPosts(sourceAccountToken, postsToMigrate, types.UnsaveAction, concurrencyForPosts)
				finalResponse.Data.UnsavePost = unsaveResult
			}
		} else {
			config.InfoLogger.Println("No new posts to migrate from selection.")
		}
	} else {
		config.InfoLogger.Println("No posts selected for migration")
	}

	// Handle selected comments migration
	if len(req.SelectedComments) > 0 {
		config.InfoLogger.Printf("Migrating %d selected comments", len(req.SelectedComments))

		// Fetch saved comments from destination account to avoid duplicates
		config.InfoLogger.Printf("Fetching saved comments from destination account %s to avoid duplicates...", destAccountUsername)
		destSavedComments, err := reddit.FetchSavedCommentFullNames(destAccountToken, destAccountUsername)
		commentsToMigrate := req.SelectedComments
		if err != nil {
			config.ErrorLogger.Printf("Could not fetch saved comments from destination account. Proceeding with all %d selected comments. Error: %v", len(req.SelectedComments), err)
		} else {
			commentsToMigrate = filterSlice(req.SelectedComments, destSavedComments)
			config.InfoLogger.Printf("Filtered selection: %d comments to migrate after removing %d duplicates.", len(commentsToMigrate), len(req.SelectedComments)-len(commentsToMigrate))
		}

		if len(commentsToMigrate) > 0 {
			// Reverse order for chronological preservation
			for i, j := 0, len(commentsToMigrate)-1; i < j; i, j = i+1, j-1 {
				commentsToMigrate[i], commentsToMigrate[j] = commentsToMigrate[j], commentsToMigrate[i]
			}

			concurrency := config.DefaultPostConcurrency
			saveResult := reddit.ManageSavedPosts(destAccountToken, commentsToMigrate, types.SaveAction, concurrency)
			finalResponse.Data.SaveComment = saveResult

			if req.DeleteSourceComments && sourceAccountToken != "" {
				config.InfoLogger.Printf("Deleting %d selected comments from source account", len(commentsToMigrate))
				unsaveResult := reddit.ManageSavedPosts(sourceAccountToken, commentsToMigrate, types.UnsaveAction, concurrency)
				finalResponse.Data.UnsaveComment = unsaveResult
			}
		} else {
			config.InfoLogger.Println("No new comments to migrate from selection.")
		}
	} else {
		config.InfoLogger.Println("No comments selected for migration")
	}

	// Handle selected multireddits migration
	if len(req.SelectedMultireddits) > 0 {
		config.InfoLogger.Printf("Migrating %d selected multireddits", len(req.SelectedMultireddits))

		// Fetch full multireddit data from source to get subreddit lists
		sourceMultis, err := reddit.FetchMultireddits(sourceAccountToken)
		if err != nil {
			config.ErrorLogger.Printf("Failed to fetch multireddit details from source: %v", err)
		} else {
			// Filter to only selected multireddits
			selectedSet := make(map[string]bool)
			for _, name := range req.SelectedMultireddits {
				selectedSet[name] = true
			}
			var multisToCreate []types.MultiredditInfo
			for _, m := range sourceMultis {
				if selectedSet[m.Name] {
					multisToCreate = append(multisToCreate, m)
				}
			}

			// Filter out duplicates on destination
			destMultis, err := reddit.FetchMultireddits(destAccountToken)
			if err != nil {
				config.ErrorLogger.Printf("Could not fetch multireddits from destination. Proceeding with all %d selected. Error: %v", len(multisToCreate), err)
			} else {
				destNames := make(map[string]bool)
				for _, m := range destMultis {
					destNames[m.Name] = true
				}
				var filtered []types.MultiredditInfo
				for _, m := range multisToCreate {
					if !destNames[m.Name] {
						filtered = append(filtered, m)
					}
				}
				multisToCreate = filtered
			}

			if len(multisToCreate) > 0 {
				result := reddit.ManageMultireddits(destAccountToken, destAccountUsername, multisToCreate)
				finalResponse.Data.CreateMultireddit = result
			}
		}
	} else {
		config.InfoLogger.Println("No multireddits selected for migration")
	}

	// Determine overall success and message
	hasErrors := finalResponse.Data.SubscribeSubreddit.Error ||
		finalResponse.Data.UnsubscribeSubreddit.Error ||
		finalResponse.Data.SavePost.FailedCount > 0 ||
		finalResponse.Data.UnsavePost.FailedCount > 0 ||
		finalResponse.Data.SaveComment.FailedCount > 0 ||
		finalResponse.Data.UnsaveComment.FailedCount > 0 ||
		finalResponse.Data.CreateMultireddit.FailedCount > 0

	if hasErrors {
		finalResponse.Success = false
		finalResponse.Message = "Custom migration completed with some errors. Check individual operation statuses."
		config.InfoLogger.Println("Custom migration process completed with some errors.")
	} else {
		finalResponse.Success = true
		finalResponse.Message = "Custom migration completed successfully."
		config.InfoLogger.Println("Custom migration process completed successfully.")
	}

	config.InfoLogger.Printf("Custom migration summary - Subreddits subscribed: %d, Posts saved: %d, Comments saved: %d",
		finalResponse.Data.SubscribeSubreddit.SuccessCount, finalResponse.Data.SavePost.SuccessCount, finalResponse.Data.SaveComment.SuccessCount)

	return finalResponse
}
