package api

import (
	"encoding/json"
	"net/http"

	"github.com/nileshnk/reddit-migrate/internal/config"
)

// ValidateContentType checks if the request has the correct Content-Type
func ValidateContentType(r *http.Request) bool {
	return r.Header.Get("Content-Type") == "application/json"
}

// SendJSONResponse sends a JSON response with the given data
func SendJSONResponse(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(data)
}

// SendErrorResponse sends a structured error response
func SendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}{
		Success: false,
		Message: message,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		config.ErrorLogger.Printf("Failed to encode error response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// DecodeJSONRequest decodes a JSON request body into the provided struct
func DecodeJSONRequest(r *http.Request, dest interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dest)
}
