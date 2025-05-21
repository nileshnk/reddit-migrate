package reddit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/nileshnk/reddit-migrate/internal/config"
	"github.com/nileshnk/reddit-migrate/internal/types"
	"io"
	"net/http"
	"strings"
)

// ManageSubreddits performs subscribe or unsubscribe actions on a list of subreddits in chunks.
// It aggregates results from chunk operations.
func ManageSubreddits(token string, subredditDisplayNames []string, action types.SubredditActionType, chunkSize int) types.ManageSubredditResponseType {
	if len(subredditDisplayNames) == 0 {
		config.DebugLogger.Printf("No subreddits to %s.", action)
		return types.ManageSubredditResponseType{SuccessCount: 0, FailedCount: 0}
	}
	if chunkSize <= 0 {
		chunkSize = 100 // Default chunk size if invalid.
		config.DebugLogger.Printf("Invalid chunk size for manageSubreddits, defaulting to %d", chunkSize)
	}

	config.InfoLogger.Printf("Managing %d subreddits with action '%s', chunk size %d.", len(subredditDisplayNames), action, chunkSize)
	chunks := chunkStringArray(subredditDisplayNames, chunkSize)
	var finalResponse types.ManageSubredditResponseType

	for i, chunk := range chunks {
		config.DebugLogger.Printf("Processing chunk %d/%d for %s action (size: %d).", i+1, len(chunks), action, len(chunk))
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
	config.DebugLogger.Printf("Finished managing subreddits with action '%s'. Success: %d, Failed: %d.", action, finalResponse.SuccessCount, finalResponse.FailedCount)
	return finalResponse
}

// manageSubredditChunk sends a request to Reddit API to subscribe/unsubscribe a single chunk of subreddits.
func manageSubredditChunk(token string, subredditDisplayNamesChunk []string, action types.SubredditActionType) types.ManageSubredditResponseType {
	if len(subredditDisplayNamesChunk) == 0 {
		return types.ManageSubredditResponseType{SuccessCount: 0, FailedCount: 0}
	}

	subredditNames := strings.Join(subredditDisplayNamesChunk, ",")
	requestBodyStr := fmt.Sprintf("sr_name=%s&action=%s&api_type=json", subredditNames, action)
	requestBodyBytes := []byte(requestBodyStr)

	req, err := http.NewRequest(http.MethodPost, "https://oauth.reddit.com/api/subscribe", bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		config.ErrorLogger.Printf("Error creating request for %s subreddits: %v. Subreddits: %v", action, err, subredditDisplayNamesChunk)
		return types.ManageSubredditResponseType{
			Error:            true,
			StatusCode:       0, // No HTTP status code as request creation failed.
			FailedCount:      len(subredditDisplayNamesChunk),
			FailedSubreddits: subredditDisplayNamesChunk,
		}
	}

	req.Header = http.Header{
		"Authorization": {"Bearer " + token},
		"Content-Type":  {"application/x-www-form-urlencoded"},
		"User-Agent":    {config.UserAgent}, // Assuming userAgent is a global constant or variable.
	}

	config.DebugLogger.Printf("Sending %s request for subreddits: %s", action, subredditNames)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		config.ErrorLogger.Printf("Error sending %s request for subreddits: %v. Subreddits: %v", action, err, subredditDisplayNamesChunk)
		return types.ManageSubredditResponseType{
			Error:            true,
			StatusCode:       0, // No HTTP status code.
			FailedCount:      len(subredditDisplayNamesChunk),
			FailedSubreddits: subredditDisplayNamesChunk,
		}
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		config.ErrorLogger.Printf("Error reading response body for %s subreddits (status %d): %v. Subreddits: %v", action, resp.StatusCode, err, subredditDisplayNamesChunk)
		// Still process status code, but mark as error.
		return types.ManageSubredditResponseType{
			Error:            true,
			StatusCode:       resp.StatusCode,
			FailedCount:      len(subredditDisplayNamesChunk),
			FailedSubreddits: subredditDisplayNamesChunk,
		}
	}
	config.DebugLogger.Printf("Response for %s subreddits (Status: %d): %s", action, resp.StatusCode, string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		config.ErrorLogger.Printf("Failed to %s subreddits (status %d): %s. Subreddits: %v", action, resp.StatusCode, string(bodyBytes), subredditDisplayNamesChunk)
		return types.ManageSubredditResponseType{
			Error:            true,
			StatusCode:       resp.StatusCode,
			SuccessCount:     0,
			FailedCount:      len(subredditDisplayNamesChunk),
			FailedSubreddits: subredditDisplayNamesChunk,
		}
	}

	// Assuming success if status is 200 OK. Reddit API for subscribe usually returns an empty JSON object {} on success.
	// More sophisticated parsing of response body might be needed if partial success/failure within a chunk is possible.
	return types.ManageSubredditResponseType{
		Error:            false,
		StatusCode:       resp.StatusCode,
		SuccessCount:     len(subredditDisplayNamesChunk),
		FailedCount:      0,
		FailedSubreddits: nil,
	}
}

// ManageFollowedUsers performs follow (subscribe) or unfollow (unsubscribe) actions for a list of user display names.
func ManageFollowedUsers(token string, userDisplayNames []string, action types.SubredditActionType) types.ManageSubredditResponseType {
	if len(userDisplayNames) == 0 {
		config.DebugLogger.Printf("No users to %s.", action)
		return types.ManageSubredditResponseType{SuccessCount: 0, FailedCount: 0}
	}

	config.InfoLogger.Printf("Managing %d followed users with action '%s'.", len(userDisplayNames), action)
	var finalResponse types.ManageSubredditResponseType
	var failedUsernames []string

	requestMethod := http.MethodPut        // For "sub" (follow)
	if action == types.UnsubscribeAction { // Corrected: was types.SubscribeAction, should be UnsubscribeAction for DELETE
		requestMethod = http.MethodDelete // For "unsub" (unfollow)
	}

	for _, username := range userDisplayNames {
		// Reddit API expects username without "u_" prefix for this endpoint.
		cleanUsername := strings.TrimPrefix(username, "u_")
		if cleanUsername == "" {
			config.DebugLogger.Printf("Skipping empty username for %s action.", action)
			continue
		}

		// The endpoint is /api/v1/me/friends/{username}
		// For following, it's PUT with JSON body {"name": "username"}
		// For unfollowing, it's DELETE.
		apiURL := fmt.Sprintf("https://oauth.reddit.com/api/v1/me/friends/%s", cleanUsername)
		var requestBodyBytes []byte
		var err error

		if requestMethod == http.MethodPut {
			jsonBody := map[string]string{"name": cleanUsername}
			requestBodyBytes, err = json.Marshal(jsonBody)
			if err != nil {
				config.ErrorLogger.Printf("Error marshalling body for %s user %s: %v", action, cleanUsername, err)
				failedUsernames = append(failedUsernames, username) // Original name for reporting
				continue
			}
		}

		req, err := http.NewRequest(requestMethod, apiURL, bytes.NewBuffer(requestBodyBytes)) // Pass nil buffer for DELETE if no body
		if err != nil {
			config.ErrorLogger.Printf("Error creating request to %s user %s: %v", action, cleanUsername, err)
			failedUsernames = append(failedUsernames, username)
			continue
		}

		req.Header = http.Header{
			"Authorization": {"Bearer " + token},
			"User-Agent":    {config.UserAgent},
		}
		if requestMethod == http.MethodPut { // Content-Type only for PUT
			req.Header.Set("Content-Type", "application/json")
		}

		config.DebugLogger.Printf("Sending %s request for user: %s (URL: %s)", action, cleanUsername, apiURL)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			config.ErrorLogger.Printf("Error sending %s request for user %s: %v", action, cleanUsername, err)
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
			config.ErrorLogger.Printf("Failed to %s user %s (status %d): %s", action, cleanUsername, resp.StatusCode, string(bodyBytes))
			failedUsernames = append(failedUsernames, username)
			finalResponse.Error = true                 // Mark overall error if any user fails.
			finalResponse.StatusCode = resp.StatusCode // Report the last erroring status.
		} else {
			config.DebugLogger.Printf("Successfully %s user %s (status %d). Response: %s", action, cleanUsername, resp.StatusCode, string(bodyBytes))
			finalResponse.SuccessCount++
		}
	}

	finalResponse.FailedCount = len(failedUsernames)
	finalResponse.FailedSubreddits = failedUsernames // Re-using FailedSubreddits field for failed usernames here.
	config.InfoLogger.Printf("Finished managing users with action '%s'. Success: %d, Failed: %d.", action, finalResponse.SuccessCount, finalResponse.FailedCount)
	return finalResponse
}

// chunkStringArray splits a slice of strings into chunks of a specified size.
// This is a utility function that can be kept package-private or moved to a common utils package later.
func chunkStringArray(array []string, chunkSize int) [][]string {
	if chunkSize <= 0 {
		config.ErrorLogger.Printf("chunkStringArray called with invalid chunkSize %d. Returning single chunk.", chunkSize)
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

// FetchSubredditFullNames retrieves lists of full names and display names for subreddits the user is subscribed to,
// including followed users (which appear as subreddits of type "user").
// It handles pagination from the Reddit API.
func FetchSubredditFullNames(token string) (types.RedditNameType, error) {
	config.InfoLogger.Println("Fetching subscribed subreddits and followed users.")
	// The endpoint is /subreddits/mine.json (variant like /subreddits/mine/subscriber.json also exists)
	// For this use case, /subreddits/mine.json usually lists subscribed "true" subreddits and followed users.
	apiURL := "https://oauth.reddit.com/subreddits/mine.json"
	nameList, err := fetchAllNames(apiURL, token, true) // true indicates it's for subreddits (enables u_ filtering)
	if err != nil {
		return types.RedditNameType{}, fmt.Errorf("error fetching subscribed subreddits: %w", err)
	}
	config.InfoLogger.Printf("Fetched %d subscribed subreddits (display_name) and %d followed users (display_name).",
		len(nameList.DisplayNamesList), len(nameList.UserDisplayNameList))
	return nameList, nil
}
