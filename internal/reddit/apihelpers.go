package reddit

import (
	"encoding/json"
	"fmt"
	"github.com/nileshnk/reddit-migrate/internal/config"
	"github.com/nileshnk/reddit-migrate/internal/types"
	"io"
	"net/http"
)

// fetchAllNames is a generic function to fetch items (subreddits, posts) from a Reddit API listing endpoint.
// It handles pagination and extracts full names and display names.
// isSubredditContext flag helps differentiate processing for user "subreddits" (followed users).
func fetchAllNames(baseAPIURL, token string, isSubredditContext bool) (types.RedditNameType, error) {
	var result types.RedditNameType
	lastFullName := "" // For "after" parameter in pagination.

	config.DebugLogger.Printf("Starting to fetch all names from URL: %s (isSubredditContext: %t)", baseAPIURL, isSubredditContext)

	for i := 0; ; i++ { // Loop indefinitely until no "after" token is returned. Safety break below.
		paginatedURL := fmt.Sprintf("%s?limit=100&after=%s", baseAPIURL, lastFullName)
		config.DebugLogger.Printf("Fetching page %d from %s", i+1, paginatedURL)

		req, err := http.NewRequest(http.MethodGet, paginatedURL, nil)
		if err != nil {
			return result, fmt.Errorf("error creating request for %s: %w", paginatedURL, err)
		}
		req.Header = http.Header{
			"Authorization": {"Bearer " + token},
			"User-Agent":    {config.UserAgent},
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return result, fmt.Errorf("error fetching data from %s: %w", paginatedURL, err)
		}

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			config.ErrorLogger.Printf("Failed to fetch names from %s. Status: %d, Body: %s", paginatedURL, resp.StatusCode, string(bodyBytes))
			return result, fmt.Errorf("failed to fetch data from %s, status code: %d", paginatedURL, resp.StatusCode)
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close() // Close body immediately after reading.
		if err != nil {
			return result, fmt.Errorf("error reading response body from %s: %w", paginatedURL, err)
		}

		var listing types.FullNameListType
		if err := json.Unmarshal(bodyBytes, &listing); err != nil {
			config.ErrorLogger.Printf("Error unmarshalling response from %s: %v. Body: %s", paginatedURL, err, string(bodyBytes))
			return result, fmt.Errorf("error unmarshalling response from %s: %w", paginatedURL, err)
		}
		config.DebugLogger.Printf("Page %d data from %s: %+v", i+1, paginatedURL, listing.Data)

		if len(listing.Data.Children) == 0 && listing.Data.After == "" && lastFullName != "" {
			config.DebugLogger.Printf("No children found on page %d and no 'after' token. Assuming end of list for %s.", i+1, paginatedURL)
			break
		}

		for _, child := range listing.Data.Children {
			if isSubredditContext && child.Data.SubredditType == "user" {
				if child.Data.DisplayName != "" {
					result.UserDisplayNameList = append(result.UserDisplayNameList, child.Data.DisplayName)
				}
			} else {
				if child.Data.Name != "" {
					result.FullNamesList = append(result.FullNamesList, child.Data.Name)
				}
				if child.Data.DisplayName != "" {
					result.DisplayNamesList = append(result.DisplayNamesList, child.Data.DisplayName)
				}
			}
		}
		config.DebugLogger.Printf("Page %d: collected %d fullNames, %d displayNames, %d userDisplayNames. Current totals: Full: %d, Display: %d, User: %d",
			i+1,
			countItems(listing.Data.Children, func(c types.FullListChild) bool {
				return c.Data.Name != "" && !(isSubredditContext && c.Data.SubredditType == "user")
			}),
			countItems(listing.Data.Children, func(c types.FullListChild) bool {
				return c.Data.DisplayName != "" && !(isSubredditContext && c.Data.SubredditType == "user")
			}),
			countItems(listing.Data.Children, func(c types.FullListChild) bool {
				return isSubredditContext && c.Data.SubredditType == "user" && c.Data.DisplayName != ""
			}),
			len(result.FullNamesList), len(result.DisplayNamesList), len(result.UserDisplayNameList))

		if listing.Data.After == "" {
			config.DebugLogger.Printf("No 'after' token in response from %s. Assuming end of list.", paginatedURL)
			break // No more pages.
		}
		lastFullName = listing.Data.After

		if i > 100 { // Safety break to prevent potential infinite loops due to API quirks or logic errors.
			config.ErrorLogger.Printf("fetchAllNames exceeded 100 pages for %s. Aborting to prevent infinite loop. Last 'after': %s", baseAPIURL, lastFullName)
			return result, fmt.Errorf("exceeded 100 pages fetching from %s, possible infinite loop", baseAPIURL)
		}
	}
	config.InfoLogger.Printf("Finished fetching all names from %s. Total full names: %d, display names: %d, user display names: %d.",
		baseAPIURL, len(result.FullNamesList), len(result.DisplayNamesList), len(result.UserDisplayNameList))
	return result, nil
}

// countItems helper for logging in fetchAllNames
func countItems(children []types.FullListChild, predicate func(child types.FullListChild) bool) int {
	count := 0
	for _, child := range children {
		if predicate(child) {
			count++
		}
	}
	return count
}
