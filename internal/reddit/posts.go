package reddit

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
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

// Result holds the outcome of processing a single post.
// It includes the PostID, whether the operation was successful, and any error encountered.
// type Result struct {
// 	PostID  string
// 	Success bool
// 	Error   error
// }

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
