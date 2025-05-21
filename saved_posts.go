package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
	DebugLogger *log.Logger
)

func init() {
	InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lmicroseconds)
	ErrorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lmicroseconds)
	DebugLogger = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lmicroseconds)
	loadConfig() // Load configuration after loggers are initialized
}

// Result holds the outcome of processing a single post.
// It includes the PostID, whether the operation was successful, and any error encountered.
type Result struct {
	PostID  string
	Success bool
	Error   error
}

// RateLimiter provides a token bucket based rate limiting mechanism.
// It controls the frequency of operations to avoid overwhelming an external API.
type RateLimiter struct {
	requests     chan struct{} // Channel acting as the token bucket.
	interval     time.Duration // Interval at which tokens are refreshed/refilled.
	maxTokens    int           // Maximum number of tokens (requests) allowed per interval.
	pauseSignal  chan struct{} // Signal to pause the rate limiter.
	resumeSignal chan struct{} // Signal to resume the rate limiter.
	isPaused     bool          // Current paused state of the rate limiter.
	mu           sync.RWMutex  // Mutex to protect access to isPaused state.
}

// NewRateLimiter creates and starts a new RateLimiter.
// maxTokens defines how many requests can be made within the given interval.
func NewRateLimiter(maxTokens int, interval time.Duration) *RateLimiter {
	InfoLogger.Printf("RateLimiter: Initializing with maxTokens=%d, interval=%v", maxTokens, interval)
	rl := &RateLimiter{
		requests:     make(chan struct{}, maxTokens), // Buffered channel for tokens.
		interval:     interval,
		maxTokens:    maxTokens,
		pauseSignal:  make(chan struct{}, 1), // Buffered to prevent blocking on send.
		resumeSignal: make(chan struct{}, 1), // Buffered to prevent blocking on send.
		isPaused:     false,
	}
	go rl.refillTokens() // Start the token refilling goroutine.
	return rl
}

// refillTokens is a goroutine that periodically clears the token bucket, effectively refilling it.
// This runs in the background for the lifetime of the RateLimiter.
func (rl *RateLimiter) refillTokens() {
	ticker := time.NewTicker(rl.interval)
	defer ticker.Stop()

	DebugLogger.Printf("RateLimiter: Starting token refill goroutine with interval %v", rl.interval)
	for range ticker.C {
		rl.mu.RLock() // Read lock to check isPaused.
		isPausedCurrent := rl.isPaused
		rl.mu.RUnlock()

		if !isPausedCurrent {
			// Drain all current tokens from the bucket to simulate a refill.
			// This means after each interval, a burst of up to maxTokens is allowed.
			numToDrain := len(rl.requests)
			for i := 0; i < numToDrain; i++ {
				select {
				case <-rl.requests:
				default: // Channel might have been emptied by concurrent Wait() calls.
					break
				}
			}
			if numToDrain > 0 {
				// This logging can be noisy if the bucket is often empty.
				// DebugLogger.Printf("RateLimiter: Token bucket refilled (drained %d tokens).", numToDrain)
			}
		} else {
			DebugLogger.Printf("RateLimiter: Token refill skipped because limiter is paused.")
		}
	}
}

// Pause stops the rate limiter from issuing new tokens.
// Ongoing operations are not affected, but new calls to Wait will block until Resume is called.
func (rl *RateLimiter) Pause() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if !rl.isPaused {
		rl.isPaused = true
		InfoLogger.Println("RateLimiter: Paused.")
		// Non-blocking send on pauseSignal. This helps unblock Wait if it's selecting on it.
		select {
		case rl.pauseSignal <- struct{}{}:
			DebugLogger.Println("RateLimiter: Pause signal sent to unblock waiters.")
		default:
			DebugLogger.Println("RateLimiter: Pause signal channel full or no active waiters on pauseSignal.")
		}
	}
}

// Resume allows the rate limiter to start issuing tokens again.
func (rl *RateLimiter) Resume() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.isPaused {
		rl.isPaused = false
		InfoLogger.Println("RateLimiter: Resumed.")
		// Non-blocking send on resumeSignal. This helps unblock Wait if it's selecting on it.
		select {
		case rl.resumeSignal <- struct{}{}:
			DebugLogger.Println("RateLimiter: Resume signal sent to unblock waiters.")
		default:
			DebugLogger.Println("RateLimiter: Resume signal channel full or no active waiters on resumeSignal.")
		}
	}
}

// Wait blocks until a token is available from the rate limiter or the limiter is paused/resumed.
// This should be called before performing a rate-limited operation.
func (rl *RateLimiter) Wait() {
	DebugLogger.Println("RateLimiter: Attempting to acquire token...")
	startTime := time.Now()

	for { // Loop indefinitely until a token is acquired or context is cancelled externally.
		rl.mu.RLock()
		isPausedCurrent := rl.isPaused
		rl.mu.RUnlock()

		if !isPausedCurrent {
			select {
			case rl.requests <- struct{}{}: // Try to send a request (acquire a token).
				DebugLogger.Printf("RateLimiter: Token acquired. Wait time: %v", time.Since(startTime))
				return
			case <-rl.pauseSignal: // If pauseSignal is triggered while waiting for a token.
				DebugLogger.Println("RateLimiter: Notified by pauseSignal while attempting to acquire token. Re-evaluating state.")
				continue // Re-check pause status.
			case <-time.After(RateLimitSleepInterval): // Periodically re-evaluate if no token or signal.
				DebugLogger.Printf("RateLimiter: Timed out waiting for token/pause signal, re-checking state.")
				continue
			}
		} else {
			DebugLogger.Println("RateLimiter: Currently paused. Waiting for resume signal or timeout.")
			select {
			case <-rl.resumeSignal: // If resumeSignal is triggered while paused.
				DebugLogger.Println("RateLimiter: Notified by resumeSignal while paused. Re-evaluating state.")
				continue // Re-check pause status, should now be unpaused.
			case <-time.After(RateLimitInterval): // Periodically re-evaluate if still paused.
				DebugLogger.Printf("RateLimiter: Timed out waiting for resume signal, re-checking pause state.")
				continue
			}
		}
	}
}

// manageSavedPosts coordinates the saving or unsaving of posts concurrently using worker goroutines.
// It employs a rate limiter and a mechanism to pause/resume workers if API rate limits are hit.
// token: The OAuth token for API authentication.
// postIDs: A slice of post full names (e.g., "t3_xxxxx") to be processed.
// saveType: The action to perform (SAVE or UNSAVE).
// concurrency: The number of worker goroutines to use.
func manageSavedPosts(token string, postIDs []string, saveType post_save_type, concurrency int) manage_post_type {
	numPosts := len(postIDs)
	InfoLogger.Printf("ManageSavedPosts: Starting to %s %d posts. Concurrency: %d.", saveType, numPosts, concurrency)

	if concurrency < 1 {
		DebugLogger.Printf("ManageSavedPosts: Concurrency was %d, adjusted to minimum of 1.", concurrency)
		concurrency = 1 // Ensure at least one worker.
	}
	if numPosts == 0 {
		InfoLogger.Printf("ManageSavedPosts: No posts to %s. Operation skipped.", saveType)
		return manage_post_type{SuccessCount: 0, FailedCount: 0}
	}

	rateLimiter := NewRateLimiter(MaxTokensPerInterval, RateLimitInterval)

	jobs := make(chan string, numPosts)
	results := make(chan Result, numPosts)
	rateLimitControl := make(chan bool) // Unbuffered: worker signals pause, controller reacts then worker proceeds.

	var wg sync.WaitGroup
	// Create a context that can be cancelled to signal workers to shut down gracefully.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cancel is called when manageSavedPosts exits.

	// Start the rate limit controller goroutine.
	// This goroutine reacts to rate limit signals from workers.
	go func() {
		DebugLogger.Println("ManageSavedPosts: Rate limit controller goroutine started.")
		for shouldPause := range rateLimitControl { // Loop until rateLimitControl is closed.
			if shouldPause {
				InfoLogger.Println("ManageSavedPosts: Rate limit controller received pause signal from a worker.")
				rateLimiter.Pause()
				InfoLogger.Printf("ManageSavedPosts: Rate limiter paused. Sleeping for %v before testing API.", RateLimitSleepInterval)
				time.Sleep(RateLimitSleepInterval)

				// After initial sleep, repeatedly test if the Reddit API is accessible.
				for !testRedditAPI(token, "t3_testdummy") { // Use a generic, non-modifying API call for testing.
					ErrorLogger.Printf("ManageSavedPosts: Test request failed. Rate limit likely still active. Sleeping again for %v.", RateLimitSleepInterval)
					time.Sleep(RateLimitSleepInterval)
				}
				InfoLogger.Println("ManageSavedPosts: Test request successful. Resuming rate limiter.")
				rateLimiter.Resume()
			} else {
				// This path (false signal) is not currently used by workers for rateLimitControl.
				DebugLogger.Println("ManageSavedPosts: Rate limit controller received 'false' signal (currently unused).")
			}
		}
		DebugLogger.Println("ManageSavedPosts: Rate limit controller goroutine exiting as control channel was closed.")
	}()

	// Start worker goroutines.
	InfoLogger.Printf("ManageSavedPosts: Starting %d workers for %s operation.", concurrency, saveType)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			DebugLogger.Printf("Worker %d: Started for %s operation.", workerID, saveType)
			postWorker(ctx, token, saveType, rateLimiter, jobs, results, rateLimitControl, workerID)
			DebugLogger.Printf("Worker %d: Finished processing jobs.", workerID)
		}(i)
	}

	// Queue all postIDs into the jobs channel.
	InfoLogger.Printf("ManageSavedPosts: Queueing %d posts for %s operation.", numPosts, saveType)
	for _, postID := range postIDs {
		select {
		case jobs <- postID:
		case <-ctx.Done(): // If context is cancelled while queueing jobs.
			ErrorLogger.Printf("ManageSavedPosts: Context cancelled while queueing jobs. %d posts not queued.", numPosts-len(jobs))
			break // Exit loop, remaining jobs won't be queued.
		}
	}
	close(jobs) // Signal workers that all jobs have been sent.
	DebugLogger.Println("ManageSavedPosts: All posts queued. Waiting for workers to complete...")

	// Create a goroutine to wait for all workers to finish, then close channels.
	go func() {
		wg.Wait() // Block until all worker goroutines have called wg.Done().
		DebugLogger.Println("ManageSavedPosts: All workers have completed.")
		close(results)          // Close results channel: signals the result collection loop to terminate.
		close(rateLimitControl) // Close control channel: signals the rate limit controller goroutine to terminate.
		DebugLogger.Println("ManageSavedPosts: Results and rateLimitControl channels closed.")
	}()

	// Collect and summarize results from the results channel.
	var successfulCount int
	var failedCount int
	var failedPostIDs []string // Store IDs of posts that failed for logging.

	for result := range results { // Loop until the results channel is closed.
		if result.Success {
			successfulCount++
			DebugLogger.Printf("ManageSavedPosts: Post %s processed successfully.", result.PostID)
		} else {
			failedCount++
			failedPostIDs = append(failedPostIDs, result.PostID)
			if result.Error != nil {
				ErrorLogger.Printf("ManageSavedPosts: Failed to %s post %s: %v", saveType, result.PostID, result.Error)
			} else {
				// This case should ideally not happen if errors are always populated on failure.
				ErrorLogger.Printf("ManageSavedPosts: Failed to %s post %s (no specific error returned, likely HTTP non-200).", saveType, result.PostID)
			}
		}
	}

	InfoLogger.Printf("ManageSavedPosts: %s operation complete. Success: %d, Failed: %d.", saveType, successfulCount, failedCount)
	if failedCount > 0 {
		// Log only a summary of failed IDs if there are many, to avoid overly verbose logs.
		maxFailedIDsToLog := 10
		if len(failedPostIDs) > maxFailedIDsToLog {
			InfoLogger.Printf("ManageSavedPosts: Failed to %s %d posts. First %d failed IDs: %v...", saveType, failedCount, maxFailedIDsToLog, failedPostIDs[:maxFailedIDsToLog])
		} else {
			InfoLogger.Printf("ManageSavedPosts: Failed to %s %d posts. IDs: %v", saveType, failedCount, failedPostIDs)
		}
	}

	return manage_post_type{
		SuccessCount: successfulCount,
		FailedCount:  failedCount,
	}
}

// postWorker is the core function executed by each worker goroutine.
// It continuously fetches postIDs from the `jobs` channel, processes them by calling `processSinglePost`,
// and sends the `Result` to the `results` channel.
// It respects rate limits using `rateLimiter` and handles graceful shutdown via `ctx`.
// If a rate limit (429) is hit, it signals the `rateLimitControl` channel.
func postWorker(
	ctx context.Context, // Context for graceful shutdown.
	token string, // OAuth token for Reddit API.
	saveType post_save_type, // Action to perform: SAVE or UNSAVE.
	rateLimiter *RateLimiter, // Shared rate limiter instance.
	jobs <-chan string, // Channel to receive postIDs (jobs) from.
	results chan<- Result, // Channel to send processing results to.
	rateLimitControl chan<- bool, // Channel to signal rate limit issues to the controller.
	workerID int, // Identifier for logging purposes.
) {
	apiURL := fmt.Sprintf("https://oauth.reddit.com/api/%s", saveType)
	DebugLogger.Printf("Worker %d: Initialized for %s operation. API URL: %s", workerID, saveType, apiURL)

	for postID := range jobs { // Loop until the jobs channel is closed and drained.
		// Check for context cancellation at the beginning of each job processing cycle.
		select {
		case <-ctx.Done():
			DebugLogger.Printf("Worker %d: Context cancelled. Job for post %s will not be processed. Exiting.", workerID, postID)
			// Send a result indicating cancellation if needed, or just exit.
			// Depending on requirements, you might want to ensure a Result is sent for pending jobs.
			// For now, we assume if a job was pulled, its result must be sent or it means the worker exited before sending.
			return // Exit the worker goroutine.
		default:
			// Context is not yet cancelled, proceed.
		}

		if postID == "" {
			DebugLogger.Printf("Worker %d: Received empty post ID. Skipping.", workerID)
			continue // Get the next job.
		}

		DebugLogger.Printf("Worker %d: Waiting for rate limit token to process post %s.", workerID, postID)
		rateLimiter.Wait() // Block here until a token is available.

		// After acquiring a rate limit token, re-check for context cancellation.
		// This is important if the Wait() was long and the context was cancelled during that time.
		select {
		case <-ctx.Done():
			DebugLogger.Printf("Worker %d: Context cancelled after acquiring rate limit token for post %s. Exiting.", workerID, postID)
			return
		default:
		}

		DebugLogger.Printf("Worker %d: Processing post %s (%s).", workerID, postID, saveType)
		result := processSinglePost(ctx, token, postID, apiURL, saveType, rateLimitControl, workerID)

		// Attempt to send the result. If the context is cancelled, sending might fail or block indefinitely
		// if the results channel is not being read from (e.g., main goroutine also exited).
		select {
		case results <- result:
			if !result.Success {
				DebugLogger.Printf("Worker %d: Sent failure result for post %s to results channel.", workerID, postID)
			} else {
				DebugLogger.Printf("Worker %d: Sent success result for post %s to results channel.", workerID, postID)
			}
		case <-ctx.Done():
			DebugLogger.Printf("Worker %d: Context cancelled while trying to send result for post %s. Result discarded. Exiting.", workerID, postID)
			return // Exit if context is done to prevent blocking on results channel send.
		}
	}
	DebugLogger.Printf("Worker %d: Job channel closed and all jobs processed. Exiting.", workerID)
}

// processSinglePost handles the actual HTTP request to save or unsave a single post on Reddit.
// It constructs the request, sends it, and interprets the response.
// If a rate limit (HTTP 429) is encountered, it sends `true` to the `rateLimitControl` channel.
// Returns a `Result` struct summarizing the outcome.
func processSinglePost(ctx context.Context, token, postID, apiURL string, saveType post_save_type, rateLimitControl chan<- bool, workerID int) Result {
	payload := []byte(fmt.Sprintf("id=%s", postID)) // e.g., "id=t3_xxxxx"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(payload)) // Use NewRequestWithContext
	if err != nil {
		ErrorLogger.Printf("Worker %d: Failed to create HTTP request for post %s (%s): %v", workerID, postID, saveType, err)
		return Result{PostID: postID, Success: false, Error: fmt.Errorf("creating request for post %s: %w", postID, err)}
	}

	req.Header = http.Header{
		"Authorization": {"Bearer " + token},
		"User-Agent":    {UserAgent},                           // Use UserAgent from config
		"Content-Type":  {"application/x-www-form-urlencoded"}, // Reddit API for save/unsave expects this.
	}

	DebugLogger.Printf("Worker %d: Sending API request for post %s (%s) to %s.", workerID, postID, saveType, apiURL)
	// Use a client with a timeout to prevent indefinite blocking.
	client := http.Client{Timeout: DefaultAPITimeout} // Use configured timeout
	resp, err := client.Do(req)
	if err != nil {
		ErrorLogger.Printf("Worker %d: HTTP client error for post %s (%s) during API call to %s: %v", workerID, postID, saveType, apiURL, err)
		return Result{PostID: postID, Success: false, Error: fmt.Errorf("HTTP client error for post %s: %w", postID, err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests { // HTTP 429: Rate limit hit.
		InfoLogger.Printf("Worker %d: Rate limit (429) hit while processing post %s (%s). Signaling controller.", workerID, postID, saveType)
		// Send a signal to the rate limit controller. This is a blocking send if the channel is unbuffered
		// and the controller isn't ready. Consider if this should be non-blocking or if context cancellation applies here.
		// For now, assuming controller is responsive or channel has buffer (it's unbuffered currently).
		select {
		case rateLimitControl <- true: // Signal that a pause is needed.
			DebugLogger.Printf("Worker %d: Successfully signaled rate limit pause to controller for post %s.", workerID, postID)
		// No default needed if we expect the controller to always be listening until closed.
		// However, if context might be cancelled, this select could block.
		// Adding ctx.Done() case for robustness:
		case <-ctx.Done():
			DebugLogger.Printf("Worker %d: Context cancelled while trying to signal rate limit for post %s. Signal not sent.", workerID, postID)
			return Result{PostID: postID, Success: false, Error: fmt.Errorf("rate limited (429) and context cancelled on post %s", postID)}
		}
		// This job failed due to rate limiting. It might be retried by a higher-level mechanism if implemented.
		return Result{PostID: postID, Success: false, Error: fmt.Errorf("rate limited (429) on post %s", postID)}
	}

	// For save/unsave, Reddit API typically returns 200 OK on success.
	if resp.StatusCode != http.StatusOK {
		bodyBytes, readErr := ioutil.ReadAll(resp.Body)
		if readErr != nil {
			ErrorLogger.Printf("Worker %d: Failed to %s post %s. Status: %d. Also failed to read error body: %v", workerID, saveType, postID, resp.StatusCode, readErr)
			return Result{PostID: postID, Success: false, Error: fmt.Errorf("API error for post %s, status %d (body read failed: %v)", postID, resp.StatusCode, readErr)}
		}
		ErrorLogger.Printf("Worker %d: Failed to %s post %s. Status: %d. Body: %s", workerID, saveType, postID, resp.StatusCode, string(bodyBytes))
		return Result{PostID: postID, Success: false, Error: fmt.Errorf("API error for post %s, status %d: %s", postID, resp.StatusCode, string(bodyBytes))}
	}

	// If we reach here, the operation was successful (HTTP 200 OK).
	DebugLogger.Printf("Worker %d: Successfully processed post %s (%s). Status: %d", workerID, postID, saveType, resp.StatusCode)
	return Result{PostID: postID, Success: true, Error: nil}
}

// testRedditAPI makes a lightweight, authenticated GET request to a generic Reddit API endpoint (e.g., /api/info).
// It's used by the rate limit controller to check if rate limits (HTTP 429) have cleared after a pause.
// `targetName` can be a dummy post ID like "t3_testdummy" or any valid fullname for /api/info.
// Returns `true` if the API seems accessible (not 429), `false` otherwise.
func testRedditAPI(token string, targetName string) bool {
	// Using /api/info with a (potentially non-existent) fullname is a good lightweight test.
	// It requires authentication but doesn't modify data.
	testAPIURL := fmt.Sprintf("%s/api/info.json?id=%s", RedditOauthURL, targetName) // Use RedditOauthURL from config
	DebugLogger.Printf("RateLimitController: Sending test request to %s to check API status.", testAPIURL)

	if token == "" {
		ErrorLogger.Println("RateLimitController: Test request cannot be made - empty token provided.")
		return false // Cannot make an authenticated request, so assume API is not ready.
	}

	// Create a context with a timeout for the test request itself.
	reqCtx, cancelReqCtx := context.WithTimeout(context.Background(), TestAPITimeout) // Use configured timeout
	defer cancelReqCtx()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, testAPIURL, nil) // Use NewRequestWithContext
	if err != nil {
		ErrorLogger.Printf("RateLimitController: Error creating test request to %s: %v", testAPIURL, err)
		return false // If request can't be created, conservatively assume API is not ready.
	}

	req.Header = http.Header{
		"Authorization": {"Bearer " + token},
		"User-Agent":    {UserAgent}, // Use UserAgent from config
	}

	// Use a client with a reasonable timeout for this test request.
	// The request context will handle finer-grained timeout for the request lifecycle.
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		ErrorLogger.Printf("RateLimitController: Error making test request to %s: %v", testAPIURL, err)
		return false // Network or other error, assume API is not ready.
	}
	defer resp.Body.Close()

	bodyBytes, readErr := ioutil.ReadAll(resp.Body) // Read body for logging, especially on unexpected status.
	if readErr != nil {
		ErrorLogger.Printf("RateLimitController: Error reading response body from test request to %s (Status %d): %v", testAPIURL, resp.StatusCode, readErr)
		// If we can't read the body, it's hard to know the state, but non-429 might still be okay.
	}

	if resp.StatusCode == http.StatusTooManyRequests { // HTTP 429: Explicitly rate limited.
		InfoLogger.Printf("RateLimitController: Test request to %s hit rate limit (429). Body: %s", testAPIURL, string(bodyBytes))
		return false // Still rate limited.
	}

	// For /api/info, a 200 OK is expected if the API is generally working, even if the ID is not found (empty `children` list).
	// Other statuses (e.g., 401 Unauthorized, 403 Forbidden, 5xx Server Error) indicate problems beyond just rate limiting.
	if resp.StatusCode != http.StatusOK {
		ErrorLogger.Printf("RateLimitController: Test request to %s received unexpected status %d (not 200 or 429). Body: %s. Assuming API not fully ready.",
			testAPIURL, resp.StatusCode, string(bodyBytes))
		return false // Any other non-OK status is treated as the API not being ready.
	}

	DebugLogger.Printf("RateLimitController: Test request to %s successful (Status: %d). Assuming rate limit cleared.", testAPIURL, resp.StatusCode)
	return true // API is accessible (not 429 and was 200 OK).
}
