package types

// MigrationRequestType defines the structure for the migration request body.
// It includes cookies for old and new accounts, and user preferences for migration.
type MigrationRequestType struct {
	OldAccountCookie string          `json:"old_account_cookie"`
	NewAccountCookie string          `json:"new_account_cookie"`
	Preferences      PreferencesType `json:"preferences"`
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
	Data    MigrationDetails `json:"data"` // Renamed from MigrationData for clarity
}

// MigrationDetails holds the detailed results of migration operations.
// This structure is embedded within MigrationResponseType.
type MigrationDetails struct { // Renamed from MigrationData
	SubscribeSubreddit   ManageSubredditResponseType `json:"subscribeSubreddit"`
	UnsubscribeSubreddit ManageSubredditResponseType `json:"unsubscribeSubreddit"`
	SavePost             ManagePostResponseType      `json:"savePost"`   // Renamed from manage_post_type
	UnsavePost           ManagePostResponseType      `json:"unsavePost"` // Renamed from manage_post_type
}

// SubredditActionType defines the action to be performed on a subreddit (subscribe or unsubscribe).
type SubredditActionType string // Renamed from subscribe_type

const (
	// SubscribeAction indicates an action to subscribe to a subreddit.
	SubscribeAction SubredditActionType = "sub" // Renamed from subscribe
	// UnsubscribeAction indicates an action to unsubscribe from a subreddit.
	UnsubscribeAction SubredditActionType = "unsub" // Renamed from unsubscribe
)

// ManageSubredditResponseType defines the structure for the response of managing subreddits.
// It includes error status, HTTP status code, counts of successful and failed operations, and a list of failed subreddits.
type ManageSubredditResponseType struct { // Renamed from manage_subreddit_response_type
	Error            bool
	StatusCode       int
	SuccessCount     int
	FailedCount      int
	FailedSubreddits []string
}

// PostActionType defines the action to be performed on a post (save or unsave).
type PostActionType string // Renamed from post_save_type

const (
	// SaveAction indicates an action to save a post.
	SaveAction PostActionType = "save" // Renamed from SAVE
	// UnsaveAction indicates an action to unsave a post.
	UnsaveAction PostActionType = "unsave" // Renamed from UNSAVE
)

// ManagePostResponseType defines the structure for the response of managing posts.
// It includes counts of successful and failed operations.
type ManagePostResponseType struct { // Renamed from manage_post_type
	SuccessCount int
	FailedCount  int
}

// RedditNameType holds lists of subreddit and user display names and full names.
// This is used internally to pass around collections of names fetched from Reddit.
type RedditNameType struct { // Renamed from reddit_name_type
	FullNamesList       []string
	DisplayNamesList    []string
	UserDisplayNameList []string
}

// FullNameListType defines the structure for a list of items (subreddits or posts) from Reddit API.
// It's used for unmarshalling JSON responses that contain a list of children objects.
type FullNameListType struct { // Renamed from full_name_list_type
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
type VerifyCookieType struct { // Renamed from verify_cookie_type
	Cookie string `json:"cookie"`
}

// ProfileResponseType defines the structure for the response from Reddit's /api/me.json endpoint.
// It contains basic profile information of the authenticated user.
type ProfileResponseType struct { // Renamed from profile_response_type
	Type string `json:"type"`
	Data struct {
		Name       string `json:"name"`
		IsEmployee bool   `json:"is_employee"` // Corrected json tag
		IsFriend   bool   `json:"is_friend"`   // Corrected json tag
	} `json:"data"`
}

// TokenResponseType defines the structure for the response when verifying a token/cookie.
// It indicates success, a message, and the username associated with the token/cookie if valid.
type TokenResponseType struct { // Renamed from token_response_type
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Username string `json:"username"`
	} `json:"data"`
}

// ErrorResponseType defines a generic error response structure from the Reddit API.
// It usually contains an error code and a descriptive message.
type ErrorResponseType struct { // Renamed from error_response_type
	Error   string `json:"error"`
	Message string `json:"message"`
}
