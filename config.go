package main

import (
	"os"
	"strconv"
	"time"
	// Loggers like InfoLogger, ErrorLogger, DebugLogger are expected to be initialized
	// in another file within the 'main' package (e.g., saved_posts.go or main.go)
	// before loadConfig is called.
)

// getEnvOrDefault retrieves an environment variable or returns a default value.
func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvOrDefaultInt retrieves an integer environment variable or returns a default value.
// It logs an error and uses the default if the environment variable is present but not a valid integer.
func getEnvOrDefaultInt(key string, defaultValue int) int {
	if valueStr, exists := os.LookupEnv(key); exists {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		} else if ErrorLogger != nil {
			ErrorLogger.Printf("Error converting env var %s (value: '%s') to int: %v. Using default %d.", key, valueStr, err, defaultValue)
		}
	}
	return defaultValue
}

// getEnvOrDefaultDuration retrieves a duration environment variable (expected in seconds) or returns a default value.
// The default value should be provided as a time.Duration.
// It logs an error and uses the default if the environment variable is present but not a valid integer (for seconds).
func getEnvOrDefaultDuration(key string, defaultValue time.Duration) time.Duration {
	if valueStr, exists := os.LookupEnv(key); exists {
		// Assuming the duration from env var is in seconds.
		if valueInt, err := strconv.Atoi(valueStr); err == nil {
			return time.Duration(valueInt) * time.Second
		} else if ErrorLogger != nil {
			ErrorLogger.Printf("Error converting env var %s (value: '%s') to duration (seconds): %v. Using default %v.", key, valueStr, err, defaultValue)
		}
	}
	return defaultValue
}

// Global configuration variables
var (
	// General
	UserAgent      string
	RedditOauthURL string
	RedditBaseURL  string // For non-OAuth endpoints like /api/me.json

	// Migration settings for migrate.go
	DefaultSubredditChunkSize int
	MaxSubredditRetryAttempts int
	DefaultPostConcurrency    int
	DefaultAPITimeout         time.Duration // General API client timeout
	TestAPITimeout            time.Duration // Timeout for testRedditAPI in saved_posts.go and similar tests

	// Rate Limiter settings for saved_posts.go
	RateLimitSleepInterval time.Duration // Derived from RATE_LIMIT_SLEEP_INTERVAL_MINUTES
	RateLimitInterval      time.Duration // Derived from RATE_LIMIT_INTERVAL_MINUTES
	MaxTokensPerInterval   int           // MAX_TOKENS_PER_INTERVAL
)

// loadConfig loads configuration from environment variables.
// It should be called once at application startup, after loggers are initialized.
func loadConfig() {
	// Ensure loggers are initialized before this function is called.
	if InfoLogger == nil || DebugLogger == nil || ErrorLogger == nil {
		// This is a fallback if loggers aren't ready, real logging might not happen.
		println("Warning: Loggers not initialized prior to loadConfig(). Configuration loading logs might be incomplete.")
	} else {
		InfoLogger.Println("Loading configuration from environment variables...")
	}

	UserAgent = getEnvOrDefault("USER_AGENT", "GoMigrateClient/1.1 by RedditUser (dev build)")
	RedditOauthURL = getEnvOrDefault("REDDIT_OAUTH_URL", "https://oauth.reddit.com")
	RedditBaseURL = getEnvOrDefault("REDDIT_BASE_URL", "https://www.reddit.com")

	DefaultSubredditChunkSize = getEnvOrDefaultInt("DEFAULT_SUBREDDIT_CHUNK_SIZE", 100)
	MaxSubredditRetryAttempts = getEnvOrDefaultInt("MAX_SUBREDDIT_RETRY_ATTEMPTS", 5)
	DefaultPostConcurrency = getEnvOrDefaultInt("DEFAULT_POST_CONCURRENCY", 10)

	// Durations from env are expected in seconds
	DefaultAPITimeout = getEnvOrDefaultDuration("DEFAULT_API_TIMEOUT_SECONDS", 30*time.Second)
	TestAPITimeout = getEnvOrDefaultDuration("TEST_API_TIMEOUT_SECONDS", 15*time.Second)

	// Rate Limiter settings
	// Store them as time.Duration directly where applicable
	rateLimitSleepSeconds := getEnvOrDefaultInt("RATE_LIMIT_SLEEP_INTERVAL_SECONDS", 30)
	RateLimitSleepInterval = time.Duration(rateLimitSleepSeconds) * time.Minute

	rateLimitIntervalSeconds := getEnvOrDefaultInt("RATE_LIMIT_INTERVAL_SECONDS", 30)
	RateLimitInterval = time.Duration(rateLimitIntervalSeconds) * time.Minute

	MaxTokensPerInterval = getEnvOrDefaultInt("MAX_TOKENS_PER_INTERVAL", 50)

	if DebugLogger != nil {
		DebugLogger.Printf("UserAgent: %s", UserAgent)
		DebugLogger.Printf("RedditOauthURL: %s", RedditOauthURL)
		DebugLogger.Printf("RedditBaseURL: %s", RedditBaseURL)
		DebugLogger.Printf("DefaultSubredditChunkSize: %d", DefaultSubredditChunkSize)
		DebugLogger.Printf("MaxSubredditRetryAttempts: %d", MaxSubredditRetryAttempts)
		DebugLogger.Printf("DefaultPostConcurrency: %d", DefaultPostConcurrency)
		DebugLogger.Printf("DefaultAPITimeout: %v", DefaultAPITimeout)
		DebugLogger.Printf("TestAPITimeout: %v", TestAPITimeout)
		DebugLogger.Printf("RateLimitSleepInterval: %v (from %d minutes)", RateLimitSleepInterval, rateLimitSleepSeconds)
		DebugLogger.Printf("RateLimitInterval: %v (from %d minutes)", RateLimitInterval, rateLimitIntervalSeconds)
		DebugLogger.Printf("MaxTokensPerInterval: %d", MaxTokensPerInterval)
	}

	if InfoLogger != nil {
		InfoLogger.Println("Configuration loaded.")
	}
}
