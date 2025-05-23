package reddit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/nileshnk/reddit-migrate/internal/config"
	"github.com/nileshnk/reddit-migrate/internal/types"
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

// FetchSubredditsWithDetails retrieves detailed information about all subreddits the user is subscribed to
// including subscriber counts, descriptions, icons, and metadata needed for the selection UI
func FetchSubredditsWithDetails(token string) ([]types.SubredditInfo, error) {
	config.InfoLogger.Println("Fetching detailed subscribed subreddits information.")
	apiURL := "https://oauth.reddit.com/subreddits/mine.json"

	var allSubreddits []types.SubredditInfo
	lastFullName := ""

	for i := 0; ; i++ {
		paginatedURL := fmt.Sprintf("%s?limit=100&after=%s", apiURL, lastFullName)
		config.DebugLogger.Printf("Fetching subreddits page %d from %s", i+1, paginatedURL)

		req, err := http.NewRequest(http.MethodGet, paginatedURL, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request for %s: %w", paginatedURL, err)
		}

		req.Header = http.Header{
			"Authorization": {"Bearer " + token},
			"User-Agent":    {config.UserAgent},
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error fetching subreddits from %s: %w", paginatedURL, err)
		}

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			config.ErrorLogger.Printf("Failed to fetch subreddits from %s. Status: %d, Body: %s", paginatedURL, resp.StatusCode, string(bodyBytes))
			return nil, fmt.Errorf("failed to fetch subreddits from %s, status code: %d", paginatedURL, resp.StatusCode)
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("error reading subreddits response body from %s: %w", paginatedURL, err)
		}

		var listing struct {
			Kind string `json:"kind"`
			Data struct {
				After    string                        `json:"after"`
				Children []types.DetailedSubredditData `json:"children"`
			} `json:"data"`
		}

		if err := json.Unmarshal(bodyBytes, &listing); err != nil {
			config.ErrorLogger.Printf("Error unmarshalling subreddits response from %s: %v. Body: %s", paginatedURL, err, string(bodyBytes))
			return nil, fmt.Errorf("error unmarshalling subreddits response from %s: %w", paginatedURL, err)
		}

		config.DebugLogger.Printf("Page %d: found %d subreddits", i+1, len(listing.Data.Children))

		if len(listing.Data.Children) == 0 && listing.Data.After == "" && lastFullName != "" {
			config.DebugLogger.Printf("No subreddits found on page %d and no 'after' token. End of subreddits list.", i+1)
			break
		}

		// Process each subreddit and extract detailed information
		for _, child := range listing.Data.Children {
			if child.Kind == "t5" && child.Data.SubredditType != "user" { // Only process actual subreddits, not user profiles
				subredditInfo := parseDetailedSubredditData(child)
				allSubreddits = append(allSubreddits, subredditInfo)
			}
		}

		if listing.Data.After == "" {
			config.DebugLogger.Printf("No 'after' token in subreddits response. End of list.")
			break
		}
		lastFullName = listing.Data.After

		if i > 100 {
			config.ErrorLogger.Printf("fetchSubredditsWithDetails exceeded 100 pages. Aborting.")
			return nil, fmt.Errorf("exceeded 100 pages fetching subreddits")
		}
	}

	config.InfoLogger.Printf("Fetched %d detailed subreddits.", len(allSubreddits))
	return allSubreddits, nil
}

// parseDetailedSubredditData converts Reddit API subreddit data into our SubredditInfo structure
func parseDetailedSubredditData(subredditData types.DetailedSubredditData) types.SubredditInfo {
	// Choose the best available icon URL
	iconURL := ""
	if subredditData.Data.CommunityIcon != "" {
		iconURL = unescapeHTMLEntities(subredditData.Data.CommunityIcon)
	} else if subredditData.Data.IconImg != "" {
		iconURL = subredditData.Data.IconImg
	}

	// Choose the best available banner URL
	bannerURL := ""
	if subredditData.Data.BannerImg != "" {
		bannerURL = subredditData.Data.BannerImg
	} else if subredditData.Data.HeaderImg != "" {
		bannerURL = subredditData.Data.HeaderImg
	}

	return types.SubredditInfo{
		Name:          subredditData.Data.Name,
		DisplayName:   subredditData.Data.DisplayName,
		Title:         subredditData.Data.Title,
		Description:   subredditData.Data.PublicDescription,
		Subscribers:   subredditData.Data.Subscribers,
		IconURL:       iconURL,
		BannerURL:     bannerURL,
		PrimaryColor:  subredditData.Data.PrimaryColor,
		KeyColor:      subredditData.Data.KeyColor,
		SubredditType: subredditData.Data.SubredditType,
		NSFW:          subredditData.Data.Over18,
		Created:       int64(subredditData.Data.CreatedUTC),
	}
}

// unescapeHTMLEntities unescapes HTML entities in URLs (Reddit often HTML-escapes URLs)
func unescapeHTMLEntities(url string) string {
	url = strings.ReplaceAll(url, "&amp;", "&")
	url = strings.ReplaceAll(url, "&lt;", "<")
	url = strings.ReplaceAll(url, "&gt;", ">")
	url = strings.ReplaceAll(url, "&quot;", "\"")
	url = strings.ReplaceAll(url, "&#39;", "'")
	return url
}

// GetSubredditCount returns the total count of subscribed subreddits (excluding followed users)
func GetSubredditCount(token string) (int, error) {
	config.DebugLogger.Println("Getting subscribed subreddits count.")
	apiURL := "https://oauth.reddit.com/subreddits/mine.json"

	count := 0
	lastFullName := ""

	for i := 0; ; i++ {
		paginatedURL := fmt.Sprintf("%s?limit=100&after=%s", apiURL, lastFullName)

		req, err := http.NewRequest(http.MethodGet, paginatedURL, nil)
		if err != nil {
			return 0, fmt.Errorf("error creating request for subreddit count: %w", err)
		}

		req.Header = http.Header{
			"Authorization": {"Bearer " + token},
			"User-Agent":    {config.UserAgent},
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return 0, fmt.Errorf("error fetching subreddit count: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return 0, fmt.Errorf("failed to fetch subreddit count, status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return 0, fmt.Errorf("error reading subreddit count response: %w", err)
		}

		var listing struct {
			Kind string `json:"kind"`
			Data struct {
				After    string `json:"after"`
				Children []struct {
					Kind string `json:"kind"`
					Data struct {
						SubredditType string `json:"subreddit_type"`
					} `json:"data"`
				} `json:"children"`
			} `json:"data"`
		}

		if err := json.Unmarshal(bodyBytes, &listing); err != nil {
			return 0, fmt.Errorf("error unmarshalling subreddit count response: %w", err)
		}

		// Count only actual subreddits (not user profiles)
		for _, child := range listing.Data.Children {
			if child.Kind == "t5" && child.Data.SubredditType != "user" {
				count++
			}
		}

		if listing.Data.After == "" {
			break
		}
		lastFullName = listing.Data.After

		if i > 100 {
			return 0, fmt.Errorf("exceeded 100 pages fetching subreddit count")
		}
	}

	config.DebugLogger.Printf("Found %d subscribed subreddits.", count)
	return count, nil
}
