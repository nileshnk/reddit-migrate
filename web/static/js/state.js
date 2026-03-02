// Centralized application state
// All global state lives here — other modules import from this file.

export const API_BASE_URL = "";

export let BOOL_SOURCE_TOKEN_VERIFIED = false;
export let BOOL_DEST_TOKEN_VERIFIED = false;

// Authentication method tracking
export let CURRENT_AUTH_METHOD = "cookie"; // "cookie" or "oauth"
export let OAUTH_SOURCE_VERIFIED = false;
export let OAUTH_DEST_VERIFIED = false;
export let SOURCE_ACCESS_TOKEN = "";
export let DEST_ACCESS_TOKEN = "";
export let SOURCE_USERNAME = "";
export let DEST_USERNAME = "";

// OAuth modal state
export let SOURCE_AUTH_METHOD = "oauth"; // "oauth" or "direct"
export let DEST_AUTH_METHOD = "oauth"; // "oauth" or "direct"

// Selection state
export let SUBREDDIT_SELECTION = "none"; // "all", "custom", "none"
export let POSTS_SELECTION = "none"; // "all", "custom", "none"
export let COMMENTS_SELECTION = "none"; // "all", "custom", "none"
export let SELECTED_SUBREDDITS = [];
export let SELECTED_POSTS = [];
export let SELECTED_COMMENTS = [];
export let ALL_SUBREDDITS = [];
export let ALL_POSTS = [];
export let ALL_COMMENTS = [];

// Cookie token storage
export let SOURCE_COOKIE_TOKEN = "";
export let DEST_COOKIE_TOKEN = "";

// Modal state
export let currentModalType = null; // "subreddits", "posts", or "comments"
export let filteredItems = [];

// Setters — needed because ES module exports are read-only bindings
export function setBoolSourceTokenVerified(v) { BOOL_SOURCE_TOKEN_VERIFIED = v; }
export function setBoolDestTokenVerified(v) { BOOL_DEST_TOKEN_VERIFIED = v; }
export function setCurrentAuthMethod(v) { CURRENT_AUTH_METHOD = v; }
export function setOAuthSourceVerified(v) { OAUTH_SOURCE_VERIFIED = v; }
export function setOAuthDestVerified(v) { OAUTH_DEST_VERIFIED = v; }
export function setSourceAccessToken(v) { SOURCE_ACCESS_TOKEN = v; }
export function setDestAccessToken(v) { DEST_ACCESS_TOKEN = v; }
export function setSourceUsername(v) { SOURCE_USERNAME = v; }
export function setDestUsername(v) { DEST_USERNAME = v; }
export function setSourceAuthMethod(v) { SOURCE_AUTH_METHOD = v; }
export function setDestAuthMethod(v) { DEST_AUTH_METHOD = v; }
export function setSubredditSelection(v) { SUBREDDIT_SELECTION = v; }
export function setPostsSelection(v) { POSTS_SELECTION = v; }
export function setCommentsSelection(v) { COMMENTS_SELECTION = v; }
export function setSelectedSubreddits(v) { SELECTED_SUBREDDITS = v; }
export function setSelectedPosts(v) { SELECTED_POSTS = v; }
export function setSelectedComments(v) { SELECTED_COMMENTS = v; }
export function setAllSubreddits(v) { ALL_SUBREDDITS = v; }
export function setAllPosts(v) { ALL_POSTS = v; }
export function setAllComments(v) { ALL_COMMENTS = v; }
export function setSourceCookieToken(v) { SOURCE_COOKIE_TOKEN = v; }
export function setDestCookieToken(v) { DEST_COOKIE_TOKEN = v; }
export function setCurrentModalType(v) { currentModalType = v; }
export function setFilteredItems(v) { filteredItems = v; }
