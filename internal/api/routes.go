package api

import (
	"encoding/json"
	"github.com/nileshnk/reddit-migrate/internal/config"
	"github.com/nileshnk/reddit-migrate/internal/migration"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Router sets up the API routes for the application.
// It defines endpoints for testing, cookie verification, and data migration.
func Router(router chi.Router) {

	// Test endpoint to check if the API is responsive.
	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		config.DebugLogger.Printf("Received request for /api/test from %s", r.RemoteAddr)
		type testData struct {
			Hello string `json:"hello"`
		}
		response := testData{Hello: "world!"}

		w.Header().Set("Content-Type", "application/json")
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			config.ErrorLogger.Printf("Error marshalling test response: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		_, err = w.Write(jsonResponse)
		if err != nil {
			config.ErrorLogger.Printf("Error writing test response: %v", err)
		}
		config.DebugLogger.Printf("Successfully responded to /api/test request from %s", r.RemoteAddr)
	})

	// Endpoint to verify the validity of a Reddit account cookie.
	router.Post("/verify-cookie", migration.VerifyTokenResponse) // TODO: verifyTokenResponse needs to be defined or imported
	config.InfoLogger.Println("Registered /api/verify-cookie POST endpoint")

	// Endpoint to handle the data migration process between Reddit accounts.
	router.Post("/migrate", migration.MigrationHandler) // TODO: MigrationHandler needs to be defined or imported
	config.InfoLogger.Println("Registered /api/migrate POST endpoint")
}
