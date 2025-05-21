package worker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/nileshnk/reddit-migrate/internal/config"
	"github.com/nileshnk/reddit-migrate/internal/ratelimiter"
	"github.com/nileshnk/reddit-migrate/internal/types"
)

// Result holds the outcome of processing a single post.
// It includes the PostID, whether the operation was successful, and any error encountered.
type Result struct {
	PostID  string
	Success bool
	Error   error
}

// PostWorker processes individual post jobs (save/unsave) using a rate limiter.
// It's designed to be run as a goroutine.
func PostWorker(
	ctx context.Context,
	token string,
	actionType types.PostActionType,
	rateLimiter *ratelimiter.RateLimiter,
	jobs <-chan string,
	results chan<- Result,
	rateLimitControl chan<- bool,
	workerID int,
) {
	redditOauthURL := config.RedditOauthURL
	apiEndpoint := ""
	switch actionType {
	case types.SaveAction:
		apiEndpoint = fmt.Sprintf("%s/api/save", redditOauthURL)
	case types.UnsaveAction:
		apiEndpoint = fmt.Sprintf("%s/api/unsave", redditOauthURL)
	default:
		config.ErrorLogger.Printf("Worker %d: Unknown action type: %v", workerID, actionType)
		// Send results for any jobs already pulled if an unknown action type is somehow passed.
		for postID := range jobs {
			results <- Result{PostID: postID, Success: false, Error: fmt.Errorf("unknown action type: %v", actionType)}
		}
		return
	}

	for postID := range jobs {
		select {
		case <-ctx.Done(): // Check if context was cancelled before processing job.
			config.DebugLogger.Printf("Worker %d: Context cancelled. Exiting. Post %s not processed.", workerID, postID)
			results <- Result{PostID: postID, Success: false, Error: ctx.Err()} // Report as error due to cancellation.
			return                                                              // Exit worker.
		default:
			// Context not cancelled, proceed to process the job.
			config.DebugLogger.Printf("Worker %d: Processing post %s for %s action.", workerID, postID, actionType)
			rateLimiter.Wait() // Wait for rate limiter token.
			// Double-check context after acquiring rate limit token, in case of long wait.
			select {
			case <-ctx.Done():
				config.DebugLogger.Printf("Worker %d: Context cancelled after acquiring rate limit token. Post %s not processed.", workerID, postID)
				results <- Result{PostID: postID, Success: false, Error: ctx.Err()}
				return
			default:
				results <- processSinglePost(ctx, token, postID, apiEndpoint, actionType, rateLimitControl, workerID)
			}
		}
	}
	config.DebugLogger.Printf("Worker %d: No more jobs. Exiting.", workerID)
}

func processSinglePost(ctx context.Context, token, postID, apiURL string, actionType types.PostActionType, rateLimitControl chan<- bool, workerID int) Result {
	config.DebugLogger.Printf("Worker %d: Action %s on post %s using URL %s", workerID, actionType, postID, apiURL)

	userAgent := config.UserAgent
	httpClient := http.Client{Timeout: config.DefaultAPITimeout}

	payload := []byte(fmt.Sprintf("id=%s", postID))
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(payload))
	if err != nil {
		config.ErrorLogger.Printf("Worker %d: Failed to create request for post %s: %v", workerID, postID, err)
		return Result{PostID: postID, Success: false, Error: err}
	}

	req.Header.Set("Authorization", "bearer "+token)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		config.ErrorLogger.Printf("Worker %d: Failed to %s post %s: %v", workerID, actionType, postID, err)
		return Result{PostID: postID, Success: false, Error: err}
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		config.ErrorLogger.Printf("Worker %d: Failed to read response body for post %s: %v", workerID, postID, err)
		return Result{PostID: postID, Success: false, Error: err}
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == 429 {
		config.ErrorLogger.Printf("Worker %d: Rate limit hit (status %d) when trying to %s post %s. Signaling pause.", workerID, resp.StatusCode, actionType, postID)
		select {
		case rateLimitControl <- true:
			config.DebugLogger.Printf("Worker %d: Pause signal sent to controller.", workerID)
		case <-ctx.Done():
			config.ErrorLogger.Printf("Worker %d: Context cancelled while trying to signal pause for post %s.", workerID, postID)
			return Result{PostID: postID, Success: false, Error: fmt.Errorf("rate limit hit, context cancelled before pause: %w", ctx.Err())}
		}
		// Return as error, post will be retried if this is part of a larger retry mechanism (not implemented here for single post)
		return Result{PostID: postID, Success: false, Error: fmt.Errorf("rate limited (status %d): %s", resp.StatusCode, string(bodyBytes))}
	}

	if resp.StatusCode != http.StatusOK {
		config.ErrorLogger.Printf("Worker %d: Failed to %s post %s. Status: %s, Body: %s", workerID, actionType, postID, resp.Status, string(bodyBytes))
		return Result{PostID: postID, Success: false, Error: fmt.Errorf("failed with status %s: %s", resp.Status, string(bodyBytes))}
	}

	config.DebugLogger.Printf("Worker %d: Successfully %s post %s. Status: %s", workerID, actionType, postID, resp.Status)
	return Result{PostID: postID, Success: true}
}
