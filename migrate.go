package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// MigrationHandler is the HTTP handler for the /migrate endpoint.
// It orchestrates the entire migration process based on the provided old and new account cookies and user preferences.
func MigrationHandler(w http.ResponseWriter, r *http.Request) {
	DebugLogger.Printf("Received migration request from %s", r.RemoteAddr)

	// Validate request content type.
	if r.Header.Get("Content-Type") != "application/json" {
		ErrorLogger.Printf("Invalid content type from %s: %s", r.RemoteAddr, r.Header.Get("Content-Type"))
		errorResponse(w, "Content Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	// Decode request body.
	var requestBody migration_request_type
	if err := decodeMigrationRequest(r, &requestBody); err != nil {
		ErrorLogger.Printf("Error decoding migration request from %s: %v", r.RemoteAddr, err)
		// errorResponse is called within decodeMigrationRequest for specific errors
		if !strings.Contains(err.Error(), "Bad Request") { // Avoid double response if errorResponse was already called
			errorResponse(w, fmt.Sprintf("Bad Request: %v", err), http.StatusBadRequest)
		}
		return
	}

	InfoLogger.Printf("Migration request validated for %s. Old cookie ends: ...%s, New cookie ends: ...%s",
		r.RemoteAddr,
		safeSuffix(requestBody.Old_account_cookie, 6),
		safeSuffix(requestBody.New_account_cookie, 6))
	DebugLogger.Printf("Migration preferences: %+v", requestBody.Preferences)

	// Perform the migration.
	finalResponse := initializeMigration(requestBody.Old_account_cookie, requestBody.New_account_cookie, requestBody.Preferences)

	// Send response.
	w.Header().Set("Content-Type", "application/json")
	jsonResp, err := json.Marshal(finalResponse)
	if err != nil {
		ErrorLogger.Printf("Error marshalling migration response for %s: %v", r.RemoteAddr, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(jsonResp); err != nil {
		ErrorLogger.Printf("Error writing migration response for %s: %v", r.RemoteAddr, err)
	} else {
		InfoLogger.Printf("Successfully sent migration response to %s. Success: %t", r.RemoteAddr, finalResponse.Success)
	}
}

// decodeMigrationRequest decodes the JSON request body into the migration_request_type struct.
// It handles potential unmarshalling errors and unknown fields.
func decodeMigrationRequest(r *http.Request, requestBody *migration_request_type) error {
	decoder := json.NewDecoder(r.Body)
	// decoder.DisallowUnknownFields() // Consider enabling this for stricter validation.

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
func initializeMigration(oldAccountCookie, newAccountCookie string, preferences preferences_type) migration_response_type {
	var finalResponse migration_response_type
	finalResponse.Success = false // Default to false

	InfoLogger.Println("Starting migration process...")

	// Verify cookies and get usernames.
	oldAccountUsername, err := getUsernameFromCookie(oldAccountCookie)
	if err != nil {
		ErrorLogger.Printf("Failed to verify old account cookie: %v", err)
		finalResponse.Message = fmt.Sprintf("Failed to verify old account cookie: %v", err)
		return finalResponse
	}
	newAccountUsername, err := getUsernameFromCookie(newAccountCookie)
	if err != nil {
		ErrorLogger.Printf("Failed to verify new account cookie: %v", err)
		finalResponse.Message = fmt.Sprintf("Failed to verify new account cookie: %v", err)
		return finalResponse
	}
	InfoLogger.Printf("Verified old account: %s, new account: %s", oldAccountUsername, newAccountUsername)

	// Parse OAuth tokens from cookies.
	oldAccountToken := parseTokenFromCookie(oldAccountCookie)
	newAccountToken := parseTokenFromCookie(newAccountCookie)
	if oldAccountToken == "" || newAccountToken == "" {
		ErrorLogger.Println("Failed to parse OAuth tokens from one or both cookies.")
		finalResponse.Message = "Failed to parse OAuth tokens from cookies. Ensure 'token_v2' is present."
		return finalResponse
	}
	DebugLogger.Printf("Old account token (suffix): ...%s", safeSuffix(oldAccountToken, 6))
	DebugLogger.Printf("New account token (suffix): ...%s", safeSuffix(newAccountToken, 6))

	// Handle subreddit migration/deletion.
	if preferences.Migrate_subreddit_bool || preferences.Delete_subreddit_bool {
		if err := processSubreddits(oldAccountToken, newAccountToken, oldAccountUsername, newAccountUsername, preferences, &finalResponse.Data); err != nil {
			ErrorLogger.Printf("Error processing subreddits: %v", err)
			// Message is set within processSubreddits or its sub-functions for partial success.
			// If a critical error occurs, it might stop here.
		}
	}

	// Handle post migration/deletion.
	if preferences.Migrate_post_bool || preferences.Delete_post_bool {
		if err := processPosts(oldAccountToken, newAccountToken, oldAccountUsername, newAccountUsername, preferences, &finalResponse.Data); err != nil {
			ErrorLogger.Printf("Error processing posts: %v", err)
			// Similar to subreddits, messages handled internally for partial success.
		}
	}

	// Determine overall success and message.
	// A more sophisticated check might be needed if partial successes are not considered overall success.
	if finalResponse.Data.SubscribeSubreddit.Error || finalResponse.Data.UnsubscribeSubreddit.Error ||
		finalResponse.Data.SavePost.FailedCount > 0 || finalResponse.Data.UnsavePost.FailedCount > 0 {
		finalResponse.Success = false
		finalResponse.Message = "Migration completed with some errors. Check individual operation statuses."
		InfoLogger.Println("Migration process completed with some errors.")
	} else {
		finalResponse.Success = true
		finalResponse.Message = "Migration completed successfully."
		InfoLogger.Println("Migration process completed successfully.")
	}

	return finalResponse
}

// processSubreddits handles the migration and/or deletion of subreddits.
func processSubreddits(oldToken, newToken, oldUser, newUser string, prefs preferences_type, responseData *MigrationData) error {
	InfoLogger.Println("Fetching all subreddit and followed user names from old account...")
	subredditNameList, err := fetchSubredditFullNames(oldToken)
	if err != nil {
		return fmt.Errorf("failed to fetch subreddit names from old account: %w", err)
	}
	InfoLogger.Printf("Fetched %d subreddits and %d followed users from %s.",
		len(subredditNameList.displayNamesList), len(subredditNameList.userDisplayNameList), oldUser)

	// Migrate (subscribe) subreddits to the new account.
	if prefs.Migrate_subreddit_bool {
		InfoLogger.Printf("Starting subreddit migration for %s -> %s.", oldUser, newUser)
		responseData.SubscribeSubreddit = migrateSubredditsWithRetry(newToken, subredditNameList.displayNamesList, newUser)

		InfoLogger.Printf("Starting followed user migration for %s -> %s.", oldUser, newUser)
		followedUsersResult := manageFollowedUsers(newToken, subredditNameList.userDisplayNameList, subscribe)
		InfoLogger.Printf("Followed %d users for %s (failed: %d).", followedUsersResult.SuccessCount, newUser, followedUsersResult.FailedCount)
		// Combine or report user following results separately if needed.
		// For now, it's logged, but not directly part of SubscribeSubreddit response.
		// Consider adding a field to migration_response_type.Data for followed users if detailed reporting is needed.
	}

	// Delete (unsubscribe) subreddits from the old account.
	if prefs.Delete_subreddit_bool {
		InfoLogger.Printf("Starting subreddit deletion (unsubscribing) from %s.", oldUser)
		// Using a larger chunk size for deletion as it's usually less critical if some fail compared to subscription.
		unsubscribeData := manageSubreddits(oldToken, subredditNameList.displayNamesList, unsubscribe, 500)
		InfoLogger.Printf("Unsubscribed %d subreddits from %s (failed: %d).", unsubscribeData.SuccessCount, oldUser, unsubscribeData.FailedCount)
		responseData.UnsubscribeSubreddit = unsubscribeData
	}
	return nil
}

// migrateSubredditsWithRetry attempts to subscribe to subreddits with a retry mechanism.
func migrateSubredditsWithRetry(token string, displayNames []string, username string) manage_subreddit_response_type {
	subredditChunkSize := 100 // Initial chunk size for subscribing.
	maxRetryAttempts := 5     // Maximum number of retry attempts.

	InfoLogger.Printf("Migrating %d subreddits to account %s.", len(displayNames), username)
	subscribeData := manageSubreddits(token, displayNames, subscribe, subredditChunkSize)
	InfoLogger.Printf("Initial subscription attempt for %s: %d successful, %d failed.", username, subscribeData.SuccessCount, subscribeData.FailedCount)

	retryAttempts := 1
	for subscribeData.FailedCount > 0 && retryAttempts <= maxRetryAttempts {
		InfoLogger.Printf("Retrying %d failed subreddits for %s (attempt %d/%d). Chunk size: %d",
			subscribeData.FailedCount, username, retryAttempts, maxRetryAttempts, subredditChunkSize/retryAttempts)

		// Retry only the failed subreddits.
		// The chunk size is reduced for retries to potentially avoid overwhelming the API.
		failedToRetry := subscribeData.FailedSubreddits
		subscribeData.FailedSubreddits = nil // Clear for the new retry attempt.
		subscribeData.FailedCount = 0        // Reset for the new retry attempt.

		retryResult := manageSubreddits(token, failedToRetry, subscribe, subredditChunkSize/retryAttempts)

		// Accumulate results:
		// SuccessCount from the initial attempt is already there. Add successes from retry.
		subscribeData.SuccessCount += retryResult.SuccessCount
		// FailedCount and FailedSubreddits are for the *current overall* status after this retry.
		subscribeData.FailedCount = retryResult.FailedCount
		subscribeData.FailedSubreddits = retryResult.FailedSubreddits // These are the ones still failing.

		InfoLogger.Printf("Retry attempt %d for %s: %d successful, %d still failed.",
			retryAttempts, username, retryResult.SuccessCount, retryResult.FailedCount)
		retryAttempts++
	}

	if subscribeData.FailedCount > 0 {
		ErrorLogger.Printf("Failed to migrate %d subreddits for %s after %d attempts. Failures: %v",
			subscribeData.FailedCount, username, maxRetryAttempts, subscribeData.FailedSubreddits)
	} else {
		InfoLogger.Printf("Successfully migrated all %d initially targeted subreddits for %s.", len(displayNames), username)
	}
	return subscribeData
}

// processPosts handles the migration and/or deletion of saved posts.
func processPosts(oldToken, newToken, oldUser, newUser string, prefs preferences_type, responseData *MigrationData) error {
	InfoLogger.Printf("Fetching saved post full names from old account %s...", oldUser)
	savedPostsFullNamesList, err := fetchSavedPostsFullNames(oldToken, oldUser)
	if err != nil {
		return fmt.Errorf("failed to fetch saved post names from %s: %w", oldUser, err)
	}
	InfoLogger.Printf("Fetched %d saved posts from %s.", len(savedPostsFullNamesList), oldUser)

	concurrencyForPosts := 10 // Concurrency level for post operations.

	// Migrate (save) posts to the new account.
	if prefs.Migrate_post_bool {
		InfoLogger.Printf("Starting saved post migration for %s -> %s (%d posts).", oldUser, newUser, len(savedPostsFullNamesList))
		savePostsResponse := manageSavedPosts(newToken, savedPostsFullNamesList, SAVE, concurrencyForPosts)
		InfoLogger.Printf("Saved %d posts to %s (failed: %d).", savePostsResponse.SuccessCount, newUser, savePostsResponse.FailedCount)
		responseData.SavePost = savePostsResponse
	}

	// Delete (unsave) posts from the old account.
	if prefs.Delete_post_bool {
		InfoLogger.Printf("Starting saved post deletion (unsaving) from %s (%d posts).", oldUser, len(savedPostsFullNamesList))
		unsavePostsResponse := manageSavedPosts(oldToken, savedPostsFullNamesList, UNSAVE, concurrencyForPosts)
		InfoLogger.Printf("Unsaved %d posts from %s (failed: %d).", unsavePostsResponse.SuccessCount, oldUser, unsavePostsResponse.FailedCount)
		responseData.UnsavePost = unsavePostsResponse
	}
	return nil
}

// manageSubreddits performs subscribe or unsubscribe actions on a list of subreddits in chunks.
// It aggregates results from chunk operations.
func manageSubreddits(token string, subredditDisplayNames []string, action subscribe_type, chunkSize int) manage_subreddit_response_type {
	if len(subredditDisplayNames) == 0 {
		DebugLogger.Printf("No subreddits to %s.", action)
		return manage_subreddit_response_type{SuccessCount: 0, FailedCount: 0}
	}
	if chunkSize <= 0 {
		chunkSize = 100 // Default chunk size if invalid.
		DebugLogger.Printf("Invalid chunk size for manageSubreddits, defaulting to %d", chunkSize)
	}

	InfoLogger.Printf("Managing %d subreddits with action '%s', chunk size %d.", len(subredditDisplayNames), action, chunkSize)
	chunks := chunkStringArray(subredditDisplayNames, chunkSize)
	var finalResponse manage_subreddit_response_type

	for i, chunk := range chunks {
		DebugLogger.Printf("Processing chunk %d/%d for %s action (size: %d).", i+1, len(chunks), action, len(chunk))
		response := manageSubredditChunk(token, chunk, action)
		finalResponse.SuccessCount += response.SuccessCount
		finalResponse.FailedCount += response.FailedCount
		if response.Error { // If any chunk has an error, mark the overall as having an error.
			finalResponse.Error = true
			finalResponse.StatusCode = response.StatusCode // Might be overwritten by later chunks, consider how to aggregate.
		}
		if response.FailedCount > 0 {
			finalResponse.FailedSubreddits = append(finalResponse.FailedSubreddits, response.FailedSubreddits...)
		}
	}
	DebugLogger.Printf("Finished managing subreddits with action '%s'. Success: %d, Failed: %d.", action, finalResponse.SuccessCount, finalResponse.FailedCount)
	return finalResponse
}

// manageSubredditChunk sends a request to Reddit API to subscribe/unsubscribe a single chunk of subreddits.
func manageSubredditChunk(token string, subredditDisplayNamesChunk []string, action subscribe_type) manage_subreddit_response_type {
	if len(subredditDisplayNamesChunk) == 0 {
		return manage_subreddit_response_type{SuccessCount: 0, FailedCount: 0}
	}

	subredditNames := strings.Join(subredditDisplayNamesChunk, ",")
	requestBodyStr := fmt.Sprintf("sr_name=%s&action=%s&api_type=json", subredditNames, action)
	requestBodyBytes := []byte(requestBodyStr)

	req, err := http.NewRequest(http.MethodPost, "https://oauth.reddit.com/api/subscribe", bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		ErrorLogger.Printf("Error creating request for %s subreddits: %v. Subreddits: %v", action, err, subredditDisplayNamesChunk)
		return manage_subreddit_response_type{
			Error:            true,
			StatusCode:       0, // No HTTP status code as request creation failed.
			FailedCount:      len(subredditDisplayNamesChunk),
			FailedSubreddits: subredditDisplayNamesChunk,
		}
	}

	req.Header = http.Header{
		"Authorization": {"Bearer " + token},
		"Content-Type":  {"application/x-www-form-urlencoded"},
		"User-Agent":    {userAgent}, // Assuming userAgent is a global constant or variable.
	}

	DebugLogger.Printf("Sending %s request for subreddits: %s", action, subredditNames)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ErrorLogger.Printf("Error sending %s request for subreddits: %v. Subreddits: %v", action, err, subredditDisplayNamesChunk)
		return manage_subreddit_response_type{
			Error:            true,
			StatusCode:       0, // No HTTP status code.
			FailedCount:      len(subredditDisplayNamesChunk),
			FailedSubreddits: subredditDisplayNamesChunk,
		}
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		ErrorLogger.Printf("Error reading response body for %s subreddits (status %d): %v. Subreddits: %v", action, resp.StatusCode, err, subredditDisplayNamesChunk)
		// Still process status code, but mark as error.
		return manage_subreddit_response_type{
			Error:            true,
			StatusCode:       resp.StatusCode,
			FailedCount:      len(subredditDisplayNamesChunk),
			FailedSubreddits: subredditDisplayNamesChunk,
		}
	}
	DebugLogger.Printf("Response for %s subreddits (Status: %d): %s", action, resp.StatusCode, string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		ErrorLogger.Printf("Failed to %s subreddits (status %d): %s. Subreddits: %v", action, resp.StatusCode, string(bodyBytes), subredditDisplayNamesChunk)
		return manage_subreddit_response_type{
			Error:            true,
			StatusCode:       resp.StatusCode,
			SuccessCount:     0,
			FailedCount:      len(subredditDisplayNamesChunk),
			FailedSubreddits: subredditDisplayNamesChunk,
		}
	}

	// Assuming success if status is 200 OK. Reddit API for subscribe usually returns an empty JSON object {} on success.
	// More sophisticated parsing of response body might be needed if partial success/failure within a chunk is possible.
	return manage_subreddit_response_type{
		Error:            false,
		StatusCode:       resp.StatusCode,
		SuccessCount:     len(subredditDisplayNamesChunk),
		FailedCount:      0,
		FailedSubreddits: nil,
	}
}

// manageFollowedUsers performs follow (subscribe) or unfollow (unsubscribe) actions for a list of user display names.
func manageFollowedUsers(token string, userDisplayNames []string, action subscribe_type) manage_subreddit_response_type {
	if len(userDisplayNames) == 0 {
		DebugLogger.Printf("No users to %s.", action)
		return manage_subreddit_response_type{SuccessCount: 0, FailedCount: 0}
	}

	InfoLogger.Printf("Managing %d followed users with action '%s'.", len(userDisplayNames), action)
	var finalResponse manage_subreddit_response_type
	var failedUsernames []string

	requestMethod := http.MethodPut // For "sub" (follow)
	if action == unsubscribe {
		requestMethod = http.MethodDelete // For "unsub" (unfollow)
	}

	for _, username := range userDisplayNames {
		// Reddit API expects username without "u_" prefix for this endpoint.
		cleanUsername := strings.TrimPrefix(username, "u_")
		if cleanUsername == "" {
			DebugLogger.Printf("Skipping empty username for %s action.", action)
			continue
		}

		// The endpoint is /api/v1/me/friends/{username}
		// For following, it's PUT with JSON body {"name": "username"}
		// For unfollowing, it's DELETE. The body might not be strictly necessary for DELETE by some interpretations,
		// but Reddit's docs sometimes show it. Let's be consistent.
		apiURL := fmt.Sprintf("https://oauth.reddit.com/api/v1/me/friends/%s", cleanUsername)
		jsonBody := map[string]string{"name": cleanUsername} // Required for PUT, possibly ignored for DELETE.
		requestBodyBytes, err := json.Marshal(jsonBody)
		if err != nil {
			ErrorLogger.Printf("Error marshalling body for %s user %s: %v", action, cleanUsername, err)
			failedUsernames = append(failedUsernames, username) // Original name for reporting
			continue
		}

		req, err := http.NewRequest(requestMethod, apiURL, bytes.NewBuffer(requestBodyBytes))
		if err != nil {
			ErrorLogger.Printf("Error creating request to %s user %s: %v", action, cleanUsername, err)
			failedUsernames = append(failedUsernames, username)
			continue
		}

		req.Header = http.Header{
			"Authorization": {"Bearer " + token},
			"Content-Type":  {"application/json"}, // Important for this endpoint
			"User-Agent":    {userAgent},
		}

		DebugLogger.Printf("Sending %s request for user: %s (URL: %s)", action, cleanUsername, apiURL)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			ErrorLogger.Printf("Error sending %s request for user %s: %v", action, cleanUsername, err)
			failedUsernames = append(failedUsernames, username)
			continue
		}

		bodyBytes, _ := io.ReadAll(resp.Body) // Read body for logging, even on error.
		resp.Body.Close()                     // Close body immediately.

		// According to Reddit API for follow:
		// PUT to /api/v1/me/friends/username with {"name": "username"} returns 200 OK (empty body or relationship object).
		// DELETE to /api/v1/me/friends/username returns 204 No Content.
		isSuccess := (requestMethod == http.MethodPut && resp.StatusCode == http.StatusOK) ||
			(requestMethod == http.MethodDelete && resp.StatusCode == http.StatusNoContent)

		if !isSuccess {
			ErrorLogger.Printf("Failed to %s user %s (status %d): %s", action, cleanUsername, resp.StatusCode, string(bodyBytes))
			failedUsernames = append(failedUsernames, username)
			finalResponse.Error = true                 // Mark overall error if any user fails.
			finalResponse.StatusCode = resp.StatusCode // Report the last erroring status.
		} else {
			DebugLogger.Printf("Successfully %s user %s (status %d). Response: %s", action, cleanUsername, resp.StatusCode, string(bodyBytes))
			finalResponse.SuccessCount++
		}
	}

	finalResponse.FailedCount = len(failedUsernames)
	finalResponse.FailedSubreddits = failedUsernames // Re-using FailedSubreddits field for failed usernames here.
	InfoLogger.Printf("Finished managing users with action '%s'. Success: %d, Failed: %d.", action, finalResponse.SuccessCount, finalResponse.FailedCount)
	return finalResponse
}

// chunkStringArray splits a slice of strings into chunks of a specified size.
func chunkStringArray(array []string, chunkSize int) [][]string {
	if chunkSize <= 0 {
		// Avoid infinite loop or panic if chunkSize is invalid. Return array as a single chunk.
		ErrorLogger.Printf("chunkStringArray called with invalid chunkSize %d. Returning single chunk.", chunkSize)
		return [][]string{array}
	}
	var chunks [][]string
	for i := 0; i < len(array); i += chunkSize {
		end := i + chunkSize
		if end > len(array) {
			end = len(array)
		}
		chunks = append(chunks, array[i:end])
	}
	return chunks
}

// fetchSavedPostsFullNames retrieves a list of full names for all posts saved by the user.
// It handles pagination from the Reddit API.
func fetchSavedPostsFullNames(token, username string) ([]string, error) {
	if username == "" {
		return nil, errors.New("username cannot be empty for fetching saved posts")
	}
	InfoLogger.Printf("Fetching saved posts for user %s.", username)
	// The endpoint is /user/{username}/saved.json
	apiURL := fmt.Sprintf("https://oauth.reddit.com/user/%s/saved.json", username)
	nameList, err := fetchAllFullNames(apiURL, token, false) // false indicates not specifically for subreddits (affects u_ filtering)
	if err != nil {
		return nil, fmt.Errorf("error fetching saved posts for %s: %w", username, err)
	}
	// For saved posts, we expect 'name' (t3_xxxxx), not 'display_name'. The 'fullNamesList' from fetchAllFullNames contains these.
	// The userDisplayNameList is not relevant for saved posts.
	InfoLogger.Printf("Fetched %d saved post full names for user %s.", len(nameList.fullNamesList), username)
	return nameList.fullNamesList, nil
}

// fetchSubredditFullNames retrieves lists of full names and display names for subreddits the user is subscribed to,
// including followed users (which appear as subreddits of type "user").
// It handles pagination from the Reddit API.
func fetchSubredditFullNames(token string) (reddit_name_type, error) {
	InfoLogger.Println("Fetching subscribed subreddits and followed users.")
	// The endpoint is /subreddits/mine.json (variant like /subreddits/mine/subscriber.json also exists)
	// For this use case, /subreddits/mine.json usually lists subscribed "true" subreddits and followed users.
	apiURL := "https://oauth.reddit.com/subreddits/mine.json"
	nameList, err := fetchAllFullNames(apiURL, token, true) // true indicates it's for subreddits (enables u_ filtering)
	if err != nil {
		return reddit_name_type{}, fmt.Errorf("error fetching subscribed subreddits: %w", err)
	}
	InfoLogger.Printf("Fetched %d subscribed subreddits (display_name) and %d followed users (display_name).",
		len(nameList.displayNamesList), len(nameList.userDisplayNameList))
	return nameList, nil
}

// fetchAllFullNames is a generic function to fetch items (subreddits, posts) from a Reddit API listing endpoint.
// It handles pagination and extracts full names and display names.
// is_subreddit flag helps differentiate processing for user "subreddits" (followed users).
func fetchAllFullNames(baseAPIURL, token string, isSubredditContext bool) (reddit_name_type, error) {
	var result reddit_name_type
	lastFullName := "" // For "after" parameter in pagination.

	DebugLogger.Printf("Starting to fetch all names from URL: %s (isSubredditContext: %t)", baseAPIURL, isSubredditContext)

	for i := 0; ; i++ { // Loop indefinitely until no "after" token is returned. Safety break below.
		paginatedURL := fmt.Sprintf("%s?limit=100&after=%s", baseAPIURL, lastFullName)
		DebugLogger.Printf("Fetching page %d from %s", i+1, paginatedURL)

		req, err := http.NewRequest(http.MethodGet, paginatedURL, nil)
		if err != nil {
			return result, fmt.Errorf("error creating request for %s: %w", paginatedURL, err)
		}
		req.Header = http.Header{
			"Authorization": {"Bearer " + token},
			"User-Agent":    {userAgent},
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return result, fmt.Errorf("error fetching data from %s: %w", paginatedURL, err)
		}

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			ErrorLogger.Printf("Failed to fetch names from %s. Status: %d, Body: %s", paginatedURL, resp.StatusCode, string(bodyBytes))
			return result, fmt.Errorf("failed to fetch data from %s, status code: %d", paginatedURL, resp.StatusCode)
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close() // Close body immediately after reading.
		if err != nil {
			return result, fmt.Errorf("error reading response body from %s: %w", paginatedURL, err)
		}

		var listing full_name_list_type
		if err := json.Unmarshal(bodyBytes, &listing); err != nil {
			ErrorLogger.Printf("Error unmarshalling response from %s: %v. Body: %s", paginatedURL, err, string(bodyBytes))
			return result, fmt.Errorf("error unmarshalling response from %s: %w", paginatedURL, err)
		}
		// DebugLogger.Printf("Page %d data from %s: %+v", i+1, paginatedURL, listing.Data)

		if len(listing.Data.Children) == 0 && listing.Data.After == "" && lastFullName != "" {
			// This condition might indicate an issue or just the end of an empty list after the first page.
			// If it's the very first fetch and it's empty, that's fine.
			DebugLogger.Printf("No children found on page %d and no 'after' token. Assuming end of list for %s.", i+1, paginatedURL)
			break
		}

		for _, child := range listing.Data.Children {
			// If we are in a subreddit context and the item is a "user" subreddit,
			// it means this is a user the account is following.
			if isSubredditContext && child.Data.Subreddit_type == "user" {
				if child.Data.Display_name != "" { // Typically display_name is the username here.
					result.userDisplayNameList = append(result.userDisplayNameList, child.Data.Display_name)
				}
			} else {
				// For regular subreddits or posts.
				if child.Data.Name != "" {
					result.fullNamesList = append(result.fullNamesList, child.Data.Name)
				}
				if child.Data.Display_name != "" { // display_name for subreddits, title for posts (check usage if for posts)
					result.displayNamesList = append(result.displayNamesList, child.Data.Display_name)
				}
			}
		}
		DebugLogger.Printf("Page %d: collected %d fullNames, %d displayNames, %d userDisplayNames. Current totals: Full: %d, Display: %d, User: %d",
			i+1, len(listing.Data.Children), // This is children on current page, not names added.
			countItems(listing.Data.Children, func(c struct {
				Kind string `json:"kind"`
				Data struct {
					Name           string `json:"name"`
					Display_name   string `json:"display_name"`
					Subreddit_type string `json:"subreddit_type"`
				} `json:"data"`
			}) bool {
				return c.Data.Name != "" && !(isSubredditContext && c.Data.Subreddit_type == "user")
			}),
			countItems(listing.Data.Children, func(c struct {
				Kind string `json:"kind"`
				Data struct {
					Name           string `json:"name"`
					Display_name   string `json:"display_name"`
					Subreddit_type string `json:"subreddit_type"`
				} `json:"data"`
			}) bool {
				return c.Data.Display_name != "" && !(isSubredditContext && c.Data.Subreddit_type == "user")
			}),
			countItems(listing.Data.Children, func(c struct {
				Kind string `json:"kind"`
				Data struct {
					Name           string `json:"name"`
					Display_name   string `json:"display_name"`
					Subreddit_type string `json:"subreddit_type"`
				} `json:"data"`
			}) bool {
				return isSubredditContext && c.Data.Subreddit_type == "user" && c.Data.Display_name != ""
			}),
			len(result.fullNamesList), len(result.displayNamesList), len(result.userDisplayNameList))

		if listing.Data.After == "" {
			DebugLogger.Printf("No 'after' token in response from %s. Assuming end of list.", paginatedURL)
			break // No more pages.
		}
		lastFullName = listing.Data.After

		if i > 100 { // Safety break to prevent potential infinite loops due to API quirks or logic errors.
			ErrorLogger.Printf("fetchAllFullNames exceeded 100 pages for %s. Aborting to prevent infinite loop. Last 'after': %s", baseAPIURL, lastFullName)
			return result, fmt.Errorf("exceeded 100 pages fetching from %s, possible infinite loop", baseAPIURL)
		}
	}
	InfoLogger.Printf("Finished fetching all names from %s. Total full names: %d, display names: %d, user display names: %d.",
		baseAPIURL, len(result.fullNamesList), len(result.displayNamesList), len(result.userDisplayNameList))
	return result, nil
}

// Helper for logging in fetchAllFullNames
func countItems(children []struct {
	Kind string `json:"kind"`
	Data struct {
		Name           string `json:"name"`
		Display_name   string `json:"display_name"`
		Subreddit_type string `json:"subreddit_type"`
	} `json:"data"`
}, predicate func(child struct {
	Kind string `json:"kind"`
	Data struct {
		Name           string `json:"name"`
		Display_name   string `json:"display_name"`
		Subreddit_type string `json:"subreddit_type"`
	} `json:"data"`
}) bool) int {
	count := 0
	for _, child := range children {
		if predicate(child) {
			count++
		}
	}
	return count
}

// verifyTokenResponse is the HTTP handler for the /verify-cookie endpoint.
// It validates a Reddit cookie and returns the associated username if valid.
func verifyTokenResponse(w http.ResponseWriter, r *http.Request) {
	DebugLogger.Printf("Received /verify-cookie request from %s", r.RemoteAddr)

	if r.Header.Get("Content-Type") != "application/json" {
		ErrorLogger.Printf("Invalid content type for /verify-cookie from %s: %s", r.RemoteAddr, r.Header.Get("Content-Type"))
		errorResponse(w, "Content Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	var requestBody verify_cookie_type
	decoder := json.NewDecoder(r.Body)
	// Consider DisallowUnknownFields for stricter parsing.
	// decoder.DisallowUnknownFields()
	err := decoder.Decode(&requestBody)
	if err != nil {
		ErrorLogger.Printf("Error decoding /verify-cookie request from %s: %v", r.RemoteAddr, err)
		var unmarshalErr *json.UnmarshalTypeError
		if errors.As(err, &unmarshalErr) {
			errorResponse(w, "Bad Request. Wrong Type provided for field "+unmarshalErr.Field, http.StatusBadRequest)
		} else {
			errorResponse(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
		}
		return
	}
	InfoLogger.Printf("Verifying cookie for %s (ends with ...%s)", r.RemoteAddr, safeSuffix(requestBody.Cookie, 6))

	finalResponse := verifyCookieAndGetResponse(requestBody.Cookie)

	w.Header().Set("Content-Type", "application/json")
	jsonResp, err := json.Marshal(finalResponse)
	if err != nil {
		ErrorLogger.Printf("Error marshalling /verify-cookie response for %s: %v", r.RemoteAddr, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(jsonResp); err != nil {
		ErrorLogger.Printf("Error writing /verify-cookie response for %s: %v", r.RemoteAddr, err)
	} else {
		DebugLogger.Printf("Successfully sent /verify-cookie response to %s. Success: %t, User: %s",
			r.RemoteAddr, finalResponse.Success, finalResponse.Data.Username)
	}
}

// verifyCookieAndGetResponse takes a cookie string, calls Reddit's API to verify it, and returns a structured response.
func verifyCookieAndGetResponse(cookieStr string) token_response_type {
	var finalResponse token_response_type

	// Make request to Reddit's /api/me.json
	req, err := http.NewRequest(http.MethodGet, "https://www.reddit.com/api/me.json", nil)
	if err != nil {
		ErrorLogger.Printf("Error creating request for /api/me.json: %v", err)
		finalResponse.Success = false
		finalResponse.Message = "Internal error creating request to verify cookie."
		return finalResponse
	}

	req.Header = http.Header{
		"Cookie":     {cookieStr},
		"User-Agent": {userAgent}, // Use a global or configured user agent.
	}

	DebugLogger.Printf("Sending request to /api/me.json to verify cookie (ends ...%s)", safeSuffix(cookieStr, 6))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ErrorLogger.Printf("Error sending request to /api/me.json: %v", err)
		finalResponse.Success = false
		finalResponse.Message = "Error contacting Reddit to verify cookie."
		return finalResponse
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		ErrorLogger.Printf("Error reading response body from /api/me.json (status %d): %v", resp.StatusCode, err)
		finalResponse.Success = false
		finalResponse.Message = "Error reading Reddit's response."
		return finalResponse
	}
	// DebugLogger.Printf("/api/me.json response (Status: %d): %s", resp.StatusCode, string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		ErrorLogger.Printf("Cookie verification failed. /api/me.json status: %d. Body: %s", resp.StatusCode, string(bodyBytes))
		var errorRespData error_response_type
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

	var profile profile_response_type
	if err := json.Unmarshal(bodyBytes, &profile); err != nil {
		ErrorLogger.Printf("Error unmarshalling /api/me.json response: %v. Body: %s", err, string(bodyBytes))
		finalResponse.Success = false
		finalResponse.Message = "Error parsing Reddit's response."
		return finalResponse
	}

	if profile.Data.Name == "" {
		ErrorLogger.Printf("Cookie verified (status 200) but no username found in /api/me.json response. Body: %s", string(bodyBytes))
		finalResponse.Success = false
		finalResponse.Message = "Cookie seems valid, but username could not be retrieved."
		return finalResponse
	}

	InfoLogger.Printf("Cookie successfully verified for username: %s", profile.Data.Name)
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
				DebugLogger.Printf("Successfully parsed token_v2 from cookie (value ends ...%s)", safeSuffix(tokenPair[1], 6))
				return tokenPair[1]
			}
			ErrorLogger.Printf("Found 'token_v2=' but failed to parse value from part: '%s'", trimmedPart)
			return "" // Found prefix but value is malformed
		}
	}
	DebugLogger.Printf("Could not find 'token_v2=' in cookie string: ...%s", safeSuffix(cookie, 20))
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
		ErrorLogger.Printf("Critical: Failed to marshal error response object: %v. Original message: %s", err, message)
		http.Error(w, `{"message":"Error generating error response"}`, http.StatusInternalServerError)
		return
	}
	if _, writeErr := w.Write(jsonResp); writeErr != nil {
		ErrorLogger.Printf("Failed to write error response to client: %v. Original message: %s, Status: %d", writeErr, message, httpStatusCode)
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

// userAgent should be defined, perhaps globally or passed around.
// Using a descriptive User-Agent is good practice for Reddit API.
const userAgent = "GoMigrateClient/1.0 by YourUsername (contact your_email_or_reddit_profile)" // TODO: Replace with actual details
