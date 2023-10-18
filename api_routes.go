package main

import (
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
		fmt.Fprintf(w, "Hello World! I'm Nilesh")
	});

	router.Post("/verify-token", verifyToken)
}



type verify_token_type struct {
	Access_token string `json:"access_token"`
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
func verifyToken(w http.ResponseWriter, r *http.Request){

	headerContentTtype := r.Header.Get("Content-Type")
	if headerContentTtype != "application/json" {
		errorResponse(w, "Content Type is not application/json", http.StatusUnsupportedMediaType)
		return
	}

	// Parse token from request body
	var t verify_token_type
	var unmarshalErr *json.UnmarshalTypeError

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&t)
	if err != nil {
		if errors.As(err, &unmarshalErr) {
			errorResponse(w, "Bad Request. Wrong Type provided for field "+unmarshalErr.Field, http.StatusBadRequest)
		} else {
			errorResponse(w, "Bad Request "+err.Error(), http.StatusBadRequest)
		}
		return
	}	

	reddit_profile_uri := "https://www.reddit.com/api/me.json"
	
	
	req, err := http.NewRequest(http.MethodGet, reddit_profile_uri, nil)

	// setCookieFromToken := "token_v2=" + t.Access_token;
	req.Header = http.Header{
		"Cookie": []string{t.Access_token},
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
		jsonResp, errJsonMarshal :=  json.Marshal(FinalResponse);
		if errJsonMarshal != nil {
			fmt.Println("Error in Marshalling Response Body");
			fmt.Println(errJsonMarshal)
		}
		w.Write(jsonResp)
		return
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
	jsonResp, errJsonMarshal :=  json.Marshal(FinalResponse);
	if errJsonMarshal != nil {
		fmt.Println("Error in Marshalling Response Body");
		fmt.Println(errJsonMarshal)
	}
	w.Write(jsonResp)
	// tokenFromCookie := parseTokenFromCookie(t.Access_token)
	// fmt.Println(tokenFromCookie)

	return 	
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
