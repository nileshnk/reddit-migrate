package reddit

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/nileshnk/reddit-migrate/internal/config"
	"github.com/nileshnk/reddit-migrate/internal/types"
)

// FetchSavedCommentFullNames retrieves a list of full names for all comments saved by the user.
// It uses the same /saved.json endpoint as posts, filtering for t1_ (comment) items.
func FetchSavedCommentFullNames(token, username string) ([]string, error) {
	if username == "" {
		return nil, fmt.Errorf("username is required for fetching saved comments")
	}

	config.InfoLogger.Printf("Fetching saved comment full names for user %s.", username)
	apiURL := fmt.Sprintf("https://oauth.reddit.com/user/%s/saved.json", username)

	var allNames []string
	lastFullName := ""

	for i := 0; ; i++ {
		paginatedURL := fmt.Sprintf("%s?limit=100&after=%s", apiURL, lastFullName)

		req, err := http.NewRequest(http.MethodGet, paginatedURL, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request for saved comments: %w", err)
		}

		req.Header = http.Header{
			"Authorization": {"Bearer " + token},
			"User-Agent":    {config.UserAgent},
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error fetching saved comments: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("failed to fetch saved comments, status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("error reading saved comments response: %w", err)
		}

		var listing types.FullNameListType
		if err := json.Unmarshal(bodyBytes, &listing); err != nil {
			return nil, fmt.Errorf("error unmarshalling saved comments response: %w", err)
		}

		for _, child := range listing.Data.Children {
			if child.Kind == "t1" {
				if child.Data.Name != "" {
					allNames = append(allNames, child.Data.Name)
				}
			}
		}

		if listing.Data.After == "" {
			break
		}
		lastFullName = listing.Data.After

		if i > 100 {
			return nil, fmt.Errorf("exceeded 100 pages fetching saved comment names")
		}
	}

	config.InfoLogger.Printf("Fetched %d saved comment full names for user %s.", len(allNames), username)
	return allNames, nil
}

// FetchSavedCommentsWithDetails retrieves detailed information about all saved comments for a user
func FetchSavedCommentsWithDetails(token, username string) ([]types.SavedCommentInfo, error) {
	if username == "" {
		return nil, fmt.Errorf("username is required for fetching saved comments")
	}

	config.InfoLogger.Printf("Fetching detailed saved comments for user %s.", username)
	apiURL := fmt.Sprintf("https://oauth.reddit.com/user/%s/saved.json", username)

	var allComments []types.SavedCommentInfo
	lastFullName := ""

	for i := 0; ; i++ {
		paginatedURL := fmt.Sprintf("%s?limit=100&after=%s", apiURL, lastFullName)
		config.DebugLogger.Printf("Fetching saved comments page %d from %s", i+1, paginatedURL)

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
			return nil, fmt.Errorf("error fetching saved comments from %s: %w", paginatedURL, err)
		}

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			config.ErrorLogger.Printf("Failed to fetch saved comments from %s. Status: %d, Body: %s", paginatedURL, resp.StatusCode, string(bodyBytes))
			return nil, fmt.Errorf("failed to fetch saved comments from %s, status code: %d", paginatedURL, resp.StatusCode)
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("error reading saved comments response body from %s: %w", paginatedURL, err)
		}

		var listing struct {
			Kind string `json:"kind"`
			Data struct {
				After    string                       `json:"after"`
				Children []types.DetailedCommentData `json:"children"`
			} `json:"data"`
		}

		if err := json.Unmarshal(bodyBytes, &listing); err != nil {
			config.ErrorLogger.Printf("Error unmarshalling saved comments response from %s: %v", paginatedURL, err)
			return nil, fmt.Errorf("error unmarshalling saved comments response from %s: %w", paginatedURL, err)
		}

		config.DebugLogger.Printf("Page %d: found %d items in saved listing", i+1, len(listing.Data.Children))

		if len(listing.Data.Children) == 0 && listing.Data.After == "" && lastFullName != "" {
			break
		}

		for _, child := range listing.Data.Children {
			if child.Kind == "t1" {
				commentInfo := parseDetailedCommentData(child)
				allComments = append(allComments, commentInfo)
			}
		}

		if listing.Data.After == "" {
			break
		}
		lastFullName = listing.Data.After

		if i > 100 {
			config.ErrorLogger.Printf("fetchSavedCommentsWithDetails exceeded 100 pages for %s. Aborting.", username)
			return nil, fmt.Errorf("exceeded 100 pages fetching saved comments for %s", username)
		}
	}

	config.InfoLogger.Printf("Fetched %d detailed saved comments for user %s.", len(allComments), username)
	return allComments, nil
}

// GetSavedCommentsCount returns the total count of saved comments for a user
func GetSavedCommentsCount(token, username string) (int, error) {
	if username == "" {
		return 0, fmt.Errorf("username is required for fetching saved comments count")
	}

	config.DebugLogger.Printf("Getting saved comments count for user %s.", username)
	apiURL := fmt.Sprintf("https://oauth.reddit.com/user/%s/saved.json", username)

	count := 0
	lastFullName := ""

	for i := 0; ; i++ {
		paginatedURL := fmt.Sprintf("%s?limit=100&after=%s", apiURL, lastFullName)

		req, err := http.NewRequest(http.MethodGet, paginatedURL, nil)
		if err != nil {
			return 0, fmt.Errorf("error creating request for saved comments count: %w", err)
		}

		req.Header = http.Header{
			"Authorization": {"Bearer " + token},
			"User-Agent":    {config.UserAgent},
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return 0, fmt.Errorf("error fetching saved comments count: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return 0, fmt.Errorf("failed to fetch saved comments count, status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return 0, fmt.Errorf("error reading saved comments count response: %w", err)
		}

		var listing struct {
			Kind string `json:"kind"`
			Data struct {
				After    string `json:"after"`
				Children []struct {
					Kind string `json:"kind"`
				} `json:"children"`
			} `json:"data"`
		}

		if err := json.Unmarshal(bodyBytes, &listing); err != nil {
			return 0, fmt.Errorf("error unmarshalling saved comments count response: %w", err)
		}

		for _, child := range listing.Data.Children {
			if child.Kind == "t1" {
				count++
			}
		}

		if listing.Data.After == "" {
			break
		}
		lastFullName = listing.Data.After

		if i > 100 {
			return 0, fmt.Errorf("exceeded 100 pages fetching saved comments count")
		}
	}

	config.DebugLogger.Printf("Found %d saved comments for user %s.", count, username)
	return count, nil
}

// parseDetailedCommentData converts Reddit API comment data into our SavedCommentInfo structure
func parseDetailedCommentData(commentData types.DetailedCommentData) types.SavedCommentInfo {
	// Create a plain text snippet from the body (strip HTML, limit length)
	bodyText := commentData.Data.Body
	if len(bodyText) > 300 {
		bodyText = bodyText[:300] + "..."
	}

	permalink := commentData.Data.Permalink
	if permalink != "" && !strings.HasPrefix(permalink, "http") {
		permalink = "https://reddit.com" + permalink
	}

	return types.SavedCommentInfo{
		ID:        commentData.Data.ID,
		FullName:  commentData.Data.Name,
		Body:      commentData.Data.Body,
		BodyText:  bodyText,
		Author:    commentData.Data.Author,
		Subreddit: commentData.Data.Subreddit,
		Score:     commentData.Data.Score,
		Created:   int64(commentData.Data.CreatedUTC),
		Permalink: permalink,
		LinkTitle: commentData.Data.LinkTitle,
		LinkURL:   commentData.Data.LinkURL,
		NSFW:      commentData.Data.Over18,
	}
}
