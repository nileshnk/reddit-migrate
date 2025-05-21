package main

// migration_request_type defines the structure for the migration request body.
// It includes cookies for old and new accounts, and user preferences for migration.
type migration_request_type struct {
	Old_account_cookie string           `json:"old_account_cookie"`
	New_account_cookie string           `json:"new_account_cookie"`
	Preferences        preferences_type `json:"preferences"`
}

// preferences_type defines the user's choices for the migration process.
// Each boolean field indicates whether a specific migration or deletion action should be performed.
type preferences_type struct {
	Migrate_subreddit_bool bool `json:"migrate_subreddit_bool"`
	Migrate_post_bool      bool `json:"migrate_post_bool"`
	Delete_post_bool       bool `json:"delete_post_bool"`
	Delete_subreddit_bool  bool `json:"delete_subreddit_bool"`
}

// migration_response_type defines the structure of the response sent after a migration attempt.
// It includes success status, a message, and detailed data about the performed actions.
type migration_response_type struct {
	Success bool          `json:"success"`
	Message string        `json:"message"`
	Data    MigrationData `json:"data"`
}

// MigrationData holds the detailed results of migration operations.
// This structure is embedded within migration_response_type.
type MigrationData struct {
	SubscribeSubreddit   manage_subreddit_response_type `json:"subscribeSubreddit"`
	UnsubscribeSubreddit manage_subreddit_response_type `json:"unsubscribeSubreddit"`
	SavePost             manage_post_type               `json:"savePost"`
	UnsavePost           manage_post_type               `json:"unsavePost"`
}

// subscribe_type defines the action to be performed on a subreddit (subscribe or unsubscribe).
type subscribe_type string

const (
	// subscribe indicates an action to subscribe to a subreddit.
	subscribe subscribe_type = "sub"
	// unsubscribe indicates an action to unsubscribe from a subreddit.
	unsubscribe subscribe_type = "unsub"
)

// manage_subreddit_response_type defines the structure for the response of managing subreddits.
// It includes error status, HTTP status code, counts of successful and failed operations, and a list of failed subreddits.
type manage_subreddit_response_type struct {
	Error            bool
	StatusCode       int
	SuccessCount     int
	FailedCount      int
	FailedSubreddits []string
}

// post_save_type defines the action to be performed on a post (save or unsave).
type post_save_type string

const (
	// SAVE indicates an action to save a post.
	SAVE post_save_type = "save"
	// UNSAVE indicates an action to unsave a post.
	UNSAVE post_save_type = "unsave"
)

// manage_post_type defines the structure for the response of managing posts.
// It includes counts of successful and failed operations.
type manage_post_type struct {
	SuccessCount int
	FailedCount  int
}

// reddit_name_type holds lists of subreddit and user display names and full names.
// This is used internally to pass around collections of names fetched from Reddit.
type reddit_name_type struct {
	fullNamesList       []string
	displayNamesList    []string
	userDisplayNameList []string
}

// full_name_list_type defines the structure for a list of items (subreddits or posts) from Reddit API.
// It's used for unmarshalling JSON responses that contain a list of children objects.
type full_name_list_type struct {
	Kind string `json:"kind"`
	Data struct {
		After    string `json:"after"` // Token for pagination, indicates the next item to fetch.
		Children []struct {
			Kind string `json:"kind"` // Type of the child item (e.g., t5 for subreddit, t3 for post).
			Data struct {
				Name           string `json:"name"`           // Full name of the item (e.g., t5_abcdef).
				Display_name   string `json:"display_name"`   // User-friendly display name (e.g., AskReddit).
				Subreddit_type string `json:"subreddit_type"` // Type of subreddit (e.g., public, private, user).
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

// verify_cookie_type defines the structure for the request body when verifying a cookie.
// It contains the cookie string to be verified.
type verify_cookie_type struct {
	Cookie string `json:"cookie"`
}

// profile_response_type defines the structure for the response from Reddit's /api/me.json endpoint.
// It contains basic profile information of the authenticated user.
type profile_response_type struct {
	Type string `json:"type"` // Should be "Identified"
	Data struct {
		Name        string `json:"name"`        // Username of the account.
		Is_employee bool   `json:"is_employee"` // Whether the user is a Reddit employee.
		Is_friend   bool   `json:"is_friend"`   // Whether the user is a friend of the authenticated user (rarely used).
	} `json:"data"`
}

// token_response_type defines the structure for the response when verifying a token/cookie.
// It indicates success, a message, and the username associated with the token/cookie if valid.
type token_response_type struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Username string `json:"username"`
	} `json:"data"`
}

// error_response_type defines a generic error response structure from the Reddit API.
// It usually contains an error code and a descriptive message.
type error_response_type struct {
	Error   string `json:"error"`   // Error code (e.g., "401").
	Message string `json:"message"` // Descriptive error message.
}
