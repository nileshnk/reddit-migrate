# reddit-migrate

A simple interface to migrate Reddit account data from Old account to New account using Reddit API's.

:warning:

> Note: This requires cookie of both Old Account & New Account (which also contains Authorization access token). ⚠️ Cookies contains sensitive data(credentials), therefore, make sure you never share it with anyone. ⚠️

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

After installation on opening the site in browser. Follow the below steps.

### Steps:

- Retrieve the cookie of both accounts (old and new). [See how!](https://www.google.com)
- Paste both cookies accordingly and verify.
- Once verified, select options according to requirement.
- Submit
