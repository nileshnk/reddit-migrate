package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func testRouter(r chi.Router) {
	r.Get("/test", func(w http.ResponseWriter, r *http.Request){
		fmt.Fprintf(w, "Hello World! I'm Nilesh")
	});
}

func apiRouter(router chi.Router) {
	
	router.Get("/test", func(w http.ResponseWriter, r *http.Request){
		type test_data struct {
			Hello string `json:"hello"`
		}
		var test test_data;
		test.Hello = "world!"
		t := []byte(`{"hello": "nilesh"}`)
		// body, _ := json.Marshal(t);
		w.Header().Set("Content-Type", "application/json")
		w.Write(t);
	});

	router.Post("/verify-cookie", verifyTokenResponse)

	router.Post("/migrate", MigrationHandler)
}

type migration_request_type struct {
	Old_account_cookie string `json:"old_account_cookie"`
	New_account_cookie string `json:"new_account_cookie"`
	Preferences preferences_type `json:"preferences"`
}

// It is the core migrate handler, calling all required methods.
func MigrationHandler(w http.ResponseWriter, r *http.Request){
	
	// extract tokens and preferences from request body
	headerContentTtype := r.Header.Get("Content-Type")
	if headerContentTtype != "application/json" {
		errorResponse(w, "Content Type is not application/json", http.StatusUnsupportedMediaType)
		return
	}

	// Parse from request body
	var requestBody migration_request_type
	var unmarshalErr *json.UnmarshalTypeError

	decoder := json.NewDecoder(r.Body)
	// decoder.DisallowUnknownFields()


	err := decoder.Decode(&requestBody)
	if err != nil {
		if errors.As(err, &unmarshalErr) {
			errorResponse(w, "Bad Request. Wrong Type provided for field "+unmarshalErr.Field, http.StatusBadRequest)
		} else {
			errorResponse(w, "Bad Request "+err.Error(), http.StatusBadRequest)
		}
		return
	}
	FinalResponse := initializeMigration(requestBody.Old_account_cookie, requestBody.New_account_cookie, requestBody.Preferences)



	jsonResp, jsonMarshalErr := json.Marshal(FinalResponse);
	if jsonMarshalErr != nil {
		fmt.Println("Error in Marshalling Response Body");
		fmt.Println(jsonMarshalErr)
	}
	fmt.Println(string(jsonResp)); 
	w.Write(jsonResp)
	
	return
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

// main function to handle migration
func initializeMigration(old_account_cookie string, new_account_cookie string, preferences preferences_type) migration_response_type {

	var FinalResponse migration_response_type;

	oldAccountUsername := verifyCookie(old_account_cookie).Data.Username;
	newAccountUsername := verifyCookie(new_account_cookie).Data.Username;

	old_account_token := parseTokenFromCookie(old_account_cookie)
	new_account_token := parseTokenFromCookie(new_account_cookie)

	fmt.Println(preferences)

	
	if (preferences.Migrate_subreddit_bool || preferences.Delete_subreddit_bool) {
		fmt.Println("Fetching all subreddits full names...")
		subredditNameList := fetchSubredditFullNames(old_account_token)
		
		subredditChunkSize := 100
		if preferences.Migrate_subreddit_bool == true {
			fmt.Println("Migrations of subreddits started...")
			
			fmt.Println("Total subreddits in Old Account is", len(subredditNameList.fullNamesList))

			subscribeData := manageSubreddits(new_account_token, subredditNameList.displayNamesList, subscribe_type(subscribe), subredditChunkSize)
			fmt.Println("Subscribed from Old Account with following data")
			fmt.Println(subscribeData);
			retryMigrate := subscribeData
			retryAttempts := 1
			for retryMigrate.FailedCount > 0 {
				if(retryAttempts >= 5){
					fmt.Printf("Retry Failed. %v subreddits failed to migrate! \n", retryMigrate.FailedCount)
					break;
				}
				fmt.Printf("Total subreddits failed to migrate: %v. Retrying... \n", retryMigrate.FailedCount)
				retryMigrate = manageSubreddits(new_account_token, retryMigrate.FailedSubreddits, subscribe_type(subscribe), subredditChunkSize/retryAttempts)
				retryAttempts++
			}

			fmt.Printf("Total followed users in \"%v\" account: %v \n", oldAccountUsername, len(subredditNameList.userDisplayNameList))
			fmt.Printf("Following users in %v account...\n", newAccountUsername);

			followUsersData := manageFollowedUsers(new_account_token, subredditNameList.userDisplayNameList, subscribe_type(subscribe))
			fmt.Println(followUsersData)

			FinalResponse.Data.SubscribeSubreddit = subscribeData	
		}

		if preferences.Delete_subreddit_bool == true {
			unsubscribeData := manageSubreddits(old_account_token, subredditNameList.displayNamesList, subscribe_type(unsubscribe), 500)
			fmt.Println("Unsubscribed from Old Account with following data")
			// fmt.Println(unsubscribeData)
			FinalResponse.Data.UnsubscribeSubreddit = unsubscribeData
		}
	}
	
	if (preferences.Migrate_post_bool || preferences.Delete_post_bool) {
	
		savedPostsFullNamesList := fetchSavedPostsFullNames(old_account_token, oldAccountUsername)
		// fmt.Println(savedPostsFullNamesList)
		if preferences.Migrate_post_bool == true {
			// oldAccountUsername := verifyToken()
			fmt.Printf("Found %v posts from Old Account %v. Migrating...\n", len(savedPostsFullNamesList), oldAccountUsername)

			savePostsResponse := manageSavedPosts(new_account_token, savedPostsFullNamesList, post_save_type(SAVE))
			fmt.Printf("Saved %v posts to New Account %v \n", savePostsResponse.SuccessCount, newAccountUsername)
			FinalResponse.Data.SavePost = savePostsResponse
		}

		if preferences.Delete_post_bool == true {
			fmt.Printf("Unsaving %v Posts from Old Account %v \n",len(savedPostsFullNamesList), oldAccountUsername)
			unsavePostsResponse := manageSavedPosts(old_account_token, savedPostsFullNamesList, post_save_type(UNSAVE))
			FinalResponse.Data.UnsavePost = unsavePostsResponse
			fmt.Printf("Unsaved %v Posts from %v \n", unsavePostsResponse.SuccessCount, oldAccountUsername)
		}
	}
	
	FinalResponse.Success = true
	FinalResponse.Message = "Migration Successful"
	
	// fmt.Println(FinalResponse)

	return FinalResponse
}


// create an enum for subscribe and unsubscribe
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

// function to manage the subreddits
func manageSubreddits(token string, subredditFullNamesList []string, subscribeType subscribe_type, chunkSize int  ) manage_subreddit_response_type {
	// split the subredditFullNamesList into chunks 

	chunks := chunkArray(subredditFullNamesList, chunkSize);
	var FinalResponse manage_subreddit_response_type;
	var responses []manage_subreddit_response_type;
	// iterate over the chunks and manage the subreddits
	for _, chunk := range chunks {
		response := manageSubredditsChunk(token, chunk, subscribeType)
		responses = append(responses, response);
		if response.Error == true {
			FinalResponse.Error = true
			FinalResponse.StatusCode = 500
			FinalResponse.FailedCount += response.FailedCount 
			FinalResponse.FailedSubreddits = append(FinalResponse.FailedSubreddits, chunk...)
		} else {
			FinalResponse.SuccessCount += len(chunk);
		}
	}

	return FinalResponse
}

// function to manage the subreddits in chunks
func manageSubredditsChunk(token string, subredditFullNamesList []string, subscribeType subscribe_type) manage_subreddit_response_type {
	
	subredditNames := strings.Join(subredditFullNamesList, ",");

	requiredUri := "https://oauth.reddit.com/api/subscribe";
	// requiredBody := "sr=" + subredditNames + "&action=" + string(subscribeType) + "&api_type=json";
	stringifiedBody := fmt.Sprintf("sr_name=%v&action=%v&api_type=json", subredditNames, string(subscribeType))

	requiredBody := []byte (stringifiedBody);
	// body, err := ioutil.ReadAll(requiredBody);
	modifySubredditReq, modifySubredditReqErr := http.NewRequest(http.MethodPost, requiredUri,bytes.NewBuffer(requiredBody));
	modifySubredditReq.Header = http.Header{
		"Authorization": []string{"Bearer " + token},
		"Content-Type": []string{"application/x-www-form-urlencoded"},
		"User-Agent": []string{"Mozilla/5.0 (X11; Linux x86_64; rv:91.0) Gecko/20100101 Firefox/91.0"},
		// "User-Agent": []string{"this is a sample string to test user-agent"},
	}

	if modifySubredditReqErr != nil {
		fmt.Println("Error in creating request for modifying subreddits");
	}

	modifySubredditRes, modifySubredditResErr := http.DefaultClient.Do(modifySubredditReq);

	if modifySubredditResErr != nil {
		fmt.Println("Error from server for modifying subreddits");
		fmt.Println(modifySubredditResErr);
	}

	body, _ := ioutil.ReadAll(modifySubredditRes.Body)

	fmt.Println(string(body))
	

	response := manage_subreddit_response_type{
		Error: false,
		StatusCode: modifySubredditRes.StatusCode,
		SuccessCount: len(subredditFullNamesList),
		FailedCount: 0,
		FailedSubreddits: nil,
	}

	if modifySubredditRes.StatusCode != 200 {
		response.Error = true
		response.SuccessCount = 0
		response.FailedCount = len(subredditFullNamesList)
		response.FailedSubreddits = subredditFullNamesList
	}

	return response
}

func manageFollowedUsers(token string, userList []string, subscribeType subscribe_type) manage_subreddit_response_type {
	var FinalResponse manage_subreddit_response_type
	FinalResponse.Error = false

	var requestMethod string = (map[bool]string{true: http.MethodPut, false: http.MethodDelete })[subscribeType == subscribe_type(subscribe)] 

	var FailedResponseData []string

	for _, username := range userList {
		// create a request for each user

		username = strings.TrimPrefix(username, "u_")
		requiredUri := fmt.Sprintf("https://oauth.reddit.com/api/v1/me/friends/%v", username)
		jsonBody := map[string]string{"name": username}
		requiredBody, _ := json.Marshal(jsonBody)
		createReq, _ := http.NewRequest(requestMethod, requiredUri, bytes.NewBuffer( requiredBody))

		createReq.Header = http.Header{
			"Authorization": []string{"Bearer " + token},
			"Content-Type": []string{"application/json"},
			"User-Agent": []string{"Mozilla/5.0 (X11; Linux x86_64; rv:91.0) Gecko/20100101 Firefox/91.0"},
		}

		followResponse, followResponseErr := http.DefaultClient.Do(createReq);
		if followResponseErr != nil {
			fmt.Println("Error in follow request response")
		}

		body, _ := (ioutil.ReadAll(followResponse.Body))
		fmt.Println(string(body))

		if followResponse.StatusCode != 200 {
			fmt.Println(followResponse.StatusCode)
			FailedResponseData = append(FailedResponseData, username)
			FinalResponse.Error = true
			FinalResponse.StatusCode = followResponse.StatusCode
		}
	}
	FinalResponse = manage_subreddit_response_type{
		SuccessCount:len(userList) - len(FailedResponseData),
		FailedCount: len(FailedResponseData),
		FailedSubreddits: FailedResponseData,
	}

	return FinalResponse
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

// function to manage saved posts
func manageSavedPosts(token string, postIds []string, saveType post_save_type) manage_post_type {
	var failedSavePostIds []string
	failedCount := 0
	var requiredUri string  =  fmt.Sprintf("https://oauth.reddit.com/api/%v", saveType);
	


	for _, postId := range postIds {
		var requiredBody = []byte (fmt.Sprintf("id=%v", postId));	
		managePostReq, managePostReqErr := http.NewRequest(http.MethodPost, requiredUri, bytes.NewBuffer(requiredBody));
		managePostReq.Header = http.Header{
			"Authorization": []string{"Bearer " + token},
			"User-Agent": []string{"Mozilla/5.0 (X11; Linux x86_64; rv:91.0) Gecko/20100101 Firefox/91.0"},
			// "User-Agent": []string{"this is a sample string to test user-agent"},
		}
		if managePostReqErr != nil {
			fmt.Printf("Error %v post with full name: %v \n", saveType ,postId)
		}

		managePostRes, managePostResErr := http.DefaultClient.Do(managePostReq);

		if managePostResErr != nil {
			fmt.Printf("Error response while %v post: %v \n", saveType ,postId)
		}

		if managePostRes.StatusCode != 200 {
			failedSavePostIds = append(failedSavePostIds, postId)
			failedCount += 1
		}
	}
	
	var FinalResponse manage_post_type 
	FinalResponse.FailedCount = failedCount
	FinalResponse.SuccessCount = len(postIds) - failedCount
	return FinalResponse
}

// split array into chunks
func chunkArray(array []string, chunkSize int) [][]string {
	var chunks [][]string
	for i := 0; i < len(array); i += chunkSize {
		end := i + chunkSize
		if end > len(array) {
			end = len(array)
		}
		chunks = append(chunks, array[i:end])
	}
	return chunks
}

// function to fetch all the posts from the subreddit
func fetchSavedPostsFullNames(token string, username string) []string {

	require_uri := "https://oauth.reddit.com/user/" + username + "/saved.json";
	savedPostsFullNamesList := fetchAllFullNames(require_uri, token, false)
	return append(savedPostsFullNamesList.fullNamesList, savedPostsFullNamesList.userDisplayNameList...) ;
}

func fetchSubredditFullNames(token string) reddit_name_type {
	
	require_uri := "https://oauth.reddit.com/subreddits/mine.json";
	subredditFullNamesList := fetchAllFullNames(require_uri, token, true)
	return subredditFullNamesList;
}

type reddit_name_type struct {
	fullNamesList []string;
	displayNamesList []string;
	userDisplayNameList []string;
}

func fetchAllFullNames(require_uri string, token string, is_subreddit bool) reddit_name_type {
	var fullNamesList []string;
	var displayNamesList []string;

	lastFullName := "";
	var FollowedUsers []string

	for true {
		listing_uri := require_uri + "?limit=100&after=" + lastFullName;
		createReq, createReqErr := http.NewRequest(http.MethodGet, listing_uri, 
		nil)
		createReq.Header = http.Header{
			"Authorization": []string{"Bearer " + token},
			"User-Agent": []string{"Mozilla/5.0 (X11; Linux x86_64; rv:91.0) Gecko/20100101 Firefox/91.0"},
		}
		if createReqErr != nil {
			fmt.Println("Error Creating Request");
			fmt.Println(createReqErr)
		}

		res, err := http.DefaultClient.Do(createReq)

		if err != nil {
			fmt.Println("Error in Fetching Saved Posts");
			fmt.Println(err)
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

		var fullNameSingleList full_name_list_type;

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Println("Error in Reading Body");
			fmt.Println(err)
		}

		json.Unmarshal(body, &fullNameSingleList);
		// fmt.Println(string(body))
		// fmt.Println(fullNameSingleList)

		

		for _, child := range fullNameSingleList.Data.Children {
			if (child.Data.Subreddit_type == "user" && is_subreddit == true) {
				FollowedUsers = append(FollowedUsers, child.Data.Display_name)
				continue
			}
			fullNamesList = append(fullNamesList, child.Data.Name);
			displayNamesList = append(displayNamesList, child.Data.Display_name);
		}

		if fullNameSingleList.Data.After == "" {
			break;
		}

		lastFullName = fullNameSingleList.Data.After;

	}
	
	var FinalResponse reddit_name_type;
	FinalResponse.displayNamesList = displayNamesList
	FinalResponse.fullNamesList = fullNamesList
	FinalResponse.userDisplayNameList = FollowedUsers

	// fmt.Printf("Users subscribed: %v\n", len(FollowedUsers))
	return FinalResponse
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

// sends a response with username if token is valid
func verifyTokenResponse(w http.ResponseWriter, r *http.Request) {

	headerContentTtype := r.Header.Get("Content-Type")
	if headerContentTtype != "application/json" {
		errorResponse(w, "Content Type is not application/json", http.StatusUnsupportedMediaType)
		return
	}

	// Parse token from request body
	var requestBody verify_cookie_type
	var unmarshalErr *json.UnmarshalTypeError

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&requestBody)
	if err != nil {
		if errors.As(err, &unmarshalErr) {
			errorResponse(w, "Bad Request. Wrong Type provided for field "+unmarshalErr.Field, http.StatusBadRequest)
		} else {
			errorResponse(w, "Bad Request "+err.Error(), http.StatusBadRequest)
		}
		return
	}	

	FinalResponse := verifyCookie(requestBody.Cookie)

	jsonResp, jsonMarshalErr :=  json.Marshal(FinalResponse);
	if jsonMarshalErr != nil {
		fmt.Println("Error in Marshalling Response Body");
		fmt.Println(jsonMarshalErr)
	}

	w.Write(jsonResp)
	// tokenFromCookie := parseTokenFromCookie(t.Access_token)
	// fmt.Println(tokenFromCookie)

	return 	
}

func verifyCookie(cookie string) token_response_type {

	reddit_profile_uri := "https://www.reddit.com/api/me.json"
	req, err := http.NewRequest(http.MethodGet, reddit_profile_uri, nil)

	// setCookieFromToken := "token_v2=" + t.Access_token;
	req.Header = http.Header{
		"Cookie": []string{cookie},
		// "Cookie": []string{setCookieFromToken},
		"User-Agent": []string{"Mozilla/5.0 (X11; Linux x86_64; rv:91.0) Gecko/20100101 Firefox/91.0"},
	}
	// fmt.Println(t.Access_token);
	// req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	if err != nil {
		fmt.Println("Error Creating Request");
		fmt.Println(err)
	}

	res, err := http.DefaultClient.Do(req)
	// fmt.Println(res);

	var FinalResponse token_response_type;

	if(res.StatusCode != 200){

		type error_response_type struct {
			Error string `json:"error"`
			Message string `json:"message"`
		}

		var errorResponse error_response_type
		response, errResponseRead := ioutil.ReadAll(res.Body);
		if err != nil {
			fmt.Println("Error in Reading Response Body");
			fmt.Println(errResponseRead)
		}
		errResponse := json.Unmarshal(response, &errorResponse);
		if errResponse != nil {
			fmt.Println("Error in Marshalling Response Body");
			fmt.Println(errResponse)
		}


		FinalResponse.Success = false;
		FinalResponse.Message = "Invalid Token/Cookie";
		FinalResponse.Data.Username = errorResponse.Message;
		return FinalResponse
	}

	if err != nil {
		fmt.Println("Error in Parsing Response");
		fmt.Println(err)
	}
	var profile profile_response_type

	profileResponse, errReadResponse := ioutil.ReadAll(res.Body);
	if errReadResponse != nil {
		fmt.Println("Error in Reading Response Body");
		fmt.Println(errReadResponse)
	}
	// fmt.Println(string(profileResponse))

	errGet := json.Unmarshal(profileResponse, &profile);
	if errGet != nil {
		fmt.Println("Error in Marshalling Response Body");
		fmt.Println(errGet)
	}

	FinalResponse.Success = true;
	FinalResponse.Message = "Valid Token/Cookie";
	FinalResponse.Data.Username = profile.Data.Name;

	return FinalResponse
}

func parseTokenFromCookie(cookie string) string {

	token := strings.Split(cookie, ";")
	var access_token string = "";
	for i := 0; i < len(token); i++ {
		str := strings.TrimSpace(token[i])
		if(strings.HasPrefix(str, "token_v2")){
			access_token = strings.Split(token[i], "=")[1]
		}
	}
	return access_token
}

func errorResponse(w http.ResponseWriter, message string, httpStatusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusCode)
	resp := make(map[string]string)
	resp["message"] = message
	jsonResp, _ := json.Marshal(resp)
	w.Write(jsonResp)
}

