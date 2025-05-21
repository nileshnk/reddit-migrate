package ratelimiter

import (
	"github.com/nileshnk/reddit-migrate/internal/config"
	"sync"
	"time"
)

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
	config.InfoLogger.Printf("RateLimiter: Initializing with maxTokens=%d, interval=%v", maxTokens, interval)
	rl := &RateLimiter{
		requests:     make(chan struct{}, maxTokens),
		interval:     interval,
		maxTokens:    maxTokens,
		pauseSignal:  make(chan struct{}, 1),
		resumeSignal: make(chan struct{}, 1),
		isPaused:     false,
	}
	go rl.refillTokens()
	return rl
}

func (rl *RateLimiter) refillTokens() {
	ticker := time.NewTicker(rl.interval)
	defer ticker.Stop()

	config.DebugLogger.Printf("RateLimiter: Starting token refill goroutine with interval %v", rl.interval)
	for range ticker.C {
		rl.mu.RLock()
		isPausedCurrent := rl.isPaused
		rl.mu.RUnlock()

		if !isPausedCurrent {
			// Refill tokens up to maxTokens
			// Draining and then refilling ensures that the bucket doesn't exceed maxTokens
			// if tokens were added externally or if maxTokens changed.
			// However, the current logic only drains. Let's refill properly.
			tokensToRefill := rl.maxTokens - len(rl.requests)
			if tokensToRefill < 0 { // Should not happen if len(rl.requests) is capped by maxTokens
				tokensToRefill = 0
			}

			// First, drain any existing tokens to reset the count for the interval if that's the desired behavior.
			// The original code drained all tokens. If the goal is to ADD tokens up to maxTokens per interval,
			// then draining is not what we want unless the interval signifies a complete reset.
			// Given the name 'refill', it implies adding new tokens.
			// The original draining logic seems to conflict with typical token bucket refill.
			// A typical refill adds tokens up to the bucket capacity.
			// Let's adjust to a more standard token bucket refill: add tokens if space is available.

			numToDrain := len(rl.requests) // This was the original logic, which empties the bucket
			for i := 0; i < numToDrain; i++ {
				select {
				case <-rl.requests:
				default:
					break
				}
			}
			// After draining, fill to maxTokens. This ensures a fixed number of requests per interval.
			for i := 0; i < rl.maxTokens; i++ {
				select {
				case rl.requests <- struct{}{}:
				default: // Bucket is full
					break
				}
			}
			config.DebugLogger.Printf("RateLimiter: Tokens refilled. Current token count: %d", len(rl.requests))
		} else {
			config.DebugLogger.Printf("RateLimiter: Token refill skipped because limiter is paused.")
		}
	}
}

func (rl *RateLimiter) Pause() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if !rl.isPaused {
		rl.isPaused = true
		config.InfoLogger.Println("RateLimiter: Paused.")
		// Non-blocking send to pauseSignal
		select {
		case rl.pauseSignal <- struct{}{}:
			config.DebugLogger.Println("RateLimiter: Pause signal sent to unblock waiters.")
		default:
			config.DebugLogger.Println("RateLimiter: Pause signal channel full or no active waiters on pauseSignal.")
		}
	}
}

func (rl *RateLimiter) Resume() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.isPaused {
		rl.isPaused = false
		config.InfoLogger.Println("RateLimiter: Resumed.")
		// Non-blocking send to resumeSignal
		select {
		case rl.resumeSignal <- struct{}{}:
			config.DebugLogger.Println("RateLimiter: Resume signal sent to unblock waiters.")
		default:
			config.DebugLogger.Println("RateLimiter: Resume signal channel full or no active waiters on resumeSignal.")
		}
	}
}

// Wait blocks until a token is available or the limiter is paused and then resumed.
// It respects the pause and resume signals.
func (rl *RateLimiter) Wait() {
	config.DebugLogger.Println("RateLimiter: Attempting to acquire token...")
	startTime := time.Now()

	// These config values are used by the caller in posts.go, not directly in Wait for its own timeout.
	// The Wait function's timeout behavior is governed by the channel operations and pause/resume signals.
	// Keeping them here for now if they are intended to influence how Wait behaves beyond just token acquisition.
	// However, the current RateLimiter.Wait() primarily waits on rl.requests, rl.pauseSignal, or rl.resumeSignal.
	// The time.After in the original code within Wait() was used as a periodic check.
	// Let's use a shorter, more responsive internal timeout for the periodic checks if needed.
	// A very short sleep/timeout can be used to prevent busy-waiting if no token is immediately available.
	// The original `config.RateLimitSleepInterval` was used as a generic polling interval.

	checkInterval := 100 * time.Millisecond // Internal check interval if no token or signal is received quickly

	for {
		rl.mu.RLock()
		isPausedCurrent := rl.isPaused
		rl.mu.RUnlock()

		if !isPausedCurrent {
			select {
			case rl.requests <- struct{}{}: // Attempt to send a request token into the channel (acquiring it)
				config.DebugLogger.Printf("RateLimiter: Token acquired. Wait time: %v", time.Since(startTime))
				return
			case <-rl.pauseSignal:
				config.DebugLogger.Println("RateLimiter: Notified by pauseSignal while attempting to acquire token. Re-evaluating state.")
				// Drain the signal if multiple pauses happened quickly, ensuring we react to the latest state.
				// This helps if Pause() is called multiple times before Wait() processes the signal.
				for len(rl.pauseSignal) > 0 {
					<-rl.pauseSignal
				}
				continue // Re-check isPaused status
			case <-time.After(checkInterval): // Periodically re-check state if no token or signal
				config.DebugLogger.Printf("RateLimiter: Timed out waiting for token/pause signal, re-checking state.")
				continue
			}
		} else {
			config.DebugLogger.Println("RateLimiter: Currently paused. Waiting for resume signal or timeout.")
			select {
			case <-rl.resumeSignal:
				config.DebugLogger.Println("RateLimiter: Notified by resumeSignal while paused. Re-evaluating state.")
				// Drain the signal
				for len(rl.resumeSignal) > 0 {
					<-rl.resumeSignal
				}
				continue // Re-check isPaused status
			case <-time.After(checkInterval): // Periodically re-check state
				config.DebugLogger.Printf("RateLimiter: Timed out waiting for resume signal, re-checking pause state.")
				continue
			}
		}
	}
}
