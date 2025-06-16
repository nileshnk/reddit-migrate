package types

// MigrationRequestType defines the structure for the migration request body.
// It includes authentication data for old and new accounts, and user preferences for migration.
type MigrationRequestType struct {
	AuthMethod         string          `json:"auth_method,omitempty"`          // "cookie" or "oauth"
	OldAccountCookie   string          `json:"old_account_cookie,omitempty"`   // For cookie-based auth
	NewAccountCookie   string          `json:"new_account_cookie,omitempty"`   // For cookie-based auth
	OldAccountToken    string          `json:"old_account_token,omitempty"`    // For OAuth-based auth
	NewAccountToken    string          `json:"new_account_token,omitempty"`    // For OAuth-based auth
	OldAccountUsername string          `json:"old_account_username,omitempty"` // For OAuth-based auth
	NewAccountUsername string          `json:"new_account_username,omitempty"` // For OAuth-based auth
	Preferences        PreferencesType `json:"preferences"`
}

// PreferencesType defines the user's choices for the migration process.
// Each boolean field indicates whether a specific migration or deletion action should be performed.
type PreferencesType struct {
	MigrateSubredditBool bool `json:"migrate_subreddit_bool"`
	MigratePostBool      bool `json:"migrate_post_bool"`
	DeletePostBool       bool `json:"delete_post_bool"`
	DeleteSubredditBool  bool `json:"delete_subreddit_bool"`
}

// MigrationResponseType defines the structure of the response sent after a migration attempt.
// It includes success status, a message, and detailed data about the performed actions.
type MigrationResponseType struct {
	Success bool             `json:"success"`
	Message string           `json:"message"`
	Data    MigrationDetails `json:"data"`
}

// MigrationDetails holds the detailed results of migration operations.
// This structure is embedded within MigrationResponseType.
type MigrationDetails struct {
	SubscribeSubreddit   ManageSubredditResponseType `json:"subscribeSubreddit"`
	UnsubscribeSubreddit ManageSubredditResponseType `json:"unsubscribeSubreddit"`
	SavePost             ManagePostResponseType      `json:"savePost"`
	UnsavePost           ManagePostResponseType      `json:"unsavePost"`
}

// SubredditActionType defines the action to be performed on a subreddit (subscribe or unsubscribe).
type SubredditActionType string

const (
	// SubscribeAction indicates an action to subscribe to a subreddit.
	SubscribeAction SubredditActionType = "sub"
	// UnsubscribeAction indicates an action to unsubscribe from a subreddit.
	UnsubscribeAction SubredditActionType = "unsub"
)

// ManageSubredditResponseType defines the structure for the response of managing subreddits.
// It includes error status, HTTP status code, counts of successful and failed operations, and a list of failed subreddits.
type ManageSubredditResponseType struct {
	Error            bool
	StatusCode       int
	SuccessCount     int
	FailedCount      int
	FailedSubreddits []string
}

// PostActionType defines the action to be performed on a post (save or unsave).
type PostActionType string

const (
	// SaveAction indicates an action to save a post.
	SaveAction PostActionType = "save"
	// UnsaveAction indicates an action to unsave a post.
	UnsaveAction PostActionType = "unsave"
)

// ManagePostResponseType defines the structure for the response of managing posts.
// It includes counts of successful and failed operations.
type ManagePostResponseType struct {
	SuccessCount int
	FailedCount  int
}

// RedditNameType holds lists of subreddit and user display names and full names.
// This is used internally to pass around collections of names fetched from Reddit.
type RedditNameType struct {
	FullNamesList       []string
	DisplayNamesList    []string
	UserDisplayNameList []string
}

// FullNameListType defines the structure for a list of items (subreddits or posts) from Reddit API.
// It's used for unmarshalling JSON responses that contain a list of children objects.
type FullNameListType struct {
	Kind string `json:"kind"`
	Data struct {
		After    string          `json:"after"`
		Children []FullListChild `json:"children"`
	} `json:"data"`
}

type FullListChild struct {
	Kind string `json:"kind"`
	Data struct {
		Name          string `json:"name"`
		DisplayName   string `json:"display_name"`   // Corrected json tag from Display_name
		SubredditType string `json:"subreddit_type"` // Corrected json tag from Subreddit_type
	} `json:"data"`
}

// VerifyCookieType defines the structure for the request body when verifying a cookie.
// It contains the cookie string to be verified.
type VerifyCookieType struct {
	Cookie string `json:"cookie"`
}

// ProfileResponseType defines the structure for the response from Reddit's /api/v1/me endpoint.
// It contains basic profile information of the authenticated user.
type ProfileResponseType struct {
	Data struct {
		Name       string `json:"name"`
		IsEmployee bool   `json:"is_employee"`
		IsFriend   bool   `json:"is_friend"`
	} `json:"data"`
}

// TokenResponseType defines the structure for the response when verifying a token/cookie.
// It indicates success, a message, and the username associated with the token/cookie if valid.
type TokenResponseType struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Username string `json:"username"`
	} `json:"data"`
}

// ErrorResponseType defines a generic error response structure from the Reddit API.
// It usually contains an error code and a descriptive message.
type ErrorResponseType struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// New types for enhanced selection feature

// PostImageData holds image/media information for a Reddit post
type PostImageData struct {
	ThumbnailURL string `json:"thumbnail_url"`
	PreviewURL   string `json:"preview_url"`
	HighResURL   string `json:"high_res_url"`
	MediaType    string `json:"media_type"` // "image", "video", "link", "text", "gallery"
	Width        int    `json:"width"`
	Height       int    `json:"height"`
}

// SavedPostInfo contains detailed information about a saved post for UI display
type SavedPostInfo struct {
	ID          string        `json:"id"`        // Reddit post ID (without t3_ prefix)
	FullName    string        `json:"full_name"` // Full Reddit name (t3_xxxxx)
	Title       string        `json:"title"`
	Subreddit   string        `json:"subreddit"`
	Author      string        `json:"author"`
	URL         string        `json:"url"`
	Permalink   string        `json:"permalink"`
	Created     int64         `json:"created_utc"`
	Score       int           `json:"score"`
	NumComments int           `json:"num_comments"`
	PostHint    string        `json:"post_hint"` // "image", "link", "self", etc.
	Domain      string        `json:"domain"`
	SelfText    string        `json:"selftext"` // For text posts
	IsVideo     bool          `json:"is_video"`
	IsSelf      bool          `json:"is_self"` // True for text posts
	NSFW        bool          `json:"over_18"`
	Spoiler     bool          `json:"spoiler"`
	ImageData   PostImageData `json:"image_data"`
}

// SubredditInfo contains detailed information about a subreddit for UI display
type SubredditInfo struct {
	Name          string `json:"name"`         // Full name (t5_xxxxx)
	DisplayName   string `json:"display_name"` // r/subredditname
	Title         string `json:"title"`
	Description   string `json:"public_description"`
	Subscribers   int    `json:"subscribers"`
	IconURL       string `json:"icon_img"`
	BannerURL     string `json:"banner_img"`
	PrimaryColor  string `json:"primary_color"`
	KeyColor      string `json:"key_color"`
	SubredditType string `json:"subreddit_type"` // "public", "private", "restricted"
	NSFW          bool   `json:"over18"`
	Created       int64  `json:"created_utc"`
}

// GetSavedPostsRequest defines the request structure for fetching saved posts with details
type GetSavedPostsRequest struct {
	AuthMethod  string `json:"auth_method,omitempty"`  // "cookie" or "oauth"
	Cookie      string `json:"cookie,omitempty"`       // For cookie-based auth
	AccessToken string `json:"access_token,omitempty"` // For OAuth-based auth
	Username    string `json:"username,omitempty"`     // For OAuth-based auth
}

// GetSavedPostsResponse defines the response structure for saved posts with full details
type GetSavedPostsResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Posts   []SavedPostInfo `json:"posts"`
	Count   int             `json:"count"`
}

// GetSubredditsRequest defines the request structure for fetching subreddits with details
type GetSubredditsRequest struct {
	AuthMethod  string `json:"auth_method,omitempty"`  // "cookie" or "oauth"
	Cookie      string `json:"cookie,omitempty"`       // For cookie-based auth
	AccessToken string `json:"access_token,omitempty"` // For OAuth-based auth
	Username    string `json:"username,omitempty"`     // For OAuth-based auth
}

// GetSubredditsResponse defines the response structure for subreddits with full details
type GetSubredditsResponse struct {
	Success    bool            `json:"success"`
	Message    string          `json:"message"`
	Subreddits []SubredditInfo `json:"subreddits"`
	Count      int             `json:"count"`
}

// CustomMigrationRequest defines the structure for custom selection migration
type CustomMigrationRequest struct {
	AuthMethod          string   `json:"auth_method,omitempty"`          // "cookie" or "oauth"
	OldAccountCookie    string   `json:"old_account_cookie,omitempty"`   // For cookie-based auth
	NewAccountCookie    string   `json:"new_account_cookie,omitempty"`   // For cookie-based auth
	OldAccountToken     string   `json:"old_account_token,omitempty"`    // For OAuth-based auth
	NewAccountToken     string   `json:"new_account_token,omitempty"`    // For OAuth-based auth
	OldAccountUsername  string   `json:"old_account_username,omitempty"` // For OAuth-based auth
	NewAccountUsername  string   `json:"new_account_username,omitempty"` // For OAuth-based auth
	SelectedSubreddits  []string `json:"selected_subreddits"`            // List of display names
	SelectedPosts       []string `json:"selected_posts"`                 // List of full names (t3_xxxxx)
	DeleteOldSubreddits bool     `json:"delete_old_subreddits"`
	DeleteOldPosts      bool     `json:"delete_old_posts"`
}

// DetailedPostData represents the full Reddit post data structure for parsing API responses
type DetailedPostData struct {
	Kind string `json:"kind"`
	Data struct {
		ID                    string  `json:"id"`
		Name                  string  `json:"name"`
		Title                 string  `json:"title"`
		Subreddit             string  `json:"subreddit"`
		SubredditNamePrefixed string  `json:"subreddit_name_prefixed"`
		Author                string  `json:"author"`
		URL                   string  `json:"url"`
		Permalink             string  `json:"permalink"`
		CreatedUTC            float64 `json:"created_utc"`
		Score                 int     `json:"score"`
		NumComments           int     `json:"num_comments"`
		PostHint              string  `json:"post_hint"`
		Domain                string  `json:"domain"`
		SelfText              string  `json:"selftext"`
		IsVideo               bool    `json:"is_video"`
		IsSelf                bool    `json:"is_self"`
		Over18                bool    `json:"over_18"`
		Spoiler               bool    `json:"spoiler"`
		Thumbnail             string  `json:"thumbnail"`
		ThumbnailWidth        int     `json:"thumbnail_width"`
		ThumbnailHeight       int     `json:"thumbnail_height"`

		// Preview data for images
		Preview struct {
			Images []struct {
				Source struct {
					URL    string `json:"url"`
					Width  int    `json:"width"`
					Height int    `json:"height"`
				} `json:"source"`
				Resolutions []struct {
					URL    string `json:"url"`
					Width  int    `json:"width"`
					Height int    `json:"height"`
				} `json:"resolutions"`
			} `json:"images"`
			Enabled bool `json:"enabled"`
		} `json:"preview"`

		// Media data for videos/gifs
		Media struct {
			Type   string `json:"type"`
			Height int    `json:"height"`
			Width  int    `json:"width"`
		} `json:"media"`

		// Gallery data for image galleries
		IsGallery     bool `json:"is_gallery"`
		MediaMetadata map[string]struct {
			Status string `json:"status"`
			E      string `json:"e"` // "Image" for images
			M      string `json:"m"` // MIME type
			S      struct {
				Y int    `json:"y"` // height
				X int    `json:"x"` // width
				U string `json:"u"` // URL
			} `json:"s"`
		} `json:"media_metadata"`

		GalleryData struct {
			Items []struct {
				MediaID string `json:"media_id"`
				ID      int    `json:"id"`
			} `json:"items"`
		} `json:"gallery_data"`
	} `json:"data"`
}

// DetailedSubredditData represents the full Reddit subreddit data structure
type DetailedSubredditData struct {
	Kind string `json:"kind"`
	Data struct {
		Name                string  `json:"name"`
		DisplayName         string  `json:"display_name"`
		DisplayNamePrefixed string  `json:"display_name_prefixed"`
		Title               string  `json:"title"`
		PublicDescription   string  `json:"public_description"`
		Description         string  `json:"description"`
		Subscribers         int     `json:"subscribers"`
		IconImg             string  `json:"icon_img"`
		BannerImg           string  `json:"banner_img"`
		PrimaryColor        string  `json:"primary_color"`
		KeyColor            string  `json:"key_color"`
		SubredditType       string  `json:"subreddit_type"`
		Over18              bool    `json:"over_18"`
		CreatedUTC          float64 `json:"created_utc"`
		URL                 string  `json:"url"`

		// Community icon data
		CommunityIcon string `json:"community_icon"`
		IconSize      []int  `json:"icon_size"`

		// Header image data
		HeaderImg   string `json:"header_img"`
		HeaderSize  []int  `json:"header_size"`
		HeaderTitle string `json:"header_title"`
	} `json:"data"`
}

// AccountCountsRequest defines the request structure for getting account counts
type AccountCountsRequest struct {
	AuthMethod  string `json:"auth_method,omitempty"`  // "cookie" or "oauth"
	Cookie      string `json:"cookie,omitempty"`       // For cookie-based auth
	AccessToken string `json:"access_token,omitempty"` // For OAuth-based auth
	Username    string `json:"username,omitempty"`     // For OAuth-based auth
}

// AccountCountsResponse defines the response structure for account counts
type AccountCountsResponse struct {
	Success         bool   `json:"success"`
	Message         string `json:"message"`
	Username        string `json:"username"`
	SubredditCount  int    `json:"subreddit_count"`
	SavedPostsCount int    `json:"saved_posts_count"`
}
