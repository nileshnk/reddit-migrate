# reddit-migrate

A simple interface to migrate Reddit account data from Old account to New account using Reddit API's.

:warning:

> Note: This requires cookie of both Old Account & New Account (which also contains Authorization access token). Cookies contains sensitive data(credentials), therefore, make sure you never share it with anyone. ⚠️

## Installation

You can run the application locally or through Docker.

#### Local

Make sure golang is installed in your system. [Install here](https://go.dev/dl/) if not installed.

```
go mod tidy
go run .

# Assign different address by passing --addr flag.
go run . --addr=":3000"
```

#### Docker

Make sure docker is installed and running in background.

```
docker build -t "reddit-migrate" .
docker run -it -d -p 5005:5005 --name "migrate" reddit-migrate
```

After installation, open this site in browser http://localhost:5005 . Follow the below steps.

### Steps:

- Retrieve the cookie of both accounts (old and new).

  - You can watch [this video](https://raw.githubusercontent.com/nileshnk/reddit-migrate/main/assets/capture-cookie.mp4) to follow along.
  - Login to Reddit in the desktop web browser
  - Open a new tab.
  - Open the Network tab through developer tools. (You can right click and go to Inspect.)
  - Go to this url https://www.reddit.com/api/me.json or any other reddit page.
  - Go to that new request that just popped in network tab.
  - Scroll below to find cookie. Copy whole cookie by triple clicking on it.
  - Find cookie for other account similarly.

- Paste both cookies accordingly and verify.
- Once verified, select options according to requirement.
- Submit

### Code Description

I've used Reddit's API's to make request to Reddit using the cookies of the respective user account.
First on clicking the verify button,

- Verifies the cookie. Checks if access token is present in the cookie, and if Reddit responds with user data.
- Access Token is extracted from cookie.

For subreddit migrations, following steps are done:

- Makes a request to list all the subreddits. Display Names of subreddits are stored in array.
- Names are passed to a function that subscribes given subreddits.
- It processess the subreddits in chunks of 100 names per request.
- Separates the user accounts that are subscribed. Handles it separately.
- Returns success and failure Count.

For saved-posts migrations, following steps are done:

- Makes a request to list all the saved posts. Full Names of the posts are stored.
- They are passed to function that saves the post.
- The function call an API of reddit that only allows saving one post per request. This can causes issues in some cases where Reddit blocks request after rate limit is hit (100 Requests / Minute is the current limit by Reddit)
- Returns success and failure count.

This was first written in JS. Then switched to golang, as an exercise to get familiar with golang. I might write the code in JS for NodeJS in future.

Contact me for any queries / suggestion: mail@inilesh.com
