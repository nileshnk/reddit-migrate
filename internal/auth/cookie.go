package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/nileshnk/reddit-migrate/internal/config"
	"github.com/nileshnk/reddit-migrate/internal/types"
)

// VerifyTokenResponse is the HTTP handler for the /verify-cookie endpoint.
// It validates a Reddit cookie and returns the associated username if valid.
func VerifyTokenResponse(w http.ResponseWriter, r *http.Request) {
	config.DebugLogger.Printf("Received /verify-cookie request from %s", r.RemoteAddr)

	if r.Header.Get("Content-Type") != "application/json" {
		config.ErrorLogger.Printf("Invalid content type for /verify-cookie from %s: %s", r.RemoteAddr, r.Header.Get("Content-Type"))
		errorResponse(w, "Content Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	var requestBody types.VerifyCookieType
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&requestBody)
	if err != nil {
		config.ErrorLogger.Printf("Error decoding /verify-cookie request from %s: %v", r.RemoteAddr, err)
		var unmarshalErr *json.UnmarshalTypeError
		if errors.As(err, &unmarshalErr) {
			errorResponse(w, "Bad Request. Wrong Type provided for field "+unmarshalErr.Field, http.StatusBadRequest)
		} else {
			errorResponse(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
		}
		return
	}
	config.InfoLogger.Printf("Verifying cookie for %s (ends with ...%s)", r.RemoteAddr, SafeSuffix(requestBody.Cookie, 6))

	finalResponse := VerifyCookieAndGetResponse(requestBody.Cookie)

	w.Header().Set("Content-Type", "application/json")
	jsonResp, err := json.Marshal(finalResponse)
	if err != nil {
		config.ErrorLogger.Printf("Error marshalling /verify-cookie response for %s: %v", r.RemoteAddr, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(jsonResp); err != nil {
		config.ErrorLogger.Printf("Error writing /verify-cookie response for %s: %v", r.RemoteAddr, err)
	} else {
		config.DebugLogger.Printf("Successfully sent /verify-cookie response to %s. Success: %t, User: %s",
			r.RemoteAddr, finalResponse.Success, finalResponse.Data.Username)
	}
}

// VerifyCookieAndGetResponse takes a cookie string, calls Reddit's API to verify it, and returns a structured response.
func VerifyCookieAndGetResponse(cookieStr string) types.TokenResponseType {
	var finalResponse types.TokenResponseType

	// Make request to Reddit's /api/me.json
	req, err := http.NewRequest(http.MethodGet, "https://www.reddit.com/api/me.json", nil)
	if err != nil {
		config.ErrorLogger.Printf("Error creating request for /api/me.json: %v", err)
		finalResponse.Success = false
		finalResponse.Message = "Internal error creating request to verify cookie."
		return finalResponse
	}

	req.Header = http.Header{
		"Cookie":     {cookieStr},
		"User-Agent": {config.UserAgent},
	}

	config.DebugLogger.Printf("Sending request to /api/me.json to verify cookie (ends ...%s)", SafeSuffix(cookieStr, 6))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		config.ErrorLogger.Printf("Error sending request to /api/me.json: %v", err)
		finalResponse.Success = false
		finalResponse.Message = "Error contacting Reddit to verify cookie."
		return finalResponse
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		config.ErrorLogger.Printf("Error reading response body from /api/me.json (status %d): %v", resp.StatusCode, err)
		finalResponse.Success = false
		finalResponse.Message = "Error reading Reddit's response."
		return finalResponse
	}
	config.DebugLogger.Printf("/api/me.json response (Status: %d): %s", resp.StatusCode, string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		config.ErrorLogger.Printf("Cookie verification failed. /api/me.json status: %d. Body: %s", resp.StatusCode, string(bodyBytes))
		var errorRespData types.ErrorResponseType
		if err := json.Unmarshal(bodyBytes, &errorRespData); err == nil && errorRespData.Message != "" {
			finalResponse.Message = fmt.Sprintf("Invalid Token/Cookie: %s", errorRespData.Message)
			finalResponse.Data.Username = errorRespData.Message
		} else {
			finalResponse.Message = fmt.Sprintf("Invalid Token/Cookie (status %d)", resp.StatusCode)
			finalResponse.Data.Username = "Unknown error"
		}
		finalResponse.Success = false
		return finalResponse
	}

	var profile types.ProfileResponseType
	if err := json.Unmarshal(bodyBytes, &profile); err != nil {
		config.ErrorLogger.Printf("Error unmarshalling /api/me.json response: %v. Body: %s", err, string(bodyBytes))
		finalResponse.Success = false
		finalResponse.Message = "Error parsing Reddit's response."
		return finalResponse
	}

	if profile.Data.Name == "" {
		config.ErrorLogger.Printf("Cookie verified (status 200) but no username found in /api/me.json response. Body: %s", string(bodyBytes))
		finalResponse.Success = false
		finalResponse.Message = "Cookie seems valid, but username could not be retrieved."
		return finalResponse
	}

	config.InfoLogger.Printf("Cookie successfully verified for username: %s", profile.Data.Name)
	finalResponse.Success = true
	finalResponse.Message = "Valid Token/Cookie"
	finalResponse.Data.Username = profile.Data.Name
	return finalResponse
}

// GetUsernameFromCookie verifies a cookie and returns the username or an error.
// This is a helper for internal use within the migration logic.
func GetUsernameFromCookie(cookieStr string) (string, error) {
	response := VerifyCookieAndGetResponse(cookieStr)
	if !response.Success {
		return "", fmt.Errorf("cookie verification failed: %s", response.Message)
	}
	if response.Data.Username == "" {
		return "", errors.New("cookie verified but username is empty")
	}
	return response.Data.Username, nil
}

// ParseTokenFromCookie extracts the 'token_v2' value from a full cookie string.
// Returns an empty string if 'token_v2' is not found or the cookie format is unexpected.
func ParseTokenFromCookie(cookie string) string {
	parts := strings.Split(cookie, ";")
	for _, part := range parts {
		trimmedPart := strings.TrimSpace(part)
		if strings.HasPrefix(trimmedPart, "token_v2=") {
			tokenPair := strings.SplitN(trimmedPart, "=", 2)
			if len(tokenPair) == 2 && tokenPair[1] != "" {
				config.DebugLogger.Printf("Successfully parsed token_v2 from cookie (value ends ...%s)", SafeSuffix(tokenPair[1], 6))
				return tokenPair[1]
			}
			config.ErrorLogger.Printf("Found 'token_v2=' but failed to parse value from part: '%s'", trimmedPart)
			return ""
		}
	}
	config.DebugLogger.Printf("Could not find 'token_v2=' in cookie string: ...%s", SafeSuffix(cookie, 20))
	return ""
}

// errorResponse sends a JSON error message to the client with a given HTTP status code.
func errorResponse(w http.ResponseWriter, message string, httpStatusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusCode)
	resp := make(map[string]string)
	resp["message"] = message
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		config.ErrorLogger.Printf("Critical: Failed to marshal error response object: %v. Original message: %s", err, message)
		http.Error(w, `{"message":"Error generating error response"}`, http.StatusInternalServerError)
		return
	}
	if _, writeErr := w.Write(jsonResp); writeErr != nil {
		config.ErrorLogger.Printf("Failed to write error response to client: %v. Original message: %s, Status: %d", writeErr, message, httpStatusCode)
	}
}

// SafeSuffix returns the last N characters of a string, or the whole string if shorter than N.
// Useful for logging sensitive data like tokens without exposing the full value.
func SafeSuffix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
