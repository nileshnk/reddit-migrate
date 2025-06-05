package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nileshnk/reddit-migrate/internal/config"
	"github.com/nileshnk/reddit-migrate/internal/types"
)

// OAuthConfig holds the Reddit OAuth configuration
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
	UserAgent    string
}

// OAuthToken represents an OAuth token response from Reddit
type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	Scope        string    `json:"scope"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	CreatedAt    time.Time `json:"-"`
}

// IsExpired checks if the token has expired
func (t *OAuthToken) IsExpired() bool {
	return time.Now().After(t.CreatedAt.Add(time.Duration(t.ExpiresIn) * time.Second))
}

// OAuthState stores temporary state for OAuth flow
type OAuthState struct {
	State       string
	CreatedAt   time.Time
	CallbackURL string
}

// Global OAuth configuration (should be initialized from config)
var redditOAuth *OAuthConfig

// Initialize OAuth configuration
func InitOAuth(clientID, clientSecret, redirectURI string) {
	redditOAuth = &OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
		Scopes: []string{
			"identity",     // Access user's identity
			"read",         // Access posts and comments
			"subscribe",    // Manage subreddit subscriptions
			"save",         // Save and unsave posts/comments
			"submit",       // Submit posts and comments
			"vote",         // Vote on posts and comments
			"mysubreddits", // Access user's subreddits
			"history",      // Access user's voting history
		},
		UserAgent: config.UserAgent,
	}
}

// GenerateState creates a secure random state parameter for OAuth
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GetAuthorizationURL returns the Reddit OAuth authorization URL
func GetAuthorizationURL(state string) string {
	if redditOAuth == nil {
		config.ErrorLogger.Println("OAuth not initialized")
		return ""
	}

	params := url.Values{
		"client_id":     {redditOAuth.ClientID},
		"response_type": {"code"},
		"state":         {state},
		"redirect_uri":  {redditOAuth.RedirectURI},
		"duration":      {"permanent"}, // Request refresh token
		"scope":         {strings.Join(redditOAuth.Scopes, " ")},
	}

	return fmt.Sprintf("https://www.reddit.com/api/v1/authorize?%s", params.Encode())
}

// ExchangeCodeForToken exchanges an authorization code for an access token
func ExchangeCodeForToken(code string) (*OAuthToken, error) {
	if redditOAuth == nil {
		return nil, fmt.Errorf("OAuth not initialized")
	}

	data := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {redditOAuth.RedirectURI},
	}

	req, err := http.NewRequest("POST", "https://www.reddit.com/api/v1/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("error creating token request: %w", err)
	}

	req.SetBasicAuth(redditOAuth.ClientID, redditOAuth.ClientSecret)
	req.Header.Set("User-Agent", redditOAuth.UserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	config.DebugLogger.Printf("Exchanging authorization code for token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		config.ErrorLogger.Printf("Token exchange failed with status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	var token OAuthToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("error parsing token response: %w", err)
	}

	token.CreatedAt = time.Now()
	config.InfoLogger.Printf("Successfully exchanged code for token. Expires in %d seconds", token.ExpiresIn)

	return &token, nil
}

// RefreshAccessToken uses a refresh token to get a new access token
func RefreshAccessToken(refreshToken string) (*OAuthToken, error) {
	if redditOAuth == nil {
		return nil, fmt.Errorf("OAuth not initialized")
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}

	req, err := http.NewRequest("POST", "https://www.reddit.com/api/v1/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("error creating refresh request: %w", err)
	}

	req.SetBasicAuth(redditOAuth.ClientID, redditOAuth.ClientSecret)
	req.Header.Set("User-Agent", redditOAuth.UserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	config.DebugLogger.Printf("Refreshing access token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending refresh request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		config.ErrorLogger.Printf("Token refresh failed with status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("token refresh failed with status %d", resp.StatusCode)
	}

	var token OAuthToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("error parsing refresh response: %w", err)
	}

	token.CreatedAt = time.Now()
	token.RefreshToken = refreshToken // Reddit doesn't return a new refresh token
	config.InfoLogger.Printf("Successfully refreshed token. Expires in %d seconds", token.ExpiresIn)

	return &token, nil
}

// GetUserInfoWithToken fetches user information using an OAuth token
func GetUserInfoWithToken(accessToken string) (*types.ProfileResponseType, error) {
	req, err := http.NewRequest("GET", "https://oauth.reddit.com/api/v1/me", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating user info request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("User-Agent", config.UserAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending user info request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading user info response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		config.ErrorLogger.Printf("User info request failed with status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("user info request failed with status %d", resp.StatusCode)
	}

	var userInfo types.ProfileResponseType
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("error parsing user info response: %w", err)
	}

	return &userInfo, nil
}

// OAuthLoginHandler handles the OAuth login initiation
func OAuthLoginHandler(w http.ResponseWriter, r *http.Request) {
	state, err := GenerateState()
	if err != nil {
		config.ErrorLogger.Printf("Error generating OAuth state: %v", err)
		errorResponse(w, "Error initiating OAuth login", http.StatusInternalServerError)
		return
	}

	// Store state in session or temporary storage (implementation depends on your session management)
	// For now, we'll set it as a secure cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10 minutes
	})

	authURL := GetAuthorizationURL(state)
	if authURL == "" {
		errorResponse(w, "OAuth not properly configured", http.StatusInternalServerError)
		return
	}

	config.InfoLogger.Printf("Redirecting user to Reddit OAuth from %s", r.RemoteAddr)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// OAuthCallbackHandler handles the OAuth callback from Reddit
func OAuthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	config.DebugLogger.Printf("Received OAuth callback from %s", r.RemoteAddr)

	// Verify state parameter
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		config.ErrorLogger.Printf("OAuth state cookie not found: %v", err)
		errorResponse(w, "Invalid OAuth state", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" || state != stateCookie.Value {
		config.ErrorLogger.Printf("OAuth state mismatch. Expected: %s, Got: %s", stateCookie.Value, state)
		errorResponse(w, "Invalid OAuth state", http.StatusBadRequest)
		return
	}

	// Clear the state cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	// Check for errors from Reddit
	if errCode := r.URL.Query().Get("error"); errCode != "" {
		config.ErrorLogger.Printf("OAuth error from Reddit: %s", errCode)
		errorResponse(w, fmt.Sprintf("OAuth authorization denied: %s", errCode), http.StatusBadRequest)
		return
	}

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		config.ErrorLogger.Printf("No authorization code in OAuth callback")
		errorResponse(w, "No authorization code received", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	token, err := ExchangeCodeForToken(code)
	if err != nil {
		config.ErrorLogger.Printf("Error exchanging code for token: %v", err)
		errorResponse(w, "Error obtaining access token", http.StatusInternalServerError)
		return
	}

	// Get user information
	userInfo, err := GetUserInfoWithToken(token.AccessToken)
	if err != nil {
		config.ErrorLogger.Printf("Error fetching user info: %v", err)
		errorResponse(w, "Error fetching user information", http.StatusInternalServerError)
		return
	}

	config.InfoLogger.Printf("OAuth login successful for user: %s", userInfo.Data.Name)

	// Return token and user info to client
	response := struct {
		Success bool `json:"success"`
		Data    struct {
			Username     string `json:"username"`
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
		} `json:"data"`
		Message string `json:"message"`
	}{
		Success: true,
		Message: "OAuth authentication successful",
	}
	response.Data.Username = userInfo.Data.Name
	response.Data.AccessToken = token.AccessToken
	response.Data.RefreshToken = token.RefreshToken
	response.Data.ExpiresIn = token.ExpiresIn

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		config.ErrorLogger.Printf("Error encoding OAuth response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// CreateOAuthClient creates an HTTP client that automatically adds OAuth authentication
func CreateOAuthClient(accessToken string) *http.Client {
	return &http.Client{
		Transport: &OAuthTransport{
			Token:     accessToken,
			UserAgent: config.UserAgent,
			Base:      http.DefaultTransport,
		},
	}
}

// OAuthTransport is an http.RoundTripper that adds OAuth headers to requests
type OAuthTransport struct {
	Token     string
	UserAgent string
	Base      http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface
func (t *OAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	req2 := req.Clone(context.Background())
	req2.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.Token))
	req2.Header.Set("User-Agent", t.UserAgent)

	// Update the host for OAuth endpoints
	if strings.Contains(req2.URL.Host, "reddit.com") && !strings.HasPrefix(req2.URL.Host, "oauth.") {
		req2.URL.Host = "oauth.reddit.com"
		req2.URL.Scheme = "https"
	}

	return t.Base.RoundTrip(req2)
}
