package main


type migration_request_type struct {
	Old_account_cookie string `json:"old_account_cookie"`
	New_account_cookie string `json:"new_account_cookie"`
	Preferences preferences_type `json:"preferences"`
}

type preferences_type struct {
	Migrate_subreddit_bool bool `json:"migrate_subreddit_bool"`
	Migrate_post_bool bool `json:"migrate_post_bool"`
	Delete_post_bool bool `json:"delete_post_bool"`
	Delete_subreddit_bool bool `json:"delete_subreddit_bool"`
}

type migration_response_type struct {
	Success bool `json:"success"`
	Message string `json:"message"`
	Data struct {
		SubscribeSubreddit manage_subreddit_response_type `json:"subscribeSubreddit"`
		UnsubscribeSubreddit manage_subreddit_response_type `json:"unsubscribeSubreddit"`
		SavePost manage_post_type `json:"savePost"`
		UnsavePost manage_post_type `json:"unsavePost"`
	} `json:"data"`
}

type subscribe_type string
const (
	subscribe subscribe_type = "sub"
	unsubscribe subscribe_type = "unsub"
)

type manage_subreddit_response_type struct{
	Error bool
	StatusCode int
	SuccessCount int
	FailedCount int 
	FailedSubreddits []string
}


type post_save_type string 
const (
	SAVE post_save_type = "save"
	UNSAVE post_save_type = "unsave"
)

type manage_post_type struct {
	SuccessCount int
	FailedCount int
}

type reddit_name_type struct {
	fullNamesList []string;
	displayNamesList []string;
	userDisplayNameList []string;
}

type full_name_list_type struct {
	Kind string `json:"kind"`
	Data struct {
		After string `json:"after"`
		Children []struct {
			Kind string `json:"kind"`
			Data struct {
				Name string `json:"name"`
				Display_name string `json:"display_name"`
				Subreddit_type string `json:"subreddit_type"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}


type verify_cookie_type struct {
	Cookie string `json:"cookie"`
}

type profile_response_type struct {
	Type string `json:"type"`
	Data struct {
		Name string `json:"name"`
		Is_employee bool `json:"is_employee"`
		Is_friend bool `json:"is_friend"`
	} `json:"data"`
}

type token_response_type struct {
	Success bool `json:"success"`
	Message string `json:"message"`
	Data struct {
		Username string `json:"username"`
	} `json:"data"`
}

type error_response_type struct {
	Error string `json:"error"`
	Message string `json:"message"`
}