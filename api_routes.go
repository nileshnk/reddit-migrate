package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// apiRouter sets up the API routes for the application.
// It defines endpoints for testing, cookie verification, and data migration.
func apiRouter(router chi.Router) {
	InfoLogger.Println("Initializing API routes")

	// Test endpoint to check if the API is responsive.
	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		DebugLogger.Printf("Received request for /api/test from %s", r.RemoteAddr)
		type testData struct {
			Hello string `json:"hello"`
		}
		response := testData{Hello: "world!"}

		w.Header().Set("Content-Type", "application/json")
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			ErrorLogger.Printf("Error marshalling test response: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		_, err = w.Write(jsonResponse)
		if err != nil {
			ErrorLogger.Printf("Error writing test response: %v", err)
		}
		DebugLogger.Printf("Successfully responded to /api/test request from %s", r.RemoteAddr)
	})

	// Endpoint to verify the validity of a Reddit account cookie.
	router.Post("/verify-cookie", verifyTokenResponse)
	InfoLogger.Println("Registered /api/verify-cookie POST endpoint")

	// Endpoint to handle the data migration process between Reddit accounts.
	router.Post("/migrate", MigrationHandler)
	InfoLogger.Println("Registered /api/migrate POST endpoint")
}
