package api

import (
	"github.com/nileshnk/reddit-migrate/internal/auth"
	"github.com/nileshnk/reddit-migrate/internal/config"
	"github.com/nileshnk/reddit-migrate/internal/migration"

	"github.com/go-chi/chi/v5"
)

// Router sets up the API routes for the application.
// It defines endpoints for authentication, data fetching, and migration operations.
func Router(router chi.Router) {
	// Health check endpoint
	router.Get("/health", HealthCheckHandler)
	config.InfoLogger.Println("Registered /api/health GET endpoint")

	// OAuth endpoints
	router.Get("/oauth/login", auth.OAuthLoginHandler)
	config.InfoLogger.Println("Registered /api/oauth/login GET endpoint")

	router.Get("/oauth/callback", auth.OAuthCallbackHandler)
	config.InfoLogger.Println("Registered /api/oauth/callback GET endpoint")

	// Authentication endpoints
	router.Post("/verify-cookie", auth.VerifyTokenResponse)
	config.InfoLogger.Println("Registered /api/verify-cookie POST endpoint")

	// Data fetching endpoints
	router.Post("/subreddits", SubredditsHandler)
	config.InfoLogger.Println("Registered /api/subreddits POST endpoint")

	router.Post("/saved-posts", SavedPostsHandler)
	config.InfoLogger.Println("Registered /api/saved-posts POST endpoint")

	router.Post("/account-counts", AccountCountsHandler)
	config.InfoLogger.Println("Registered /api/account-counts POST endpoint")

	// Migration endpoints
	router.Post("/migrate", migration.MigrationHandler)
	config.InfoLogger.Println("Registered /api/migrate POST endpoint")

	router.Post("/migrate-custom", CustomMigrationHandler)
	config.InfoLogger.Println("Registered /api/migrate-custom POST endpoint")
}
