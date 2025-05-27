package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nileshnk/reddit-migrate/internal/config"
	"github.com/nileshnk/reddit-migrate/internal/ratelimiter"
	"github.com/nileshnk/reddit-migrate/internal/types"
	"github.com/nileshnk/reddit-migrate/internal/worker"
)

func init() {
	config.InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lmicroseconds)
	config.ErrorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lmicroseconds)
	config.DebugLogger = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lmicroseconds)
	config.LoadConfig() // This should be called from main, and config values accessed here.
}

// ManageSavedPosts coordinates the saving or unsaving of posts concurrently using worker goroutines.
// It employs a rate limiter and a mechanism to pause/resume workers if API rate limits are hit.
// token: The OAuth token for API authentication.
// postIDs: A slice of post full names (e.g., "t3_xxxxx") to be processed.
// actionType: The action to perform (SaveAction or UnsaveAction from migration package).
// concurrency: The number of worker goroutines to use.
// Returns ManagePostResponseType from migration package
func ManageSavedPosts(token string, postIDs []string, actionType types.PostActionType, concurrency int) types.ManagePostResponseType {
	numPosts := len(postIDs)
	config.InfoLogger.Printf("ManageSavedPosts: Starting to %s %d posts. Concurrency: %d.", actionType, numPosts, concurrency)

	if concurrency < 1 {
		config.DebugLogger.Printf("ManageSavedPosts: Concurrency was %d, adjusted to minimum of 1.", concurrency)
		concurrency = 1
	}
	if numPosts == 0 {
		config.InfoLogger.Printf("ManageSavedPosts: No posts to %s. Operation skipped.", actionType)
		return types.ManagePostResponseType{SuccessCount: 0, FailedCount: 0}
	}

	rl := ratelimiter.NewRateLimiter(config.MaxTokensPerInterval, config.RateLimitInterval)

	jobs := make(chan string, numPosts)
	results := make(chan worker.Result, numPosts)
	rateLimitControl := make(chan bool)

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		config.DebugLogger.Println("ManageSavedPosts: Rate limit controller goroutine started.")

		rateLimitSleepInterval := config.RateLimitSleepInterval
		for shouldPause := range rateLimitControl {
			if shouldPause {
				config.InfoLogger.Println("ManageSavedPosts: Rate limit controller received pause signal from a worker.")
				rl.Pause()
				config.InfoLogger.Printf("ManageSavedPosts: Rate limiter paused. Sleeping for %v before testing API.", rateLimitSleepInterval)
				time.Sleep(rateLimitSleepInterval)

				for !TestRedditAPI(token, "t3_testdummy") {
					config.ErrorLogger.Printf("ManageSavedPosts: Test request failed. Rate limit likely still active. Sleeping again for %v.", rateLimitSleepInterval)
					time.Sleep(rateLimitSleepInterval)
				}
				config.InfoLogger.Println("ManageSavedPosts: Test request successful. Resuming rate limiter.")
				rl.Resume()
			} else {
				config.DebugLogger.Println("ManageSavedPosts: Rate limit controller received 'false' signal (currently unused).")
			}
		}
		config.DebugLogger.Println("ManageSavedPosts: Rate limit controller goroutine exiting as control channel was closed.")
	}()

	config.InfoLogger.Printf("ManageSavedPosts: Starting %d workers for %s operation.", concurrency, actionType)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			config.DebugLogger.Printf("Worker %d: Started for %s operation.", workerID, actionType)
			worker.PostWorker(ctx, token, actionType, rl, jobs, results, rateLimitControl, workerID)
			config.DebugLogger.Printf("Worker %d: Finished processing jobs.", workerID)
		}(i)
	}

	config.InfoLogger.Printf("ManageSavedPosts: Queueing %d posts for %s operation.", numPosts, actionType)
	for _, postID := range postIDs {
		select {
		case jobs <- postID:
		case <-ctx.Done():
			config.ErrorLogger.Printf("ManageSavedPosts: Context cancelled while queueing jobs. %d posts not queued.", numPosts-len(jobs))
			break
		}
	}
	close(jobs)
	config.DebugLogger.Println("ManageSavedPosts: All posts queued. Waiting for workers to complete...")

	go func() {
		wg.Wait()
		config.DebugLogger.Println("ManageSavedPosts: All workers finished. Closing results and rateLimitControl channels.")
		close(results)
		close(rateLimitControl)
	}()

	successCount := 0
	failedCount := 0
	config.InfoLogger.Println("ManageSavedPosts: Collecting results...")
	for result := range results {
		if result.Success {
			successCount++
		} else {
			failedCount++
			config.ErrorLogger.Printf("ManageSavedPosts: Failed to %s post %s: %v", actionType, result.PostID, result.Error)
		}
	}

	config.InfoLogger.Printf("ManageSavedPosts: Finished %s %d posts. Success: %d, Failed: %d.", actionType, numPosts, successCount, failedCount)
	return types.ManagePostResponseType{SuccessCount: successCount, FailedCount: failedCount} // TODO: Use actual type
}

// FetchSavedPostsFullNames retrieves a list of full names for all posts saved by the user.
// It handles pagination from the Reddit API.
func FetchSavedPostsFullNames(token, username string) ([]string, error) {
	if username == "" {
		return nil, fmt.Errorf("username cannot be empty for fetching saved posts") // Use fmt.Errorf for errors.New
	}
	config.InfoLogger.Printf("Fetching saved posts for user %s.", username)
	// The endpoint is /user/{username}/saved.json
	apiURL := fmt.Sprintf("https://oauth.reddit.com/user/%s/saved.json", username)
	nameList, err := fetchAllNames(apiURL, token, false) // false indicates not specifically for subreddits (affects u_ filtering)
	if err != nil {
		return nil, fmt.Errorf("error fetching saved posts for %s: %w", username, err)
	}
	// For saved posts, we expect 'name' (t3_xxxxx), not 'display_name'. The 'fullNamesList' from fetchAllNames contains these.
	// The userDisplayNameList is not relevant for saved posts.
	config.InfoLogger.Printf("Fetched %d saved post full names for user %s.", len(nameList.FullNamesList), username)
	return nameList.FullNamesList, nil
}

// TestRedditAPI sends a simple, non-modifying GET request to the Reddit API to check connectivity and authentication.
// It uses the /api/v1/me endpoint which just requires a valid token.
// targetName is not used in this version but kept for potential future use (e.g. specific resource check).
func TestRedditAPI(token string, targetName string) bool {
	config.DebugLogger.Printf("TestRedditAPI: Testing API with dummy target: %s", targetName) // targetName currently unused

	redditAPIURL := fmt.Sprintf("%s/api/v1/me", config.RedditOauthURL)
	httpClient := http.Client{Timeout: config.TestAPITimeout}
	req, err := http.NewRequest("GET", redditAPIURL, nil)
	if err != nil {
		config.ErrorLogger.Printf("TestRedditAPI: Failed to create request: %v", err)
		return false
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", config.UserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		config.ErrorLogger.Printf("TestRedditAPI: Request failed: %v", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		config.DebugLogger.Println("TestRedditAPI: API test successful (status 200 OK).")
		return true
	} else if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == 429 {
		config.ErrorLogger.Printf("TestRedditAPI: API test failed due to rate limiting (status %d).", resp.StatusCode)
		return false
	} else {
		config.ErrorLogger.Printf("TestRedditAPI: API test failed with status %s.", resp.Status)
		return false
	}
}

// FetchSavedPostsWithDetails retrieves detailed information about all saved posts for a user
// including titles, images, thumbnails, and metadata needed for the selection UI
func FetchSavedPostsWithDetails(token, username string) ([]types.SavedPostInfo, error) {
	if username == "" {
		return nil, fmt.Errorf("username cannot be empty for fetching saved posts")
	}

	config.InfoLogger.Printf("Fetching detailed saved posts for user %s.", username)
	apiURL := fmt.Sprintf("https://oauth.reddit.com/user/%s/saved.json", username)

	var allPosts []types.SavedPostInfo
	lastFullName := ""

	for i := 0; ; i++ {
		paginatedURL := fmt.Sprintf("%s?limit=100&after=%s", apiURL, lastFullName)
		config.DebugLogger.Printf("Fetching saved posts page %d from %s", i+1, paginatedURL)

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
			return nil, fmt.Errorf("error fetching saved posts from %s: %w", paginatedURL, err)
		}

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			config.ErrorLogger.Printf("Failed to fetch saved posts from %s. Status: %d, Body: %s", paginatedURL, resp.StatusCode, string(bodyBytes))
			return nil, fmt.Errorf("failed to fetch saved posts from %s, status code: %d", paginatedURL, resp.StatusCode)
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("error reading saved posts response body from %s: %w", paginatedURL, err)
		}

		var listing struct {
			Kind string `json:"kind"`
			Data struct {
				After    string                   `json:"after"`
				Children []types.DetailedPostData `json:"children"`
			} `json:"data"`
		}

		if err := json.Unmarshal(bodyBytes, &listing); err != nil {
			config.ErrorLogger.Printf("Error unmarshalling saved posts response from %s: %v. Body: %s", paginatedURL, err, string(bodyBytes))
			return nil, fmt.Errorf("error unmarshalling saved posts response from %s: %w", paginatedURL, err)
		}

		config.DebugLogger.Printf("Page %d: found %d saved posts", i+1, len(listing.Data.Children))

		if len(listing.Data.Children) == 0 && listing.Data.After == "" && lastFullName != "" {
			config.DebugLogger.Printf("No posts found on page %d and no 'after' token. End of saved posts for %s.", i+1, username)
			break
		}

		// Process each post and extract detailed information
		for _, child := range listing.Data.Children {
			if child.Kind == "t3" { // Only process actual posts, not comments
				postInfo := parseDetailedPostData(child)
				allPosts = append(allPosts, postInfo)
			}
		}

		if listing.Data.After == "" {
			config.DebugLogger.Printf("No 'after' token in saved posts response. End of list for %s.", username)
			break
		}
		lastFullName = listing.Data.After

		if i > 100 {
			config.ErrorLogger.Printf("fetchSavedPostsWithDetails exceeded 100 pages for %s. Aborting.", username)
			return nil, fmt.Errorf("exceeded 100 pages fetching saved posts for %s", username)
		}
	}

	config.InfoLogger.Printf("Fetched %d detailed saved posts for user %s.", len(allPosts), username)
	return allPosts, nil
}

// parseDetailedPostData converts Reddit API post data into our SavedPostInfo structure
func parseDetailedPostData(postData types.DetailedPostData) types.SavedPostInfo {
	imageData := extractImageData(postData)

	return types.SavedPostInfo{
		ID:          postData.Data.ID,
		FullName:    postData.Data.Name,
		Title:       postData.Data.Title,
		Subreddit:   postData.Data.Subreddit,
		Author:      postData.Data.Author,
		URL:         postData.Data.URL,
		Permalink:   "https://reddit.com" + postData.Data.Permalink,
		Created:     int64(postData.Data.CreatedUTC),
		Score:       postData.Data.Score,
		NumComments: postData.Data.NumComments,
		PostHint:    postData.Data.PostHint,
		Domain:      postData.Data.Domain,
		SelfText:    postData.Data.SelfText,
		IsVideo:     postData.Data.IsVideo,
		IsSelf:      postData.Data.IsSelf,
		NSFW:        postData.Data.Over18,
		Spoiler:     postData.Data.Spoiler,
		ImageData:   imageData,
	}
}

// extractImageData extracts image/media information from Reddit post data
func extractImageData(postData types.DetailedPostData) types.PostImageData {
	imageData := types.PostImageData{
		MediaType: "text", // default
	}

	// Determine media type based on post characteristics
	if postData.Data.IsSelf {
		imageData.MediaType = "text"
	} else if postData.Data.IsVideo {
		imageData.MediaType = "video"
	} else if postData.Data.IsGallery {
		imageData.MediaType = "gallery"
	} else if postData.Data.PostHint == "image" {
		imageData.MediaType = "image"
	} else if postData.Data.PostHint == "link" {
		imageData.MediaType = "link"
	}

	// Extract thumbnail URL
	if postData.Data.Thumbnail != "" &&
		postData.Data.Thumbnail != "self" &&
		postData.Data.Thumbnail != "default" &&
		postData.Data.Thumbnail != "nsfw" &&
		postData.Data.Thumbnail != "spoiler" {
		imageData.ThumbnailURL = postData.Data.Thumbnail
		imageData.Width = postData.Data.ThumbnailWidth
		imageData.Height = postData.Data.ThumbnailHeight
	}

	// Extract preview images if available
	if postData.Data.Preview.Enabled && len(postData.Data.Preview.Images) > 0 {
		firstImage := postData.Data.Preview.Images[0]

		// Use source image as high-res URL
		if firstImage.Source.URL != "" {
			imageData.HighResURL = unescapeHTMLEntities(firstImage.Source.URL)
			imageData.Width = firstImage.Source.Width
			imageData.Height = firstImage.Source.Height
		}

		// Use largest resolution as preview URL
		if len(firstImage.Resolutions) > 0 {
			largestRes := firstImage.Resolutions[len(firstImage.Resolutions)-1]
			imageData.PreviewURL = unescapeHTMLEntities(largestRes.URL)
		} else if imageData.HighResURL != "" {
			imageData.PreviewURL = imageData.HighResURL
		}
	}

	// Handle gallery posts
	if postData.Data.IsGallery && len(postData.Data.GalleryData.Items) > 0 {
		// Use first image from gallery
		firstItem := postData.Data.GalleryData.Items[0]
		if mediaInfo, exists := postData.Data.MediaMetadata[firstItem.MediaID]; exists {
			if mediaInfo.S.U != "" {
				imageData.PreviewURL = unescapeHTMLEntities(mediaInfo.S.U)
				imageData.Width = mediaInfo.S.X
				imageData.Height = mediaInfo.S.Y
			}
		}
	}

	// Fallback: if no preview/thumbnail, try to extract from URL for known image hosts
	if imageData.ThumbnailURL == "" && imageData.PreviewURL == "" {
		imageData.ThumbnailURL = generateThumbnailFromURL(postData.Data.URL, postData.Data.Domain)
	}

	return imageData
}

// generateThumbnailFromURL attempts to generate thumbnail URLs for known image hosting services
func generateThumbnailFromURL(url, domain string) string {
	switch domain {
	case "i.redd.it":
		// Reddit's own image hosting
		return url
	case "imgur.com":
		// Convert imgur URLs to thumbnail format
		if strings.Contains(url, "/") {
			parts := strings.Split(url, "/")
			if len(parts) > 0 {
				filename := parts[len(parts)-1]
				if strings.Contains(filename, ".") {
					nameAndExt := strings.Split(filename, ".")
					if len(nameAndExt) == 2 {
						return fmt.Sprintf("https://i.imgur.com/%st.%s", nameAndExt[0], nameAndExt[1])
					}
				}
			}
		}
	case "i.imgur.com":
		// Already an imgur image, create thumbnail version
		if strings.Contains(url, ".") && !strings.Contains(url, "t.") {
			return strings.Replace(url, ".", "t.", -1)
		}
		return url
	}

	// For other domains, return original URL
	return url
}

// GetSavedPostsCount returns the total count of saved posts for a user
func GetSavedPostsCount(token, username string) (int, error) {
	if username == "" {
		return 0, fmt.Errorf("username cannot be empty for fetching saved posts count")
	}

	config.DebugLogger.Printf("Getting saved posts count for user %s.", username)
	apiURL := fmt.Sprintf("https://oauth.reddit.com/user/%s/saved.json", username)

	count := 0
	lastFullName := ""

	for i := 0; ; i++ {
		paginatedURL := fmt.Sprintf("%s?limit=100&after=%s", apiURL, lastFullName)

		req, err := http.NewRequest(http.MethodGet, paginatedURL, nil)
		if err != nil {
			return 0, fmt.Errorf("error creating request for saved posts count: %w", err)
		}

		req.Header = http.Header{
			"Authorization": {"Bearer " + token},
			"User-Agent":    {config.UserAgent},
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return 0, fmt.Errorf("error fetching saved posts count: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return 0, fmt.Errorf("failed to fetch saved posts count, status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return 0, fmt.Errorf("error reading saved posts count response: %w", err)
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
			return 0, fmt.Errorf("error unmarshalling saved posts count response: %w", err)
		}

		// Count only actual posts (t3_)
		for _, child := range listing.Data.Children {
			if child.Kind == "t3" {
				count++
			}
		}

		if listing.Data.After == "" {
			break
		}
		lastFullName = listing.Data.After

		if i > 100 {
			return 0, fmt.Errorf("exceeded 100 pages fetching saved posts count")
		}
	}

	config.DebugLogger.Printf("Found %d saved posts for user %s.", count, username)
	return count, nil
}
