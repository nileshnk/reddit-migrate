# API Package

This package contains the HTTP API handlers for the Reddit migration application.

## Structure

The API package is organized into the following files:

### `routes.go`

The main router configuration file that sets up all API routes and maps them to their respective handlers. This file is kept clean and focused only on route registration.

### `handlers.go`

Contains system-level handlers:

- `HealthCheckHandler` - Health check endpoint for monitoring

### `data_handlers.go`

Contains handlers for data fetching operations:

- `SubredditsHandler` - Fetches detailed subreddit information
- `SavedPostsHandler` - Fetches detailed saved posts information
- `AccountCountsHandler` - Fetches counts of subreddits and saved posts

### `migration_handlers.go`

Contains migration-specific handlers:

- `CustomMigrationHandler` - Handles custom selection migration requests

### `utils.go`

Contains utility functions shared across handlers:

- `ValidateContentType` - Validates request content type
- `SendJSONResponse` - Sends structured JSON responses
- `SendErrorResponse` - Sends structured error responses
- `DecodeJSONRequest` - Decodes JSON request bodies

## Route Groups

### System Routes

- `GET /health` - Health check endpoint

### Authentication Routes

- `GET /oauth/login` - OAuth login initiation
- `GET /oauth/callback` - OAuth callback handler
- `POST /verify-cookie` - Cookie verification

### Data Fetching Routes

- `POST /subreddits` - Get detailed subreddit information
- `POST /saved-posts` - Get detailed saved posts information
- `POST /account-counts` - Get account data counts

### Migration Routes

- `POST /migrate` - Full migration (handled in migration package)
- `POST /migrate-custom` - Custom selection migration

## Design Principles

1. **Separation of Concerns**: Each handler type is in its own file
2. **Single Responsibility**: Each handler function handles one specific endpoint
3. **Reusability**: Common functionality is extracted to utils.go
4. **Maintainability**: Clean, focused code with clear documentation
5. **Consistency**: All handlers follow the same error handling and response patterns

## Handler Pattern

All handlers follow a consistent pattern:

1. Validate content type
2. Decode request body
3. Validate authentication/authorization
4. Process the request
5. Send structured response

Error handling is standardized using the utility functions to ensure consistent API responses.
