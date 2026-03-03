package types

// MigrationRequestType defines the structure for the migration request body.
// It includes authentication data for source and destination accounts, and user preferences for migration.
type MigrationRequestType struct {
	AuthMethod            string          `json:"auth_method,omitempty"`             // "cookie" or "oauth"
	SourceAccountCookie   string          `json:"source_account_cookie,omitempty"`   // For cookie-based auth
	DestAccountCookie     string          `json:"dest_account_cookie,omitempty"`     // For cookie-based auth
	SourceAccountToken    string          `json:"source_account_token,omitempty"`    // For OAuth-based auth
	DestAccountToken      string          `json:"dest_account_token,omitempty"`      // For OAuth-based auth
	SourceAccountUsername string          `json:"source_account_username,omitempty"` // For OAuth-based auth
	DestAccountUsername   string          `json:"dest_account_username,omitempty"`   // For OAuth-based auth
	Preferences           PreferencesType `json:"preferences"`
}

// PreferencesType defines the user's choices for the migration process.
// Each boolean field indicates whether a specific migration or deletion action should be performed.
type PreferencesType struct {
	MigrateSubredditBool   bool `json:"migrate_subreddit_bool"`
	MigratePostBool        bool `json:"migrate_post_bool"`
	MigrateCommentBool     bool `json:"migrate_comment_bool"`
	MigrateMultiredditBool bool `json:"migrate_multireddit_bool"`
	DeletePostBool         bool `json:"delete_post_bool"`
	DeleteCommentBool      bool `json:"delete_comment_bool"`
	DeleteSubredditBool    bool `json:"delete_subreddit_bool"`
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
	SubscribeSubreddit   ManageSubredditResponseType   `json:"subscribeSubreddit"`
	UnsubscribeSubreddit ManageSubredditResponseType   `json:"unsubscribeSubreddit"`
	SavePost             ManagePostResponseType        `json:"savePost"`
	UnsavePost           ManagePostResponseType        `json:"unsavePost"`
	SaveComment          ManagePostResponseType        `json:"saveComment"`
	UnsaveComment        ManagePostResponseType        `json:"unsaveComment"`
	CreateMultireddit    ManageMultiredditResponseType `json:"createMultireddit"`
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
	AuthMethod             string   `json:"auth_method,omitempty"`             // "cookie" or "oauth"
	SourceAccountCookie    string   `json:"source_account_cookie,omitempty"`   // For cookie-based auth
	DestAccountCookie      string   `json:"dest_account_cookie,omitempty"`     // For cookie-based auth
	SourceAccountToken     string   `json:"source_account_token,omitempty"`    // For OAuth-based auth
	DestAccountToken       string   `json:"dest_account_token,omitempty"`      // For OAuth-based auth
	SourceAccountUsername  string   `json:"source_account_username,omitempty"` // For OAuth-based auth
	DestAccountUsername    string   `json:"dest_account_username,omitempty"`   // For OAuth-based auth
	SelectedSubreddits     []string `json:"selected_subreddits"`               // List of display names
	SelectedPosts          []string `json:"selected_posts"`                    // List of full names (t3_xxxxx)
	SelectedComments       []string `json:"selected_comments"`                 // List of full names (t1_xxxxx)
	SelectedMultireddits   []string `json:"selected_multireddits"`             // List of multireddit names
	DeleteSourceSubreddits bool     `json:"delete_source_subreddits"`
	DeleteSourcePosts      bool     `json:"delete_source_posts"`
	DeleteSourceComments   bool     `json:"delete_source_comments"`
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
	Success            bool   `json:"success"`
	Message            string `json:"message"`
	Username           string `json:"username"`
	SubredditCount     int    `json:"subreddit_count"`
	SavedPostsCount    int    `json:"saved_posts_count"`
	SavedCommentsCount int    `json:"saved_comments_count"`
	MultiredditCount   int    `json:"multireddit_count"`
}

// SavedCommentInfo contains detailed information about a saved comment for UI display
type SavedCommentInfo struct {
	ID        string `json:"id"`        // Reddit comment ID (without t1_ prefix)
	FullName  string `json:"full_name"` // Full Reddit name (t1_xxxxx)
	Body      string `json:"body"`      // Comment text (HTML)
	BodyText  string `json:"body_text"` // Plain text snippet
	Author    string `json:"author"`
	Subreddit string `json:"subreddit"`
	Score     int    `json:"score"`
	Created   int64  `json:"created_utc"`
	Permalink string `json:"permalink"`
	LinkTitle string `json:"link_title"` // Title of parent post
	LinkURL   string `json:"link_url"`   // URL of parent post
	NSFW      bool   `json:"over_18"`
}

// DetailedCommentData represents the full Reddit comment data structure for parsing API responses
type DetailedCommentData struct {
	Kind string `json:"kind"`
	Data struct {
		ID                    string  `json:"id"`
		Name                  string  `json:"name"`
		Body                  string  `json:"body"`
		BodyHTML              string  `json:"body_html"`
		Author                string  `json:"author"`
		Subreddit             string  `json:"subreddit"`
		SubredditNamePrefixed string  `json:"subreddit_name_prefixed"`
		Score                 int     `json:"score"`
		CreatedUTC            float64 `json:"created_utc"`
		Permalink             string  `json:"permalink"`
		LinkTitle             string  `json:"link_title"`
		LinkURL               string  `json:"link_url"`
		LinkPermalink         string  `json:"link_permalink"`
		Over18                bool    `json:"over_18"`
	} `json:"data"`
}

// GetSavedCommentsRequest defines the request structure for fetching saved comments with details
type GetSavedCommentsRequest struct {
	AuthMethod  string `json:"auth_method,omitempty"`  // "cookie" or "oauth"
	Cookie      string `json:"cookie,omitempty"`       // For cookie-based auth
	AccessToken string `json:"access_token,omitempty"` // For OAuth-based auth
	Username    string `json:"username,omitempty"`     // For OAuth-based auth
}

// GetSavedCommentsResponse defines the response structure for saved comments with full details
type GetSavedCommentsResponse struct {
	Success  bool               `json:"success"`
	Message  string             `json:"message"`
	Comments []SavedCommentInfo `json:"comments"`
	Count    int                `json:"count"`
}

// ExportRequest defines the request structure for the export endpoint
type ExportRequest struct {
	AuthMethod  string   `json:"auth_method,omitempty"`  // "cookie" or "oauth"
	Cookie      string   `json:"cookie,omitempty"`       // For cookie-based auth
	AccessToken string   `json:"access_token,omitempty"` // For OAuth-based auth
	Username    string   `json:"username,omitempty"`     // For OAuth-based auth
	Sections    []string `json:"sections,omitempty"`     // Which sections to export (empty = all)
}

// ExportData contains all exported account data
type ExportData struct {
	ExportedAt    string             `json:"exported_at"`
	Username      string             `json:"username"`
	Subreddits    []SubredditInfo    `json:"subreddits,omitempty"`
	SavedPosts    []SavedPostInfo    `json:"saved_posts,omitempty"`
	SavedComments []SavedCommentInfo `json:"saved_comments,omitempty"`
	Multireddits  []MultiredditInfo  `json:"multireddits,omitempty"`
	Errors        []string           `json:"errors,omitempty"`
}

// MultiredditInfo contains detailed information about a multireddit for UI display
type MultiredditInfo struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Path        string   `json:"path"`
	Description string   `json:"description_md"`
	Subreddits  []string `json:"subreddits"`
	IconURL     string   `json:"icon_url"`
	Visibility  string   `json:"visibility"`
	Created     float64  `json:"created_utc"`
	NumSubs     int      `json:"num_subscribers"`
}

// GetMultiredditsRequest defines the request structure for fetching multireddits
type GetMultiredditsRequest struct {
	AuthMethod  string `json:"auth_method,omitempty"`
	Cookie      string `json:"cookie,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
	Username    string `json:"username,omitempty"`
}

// GetMultiredditsResponse defines the response structure for multireddits
type GetMultiredditsResponse struct {
	Success      bool              `json:"success"`
	Message      string            `json:"message"`
	Multireddits []MultiredditInfo `json:"multireddits"`
	Count        int               `json:"count"`
}

// ManageMultiredditResponseType defines the response for multireddit migration operations
type ManageMultiredditResponseType struct {
	SuccessCount int
	FailedCount  int
	FailedMultis []string
}
