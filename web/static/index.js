// API Base URL Configuration
const API_BASE_URL = "";

let BOOL_OLD_TOKEN_VERIFIED = false;
let BOOL_NEW_TOKEN_VERIFIED = false;
let BOOL_MIGRATE_SUBREDDITS = false;
let BOOL_MIGRATE_SAVED_POSTS = false;
let BOOL_DELETE_SUBREDDITS = false;
let BOOL_DELETE_SAVED_POSTS = false;

// Authentication method tracking
let CURRENT_AUTH_METHOD = "cookie"; // "cookie" or "oauth"
let OAUTH_SOURCE_VERIFIED = false;
let OAUTH_DEST_VERIFIED = false;
let SOURCE_ACCESS_TOKEN = "";
let DEST_ACCESS_TOKEN = "";
let SOURCE_USERNAME = "";
let DEST_USERNAME = "";

// OAuth modal state
let SOURCE_AUTH_METHOD = "oauth"; // "oauth" or "direct"
let DEST_AUTH_METHOD = "oauth"; // "oauth" or "direct"

// New selection state variables
let SUBREDDIT_SELECTION = "none"; // "all", "custom", "none"
let POSTS_SELECTION = "none"; // "all", "custom", "none"
let SELECTED_SUBREDDITS = [];
let SELECTED_POSTS = [];
let ALL_SUBREDDITS = [];
let ALL_POSTS = [];

let OLD_ACCESS_TOKEN = "";
let NEW_ACCESS_TOKEN = "";

// Modal state
let currentModalType = null; // "subreddits" or "posts"
let filteredItems = [];

// Dark Mode Management
class DarkModeManager {
  constructor() {
    this.darkModeToggle = document.getElementById("darkModeToggle");
    this.themeIcon = document.getElementById("themeIcon");
    this.init();
  }

  init() {
    // Since we're using a primarily dark theme, let's adjust this logic
    // The toggle will switch between the dark theme and a lighter version
    const isLightMode = localStorage.getItem("lightMode") === "true";

    if (isLightMode) {
      this.enableLightMode();
    } else {
      this.enableDarkMode();
    }

    // Add event listener
    this.darkModeToggle.addEventListener("click", () => {
      this.toggle();
    });
  }

  toggle() {
    const isCurrentlyLight = document.body.classList.contains("light-mode");

    if (isCurrentlyLight) {
      this.enableDarkMode();
      localStorage.setItem("lightMode", "false");
    } else {
      this.enableLightMode();
      localStorage.setItem("lightMode", "true");
    }
  }

  enableDarkMode() {
    document.body.classList.remove("light-mode");
    document.body.className =
      "bg-gradient-to-br from-zinc-900 via-neutral-900 to-black text-white min-h-screen transition-all duration-300";
    this.themeIcon.textContent = "light_mode";
  }

  enableLightMode() {
    document.body.classList.add("light-mode");
    document.body.className =
      "bg-gradient-to-br from-white via-gray-50 to-gray-100 text-gray-900 min-h-screen transition-all duration-300 light-mode";
    this.themeIcon.textContent = "dark_mode";
  }
}

const optionSubmit = document.getElementById("option-submit");
const loadingBtn = document.getElementById("loading-btn");
const migrateResponseBlock = document.getElementById("migrate-response-block");
const migrateResponseData = document.getElementById("migrate-response-data");

// Selection Modal Management
class SelectionModal {
  constructor() {
    this.modal = document.getElementById("selectionModal");
    this.modalTitle = document.getElementById("modalTitle");
    this.searchInput = document.getElementById("searchInput");
    this.itemsList = document.getElementById("itemsList");
    this.selectedCount = document.getElementById("selectedCount");
    this.totalCount = document.getElementById("totalCount");
    this.selectAllBtn = document.getElementById("selectAllBtn");
    this.selectNoneBtn = document.getElementById("selectNoneBtn");
    this.confirmBtn = document.getElementById("confirmSelection");
    this.cancelBtn = document.getElementById("cancelSelection");
    this.closeBtn = document.getElementById("closeModal");
    this.loadingEl = document.getElementById("modalLoading");

    this.initEventListeners();
  }

  initEventListeners() {
    this.closeBtn.addEventListener("click", () => this.close());
    this.cancelBtn.addEventListener("click", () => this.close());
    this.confirmBtn.addEventListener("click", () => this.confirmSelection());
    this.selectAllBtn.addEventListener("click", () => this.selectAll());
    this.selectNoneBtn.addEventListener("click", () => this.selectNone());
    this.searchInput.addEventListener("input", (e) =>
      this.filterItems(e.target.value)
    );

    // Close modal when clicking outside
    this.modal.addEventListener("click", (e) => {
      if (e.target === this.modal) this.close();
    });
  }

  async open(type, token) {
    currentModalType = type;
    this.modal.classList.remove("hidden");

    if (type === "subreddits") {
      this.modalTitle.textContent = "Select Subreddits";
      await this.loadSubreddits(token);
    } else if (type === "posts") {
      this.modalTitle.textContent = "Select Saved Posts";
      await this.loadPosts(token);
    }
  }

  close() {
    this.modal.classList.add("hidden");
    currentModalType = null;
    this.searchInput.value = "";
  }

  showLoading() {
    this.loadingEl.classList.remove("hidden");
    this.itemsList.classList.add("hidden");
  }

  hideLoading() {
    this.loadingEl.classList.add("hidden");
    this.itemsList.classList.remove("hidden");
  }

  async loadSubreddits(token) {
    this.showLoading();

    try {
      const response = await fetch(`${API_BASE_URL}/api/subreddits`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(getAuthRequestBody()),
      });

      // Check if response is ok before trying to parse JSON
      if (!response.ok) {
        const errorText = await response.text();

        // Check for authentication/cookie errors
        if (
          response.status === 401 ||
          response.status === 403 ||
          errorText.toLowerCase().includes("token") ||
          errorText.toLowerCase().includes("expired") ||
          errorText.toLowerCase().includes("invalid") ||
          errorText.toLowerCase().includes("unauthorized")
        ) {
          throw new Error("COOKIE_EXPIRED");
        }

        throw new Error(`Server error: ${errorText}`);
      }

      const contentType = response.headers.get("content-type");
      if (!contentType || !contentType.includes("application/json")) {
        const text = await response.text();
        throw new Error(`Server returned non-JSON response: ${text}`);
      }

      const data = await response.json();

      if (data.success) {
        ALL_SUBREDDITS = data.subreddits || []; // Ensure it's always an array
        filteredItems = [...ALL_SUBREDDITS];
        this.renderSubreddits();
      } else {
        // Check if the error message indicates a cookie issue
        if (
          data.message &&
          (data.message.toLowerCase().includes("token") ||
            data.message.toLowerCase().includes("expired") ||
            data.message.toLowerCase().includes("invalid"))
        ) {
          throw new Error("COOKIE_EXPIRED");
        }
        throw new Error(data.message || "Failed to load subreddits");
      }
    } catch (error) {
      console.error("Error loading subreddits:", error);

      if (error.message === "COOKIE_EXPIRED") {
        this.itemsList.innerHTML = `
          <div class="p-8 text-center">
            <span class="material-icons text-6xl text-amber-400 mb-4 block">cookie</span>
            <p class="text-amber-400 font-semibold mb-2 text-lg">Cookie Expired or Invalid</p>
            <p class="text-slate-400 text-sm mb-4">Your Reddit authentication cookie has expired or is invalid.</p>
            <div class="bg-slate-700/30 rounded-lg p-4 text-left max-w-md mx-auto">
              <p class="text-slate-300 text-sm font-semibold mb-2">To get a new cookie:</p>
              <ol class="text-slate-400 text-xs space-y-1 list-decimal list-inside">
                <li>Open Reddit in a new tab and log in</li>
                <li>Open browser Developer Tools (F12)</li>
                <li>Go to Application/Storage → Cookies</li>
                <li>Find and copy the entire cookie string</li>
                <li>Paste it in the cookie field above</li>
              </ol>
            </div>
            <button onclick="location.reload()" class="mt-4 btn-primary px-6 py-2 text-white font-semibold rounded-lg flex items-center space-x-2 mx-auto">
              <span class="material-icons">refresh</span>
              <span>Refresh Page</span>
            </button>
          </div>
        `;
      } else {
        this.itemsList.innerHTML = `
          <div class="p-8 text-center">
            <span class="material-icons text-5xl text-red-400 mb-4 block">error_outline</span>
            <p class="text-red-400 font-semibold mb-2">Error Loading Subreddits</p>
            <p class="text-slate-400 text-sm">${error.message}</p>
            <p class="text-slate-500 text-xs mt-4">Please verify your cookie is valid and try again.</p>
          </div>
        `;
      }
    }

    this.hideLoading();
  }

  async loadPosts(token) {
    this.showLoading();

    try {
      const response = await fetch(`${API_BASE_URL}/api/saved-posts`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(getAuthRequestBody()),
      });

      // Check if response is ok before trying to parse JSON
      if (!response.ok) {
        const errorText = await response.text();

        // Check for authentication/cookie errors
        if (
          response.status === 401 ||
          response.status === 403 ||
          errorText.toLowerCase().includes("token") ||
          errorText.toLowerCase().includes("expired") ||
          errorText.toLowerCase().includes("invalid") ||
          errorText.toLowerCase().includes("unauthorized")
        ) {
          throw new Error("COOKIE_EXPIRED");
        }

        throw new Error(`Server error: ${errorText}`);
      }

      const contentType = response.headers.get("content-type");
      if (!contentType || !contentType.includes("application/json")) {
        const text = await response.text();
        throw new Error(`Server returned non-JSON response: ${text}`);
      }

      const data = await response.json();

      if (data.success) {
        ALL_POSTS = data.posts || []; // Ensure it's always an array
        filteredItems = [...ALL_POSTS];
        this.renderPosts();
      } else {
        // Check if the error message indicates a cookie issue
        if (
          data.message &&
          (data.message.toLowerCase().includes("token") ||
            data.message.toLowerCase().includes("expired") ||
            data.message.toLowerCase().includes("invalid"))
        ) {
          throw new Error("COOKIE_EXPIRED");
        }
        throw new Error(data.message || "Failed to load posts");
      }
    } catch (error) {
      console.error("Error loading posts:", error);

      if (error.message === "COOKIE_EXPIRED") {
        this.itemsList.innerHTML = `
          <div class="p-8 text-center">
            <span class="material-icons text-6xl text-amber-400 mb-4 block">cookie</span>
            <p class="text-amber-400 font-semibold mb-2 text-lg">Cookie Expired or Invalid</p>
            <p class="text-slate-400 text-sm mb-4">Your Reddit authentication cookie has expired or is invalid.</p>
            <div class="bg-slate-700/30 rounded-lg p-4 text-left max-w-md mx-auto">
              <p class="text-slate-300 text-sm font-semibold mb-2">To get a new cookie:</p>
              <ol class="text-slate-400 text-xs space-y-1 list-decimal list-inside">
                <li>Open Reddit in a new tab and log in</li>
                <li>Open browser Developer Tools (F12)</li>
                <li>Go to Application/Storage → Cookies</li>
                <li>Find and copy the entire cookie string</li>
                <li>Paste it in the cookie field above</li>
              </ol>
            </div>
            <button onclick="location.reload()" class="mt-4 btn-primary px-6 py-2 text-white font-semibold rounded-lg flex items-center space-x-2 mx-auto">
              <span class="material-icons">refresh</span>
              <span>Refresh Page</span>
            </button>
          </div>
        `;
      } else {
        this.itemsList.innerHTML = `
          <div class="p-8 text-center">
            <span class="material-icons text-5xl text-red-400 mb-4 block">error_outline</span>
            <p class="text-red-400 font-semibold mb-2">Error Loading Posts</p>
            <p class="text-slate-400 text-sm">${error.message}</p>
            <p class="text-slate-500 text-xs mt-4">Please verify your cookie is valid and try again.</p>
          </div>
        `;
      }
    }

    this.hideLoading();
  }

  renderSubreddits() {
    this.totalCount.textContent = filteredItems.length;

    // Check if no subreddits found
    if (filteredItems.length === 0) {
      const searchTerm = this.searchInput?.value?.trim() || "";

      if (searchTerm) {
        this.itemsList.innerHTML = `
          <div class="p-8 text-center">
            <span class="material-icons text-5xl text-red-400 mb-4 block">search</span>
            <p class="text-red-400 font-semibold mb-2">No Search Results</p>
            <p class="text-slate-400 text-sm mb-4">No subreddits found for "<span class="font-semibold">${searchTerm}</span>".</p>
            <button onclick="selectionModal.clearSearch()" class="btn-primary px-6 py-2 text-white font-semibold rounded-lg flex items-center space-x-2 mx-auto">
              <span class="material-icons">clear</span>
              <span>Clear Search</span>
            </button>
          </div>
        `;
      } else {
        this.itemsList.innerHTML = `
          <div class="p-8 text-center">
            <span class="material-icons text-5xl text-red-400 mb-4 block">groups</span>
            <p class="text-red-400 font-semibold mb-2">No Subreddits Found</p>
            <p class="text-slate-400 text-sm mb-4">You don't seem to be subscribed to any subreddits.</p>
            <div class="bg-slate-700/30 rounded-lg p-4 text-left max-w-md mx-auto">
              <p class="text-slate-300 text-sm font-semibold mb-2">To subscribe to a subreddit:</p>
              <ol class="text-slate-400 text-xs space-y-1 list-decimal list-inside">
                <li>Open Reddit in a new tab</li>
                <li>Find the subreddit you're interested in</li>
                <li>Click the "Join" button</li>
              </ol>
            </div>
          </div>
        `;
      }
    } else {
      const html = filteredItems
        .map((subreddit) => {
          const isSelected = SELECTED_SUBREDDITS.includes(
            subreddit.display_name
          );
          const iconUrl = subreddit.icon_img;
          const description =
            subreddit.public_description || "No description available";
          const subscriberCount = subreddit.subscribers
            ? formatNumber(subreddit.subscribers)
            : "Unknown";

          return `
                <div class="group p-4 border-b border-gray-200 dark:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer item-row transition-colors duration-150" data-id="${
                  subreddit.display_name
                }">
                    <div class="flex items-center space-x-4">
                        <div class="flex-shrink-0">
                            <label class="relative inline-flex items-center cursor-pointer">
                                <input type="checkbox" class="sr-only item-checkbox" data-id="${
                                  subreddit.display_name
                                }" ${isSelected ? "checked" : ""}>
                                <div class="checkbox-visual w-5 h-5 bg-white border-2 border-gray-300 rounded flex items-center justify-center transition-all duration-200 group-hover:border-red-400">
                                    <svg class="checkmark w-3 h-3 text-white hidden" fill="currentColor" viewBox="0 0 20 20">
                                        <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"></path>
                                    </svg>
                                </div>
                            </label>
                        </div>
                        
                        <div class="w-10 h-10 rounded-full flex-shrink-0 flex items-center justify-center ${
                          iconUrl ? "" : "bg-red-100 dark:bg-red-900"
                        }">
                            ${
                              iconUrl
                                ? `<img src="${iconUrl}" alt="${subreddit.display_name}" class="w-10 h-10 rounded-full object-cover border-2 border-gray-200 dark:border-gray-600" 
                                       onerror="this.style.display='none'; this.parentElement.innerHTML='<span class=\\'text-red-500 text-sm font-bold\\'>r/</span>';">`
                                : `<span class="text-red-500 text-sm font-bold">r/</span>`
                            }
                        </div>
                        
                        <div class="flex-1 min-w-0">
                            <div class="flex items-center space-x-2">
                                <a href="https://reddit.com/r/${
                                  subreddit.display_name
                                }" 
                                   target="_blank" 
                                   rel="noopener noreferrer"
                                   class="font-semibold text-gray-900 dark:text-gray-100 hover:text-red-600 dark:hover:text-red-400 transition-colors duration-150 truncate"
                                   onclick="event.stopPropagation()">
                                    r/${subreddit.display_name}
                                </a>
                                ${
                                  subreddit.over_18
                                    ? '<span class="px-2 py-1 text-xs bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200 rounded-full">NSFW</span>'
                                    : ""
                                }
                            </div>
                            <p class="text-sm text-gray-600 dark:text-gray-300 truncate font-medium">${
                              subreddit.title || subreddit.display_name
                            }</p>
                            <p class="text-xs text-gray-500 dark:text-gray-400 truncate leading-relaxed">${description}</p>
                            <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">
                                <span class="inline-flex items-center">
                                    <svg class="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
                                        <path d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                                    </svg>
                                    ${subscriberCount} subscribers
                                </span>
                            </p>
                        </div>
                    </div>
                </div>
            `;
        })
        .join("");

      this.itemsList.innerHTML = html;
      this.updateSelectedCount();
      this.attachCheckboxListeners();
      this.attachRowClickListeners();
      this.updateCheckboxVisuals();
    }
  }

  renderPosts() {
    this.totalCount.textContent = filteredItems.length;

    // Check if no saved posts found
    if (filteredItems.length === 0) {
      const searchTerm = this.searchInput?.value?.trim() || "";

      if (searchTerm) {
        this.itemsList.innerHTML = `
          <div class="p-8 text-center">
            <span class="material-icons text-5xl text-red-400 mb-4 block">search</span>
            <p class="text-red-400 font-semibold mb-2">No Search Results</p>
            <p class="text-slate-400 text-sm mb-4">No saved posts found for "<span class="font-semibold">${searchTerm}</span>".</p>
            <button onclick="selectionModal.clearSearch()" class="btn-primary px-6 py-2 text-white font-semibold rounded-lg flex items-center space-x-2 mx-auto">
              <span class="material-icons">clear</span>
              <span>Clear Search</span>
            </button>
          </div>
        `;
      } else {
        this.itemsList.innerHTML = `
          <div class="p-8 text-center">
            <span class="material-icons text-5xl text-red-400 mb-4 block">bookmark</span>
            <p class="text-red-400 font-semibold mb-2">No Saved Posts Found</p>
            <p class="text-slate-400 text-sm mb-4">You don't seem to have any saved posts.</p>
            <div class="bg-slate-700/30 rounded-lg p-4 text-left max-w-md mx-auto">
              <p class="text-slate-300 text-sm font-semibold mb-2">To save a post:</p>
              <ol class="text-slate-400 text-xs space-y-1 list-decimal list-inside">
                <li>Open Reddit in a new tab</li>
                <li>Find the post you're interested in</li>
                <li>Click the "Save" button</li>
              </ol>
            </div>
          </div>
        `;
      }
    } else {
      const html = filteredItems
        .map((post) => {
          const isSelected = SELECTED_POSTS.includes(post.full_name);
          const imageUrl = this.getPostImageUrl(post);
          const mediaTypeIcon = this.getMediaTypeIcon(
            post.image_data.media_type
          );
          const timeAgo = this.formatTimeAgo(post.created_utc);

          // Fix URL generation - check if permalink already contains full URL
          let postUrl;
          if (post.permalink && post.permalink.startsWith("http")) {
            postUrl = post.permalink;
          } else if (post.permalink) {
            postUrl = `https://reddit.com${post.permalink}`;
          } else {
            postUrl = `https://reddit.com/r/${post.subreddit}/comments/${post.id}/`;
          }

          return `
                <div class="group p-4 border-b border-gray-200 dark:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer item-row transition-colors duration-150" data-id="${
                  post.full_name
                }">
                    <div class="flex items-start space-x-4">
                        <div class="flex-shrink-0 flex items-center">
                            <label class="relative inline-flex items-center cursor-pointer">
                                <input type="checkbox" class="sr-only item-checkbox" data-id="${
                                  post.full_name
                                }" ${isSelected ? "checked" : ""}>
                                <div class="checkbox-visual w-5 h-5 bg-white border-2 border-gray-300 rounded flex items-center justify-center transition-all duration-200 group-hover:border-red-400">
                                    <svg class="checkmark w-3 h-3 text-white hidden" fill="currentColor" viewBox="0 0 20 20">
                                        <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"></path>
                                    </svg>
                                </div>
                            </label>
                        </div>
                        
                        <div class="flex-shrink-0">
                            ${
                              imageUrl
                                ? `<img src="${imageUrl}" alt="${post.title}" class="w-20 h-20 object-cover rounded-lg border-2 border-gray-200 dark:border-gray-600" 
                                     onerror="this.style.display='none'; this.nextElementSibling.style.display='flex';">
                                 <div class="w-20 h-20 bg-gray-200 dark:bg-gray-700 rounded-lg border-2 border-gray-200 dark:border-gray-600 flex items-center justify-center text-gray-400 dark:text-gray-500 text-sm" style="display:none;">
                                     ${mediaTypeIcon}
                                 </div>`
                                : `<div class="w-20 h-20 bg-gray-200 dark:bg-gray-700 rounded-lg border-2 border-gray-200 dark:border-gray-600 flex items-center justify-center text-gray-400 dark:text-gray-500 text-sm">
                                     ${mediaTypeIcon}
                                 </div>`
                            }
                        </div>
                        
                        <div class="flex-1 min-w-0">
                            <div class="flex items-start justify-between mb-2">
                                <a href="${postUrl}" 
                                   target="_blank" 
                                   rel="noopener noreferrer"
                                   class="font-medium text-gray-900 dark:text-gray-100 hover:text-red-600 dark:hover:text-red-400 transition-colors duration-150 text-sm leading-tight line-clamp-2 pr-2"
                                   onclick="event.stopPropagation()">
                                    ${post.title}
                                </a>
                                <div class="flex items-center space-x-1 flex-shrink-0">
                                    ${
                                      post.over_18
                                        ? '<span class="px-2 py-1 text-xs bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200 rounded-full">NSFW</span>'
                                        : ""
                                    }
                                    ${
                                      post.spoiler
                                        ? '<span class="px-2 py-1 text-xs bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200 rounded-full">Spoiler</span>'
                                        : ""
                                    }
                                </div>
                            </div>
                            
                            <div class="flex items-center space-x-2 text-xs text-gray-500 dark:text-gray-400 mb-2">
                                <a href="https://reddit.com/r/${
                                  post.subreddit
                                }" 
                                   target="_blank" 
                                   rel="noopener noreferrer"
                                   class="hover:text-red-600 dark:hover:text-red-400 transition-colors duration-150 font-medium"
                                   onclick="event.stopPropagation()">
                                    r/${post.subreddit}
                                </a>
                                <span>•</span>
                                <a href="https://reddit.com/u/${post.author}" 
                                   target="_blank" 
                                   rel="noopener noreferrer"
                                   class="hover:text-red-600 dark:hover:text-red-400 transition-colors duration-150"
                                   onclick="event.stopPropagation()">
                                    u/${post.author}
                                </a>
                                <span>•</span>
                                <span>${timeAgo}</span>
                            </div>
                            
                            <div class="flex items-center space-x-4 text-xs text-gray-400 dark:text-gray-500 mb-2">
                                <span class="inline-flex items-center">
                                    <svg class="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
                                        <path d="M2 10.5a1.5 1.5 0 113 0v6a1.5 1.5 0 01-3 0v-6zM6 10.333v5.43a2 2 0 001.106 1.79l.05.025A4 4 0 008.943 18h5.416a2 2 0 001.962-1.608l1.2-6A2 2 0 0015.56 8H12V4a2 2 0 00-2-2 1 1 0 00-1 1v.667a4 4 0 01-.8 2.4L6.8 7.933a4 4 0 00-.8 2.4z"></path>
                                    </svg>
                                    ${formatNumber(post.score)}
                                </span>
                                <span class="inline-flex items-center">
                                    <svg class="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
                                        <path fill-rule="evenodd" d="M18 10c0 3.866-3.582 7-8 7a8.841 8.841 0 01-4.083-.98L2 17l1.338-3.123C2.493 12.767 2 11.434 2 10c0-3.866 3.582-7 8-7s8 3.134 8 7zM7 9H5v2h2V9zm8 0h-2v2h2V9zM9 9h2v2H9V9z" clip-rule="evenodd"></path>
                                    </svg>
                                    ${formatNumber(post.num_comments)}
                                </span>
                                <span class="truncate text-blue-600 dark:text-blue-400">${
                                  post.domain
                                }</span>
                            </div>
                            
                            ${
                              post.selftext && post.selftext.length > 0
                                ? `<div class="bg-gray-50 dark:bg-gray-700 rounded-lg p-3 mt-2">
                                     <p class="text-xs text-gray-600 dark:text-gray-300 line-clamp-2">${post.selftext.substring(
                                       0,
                                       150
                                     )}${
                                    post.selftext.length > 150 ? "..." : ""
                                  }</p>
                                   </div>`
                                : ""
                            }
                        </div>
                    </div>
                </div>
            `;
        })
        .join("");

      this.itemsList.innerHTML = html;
      this.updateSelectedCount();
      this.attachCheckboxListeners();
      this.attachRowClickListeners();
      this.updateCheckboxVisuals();
    }
  }

  getPostImageUrl(post) {
    const imageData = post.image_data;
    if (imageData.preview_url) return imageData.preview_url;
    if (imageData.thumbnail_url) return imageData.thumbnail_url;
    if (imageData.high_res_url) return imageData.high_res_url;
    return null;
  }

  getMediaTypeIcon(mediaType) {
    switch (mediaType) {
      case "image":
        return "🖼️";
      case "video":
        return "🎥";
      case "gallery":
        return "🖼️";
      case "link":
        return "🔗";
      default:
        return "📄";
    }
  }

  formatTimeAgo(timestamp) {
    const now = Date.now() / 1000;
    const diff = now - timestamp;

    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
    if (diff < 2592000) return `${Math.floor(diff / 86400)}d ago`;
    return `${Math.floor(diff / 2592000)}mo ago`;
  }

  attachCheckboxListeners() {
    const checkboxes = this.itemsList.querySelectorAll(".item-checkbox");
    checkboxes.forEach((checkbox) => {
      checkbox.addEventListener("change", (e) => {
        e.stopPropagation(); // Prevent row click when clicking checkbox directly
        const id = e.target.dataset.id;
        this.toggleSelection(id, e.target.checked);
        this.updateCheckboxVisuals(); // Update visual state after toggle
      });
    });
  }

  toggleSelection(id, isChecked) {
    if (currentModalType === "subreddits") {
      if (isChecked) {
        if (!SELECTED_SUBREDDITS.includes(id)) {
          SELECTED_SUBREDDITS.push(id);
        }
      } else {
        SELECTED_SUBREDDITS = SELECTED_SUBREDDITS.filter((item) => item !== id);
      }
    } else if (currentModalType === "posts") {
      if (isChecked) {
        if (!SELECTED_POSTS.includes(id)) {
          SELECTED_POSTS.push(id);
        }
      } else {
        SELECTED_POSTS = SELECTED_POSTS.filter((item) => item !== id);
      }
    }
    this.updateSelectedCount();
  }

  updateSelectedCount() {
    const currentSelection =
      currentModalType === "subreddits" ? SELECTED_SUBREDDITS : SELECTED_POSTS;
    this.selectedCount.textContent = currentSelection.length;
  }

  filterItems(searchTerm) {
    const term = searchTerm.toLowerCase();

    if (currentModalType === "subreddits") {
      filteredItems = ALL_SUBREDDITS.filter(
        (subreddit) =>
          subreddit.display_name.toLowerCase().includes(term) ||
          (subreddit.title && subreddit.title.toLowerCase().includes(term)) ||
          (subreddit.public_description &&
            subreddit.public_description.toLowerCase().includes(term))
      );
      this.renderSubreddits();
    } else if (currentModalType === "posts") {
      filteredItems = ALL_POSTS.filter(
        (post) =>
          post.title.toLowerCase().includes(term) ||
          post.subreddit.toLowerCase().includes(term) ||
          post.author.toLowerCase().includes(term) ||
          (post.selftext && post.selftext.toLowerCase().includes(term))
      );
      this.renderPosts();
    }
  }

  selectAll() {
    if (currentModalType === "subreddits") {
      filteredItems.forEach((subreddit) => {
        if (!SELECTED_SUBREDDITS.includes(subreddit.display_name)) {
          SELECTED_SUBREDDITS.push(subreddit.display_name);
        }
      });
      this.renderSubreddits();
    } else if (currentModalType === "posts") {
      filteredItems.forEach((post) => {
        if (!SELECTED_POSTS.includes(post.full_name)) {
          SELECTED_POSTS.push(post.full_name);
        }
      });
      this.renderPosts();
    }
  }

  selectNone() {
    if (currentModalType === "subreddits") {
      filteredItems.forEach((subreddit) => {
        SELECTED_SUBREDDITS = SELECTED_SUBREDDITS.filter(
          (item) => item !== subreddit.display_name
        );
      });
      this.renderSubreddits();
    } else if (currentModalType === "posts") {
      filteredItems.forEach((post) => {
        SELECTED_POSTS = SELECTED_POSTS.filter(
          (item) => item !== post.full_name
        );
      });
      this.renderPosts();
    }
  }

  confirmSelection() {
    if (currentModalType === "subreddits") {
      SUBREDDIT_SELECTION = "custom";
      updateSelectionSummary(
        "subreddits",
        "custom",
        SELECTED_SUBREDDITS.length
      );
    } else if (currentModalType === "posts") {
      POSTS_SELECTION = "custom";
      updateSelectionSummary("posts", "custom", SELECTED_POSTS.length);
    }
    this.close();
  }

  attachRowClickListeners() {
    const rows = this.itemsList.querySelectorAll(".item-row");
    rows.forEach((row) => {
      row.addEventListener("click", (e) => {
        // Don't trigger if clicking on checkbox, links, or any interactive elements
        if (
          e.target.type === "checkbox" ||
          e.target.tagName === "A" ||
          e.target.closest("a") ||
          e.target.closest("label")
        ) {
          return;
        }

        const id = row.dataset.id;
        const checkbox = row.querySelector(".item-checkbox");
        if (checkbox) {
          checkbox.checked = !checkbox.checked;
          this.toggleSelection(id, checkbox.checked);
          this.updateCheckboxVisuals(); // Update visual state after row click
        }
      });
    });
  }

  updateCheckboxVisuals() {
    const rows = this.itemsList.querySelectorAll(".item-row");
    rows.forEach((row) => {
      const checkbox = row.querySelector(".item-checkbox");
      const isChecked = checkbox.checked;
      const checkboxVisual = checkbox
        .closest("label")
        .querySelector(".checkbox-visual");
      const checkmark = checkboxVisual.querySelector(".checkmark");

      // Update checkbox visual state
      if (isChecked) {
        checkboxVisual.classList.add("bg-red-500", "border-red-500");
        checkboxVisual.classList.remove("bg-white", "border-gray-300");
        checkmark.classList.remove("hidden");
      } else {
        checkboxVisual.classList.remove("bg-red-500", "border-red-500");
        checkboxVisual.classList.add("bg-white", "border-gray-300");
        checkmark.classList.add("hidden");
      }

      // Update row highlighting
      if (isChecked) {
        row.classList.add("bg-gray-100", "dark:bg-gray-600");
        row.classList.remove("hover:bg-gray-50", "dark:hover:bg-gray-700");
      } else {
        row.classList.remove("bg-gray-100", "dark:bg-gray-600");
        row.classList.add("hover:bg-gray-50", "dark:hover:bg-gray-700");
      }
    });
  }

  clearSearch() {
    this.searchInput.value = "";
    this.filterItems("");
  }
}

// Initialize modal
const selectionModal = new SelectionModal();

// Helper functions
function formatNumber(num) {
  if (num >= 1000000) return (num / 1000000).toFixed(1) + "M";
  if (num >= 1000) return (num / 1000).toFixed(1) + "K";
  return num.toString();
}

function getAuthRequestBody() {
  console.log(
    "getAuthRequestBody called - Current auth method:",
    CURRENT_AUTH_METHOD
  );
  console.log("Authentication state:", {
    OAUTH_SOURCE_VERIFIED,
    SOURCE_ACCESS_TOKEN: SOURCE_ACCESS_TOKEN
      ? SOURCE_ACCESS_TOKEN.substring(0, 10) + "..."
      : "none",
    SOURCE_USERNAME,
    OLD_ACCESS_TOKEN: OLD_ACCESS_TOKEN
      ? OLD_ACCESS_TOKEN.substring(0, 10) + "..."
      : "none",
  });

  // Return the appropriate authentication data based on authentication method
  if (CURRENT_AUTH_METHOD === "oauth") {
    const body = {
      auth_method: "oauth",
      access_token: SOURCE_ACCESS_TOKEN,
      username: SOURCE_USERNAME,
    };
    console.log("OAuth auth request body:", {
      auth_method: body.auth_method,
      access_token: body.access_token
        ? body.access_token.substring(0, 10) + "..."
        : "none",
      username: body.username,
    });
    return body;
  } else {
    const body = {
      auth_method: "cookie",
      cookie: OLD_ACCESS_TOKEN,
    };
    console.log("Cookie auth request body:", {
      auth_method: body.auth_method,
      cookie: body.cookie ? body.cookie.substring(0, 10) + "..." : "none",
    });
    return body;
  }
}

// OAuth Modal Management Functions
function showOAuthModal(type) {
  const modal = document.getElementById(`${type}OAuthModal`);
  modal.classList.remove("hidden");

  // Reset modal state
  resetOAuthModal(type);
}

function hideOAuthModal(type) {
  const modal = document.getElementById(`${type}OAuthModal`);
  modal.classList.add("hidden");
}

function resetOAuthModal(type) {
  // Reset form fields
  document.getElementById(`${type}ModalClientId`).value = "";
  document.getElementById(`${type}ModalClientSecret`).value = "";
  document.getElementById(`${type}ModalUsername`).value = "";
  document.getElementById(`${type}ModalPassword`).value = "";

  // Reset UI state
  document.getElementById(`${type}ModalSuccessMessage`).classList.add("hidden");
  document.getElementById(`${type}ModalErrorMessage`).classList.add("hidden");
  document.getElementById(`${type}ModalVerifyBtn`).style.display = "block";
  document.getElementById(`${type}ModalLoadBtn`).style.display = "none";

  // Reset auth method to OAuth flow
  switchModalAuthMethod(type, "oauth");
}

function switchModalAuthMethod(type, method) {
  const isOAuth = method === "oauth";
  const oauthBtn = document.getElementById(`${type}ModalOAuthFlowBtn`);
  const directBtn = document.getElementById(`${type}ModalDirectAuthBtn`);
  const directFields = document.getElementById(`${type}ModalDirectAuthFields`);
  const verifyBtnText = document.getElementById(`${type}ModalVerifyBtnText`);

  if (isOAuth) {
    oauthBtn.classList.add("active");
    directBtn.classList.remove("active");
    directFields.classList.add("hidden");
    verifyBtnText.textContent = "Verify & Authenticate";
  } else {
    directBtn.classList.add("active");
    oauthBtn.classList.remove("active");
    directFields.classList.remove("hidden");
    verifyBtnText.textContent = "Login";
  }

  // Update method state
  if (type === "source") {
    SOURCE_AUTH_METHOD = method;
  } else {
    DEST_AUTH_METHOD = method;
  }
}

// Helper functions for authentication checks
function isSourceAccountVerified() {
  if (CURRENT_AUTH_METHOD === "cookie") {
    return BOOL_OLD_TOKEN_VERIFIED;
  } else if (CURRENT_AUTH_METHOD === "oauth") {
    return OAUTH_SOURCE_VERIFIED;
  }
  return false;
}

function isDestAccountVerified() {
  if (CURRENT_AUTH_METHOD === "cookie") {
    return BOOL_NEW_TOKEN_VERIFIED;
  } else if (CURRENT_AUTH_METHOD === "oauth") {
    return OAUTH_DEST_VERIFIED;
  }
  return false;
}

function getSourceAccessToken() {
  if (CURRENT_AUTH_METHOD === "cookie") {
    return OLD_ACCESS_TOKEN;
  } else if (CURRENT_AUTH_METHOD === "oauth") {
    return SOURCE_ACCESS_TOKEN;
  }
  return "";
}

function getDestAccessToken() {
  if (CURRENT_AUTH_METHOD === "cookie") {
    return NEW_ACCESS_TOKEN;
  } else if (CURRENT_AUTH_METHOD === "oauth") {
    return DEST_ACCESS_TOKEN;
  }
  return "";
}

function updateSelectionSummary(type, selection, count = 0) {
  if (type === "subreddits") {
    const summaryEl = document.getElementById("subredditSelectionSummary");
    const countEl = document.getElementById("selectedSubredditCount");
    const editBtn = document.getElementById("editSubredditSelection");

    if (selection === "all") {
      summaryEl.classList.remove("hidden");
      countEl.textContent = "All";
      editBtn.style.display = "none";
    } else if (selection === "custom") {
      summaryEl.classList.remove("hidden");
      countEl.textContent = count;
      editBtn.style.display = "inline-block";
    } else {
      summaryEl.classList.add("hidden");
    }
  } else if (type === "posts") {
    const summaryEl = document.getElementById("postsSelectionSummary");
    const countEl = document.getElementById("selectedPostsCount");
    const editBtn = document.getElementById("editPostsSelection");

    if (selection === "all") {
      summaryEl.classList.remove("hidden");
      countEl.textContent = "All";
      editBtn.style.display = "none";
    } else if (selection === "custom") {
      summaryEl.classList.remove("hidden");
      countEl.textContent = count;
      editBtn.style.display = "inline-block";
    } else {
      summaryEl.classList.add("hidden");
    }
  }
}

function toggleDeleteOptions(type, show) {
  const deleteEl = document.getElementById(
    type === "subreddits" ? "deleteSubreddits" : "deletePosts"
  );
  if (show) {
    deleteEl.classList.remove("hidden");
  } else {
    deleteEl.classList.add("hidden");
    // Reset delete options to "No" when hiding
    const noRadio = deleteEl.querySelector('input[value="no"]');
    if (noRadio) noRadio.checked = true;
  }
}

// Event listeners for selection radio buttons
document
  .querySelectorAll('input[name="subredditSelection"]')
  .forEach((radio) => {
    radio.addEventListener("change", async (e) => {
      SUBREDDIT_SELECTION = e.target.value;

      if (e.target.value === "all") {
        updateSelectionSummary("subreddits", "all");
        toggleDeleteOptions("subreddits", true);
      } else if (e.target.value === "custom") {
        if (!isSourceAccountVerified()) {
          alert("Please verify your source account first");
          document.getElementById("subredditNone").checked = true;
          SUBREDDIT_SELECTION = "none";
          return;
        }

        await selectionModal.open("subreddits", getSourceAccessToken());
        toggleDeleteOptions("subreddits", true);
      } else {
        updateSelectionSummary("subreddits", "none");
        toggleDeleteOptions("subreddits", false);
        SELECTED_SUBREDDITS = [];
      }
    });
  });

document.querySelectorAll('input[name="postsSelection"]').forEach((radio) => {
  radio.addEventListener("change", async (e) => {
    POSTS_SELECTION = e.target.value;
    console.log("Posts selection changed to:", e.target.value);
    console.log("Current authentication state:", {
      CURRENT_AUTH_METHOD,
      isSourceAccountVerified: isSourceAccountVerified(),
      OAUTH_SOURCE_VERIFIED,
      SOURCE_USERNAME,
      SOURCE_ACCESS_TOKEN: SOURCE_ACCESS_TOKEN
        ? SOURCE_ACCESS_TOKEN.substring(0, 10) + "..."
        : "none",
    });

    if (e.target.value === "all") {
      updateSelectionSummary("posts", "all");
      toggleDeleteOptions("posts", true);
    } else if (e.target.value === "custom") {
      if (!isSourceAccountVerified()) {
        console.log(
          "Source account not verified, blocking custom posts selection"
        );
        alert("Please verify your source account first");
        document.getElementById("postsNone").checked = true;
        POSTS_SELECTION = "none";
        return;
      }

      console.log(
        "Opening posts selection modal with token:",
        getSourceAccessToken()
          ? getSourceAccessToken().substring(0, 10) + "..."
          : "none"
      );
      await selectionModal.open("posts", getSourceAccessToken());
      toggleDeleteOptions("posts", true);
    } else {
      updateSelectionSummary("posts", "none");
      toggleDeleteOptions("posts", false);
      SELECTED_POSTS = [];
    }
  });
});

// Edit selection button event listeners
document
  .getElementById("editSubredditSelection")
  .addEventListener("click", async () => {
    if (!isSourceAccountVerified()) {
      alert("Please verify your source account first");
      return;
    }
    // Maintain the custom radio button selection
    document.getElementById("subredditCustom").checked = true;
    SUBREDDIT_SELECTION = "custom";
    await selectionModal.open("subreddits", getSourceAccessToken());
  });

document
  .getElementById("editPostsSelection")
  .addEventListener("click", async () => {
    if (!isSourceAccountVerified()) {
      alert("Please verify your source account first");
      return;
    }
    // Maintain the custom radio button selection
    document.getElementById("postsCustom").checked = true;
    POSTS_SELECTION = "custom";
    await selectionModal.open("posts", getSourceAccessToken());
  });

// Original migration logic (updated)
optionSubmit.addEventListener("click", async (e) => {
  e.preventDefault();

  // Check authentication based on current method
  if (!isSourceAccountVerified() || !isDestAccountVerified()) {
    alert("Please verify both accounts first");
    return;
  }

  optionSubmit.style.display = "none";
  loadingBtn.style.display = "block";

  migrateResponseBlock.style.display = "none";

  // Get tokens based on authentication method
  if (CURRENT_AUTH_METHOD === "cookie") {
    const oldAccAccessToken = document.getElementById("oldAccessToken");
    const newAccAccessToken = document.getElementById("newAccessToken");

    oldAccAccessToken.style.backgroundColor = "#e6e6e6";
    oldAccAccessToken.disabled = true;
    newAccAccessToken.style.backgroundColor = "#e6e6e6";
    newAccAccessToken.disabled = true;

    OLD_ACCESS_TOKEN = oldAccAccessToken.value;
    NEW_ACCESS_TOKEN = newAccAccessToken.value;
  } else if (CURRENT_AUTH_METHOD === "oauth") {
    OLD_ACCESS_TOKEN = SOURCE_ACCESS_TOKEN;
    NEW_ACCESS_TOKEN = DEST_ACCESS_TOKEN;
  }

  const deleteSubreddits = document.getElementById(
    "deleteSubredditsYes"
  ).checked;
  const deletePosts = document.getElementById("deleteSavedPostsYes").checked;

  let requestBody;
  let endpoint;

  // Determine if we're using custom selection or traditional all/none
  if (SUBREDDIT_SELECTION === "custom" || POSTS_SELECTION === "custom") {
    // Use custom migration endpoint
    endpoint = `${API_BASE_URL}/api/migrate-custom`;
    if (CURRENT_AUTH_METHOD === "oauth") {
      requestBody = {
        auth_method: "oauth",
        old_account_token: OLD_ACCESS_TOKEN,
        new_account_token: NEW_ACCESS_TOKEN,
        old_account_username: SOURCE_USERNAME,
        new_account_username: DEST_USERNAME,
        selected_subreddits:
          SUBREDDIT_SELECTION === "custom" ? SELECTED_SUBREDDITS : [],
        selected_posts: POSTS_SELECTION === "custom" ? SELECTED_POSTS : [],
        delete_old_subreddits: deleteSubreddits,
        delete_old_posts: deletePosts,
      };
    } else {
      requestBody = {
        auth_method: "cookie",
        old_account_cookie: OLD_ACCESS_TOKEN,
        new_account_cookie: NEW_ACCESS_TOKEN,
        selected_subreddits:
          SUBREDDIT_SELECTION === "custom" ? SELECTED_SUBREDDITS : [],
        selected_posts: POSTS_SELECTION === "custom" ? SELECTED_POSTS : [],
        delete_old_subreddits: deleteSubreddits,
        delete_old_posts: deletePosts,
      };
    }
  } else {
    // Use traditional migration endpoint
    endpoint = `${API_BASE_URL}/api/migrate`;
    if (CURRENT_AUTH_METHOD === "oauth") {
      requestBody = {
        auth_method: "oauth",
        old_account_token: OLD_ACCESS_TOKEN,
        new_account_token: NEW_ACCESS_TOKEN,
        old_account_username: SOURCE_USERNAME,
        new_account_username: DEST_USERNAME,
        preferences: {
          migrate_subreddit_bool: SUBREDDIT_SELECTION === "all",
          migrate_post_bool: POSTS_SELECTION === "all",
          delete_post_bool: deletePosts,
          delete_subreddit_bool: deleteSubreddits,
        },
      };
    } else {
      requestBody = {
        auth_method: "cookie",
        old_account_cookie: OLD_ACCESS_TOKEN,
        new_account_cookie: NEW_ACCESS_TOKEN,
        preferences: {
          migrate_subreddit_bool: SUBREDDIT_SELECTION === "all",
          migrate_post_bool: POSTS_SELECTION === "all",
          delete_post_bool: deletePosts,
          delete_subreddit_bool: deleteSubreddits,
        },
      };
    }
  }

  console.log("Starting migration with:", {
    endpoint,
    selections: { subreddits: SUBREDDIT_SELECTION, posts: POSTS_SELECTION },
  });

  try {
    const migrateResponse = await fetch(endpoint, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(requestBody),
    });

    const response = await migrateResponse.json();
    console.log(response);

    if (migrateResponse.status === 200) {
      displayMigrationResponse(response);
    } else {
      throw new Error(response.message || "Migration failed");
    }
  } catch (error) {
    console.error("Migration error:", error);
    alert("Migration failed: " + error.message);

    // Re-enable form
    optionSubmit.style.display = "block";
    loadingBtn.style.display = "none";

    if (CURRENT_AUTH_METHOD === "cookie") {
      const oldAccAccessToken = document.getElementById("oldAccessToken");
      const newAccAccessToken = document.getElementById("newAccessToken");
      oldAccAccessToken.disabled = false;
      newAccAccessToken.disabled = false;
      oldAccAccessToken.style.backgroundColor = "";
      newAccAccessToken.style.backgroundColor = "";
    }
  }
});

// Keep original functions (simplified)
function displayMigrationResponse(response) {
  // Clear previous response data
  migrateResponseData.innerHTML = "";

  // Check what was actually migrated and create appropriate status elements
  const migratingSubreddits =
    SUBREDDIT_SELECTION === "all" ||
    (SUBREDDIT_SELECTION === "custom" && SELECTED_SUBREDDITS.length > 0);
  const migratingPosts =
    POSTS_SELECTION === "all" ||
    (POSTS_SELECTION === "custom" && SELECTED_POSTS.length > 0);

  // Create subreddit status if subreddits were migrated
  if (migratingSubreddits && response.data.subscribeSubreddit) {
    const subredditStatusElement = document.createElement("li");
    subredditStatusElement.className =
      "flex items-center space-x-3 p-3 bg-emerald-900/20 rounded-lg border border-emerald-500/20";
    subredditStatusElement.innerHTML = `
      <span class="material-icons text-emerald-400">check_circle</span>
      <span class="text-sm font-medium text-slate-300">
        Total subreddits successfully subscribed to new account: 
        <span class="text-emerald-400 font-bold">${response.data.subscribeSubreddit.SuccessCount}</span>
      </span>
    `;
    migrateResponseData.appendChild(subredditStatusElement);
  }

  // Create post status if posts were migrated
  if (migratingPosts && response.data.savePost) {
    const postStatusElement = document.createElement("li");
    postStatusElement.className =
      "flex items-center space-x-3 p-3 bg-emerald-900/20 rounded-lg border border-emerald-500/20";
    postStatusElement.innerHTML = `
      <span class="material-icons text-emerald-400">check_circle</span>
      <span class="text-sm font-medium text-slate-300">
        Total posts successfully saved in new account: 
        <span class="text-emerald-400 font-bold">${response.data.savePost.SuccessCount}</span>
      </span>
    `;
    migrateResponseData.appendChild(postStatusElement);
  }

  // If nothing was migrated, show a message
  if (!migratingSubreddits && !migratingPosts) {
    const noMigrationElement = document.createElement("li");
    noMigrationElement.className =
      "flex items-center space-x-3 p-3 bg-amber-900/20 rounded-lg border border-amber-500/20";
    noMigrationElement.innerHTML = `
      <span class="material-icons text-amber-400">info</span>
      <span class="text-sm font-medium text-slate-300">
        No items were selected for migration
      </span>
    `;
    migrateResponseData.appendChild(noMigrationElement);
  }

  optionSubmit.style.display = "block";
  loadingBtn.style.display = "none";
  migrateResponseBlock.style.display = "block";
  migrateResponseBlock.scrollIntoView({ behavior: "smooth" });
}

function updateSubmitButtonState() {
  const isVerified = isSourceAccountVerified() && isDestAccountVerified();

  if (isVerified) {
    optionSubmit.disabled = false;
    optionSubmit.classList.remove("cursor-not-allowed", "opacity-50");
  } else {
    optionSubmit.disabled = true;
    optionSubmit.classList.add("cursor-not-allowed", "opacity-50");
  }
}

// Keep original token verification logic
const oldTokenVerifyBtn = document.getElementById("oldTokenVerifyBtn");
const verifyLoadBtn1 = document.getElementById("verify-load-btn-1");
const newTokenVerifyBtn = document.getElementById("newTokenVerifyBtn");
const verifyLoadBtn2 = document.getElementById("verify-load-btn-2");

oldTokenVerifyBtn.addEventListener("click", async (e) => {
  e.preventDefault();
  oldTokenVerifyBtn.style.display = "none";
  verifyLoadBtn1.style.display = "block";

  const oldAccAccessToken = document.getElementById("oldAccessToken");
  const oldAccAccessTokenValue = oldAccAccessToken.value;
  const verifyOldToken = await verifyCookie(oldAccAccessTokenValue);

  oldTokenVerifyBtn.style.display = "block";
  verifyLoadBtn1.style.display = "none";

  if (verifyOldToken.success) {
    BOOL_OLD_TOKEN_VERIFIED = true;
    OLD_ACCESS_TOKEN = oldAccAccessTokenValue;

    oldAccAccessToken.disabled = true;
    oldAccAccessToken.style.borderColor = "#10b981";

    // Update button to verified state
    oldTokenVerifyBtn.className =
      "btn-verified px-6 py-3 text-white font-semibold rounded-xl flex items-center space-x-2";
    oldTokenVerifyBtn.disabled = true;
    oldTokenVerifyBtn.style.cursor = "default";
    oldTokenVerifyBtn.innerHTML = `
      <span class="material-icons text-lg">verified</span>
      <span>Verified</span>
    `;

    document.getElementById("oldTokenVerifySuccessMessage").style.display =
      "flex";
    document.getElementById("oldTokenVerifyFailMessage").style.display = "none";
    document.getElementById("oldAccountUserId").innerHTML =
      verifyOldToken.data.username;

    updateSubmitButtonState();
  } else {
    oldAccAccessToken.style.borderColor = "#ef4444";
    document.getElementById("oldTokenVerifyFailMessage").style.display = "flex";
    document.getElementById("oldTokenVerifySuccessMessage").style.display =
      "none";
  }
});

newTokenVerifyBtn.addEventListener("click", async (e) => {
  e.preventDefault();
  newTokenVerifyBtn.style.display = "none";
  verifyLoadBtn2.style.display = "block";

  const newAccAccessToken = document.getElementById("newAccessToken");
  const newAccAccessTokenValue = newAccAccessToken.value;
  const verifyNewToken = await verifyCookie(newAccAccessTokenValue);

  // Always reset the buttons first
  newTokenVerifyBtn.style.display = "block";
  verifyLoadBtn2.style.display = "none";

  if (verifyNewToken.success) {
    BOOL_NEW_TOKEN_VERIFIED = true;
    NEW_ACCESS_TOKEN = newAccAccessTokenValue;

    newAccAccessToken.disabled = true;
    newAccAccessToken.style.borderColor = "#10b981";

    // Update button to verified state
    newTokenVerifyBtn.className =
      "btn-verified px-6 py-3 text-white font-semibold rounded-xl flex items-center space-x-2";
    newTokenVerifyBtn.disabled = true;
    newTokenVerifyBtn.style.cursor = "default";
    newTokenVerifyBtn.innerHTML = `
      <span class="material-icons text-lg">verified</span>
      <span>Verified</span>
    `;

    document.getElementById("newTokenVerifySuccessMessage").style.display =
      "flex";
    document.getElementById("newTokenVerifyFailMessage").style.display = "none";
    document.getElementById("newAccountUserId").innerHTML =
      verifyNewToken.data.username;

    updateSubmitButtonState();
  } else {
    newAccAccessToken.style.borderColor = "#ef4444";
    document.getElementById("newTokenVerifyFailMessage").style.display = "flex";
    document.getElementById("newTokenVerifySuccessMessage").style.display =
      "none";
  }
});

async function verifyCookie(cookie) {
  const cookieData = getCookieObject(cookie);
  if (cookieData.token_v2 === undefined) {
    return {
      success: false,
      message: "Invalid Cookie. Please get a new one.",
      data: {},
    };
  }

  const response = await fetch(`${API_BASE_URL}/api/verify-cookie`, {
    body: JSON.stringify({ cookie: cookie }),
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
  });
  const body = await response.json();

  if (response.status !== 200) {
    return {
      success: false,
      message: "Invalid Cookie",
      data: {},
    };
  }

  return body;
}

function getCookieObject(cookie) {
  const pairs = cookie.split(";");
  const cookieObject = {};
  for (const pair of pairs) {
    const [name, value] = pair.trim().split("=");
    cookieObject[name] = value;
  }
  return cookieObject;
}

function toggleTooltip(event) {
  event.stopPropagation();
  const tooltip = document.getElementById("tooltip");
  tooltip.classList.toggle("hidden");
}

// Close tooltip if clicked outside
document.addEventListener("click", function (event) {
  const tooltip = document.getElementById("tooltip");
  const tooltipButton = tooltip.previousElementSibling;
  if (
    !tooltip.classList.contains("hidden") &&
    !tooltip.contains(event.target) &&
    !tooltipButton.contains(event.target)
  ) {
    tooltip.classList.add("hidden");
  }
});

// Password Visibility Toggle Function
function togglePasswordVisibility(inputId, eyeIconId) {
  const input = document.getElementById(inputId);
  const eyeIcon = document.getElementById(eyeIconId);

  if (input.type === "password") {
    input.type = "text";
    eyeIcon.textContent = "visibility_off";
  } else {
    input.type = "password";
    eyeIcon.textContent = "visibility";
  }
}

// Cookie Help Modal Functions
function showCookieHelp() {
  const modal = document.getElementById("cookieHelpModal");
  modal.classList.remove("hidden");
  modal.classList.add("animate-slide-in");
}

function hideCookieHelp() {
  const modal = document.getElementById("cookieHelpModal");
  modal.classList.add("hidden");
}

// OAuth Help Modal Functions
function showOAuthHelp() {
  const modal = document.getElementById("oauthHelpModal");
  modal.classList.remove("hidden");
  modal.classList.add("animate-slide-in");
}

function hideOAuthHelp() {
  const modal = document.getElementById("oauthHelpModal");
  modal.classList.add("hidden");
}

// Close modal when clicking outside
document
  .getElementById("cookieHelpModal")
  ?.addEventListener("click", function (event) {
    if (event.target === this) {
      hideCookieHelp();
    }
  });

document
  .getElementById("oauthHelpModal")
  ?.addEventListener("click", function (event) {
    if (event.target === this) {
      hideOAuthHelp();
    }
  });

// Tab Management
class TabManager {
  constructor() {
    this.cookieTab = document.getElementById("cookieTab");
    this.oauthTab = document.getElementById("oauthTab");
    this.cookieContent = document.getElementById("cookieAuthContent");
    this.oauthContent = document.getElementById("oauthAuthContent");

    console.log("TabManager elements:", {
      cookieTab: this.cookieTab,
      oauthTab: this.oauthTab,
      cookieContent: this.cookieContent,
      oauthContent: this.oauthContent,
    });

    this.init();
  }

  init() {
    if (
      !this.cookieTab ||
      !this.oauthTab ||
      !this.cookieContent ||
      !this.oauthContent
    ) {
      console.error("Tab elements not found. Check HTML element IDs.");
      return;
    }

    this.cookieTab.addEventListener("click", (e) => {
      e.preventDefault();
      console.log("Cookie tab clicked");
      this.switchTab("cookie");
    });

    this.oauthTab.addEventListener("click", (e) => {
      e.preventDefault();
      console.log("OAuth tab clicked");
      this.switchTab("oauth");
    });
  }

  switchTab(method) {
    console.log("Switching to tab:", method);
    CURRENT_AUTH_METHOD = method;

    // Update tab appearance
    if (method === "cookie") {
      console.log("Switching to cookie tab");
      this.cookieTab.classList.add("active");
      this.oauthTab.classList.remove("active");
      this.cookieContent.classList.remove("hidden");
      this.oauthContent.classList.add("hidden");
    } else {
      console.log("Switching to oauth tab");
      this.oauthTab.classList.add("active");
      this.cookieTab.classList.remove("active");
      this.oauthContent.classList.remove("hidden");
      this.cookieContent.classList.add("hidden");
    }

    console.log("Tab classes after switch:", {
      cookieTab: this.cookieTab.className,
      oauthTab: this.oauthTab.className,
      cookieContent: this.cookieContent.className,
      oauthContent: this.oauthContent.className,
    });

    // Update submit button state
    updateSubmitButtonState();
  }
}

// OAuth Modal Management
class OAuthModalManager {
  constructor() {
    this.init();
  }

  init() {
    // Modal button event listeners
    document
      .getElementById("sourceOAuthModalBtn")
      ?.addEventListener("click", () => {
        showOAuthModal("source");
      });

    document
      .getElementById("destOAuthModalBtn")
      ?.addEventListener("click", () => {
        showOAuthModal("dest");
      });

    // Modal authentication method toggles
    document
      .getElementById("sourceModalOAuthFlowBtn")
      ?.addEventListener("click", () => {
        switchModalAuthMethod("source", "oauth");
      });

    document
      .getElementById("sourceModalDirectAuthBtn")
      ?.addEventListener("click", () => {
        switchModalAuthMethod("source", "direct");
      });

    document
      .getElementById("destModalOAuthFlowBtn")
      ?.addEventListener("click", () => {
        switchModalAuthMethod("dest", "oauth");
      });

    document
      .getElementById("destModalDirectAuthBtn")
      ?.addEventListener("click", () => {
        switchModalAuthMethod("dest", "direct");
      });

    // Modal verify buttons
    document
      .getElementById("sourceModalVerifyBtn")
      ?.addEventListener("click", () => {
        this.verifyModalAuth("source");
      });

    document
      .getElementById("destModalVerifyBtn")
      ?.addEventListener("click", () => {
        this.verifyModalAuth("dest");
      });
  }

  async verifyModalAuth(type) {
    const authMethod =
      type === "source" ? SOURCE_AUTH_METHOD : DEST_AUTH_METHOD;

    if (authMethod === "oauth") {
      await this.verifyModalOAuth(type);
    } else {
      await this.verifyModalDirect(type);
    }
  }

  async verifyModalOAuth(type) {
    const clientId = document
      .getElementById(`${type}ModalClientId`)
      .value.trim();
    const clientSecret = document
      .getElementById(`${type}ModalClientSecret`)
      .value.trim();

    if (!clientId || !clientSecret) {
      alert("Please enter both Client ID and Client Secret");
      return;
    }

    this.showModalLoading(type);

    try {
      const initResponse = await fetch(`${API_BASE_URL}/api/oauth/init`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          client_id: clientId,
          client_secret: clientSecret,
          account_type: type,
        }),
      });

      if (!initResponse.ok) {
        throw new Error("Failed to initialize OAuth");
      }

      const initData = await initResponse.json();
      const authWindow = window.open(
        initData.authorization_url,
        "oauth",
        "width=600,height=600"
      );

      const checkAuthComplete = setInterval(async () => {
        try {
          if (authWindow.closed) {
            clearInterval(checkAuthComplete);

            const statusResponse = await fetch(
              `${API_BASE_URL}/api/oauth/status?account_type=${type}`
            );
            if (statusResponse.ok) {
              const statusData = await statusResponse.json();

              if (statusData.success) {
                this.handleModalAuthSuccess(
                  type,
                  statusData.username,
                  statusData.access_token
                );
              } else {
                throw new Error(
                  statusData.message || "OAuth verification failed"
                );
              }
            } else {
              throw new Error("Failed to check OAuth status");
            }
          }
        } catch (error) {
          clearInterval(checkAuthComplete);
          this.handleModalAuthError(type, error);
        }
      }, 1000);
    } catch (error) {
      this.handleModalAuthError(type, error);
    }

    this.hideModalLoading(type);
  }

  async verifyModalDirect(type) {
    const clientId = document
      .getElementById(`${type}ModalClientId`)
      .value.trim();
    const clientSecret = document
      .getElementById(`${type}ModalClientSecret`)
      .value.trim();
    const username = document
      .getElementById(`${type}ModalUsername`)
      .value.trim();
    const password = document
      .getElementById(`${type}ModalPassword`)
      .value.trim();

    if (!clientId || !clientSecret || !username || !password) {
      alert("Please enter all required fields");
      return;
    }

    this.showModalLoading(type);

    try {
      const response = await fetch(`${API_BASE_URL}/api/oauth/direct`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          client_id: clientId,
          client_secret: clientSecret,
          username: username,
          password: password,
          account_type: type,
        }),
      });

      if (!response.ok) {
        let errorMessage = "Direct authentication failed";
        try {
          const errorData = await response.json();
          errorMessage = errorData.message || errorMessage;
        } catch (parseError) {
          errorMessage = `HTTP ${response.status}: ${response.statusText}`;
        }
        throw new Error(errorMessage);
      }

      const data = await response.json();

      if (data.success) {
        this.handleModalAuthSuccess(type, data.username, data.access_token);
      } else {
        throw new Error(data.message || "Direct authentication failed");
      }
    } catch (error) {
      this.handleModalAuthError(type, error);
    }

    this.hideModalLoading(type);
  }

  handleModalAuthSuccess(type, username, accessToken) {
    console.log(`OAuth authentication successful for ${type}:`, {
      username,
      accessToken: accessToken.substring(0, 10) + "...",
    });

    // Automatically switch to OAuth mode when OAuth authentication succeeds
    if (CURRENT_AUTH_METHOD !== "oauth") {
      console.log("Switching to OAuth mode after successful authentication");
      CURRENT_AUTH_METHOD = "oauth";

      // Update tab appearance if needed
      const tabManager = document.querySelector("#oauthTab");
      const cookieTab = document.querySelector("#cookieTab");
      const oauthContent = document.querySelector("#oauthAuthContent");
      const cookieContent = document.querySelector("#cookieAuthContent");

      if (tabManager && cookieTab && oauthContent && cookieContent) {
        tabManager.classList.add("active");
        cookieTab.classList.remove("active");
        oauthContent.classList.remove("hidden");
        cookieContent.classList.add("hidden");
      }
    }

    // Update global state
    if (type === "source") {
      OAUTH_SOURCE_VERIFIED = true;
      SOURCE_ACCESS_TOKEN = accessToken;
      SOURCE_USERNAME = username;
      console.log("Set SOURCE_USERNAME to:", username);
      document.getElementById("sourceOAuthUsername").textContent = username;
      document.getElementById("sourceOAuthStatus").classList.remove("hidden");
    } else {
      OAUTH_DEST_VERIFIED = true;
      DEST_ACCESS_TOKEN = accessToken;
      DEST_USERNAME = username;
      console.log("Set DEST_USERNAME to:", username);
      document.getElementById("destOAuthUsername").textContent = username;
      document.getElementById("destOAuthStatus").classList.remove("hidden");
    }

    // Show success message in modal
    document.getElementById(`${type}ModalSuccessUsername`).textContent =
      username;
    document
      .getElementById(`${type}ModalSuccessMessage`)
      .classList.remove("hidden");
    document.getElementById(`${type}ModalErrorMessage`).classList.add("hidden");

    updateSubmitButtonState();

    console.log("Updated authentication state:", {
      CURRENT_AUTH_METHOD,
      OAUTH_SOURCE_VERIFIED,
      OAUTH_DEST_VERIFIED,
      SOURCE_USERNAME,
      DEST_USERNAME,
    });
  }

  handleModalAuthError(type, error) {
    console.error("Modal authentication error:", error);
    document.getElementById(`${type}ModalErrorText`).textContent =
      error.message || "Authentication failed";
    document
      .getElementById(`${type}ModalErrorMessage`)
      .classList.remove("hidden");
    document
      .getElementById(`${type}ModalSuccessMessage`)
      .classList.add("hidden");
  }

  showModalLoading(type) {
    document.getElementById(`${type}ModalVerifyBtn`).style.display = "none";
    document.getElementById(`${type}ModalLoadBtn`).style.display = "block";
  }

  hideModalLoading(type) {
    document.getElementById(`${type}ModalVerifyBtn`).style.display = "block";
    document.getElementById(`${type}ModalLoadBtn`).style.display = "none";
  }
}

// Initialize the page
document.addEventListener("DOMContentLoaded", () => {
  console.log("DOM Content Loaded - Initializing application");

  // Set default selections
  document.getElementById("subredditNone").checked = true;
  document.getElementById("postsNone").checked = true;
  document.getElementById("deleteSubredditsNo").checked = true;
  document.getElementById("deletePostsNo").checked = true;

  updateSubmitButtonState();

  // Initialize managers
  try {
    new DarkModeManager();
    console.log("DarkModeManager initialized");

    new TabManager();
    console.log("TabManager initialized");

    new OAuthModalManager();
    console.log("OAuthModalManager initialized");
  } catch (error) {
    console.error("Error initializing managers:", error);
  }
});
