package main

import (
	"bytes"
	"context"
	"fmt"
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

type Result struct {
	PostID  string
	Success bool
	Error   error
}

func init() {
	InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lmicroseconds)
	ErrorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lmicroseconds)
	DebugLogger = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lmicroseconds)
}

type RateLimiter struct {
	requests     chan struct{}
	interval     time.Duration
	maxTokens    int
	pauseSignal  chan struct{}
	resumeSignal chan struct{}
	isPaused     bool
	mu           sync.RWMutex
}

func NewRateLimiter(maxTokens int, interval time.Duration) *RateLimiter {
	InfoLogger.Printf("Creating new rate limiter with maxTokens=%d, interval=%v", maxTokens, interval)

	rl := &RateLimiter{
		requests:     make(chan struct{}, maxTokens),
		interval:     interval,
		maxTokens:    maxTokens,
		pauseSignal:  make(chan struct{}, 1), // Added buffer
		resumeSignal: make(chan struct{}, 1), // Added buffer
		isPaused:     false,
	}

	go rl.refreshTokens()
	return rl
}

func (rl *RateLimiter) refreshTokens() {
	ticker := time.NewTicker(rl.interval)
	defer ticker.Stop()

	DebugLogger.Printf("Starting token refresh goroutine with interval %v", rl.interval)

	for range ticker.C {
		rl.mu.RLock()
		if !rl.isPaused {
			tokenCount := len(rl.requests)
			DebugLogger.Printf("Clearing %d tokens from bucket", tokenCount)
			for len(rl.requests) > 0 {
				<-rl.requests
			}
		}
		rl.mu.RUnlock()
	}
}

func (rl *RateLimiter) Pause() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if !rl.isPaused {
		rl.isPaused = true
		InfoLogger.Println("Rate limiter paused")
		select {
		case rl.pauseSignal <- struct{}{}:
			DebugLogger.Println("Pause signal sent successfully")
		default:
			DebugLogger.Println("Pause signal channel full, signal not sent")
		}
	}
}

func (rl *RateLimiter) Resume() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.isPaused {
		rl.isPaused = false
		InfoLogger.Println("Rate limiter resumed")
		select {
		case rl.resumeSignal <- struct{}{}:
			DebugLogger.Println("Resume signal sent successfully")
		default:
			DebugLogger.Println("Resume signal channel full, signal not sent")
		}
	}
}

func (rl *RateLimiter) Wait() {
	startTime := time.Now()
	DebugLogger.Println("Waiting for rate limit token")

	for {
		rl.mu.RLock()
		isPaused := rl.isPaused
		rl.mu.RUnlock()

		if !isPaused {
			select {
			case rl.requests <- struct{}{}:
				DebugLogger.Printf("Token acquired after waiting %v", time.Since(startTime))
				return
			case <-rl.pauseSignal:
				DebugLogger.Println("Received pause signal while waiting for token")
				continue
			}
		}

		select {
		case <-rl.resumeSignal:
			DebugLogger.Println("Received resume signal")
			continue
		case <-time.After(100 * time.Millisecond):
			continue
		}
	}
}

func manageSavedPosts(token string, postIds []string, saveType post_save_type, concurrency int) manage_post_type {
	InfoLogger.Printf("Starting to manage %d posts with concurrency %d", len(postIds), concurrency)

	if concurrency < 1 {
		InfoLogger.Printf("Adjusting concurrency from %d to 1", concurrency)
		concurrency = 1
	}

	rateLimiter := NewRateLimiter(50, 2*time.Minute)
	jobs := make(chan string, len(postIds))
	results := make(chan Result, len(postIds))
	rateLimitControl := make(chan bool, 1)

	// Add WaitGroup for worker goroutines
	var wg sync.WaitGroup

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start rate limit controller
	go func() {
		defer close(rateLimitControl)
		for shouldPause := range rateLimitControl {
			if shouldPause {
				InfoLogger.Println("Rate limit triggered - initiating pause sequence")
				rateLimiter.Pause()
				InfoLogger.Println("Waiting 10 minutes for rate limit reset")
				time.Sleep(10 * time.Minute)

				success := testRequest(token, saveType)
				for !success {
					InfoLogger.Println("Rate limit still active - waiting additional interval")
					time.Sleep(10 * time.Minute)
					success = testRequest(token, saveType)
				}
				InfoLogger.Println("Rate limit cleared - resuming operations")
				rateLimiter.Resume()
			}
		}
	}()

	// Start workers
	InfoLogger.Printf("Starting %d workers", concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			DebugLogger.Printf("Worker %d started", workerID)
			worker(ctx, token, saveType, rateLimiter, jobs, results, rateLimitControl)
			DebugLogger.Printf("Worker %d finished", workerID)
		}(i)
	}

	// Queue posts
	InfoLogger.Println("Queueing posts for processing")
	for _, postID := range postIds {
		jobs <- postID
	}
	close(jobs)
	InfoLogger.Println("All posts queued, waiting for completion")

	// Wait for workers to complete and close results channel
	go func() {
		wg.Wait()
		InfoLogger.Println("All workers completed, closing channels")
		close(results)
	}()

	// Process results
	var failedCount int
	var failedSavePostIds []string

	for result := range results {
		if !result.Success {
			failedSavePostIds = append(failedSavePostIds, result.PostID)
			failedCount++
			if result.Error != nil {
				ErrorLogger.Printf("Failed to process post %s: %v", result.PostID, result.Error)
			}
		} else {
			DebugLogger.Printf("Successfully processed post %s", result.PostID)
		}
	}

	InfoLogger.Printf("Processing complete. Success: %d, Failed: %d", len(postIds)-failedCount, failedCount)
	if len(failedSavePostIds) > 0 {
		InfoLogger.Printf("Failed post IDs: %v", failedSavePostIds)
	}

	return manage_post_type{
		FailedCount:  failedCount,
		SuccessCount: len(postIds) - failedCount,
	}
}

func worker(
	ctx context.Context,
	token string,
	saveType post_save_type,
	rateLimiter *RateLimiter,
	jobs <-chan string,
	results chan<- Result,
	rateLimitControl chan<- bool,
) {
	if rateLimiter == nil || jobs == nil || results == nil || rateLimitControl == nil {
		ErrorLogger.Fatal("Worker received nil channel")
	}

	requiredUri := fmt.Sprintf("https://oauth.reddit.com/api/%v", saveType)
	DebugLogger.Printf("Worker started processing with URI: %s", requiredUri)

	for postID := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
			if postID == "" {
				DebugLogger.Println("Skipping empty post ID")
				continue
			}

			DebugLogger.Printf("Processing post ID: %s", postID)
			rateLimiter.Wait()

			result := processPost(token, postID, requiredUri, rateLimitControl)

			// Only send to channels if context is not done
			select {
			case <-ctx.Done():
				return
			case results <- result:
				if !result.Success && result.Error != nil {
					ErrorLogger.Printf("Failed to process post %s: %v", postID, result.Error)
				}
			}
		}
	}
}

func processPost(token, postID, requiredUri string, rateLimitControl chan<- bool) Result {
	requiredBody := []byte(fmt.Sprintf("id=%v", postID))
	managePostReq, err := http.NewRequest(http.MethodPost, requiredUri, bytes.NewBuffer(requiredBody))
	if err != nil {
		return Result{PostID: postID, Success: false, Error: err}
	}

	managePostReq.Header = http.Header{
		"Authorization": []string{"Bearer " + token},
		"User-Agent":    []string{"Mozilla/5.0 (X11; Linux x86_64; rv:91.0) Gecko/20100101 Firefox/91.0"},
	}

	managePostRes, err := http.DefaultClient.Do(managePostReq)
	if err != nil {
		return Result{PostID: postID, Success: false, Error: err}
	}
	defer managePostRes.Body.Close()

	if managePostRes.StatusCode == 429 {
		InfoLogger.Printf("Rate limit hit while processing post %s", postID)
		select {
		case rateLimitControl <- true:
			DebugLogger.Println("Successfully signaled rate limit")
		default:
			ErrorLogger.Println("Failed to signal rate limit - channel full")
		}
		return Result{PostID: postID, Success: false, Error: fmt.Errorf("rate limited")}
	}

	success := managePostRes.StatusCode == 200
	if !success {
		ErrorLogger.Printf("Failed to process post %s with status code %d", postID, managePostRes.StatusCode)
	}

	return Result{
		PostID:  postID,
		Success: success,
		Error:   nil,
	}
}
func testRequest(token string, saveType post_save_type) bool {
	DebugLogger.Printf("Testing rate limit with save type: %v", saveType)

	if token == "" {
		ErrorLogger.Println("Test request failed: empty token")
		return false
	}

	testUri := fmt.Sprintf("https://oauth.reddit.com/api/%v", saveType)
	req, err := http.NewRequest(http.MethodGet, testUri, nil)
	if err != nil {
		ErrorLogger.Printf("Error creating test request: %v", err)
		return false
	}

	req.Header = http.Header{
		"Authorization": []string{"Bearer " + token},
		"User-Agent":    []string{"Mozilla/5.0 (X11; Linux x86_64; rv:91.0) Gecko/20100101 Firefox/91.0"},
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ErrorLogger.Printf("Error making test request: %v", err)
		return false
	}
	defer resp.Body.Close()

	isRateLimited := resp.StatusCode == 429
	if isRateLimited {
		InfoLogger.Println("Test request indicates rate limit is still active")
	} else {
		InfoLogger.Println("Test request successful - rate limit cleared")
	}
	return !isRateLimited
}
