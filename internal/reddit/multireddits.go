package reddit

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/nileshnk/reddit-migrate/internal/config"
	"github.com/nileshnk/reddit-migrate/internal/types"
)

// multiredditAPIResponse represents the Reddit API response for a single multireddit
type multiredditAPIResponse struct {
	Kind string `json:"kind"`
	Data struct {
		Name        string  `json:"name"`
		DisplayName string  `json:"display_name"`
		Path        string  `json:"path"`
		DescriptionMd string `json:"description_md"`
		IconURL     string  `json:"icon_url"`
		Visibility  string  `json:"visibility"`
		CreatedUTC  float64 `json:"created_utc"`
		NumSubs     int     `json:"num_subscribers"`
		Subreddits  []struct {
			Name string `json:"name"`
		} `json:"subreddits"`
	} `json:"data"`
}

// FetchMultireddits retrieves all multireddits for the authenticated user.
func FetchMultireddits(token string) ([]types.MultiredditInfo, error) {
	apiURL := fmt.Sprintf("%s/api/multi/mine", config.RedditOauthURL)
	config.InfoLogger.Printf("Fetching multireddits from %s", apiURL)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for multireddits: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", config.UserAgent)

	client := http.Client{Timeout: config.DefaultAPITimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching multireddits: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch multireddits, status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading multireddits response: %w", err)
	}

	var apiMultis []multiredditAPIResponse
	if err := json.Unmarshal(bodyBytes, &apiMultis); err != nil {
		return nil, fmt.Errorf("error unmarshalling multireddits response: %w", err)
	}

	var multireddits []types.MultiredditInfo
	for _, m := range apiMultis {
		var subredditNames []string
		for _, s := range m.Data.Subreddits {
			subredditNames = append(subredditNames, s.Name)
		}

		multireddits = append(multireddits, types.MultiredditInfo{
			Name:        m.Data.Name,
			DisplayName: m.Data.DisplayName,
			Path:        m.Data.Path,
			Description: m.Data.DescriptionMd,
			Subreddits:  subredditNames,
			IconURL:     m.Data.IconURL,
			Visibility:  m.Data.Visibility,
			Created:     m.Data.CreatedUTC,
			NumSubs:     m.Data.NumSubs,
		})
	}

	config.InfoLogger.Printf("Fetched %d multireddits", len(multireddits))
	return multireddits, nil
}

// GetMultiredditCount returns the number of multireddits for the authenticated user.
func GetMultiredditCount(token string) (int, error) {
	multis, err := FetchMultireddits(token)
	if err != nil {
		return 0, err
	}
	return len(multis), nil
}

// CreateMultireddit creates a multireddit on the destination account.
// It uses POST /api/multi/user/{username}/m/{multiname} with a model JSON body.
func CreateMultireddit(token, username string, multi types.MultiredditInfo) error {
	multiPath := fmt.Sprintf("/user/%s/m/%s", username, multi.Name)
	apiURL := fmt.Sprintf("%s/api/multi%s", config.RedditOauthURL, multiPath)

	// Build the model JSON
	type subEntry struct {
		Name string `json:"name"`
	}
	type multiModel struct {
		DisplayName   string     `json:"display_name"`
		DescriptionMd string     `json:"description_md"`
		Subreddits    []subEntry `json:"subreddits"`
		Visibility    string     `json:"visibility"`
	}

	var subs []subEntry
	for _, s := range multi.Subreddits {
		subs = append(subs, subEntry{Name: s})
	}

	model := multiModel{
		DisplayName:   multi.DisplayName,
		DescriptionMd: multi.Description,
		Subreddits:    subs,
		Visibility:    multi.Visibility,
	}

	modelJSON, err := json.Marshal(model)
	if err != nil {
		return fmt.Errorf("error marshalling multireddit model: %w", err)
	}

	formData := url.Values{}
	formData.Set("model", string(modelJSON))

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("error creating multireddit request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", config.UserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := http.Client{Timeout: config.DefaultAPITimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error creating multireddit: %w", err)
	}
	defer resp.Body.Close()

	// 200 = updated existing, 201 = created new — both are success
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create multireddit %s, status: %d, body: %s", multi.Name, resp.StatusCode, string(bodyBytes))
	}

	config.InfoLogger.Printf("Successfully created multireddit %s for user %s", multi.Name, username)
	return nil
}

// ManageMultireddits migrates multireddits from source to destination account.
func ManageMultireddits(token, username string, multireddits []types.MultiredditInfo) types.ManageMultiredditResponseType {
	config.InfoLogger.Printf("ManageMultireddits: Creating %d multireddits for user %s", len(multireddits), username)

	if len(multireddits) == 0 {
		return types.ManageMultiredditResponseType{SuccessCount: 0, FailedCount: 0}
	}

	successCount := 0
	failedCount := 0
	var failedMultis []string

	for _, multi := range multireddits {
		err := CreateMultireddit(token, username, multi)
		if err != nil {
			config.ErrorLogger.Printf("ManageMultireddits: Failed to create %s: %v", multi.Name, err)
			failedCount++
			failedMultis = append(failedMultis, multi.Name)
		} else {
			successCount++
		}
	}

	config.InfoLogger.Printf("ManageMultireddits: Finished. Success: %d, Failed: %d", successCount, failedCount)
	return types.ManageMultiredditResponseType{
		SuccessCount: successCount,
		FailedCount:  failedCount,
		FailedMultis: failedMultis,
	}
}
