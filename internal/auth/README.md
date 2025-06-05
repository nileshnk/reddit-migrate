# Reddit Authentication Module

This module provides two authentication methods for Reddit:

1. **Cookie-based authentication** (legacy method)
2. **OAuth2 authentication** (recommended)

## OAuth2 Authentication

### Setup

1. **Create a Reddit App**:

   - Go to https://www.reddit.com/prefs/apps
   - Click "Create App" or "Create Another App"
   - Choose "web app" as the app type
   - Set the redirect URI to `http://localhost:5005/api/oauth/callback` (or your domain)
   - Note down the client ID and client secret

2. **Configure Environment Variables**:

   ```bash
   export REDDIT_CLIENT_ID="your_client_id"
   export REDDIT_CLIENT_SECRET="your_client_secret"
   export REDDIT_REDIRECT_URI="http://localhost:5005/api/oauth/callback"  # Optional
   ```

3. **Start the Application**:
   ```bash
   go run cmd/reddit-migrate/main.go
   ```

### OAuth Flow

1. **Login**: Navigate to `/api/oauth/login` to initiate the OAuth flow
2. **Authorization**: User will be redirected to Reddit to authorize the app
3. **Callback**: After authorization, user is redirected back with tokens
4. **Token Response**: The callback endpoint returns:
   ```json
   {
     "success": true,
     "data": {
       "username": "reddit_username",
       "access_token": "token_here",
       "refresh_token": "refresh_token_here",
       "expires_in": 3600
     },
     "message": "OAuth authentication successful"
   }
   ```

### Using OAuth Tokens

#### In API Requests

When making API requests, you can use the OAuth token instead of cookies:

```javascript
// Example: Using OAuth token with the migration API
const response = await fetch("/api/migrate", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    Authorization: `Bearer ${accessToken}`,
  },
  body: JSON.stringify({
    oldAccountToken: oldAccessToken,
    newAccountToken: newAccessToken,
    preferences: {
      migrate_subreddit_bool: true,
      migrate_post_bool: true,
      delete_subreddit_bool: false,
      delete_post_bool: false,
    },
  }),
});
```

#### Token Refresh

When a token expires, use the refresh token to get a new access token:

```go
newToken, err := auth.RefreshAccessToken(refreshToken)
if err != nil {
    // Handle error - user may need to re-authenticate
}
```

### OAuth Scopes

The application requests the following Reddit OAuth scopes:

- `identity`: Access user's identity
- `read`: Access posts and comments
- `subscribe`: Manage subreddit subscriptions
- `save`: Save and unsave posts/comments
- `submit`: Submit posts and comments
- `vote`: Vote on posts and comments
- `mysubreddits`: Access user's subreddits
- `history`: Access user's voting history

## Cookie-Based Authentication (Legacy)

Cookie authentication is still supported for backward compatibility but is not recommended for new implementations.

### Usage

1. **Get Reddit Cookie**:

   - Log into Reddit in your browser
   - Open Developer Tools (F12)
   - Go to Application/Storage → Cookies
   - Copy the entire cookie string

2. **Verify Cookie**:

   ```bash
   curl -X POST http://localhost:5005/api/verify-cookie \
     -H "Content-Type: application/json" \
     -d '{"cookie": "your_cookie_string_here"}'
   ```

3. **Response**:
   ```json
   {
     "success": true,
     "data": {
       "username": "reddit_username"
     },
     "message": "Valid Token/Cookie"
   }
   ```

### Cookie Format

The cookie must contain the `token_v2` parameter. Example:

```
token_v2=eyJhbGc...; reddit_session=...; other_params=...
```

## API Endpoints

### OAuth Endpoints

- `GET /api/oauth/login` - Initiate OAuth login flow
- `GET /api/oauth/callback` - OAuth callback endpoint (handled automatically)

### Authentication Endpoints

- `POST /api/verify-cookie` - Verify a Reddit cookie and get username

## Error Handling

### OAuth Errors

- **Invalid state**: The OAuth state parameter doesn't match
- **Authorization denied**: User denied the authorization request
- **Token expired**: Access token has expired (use refresh token)

### Cookie Errors

- **Invalid Token/Cookie**: Cookie is malformed or expired
- **token_v2 not found**: Cookie doesn't contain the required token

## Security Considerations

1. **Store tokens securely**: Never expose tokens in logs or client-side code
2. **Use HTTPS in production**: OAuth requires secure connections
3. **Validate state parameter**: Prevents CSRF attacks
4. **Token expiration**: Access tokens expire after 1 hour
5. **Refresh tokens**: Store securely and use to get new access tokens

## Migration to OAuth

To migrate from cookie-based to OAuth authentication:

1. Update API calls to use `Authorization: Bearer <token>` header
2. Replace cookie verification with OAuth token validation
3. Implement token refresh logic for long-running operations
4. Update UI to use OAuth login flow instead of cookie input
