# Reddit Management Tool - Component Architecture

## Overview

This document explains how all the JavaScript components are connected and work together to create a comprehensive Reddit management tool with both OAuth and cookie authentication support.

## Component Structure

```
web/static/
├── index.html              # Main application page
├── test.html              # Component testing page
├── app.js                 # Main application controller
├── components/
│   ├── DarkModeManager.js      # Theme switching
│   ├── AuthComponent.js        # Authentication (OAuth + Cookie)
│   ├── FeatureSelector.js      # Feature tab management
│   ├── MigrationFeature.js     # Migration settings
│   └── CleanupFeature.js       # Content cleanup settings
└── COMPONENTS_README.md   # This file
```

## Component Interactions

### 1. Main Application (`app.js`)

- **Role**: Central coordinator that manages all components
- **Responsibilities**:
  - Initialize all components
  - Handle form submission
  - Manage progress tracking
  - Display results
  - Connect to backend APIs

```javascript
class RedditMigrationApp {
  constructor() {
    this.authComponent = new AuthComponent(
      document.getElementById("authContainer")
    );
    this.featureSelector = new FeatureSelector(
      document.getElementById("featureContainer")
    );
    // ... other initialization
  }
}
```

### 2. Authentication Component (`AuthComponent.js`)

- **Role**: Handles both OAuth and cookie authentication for source and destination accounts
- **Features**:
  - Toggle between OAuth and cookie authentication
  - Separate authentication for source/destination accounts
  - Cookie verification via `/api/verify-cookie`
  - OAuth login via `/api/oauth/login` and `/api/oauth/callback`
  - Real-time status updates

```javascript
// Example usage
const authData = authComponent.getAuthData();
// Returns: { method: "oauth|cookie", source: {...}, destination: {...} }
```

### 3. Feature Selector (`FeatureSelector.js`)

- **Role**: Manages switching between Migration and Cleanup features
- **Features**:
  - Tab-based interface
  - Dynamic content rendering
  - Feature-specific component management

### 4. Migration Feature (`MigrationFeature.js`)

- **Role**: Handles migration settings and selections
- **Features**:
  - Selection modes: All, Custom, None
  - Support for subreddits, posts, and follows
  - Batch settings configuration
  - Delete options for source account
  - Custom selection modal integration

### 5. Cleanup Feature (`CleanupFeature.js`)

- **Role**: Manages content deletion settings (future feature)
- **Features**:
  - Content type selection (comments, posts, etc.)
  - Deletion modes (edit-then-delete, delete-only)
  - Advanced options (dry run, custom text)

### 6. Dark Mode Manager (`DarkModeManager.js`)

- **Role**: Handles theme switching
- **Features**:
  - Toggle between dark and light modes
  - Persistent settings via localStorage
  - CSS class management

## Authentication Flow

### OAuth Authentication Flow

1. User selects OAuth method
2. User enters Client ID and Client Secret
3. User clicks "Login with Reddit OAuth"
4. Popup window opens to `/api/oauth/login`
5. User authorizes on Reddit
6. Callback returns token data
7. Component stores token and updates UI

### Cookie Authentication Flow

1. User selects Cookie method
2. User pastes Reddit cookie
3. User clicks "Verify Cookie"
4. POST request to `/api/verify-cookie`
5. Backend verifies and returns username
6. Component updates UI with verification status

## Data Flow

```
User Interaction → Component State → Event Emission → App Controller → Backend API
                                                   ↓
UI Update ← Component Re-render ← Event Listener ← Response Processing
```

### Key Events

- `authStatusChanged`: Fired when authentication status changes
- Component state changes trigger re-rendering
- Form submission handled by main app controller

## Backend Integration

### Existing API Endpoints

- `POST /api/verify-cookie` - Cookie verification
- `GET /api/oauth/login` - OAuth initiation
- `GET /api/oauth/callback` - OAuth callback
- `POST /api/migrate` - Traditional migration
- `POST /api/migrate-custom` - Custom selection migration
- `POST /api/subreddits` - Fetch user subreddits
- `POST /api/saved-posts` - Fetch saved posts

### Request Format Examples

**Cookie Authentication:**

```javascript
{
  old_account_cookie: "cookie_string_here",
  new_account_cookie: "cookie_string_here",
  preferences: { ... }
}
```

**OAuth Authentication:**

```javascript
{
  old_account_cookie: "access_token_here",
  new_account_cookie: "access_token_here",
  preferences: { ... }
}
```

## Component Communication

### Event System

Components communicate via custom DOM events:

```javascript
// Emit event
const event = new CustomEvent("authStatusChanged", {
  detail: { sourceReady: true, destinationReady: false, method: "oauth" },
});
document.dispatchEvent(event);

// Listen for event
document.addEventListener("authStatusChanged", (e) => {
  console.log("Auth status:", e.detail);
});
```

### State Management

Each component manages its own state and exposes getter methods:

```javascript
// AuthComponent
authComponent.isReady(); // boolean
authComponent.getAuthData(); // {method, source, destination}

// MigrationFeature
migrationFeature.getSelectionData(); // {selections, selectedItems, batchSettings}
```

## Styling System

### CSS Architecture

- Tailwind CSS for utility classes
- Custom CSS for component-specific styles
- CSS variables for theme consistency
- Glassmorphism effects with backdrop-filter

### Key Style Classes

- `.glass-card` - Glassmorphism container
- `.btn-primary` - Orange Reddit-style button
- `.btn-secondary` - Blue accent button
- `.form-input` - Styled form inputs
- `.custom-radio` / `.custom-checkbox` - Custom form controls

### Responsive Design

- Mobile-first approach
- Grid layouts with responsive breakpoints
- Flexible component sizing

## Testing

### Test Page (`test.html`)

A simplified test page is provided to verify component integration:

- Loads all components
- Provides debug information
- Tests event communication
- Validates styling

### Debug Features

- Real-time debug info display
- Event logging
- Component state inspection
- Console output for troubleshooting

## Future Enhancements

### Integration with Existing Selection Modal

The current custom selection for migration features can be integrated with the existing `SelectionModal` class from the original `index.js`:

```javascript
// In MigrationFeature.js
async openSelectionModal(type) {
  const modal = new SelectionModal();
  await modal.open(type, authToken);
  // Handle selection results
}
```

### Content Cleanup Backend

The cleanup feature is designed but needs backend implementation:

- New endpoint: `POST /api/cleanup`
- Support for content deletion similar to ereddicator
- Batch processing with progress tracking

### OAuth Improvements

- Token refresh handling
- Multiple Reddit app support
- Better error handling and user feedback

## File Dependencies

```
index.html
├── components/DarkModeManager.js
├── components/AuthComponent.js
├── components/FeatureSelector.js
│   ├── components/MigrationFeature.js
│   └── components/CleanupFeature.js
└── app.js (main controller)
```

## Getting Started

1. **Development**: Use `test.html` for component testing
2. **Production**: Use `index.html` for the full application
3. **Debugging**: Check browser console and debug info section
4. **Backend**: Ensure all API endpoints are running
5. **OAuth**: Set up Reddit app credentials if using OAuth

## Component Guidelines

### Adding New Components

1. Create component class with constructor, render(), and attachEventListeners()
2. Emit events for state changes
3. Provide getter methods for data access
4. Follow existing styling patterns
5. Add to main app initialization

### State Management Best Practices

1. Keep component state internal
2. Use events for communication
3. Provide clear data interfaces
4. Handle loading and error states
5. Maintain UI consistency

This architecture provides a solid foundation for a comprehensive Reddit management tool that can be easily extended with additional features while maintaining clean separation of concerns.
