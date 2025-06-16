package config

import (
	"fmt"
	"log" // Using standard log for now, actual logger injection TBD
	"os"
	"strconv"
	"strings"
	"time"
)

// TODO: Replace with a proper logging solution that can be injected or globally accessed.
// For now, using a simplified placeholder.
var ErrorLogger *log.Logger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lmicroseconds)
var InfoLogger *log.Logger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lmicroseconds)
var DebugLogger *log.Logger = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lmicroseconds)

const DefaultAddress = "localhost:5005"

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

// getServerAddress determines the server address based on environment variables,
// command-line arguments, or a default value.
func GetServerAddress() string {
	// Priority 1: Environment variable GO_ADDR.
	if addr := os.Getenv("GO_ADDR"); addr != "" {
		InfoLogger.Printf("Using address from GO_ADDR environment variable: %s", addr)
		return addr
	}

	// Priority 2: Command-line argument --addr.
	for _, arg := range os.Args[1:] { // Skip the program name.
		if strings.HasPrefix(arg, "--addr=") {
			addr := strings.TrimPrefix(arg, "--addr=")
			if addr != "" {
				InfoLogger.Printf("Using address from --addr command-line argument: %s", addr)
				return addr
			}
		}
	}

	// Priority 3: Default address.
	InfoLogger.Printf("Using default address: %s", DefaultAddress)
	return DefaultAddress
}

// Global configuration variables
var (
	// General
	UserAgent              string
	RedditOauthURL         string
	RedditBaseURL          string // For non-OAuth endpoints like /api/me.json
	ServerAddress          string
	RedditOauthRedirectUri string

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

// LoadConfig loads configuration from environment variables.
// It should be called once at application startup, after loggers are initialized.
func LoadConfig() {
	// Ensure loggers are initialized before this function is called.
	if InfoLogger == nil || DebugLogger == nil || ErrorLogger == nil {
		// This is a fallback if loggers aren't ready, real logging might not happen.
		println("Warning: Loggers not initialized prior to LoadConfig(). Configuration loading logs might be incomplete.")
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
	RateLimitSleepInterval = time.Duration(rateLimitSleepSeconds) * time.Second // Note: Original code had time.Minute here, might be error. Assuming seconds as per var name.

	rateLimitIntervalSeconds := getEnvOrDefaultInt("RATE_LIMIT_INTERVAL_SECONDS", 30)
	RateLimitInterval = time.Duration(rateLimitIntervalSeconds) * time.Second // Note: Original code had time.Minute here, might be error. Assuming seconds as per var name.

	MaxTokensPerInterval = getEnvOrDefaultInt("MAX_TOKENS_PER_INTERVAL", 50)
	ServerAddress = GetServerAddress()
	RedditOauthRedirectUri = fmt.Sprintf("http://%s/api/oauth/callback", ServerAddress)

	if DebugLogger != nil {
		DebugLogger.Printf("UserAgent: %s", UserAgent)
		DebugLogger.Printf("RedditOauthURL: %s", RedditOauthURL)
		DebugLogger.Printf("RedditBaseURL: %s", RedditBaseURL)
		DebugLogger.Printf("DefaultSubredditChunkSize: %d", DefaultSubredditChunkSize)
		DebugLogger.Printf("MaxSubredditRetryAttempts: %d", MaxSubredditRetryAttempts)
		DebugLogger.Printf("DefaultPostConcurrency: %d", DefaultPostConcurrency)
		DebugLogger.Printf("DefaultAPITimeout: %v", DefaultAPITimeout)
		DebugLogger.Printf("TestAPITimeout: %v", TestAPITimeout)
		// Corrected logging for duration: originally RATE_LIMIT_SLEEP_INTERVAL_SECONDS was multiplied by time.Minute
		DebugLogger.Printf("RateLimitSleepInterval: %v (from %d seconds)", RateLimitSleepInterval, rateLimitSleepSeconds)
		DebugLogger.Printf("RateLimitInterval: %v (from %d seconds)", RateLimitInterval, rateLimitIntervalSeconds)
		DebugLogger.Printf("MaxTokensPerInterval: %d", MaxTokensPerInterval)
	}

	if InfoLogger != nil {
		InfoLogger.Println("Configuration loaded.")
	}
}
