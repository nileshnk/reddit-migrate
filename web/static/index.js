let BOOL_OLD_TOKEN_VERIFIED = false;
let BOOL_NEW_TOKEN_VERIFIED = false;
let BOOL_MIGRATE_SUBREDDITS = false;
let BOOL_MIGRATE_SAVED_POSTS = false;
let BOOL_DELETE_SUBREDDITS = false;
let BOOL_DELETE_SAVED_POSTS = false;

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
    this.init();
  }

  init() {
    // Check for saved preference or default to light mode
    const isDarkMode = localStorage.getItem("darkMode") === "true";

    if (isDarkMode) {
      document.documentElement.classList.add("dark");
    }

    // Add event listener
    this.darkModeToggle.addEventListener("click", () => {
      this.toggle();
    });
  }

  toggle() {
    const isDarkMode = document.documentElement.classList.contains("dark");

    if (isDarkMode) {
      document.documentElement.classList.remove("dark");
      localStorage.setItem("darkMode", "false");
    } else {
      document.documentElement.classList.add("dark");
      localStorage.setItem("darkMode", "true");
    }
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
      const response = await fetch("/api/subreddits", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ cookie: getFullCookieString(token) }),
      });

      const data = await response.json();

      if (data.success) {
        ALL_SUBREDDITS = data.subreddits;
        filteredItems = [...ALL_SUBREDDITS];
        this.renderSubreddits();
      } else {
        throw new Error(data.message);
      }
    } catch (error) {
      console.error("Error loading subreddits:", error);
      this.itemsList.innerHTML = `<div class="p-4 text-red-500">Error loading subreddits: ${error.message}</div>`;
    }

    this.hideLoading();
  }

  async loadPosts(token) {
    this.showLoading();

    try {
      const response = await fetch("/api/saved-posts", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ cookie: getFullCookieString(token) }),
      });

      const data = await response.json();

      if (data.success) {
        ALL_POSTS = data.posts;
        filteredItems = [...ALL_POSTS];
        this.renderPosts();
      } else {
        throw new Error(data.message);
      }
    } catch (error) {
      console.error("Error loading posts:", error);
      this.itemsList.innerHTML = `<div class="p-4 text-red-500">Error loading posts: ${error.message}</div>`;
    }

    this.hideLoading();
  }

  renderSubreddits() {
    this.totalCount.textContent = filteredItems.length;

    const html = filteredItems
      .map((subreddit) => {
        const isSelected = SELECTED_SUBREDDITS.includes(subreddit.display_name);
        const iconUrl = subreddit.icon_img;
        const description =
          subreddit.public_description || "No description available";
        const subscriberCount = subreddit.subscribers
          ? formatNumber(subreddit.subscribers)
          : "Unknown";

        return `
                <div class="p-4 border-b hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer item-row" data-id="${
                  subreddit.display_name
                }">
                    <div class="flex items-center space-x-3">
                        <input type="checkbox" class="item-checkbox" data-id="${
                          subreddit.display_name
                        }" ${isSelected ? "checked" : ""}>
                        <div class="w-8 h-8 rounded-full flex-shrink-0 flex items-center justify-center ${
                          iconUrl ? "" : "bg-red-100 dark:bg-red-900"
                        }">
                            ${
                              iconUrl
                                ? `<img src="${iconUrl}" alt="${subreddit.display_name}" class="w-8 h-8 rounded-full object-cover" 
                                       onerror="this.style.display='none'; this.parentElement.innerHTML='<span class=\\'text-red-500 text-xs font-bold\\'>r/</span>';">`
                                : `<span class="text-red-500 text-xs font-bold">r/</span>`
                            }
                        </div>
                        <div class="flex-1 min-w-0">
                            <div class="flex items-center space-x-2">
                                <h4 class="font-medium text-gray-900 dark:text-gray-100 truncate">r/${
                                  subreddit.display_name
                                }</h4>
                                ${
                                  subreddit.over_18
                                    ? '<span class="px-2 py-1 text-xs bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200 rounded">NSFW</span>'
                                    : ""
                                }
                            </div>
                            <p class="text-sm text-gray-600 dark:text-gray-300 truncate">${
                              subreddit.title || subreddit.display_name
                            }</p>
                            <p class="text-xs text-gray-500 dark:text-gray-400 truncate">${description}</p>
                            <p class="text-xs text-gray-400 dark:text-gray-500">${subscriberCount} subscribers</p>
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
  }

  renderPosts() {
    this.totalCount.textContent = filteredItems.length;

    const html = filteredItems
      .map((post) => {
        const isSelected = SELECTED_POSTS.includes(post.full_name);
        const imageUrl = this.getPostImageUrl(post);
        const mediaTypeIcon = this.getMediaTypeIcon(post.image_data.media_type);
        const timeAgo = this.formatTimeAgo(post.created_utc);

        return `
                <div class="p-4 border-b hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer item-row" data-id="${
                  post.full_name
                }">
                    <div class="flex items-start space-x-3">
                        <input type="checkbox" class="item-checkbox mt-1" data-id="${
                          post.full_name
                        }" ${isSelected ? "checked" : ""}>
                        
                        <div class="flex-shrink-0">
                            ${
                              imageUrl
                                ? `<img src="${imageUrl}" alt="${post.title}" class="w-20 h-20 object-cover rounded-lg" 
                                     onerror="this.style.display='none'; this.nextElementSibling.style.display='flex';">
                                 <div class="w-20 h-20 bg-gray-200 dark:bg-gray-700 rounded-lg flex items-center justify-center text-gray-400 dark:text-gray-500 text-xs" style="display:none;">
                                     ${mediaTypeIcon}
                                 </div>`
                                : `<div class="w-20 h-20 bg-gray-200 dark:bg-gray-700 rounded-lg flex items-center justify-center text-gray-400 dark:text-gray-500 text-xs">
                                     ${mediaTypeIcon}
                                 </div>`
                            }
                        </div>
                        
                        <div class="flex-1 min-w-0">
                            <div class="flex items-start justify-between">
                                <h4 class="font-medium text-gray-900 dark:text-gray-100 text-sm leading-tight line-clamp-2">${
                                  post.title
                                }</h4>
                                <div class="flex items-center space-x-1 ml-2 flex-shrink-0">
                                    ${
                                      post.over_18
                                        ? '<span class="px-1 py-0.5 text-xs bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200 rounded">NSFW</span>'
                                        : ""
                                    }
                                    ${
                                      post.spoiler
                                        ? '<span class="px-1 py-0.5 text-xs bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200 rounded">Spoiler</span>'
                                        : ""
                                    }
                                </div>
                            </div>
                            
                            <div class="mt-1 flex items-center space-x-2 text-xs text-gray-500 dark:text-gray-400">
                                <span>r/${post.subreddit}</span>
                                <span>•</span>
                                <span>u/${post.author}</span>
                                <span>•</span>
                                <span>${timeAgo}</span>
                            </div>
                            
                            <div class="mt-1 flex items-center space-x-3 text-xs text-gray-400 dark:text-gray-500">
                                <span>⬆ ${formatNumber(post.score)}</span>
                                <span>💬 ${formatNumber(
                                  post.num_comments
                                )}</span>
                                <span class="truncate">${post.domain}</span>
                            </div>
                            
                            ${
                              post.selftext && post.selftext.length > 0
                                ? `<p class="mt-2 text-xs text-gray-600 dark:text-gray-300 line-clamp-2">${post.selftext.substring(
                                    0,
                                    150
                                  )}${
                                    post.selftext.length > 150 ? "..." : ""
                                  }</p>`
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
        // Don't trigger if clicking on checkbox directly
        if (e.target.type === "checkbox") return;

        const id = row.dataset.id;
        const checkbox = row.querySelector(".item-checkbox");
        if (checkbox) {
          checkbox.checked = !checkbox.checked;
          this.toggleSelection(id, checkbox.checked);
        }
      });
    });
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

function getFullCookieString(token) {
  // This is a simplified version - in practice you'd get the full cookie from the input
  return document.getElementById("oldAccessToken").value;
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
        if (!BOOL_OLD_TOKEN_VERIFIED) {
          alert("Please verify your old account cookie first");
          document.getElementById("subredditNone").checked = true;
          SUBREDDIT_SELECTION = "none";
          return;
        }

        await selectionModal.open("subreddits", OLD_ACCESS_TOKEN);
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

    if (e.target.value === "all") {
      updateSelectionSummary("posts", "all");
      toggleDeleteOptions("posts", true);
    } else if (e.target.value === "custom") {
      if (!BOOL_OLD_TOKEN_VERIFIED) {
        alert("Please verify your old account cookie first");
        document.getElementById("postsNone").checked = true;
        POSTS_SELECTION = "none";
        return;
      }

      await selectionModal.open("posts", OLD_ACCESS_TOKEN);
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
    if (!BOOL_OLD_TOKEN_VERIFIED) {
      alert("Please verify your old account cookie first");
      return;
    }
    // Maintain the custom radio button selection
    document.getElementById("subredditCustom").checked = true;
    SUBREDDIT_SELECTION = "custom";
    await selectionModal.open("subreddits", OLD_ACCESS_TOKEN);
  });

document
  .getElementById("editPostsSelection")
  .addEventListener("click", async () => {
    if (!BOOL_OLD_TOKEN_VERIFIED) {
      alert("Please verify your old account cookie first");
      return;
    }
    // Maintain the custom radio button selection
    document.getElementById("postsCustom").checked = true;
    POSTS_SELECTION = "custom";
    await selectionModal.open("posts", OLD_ACCESS_TOKEN);
  });

// Original migration logic (updated)
optionSubmit.addEventListener("click", async (e) => {
  e.preventDefault();

  if (!BOOL_OLD_TOKEN_VERIFIED || !BOOL_NEW_TOKEN_VERIFIED) {
    alert("Please verify both account cookies first");
    return;
  }

  optionSubmit.style.display = "none";
  loadingBtn.style.display = "block";

  const oldAccAccessToken = document.getElementById("oldAccessToken");
  const newAccAccessToken = document.getElementById("newAccessToken");

  migrateResponseBlock.style.display = "none";

  oldAccAccessToken.style.backgroundColor = "#e6e6e6";
  oldAccAccessToken.disabled = true;
  newAccAccessToken.style.backgroundColor = "#e6e6e6";
  newAccAccessToken.disabled = true;

  OLD_ACCESS_TOKEN = oldAccAccessToken.value;
  NEW_ACCESS_TOKEN = newAccAccessToken.value;

  const deleteSubreddits = document.getElementById(
    "deleteSubredditsYes"
  ).checked;
  const deletePosts = document.getElementById("deleteSavedPostsYes").checked;

  let requestBody;
  let endpoint;

  // Determine if we're using custom selection or traditional all/none
  if (SUBREDDIT_SELECTION === "custom" || POSTS_SELECTION === "custom") {
    // Use custom migration endpoint
    endpoint = "/api/migrate-custom";
    requestBody = {
      old_account_cookie: OLD_ACCESS_TOKEN,
      new_account_cookie: NEW_ACCESS_TOKEN,
      selected_subreddits:
        SUBREDDIT_SELECTION === "custom" ? SELECTED_SUBREDDITS : [],
      selected_posts: POSTS_SELECTION === "custom" ? SELECTED_POSTS : [],
      delete_old_subreddits: deleteSubreddits,
      delete_old_posts: deletePosts,
    };
  } else {
    // Use traditional migration endpoint
    endpoint = "/api/migrate";
    requestBody = {
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
    oldAccAccessToken.disabled = false;
    newAccAccessToken.disabled = false;
    oldAccAccessToken.style.backgroundColor = "";
    newAccAccessToken.style.backgroundColor = "";
  }
});

// Keep original functions (simplified)
function displayMigrationResponse(response) {
  const SubscribeSubreddit = document
    .getElementById("subscribe-subreddit")
    ?.querySelector("span");
  if (SubscribeSubreddit) {
    SubscribeSubreddit.innerText = `Total subreddits successfully subscribed to new account: ${response.data.subscribeSubreddit.SuccessCount}`;
  }

  const SavePost = document.getElementById("save-post")?.querySelector("span");
  if (SavePost) {
    SavePost.innerHTML = `Total posts successfully saved in new account: ${response.data.savePost.SuccessCount}`;
  }

  optionSubmit.style.display = "block";
  loadingBtn.style.display = "none";
  migrateResponseBlock.style.display = "block";
  migrateResponseBlock.scrollIntoView({ behavior: "smooth" });
}

function updateSubmitButtonState() {
  if (BOOL_OLD_TOKEN_VERIFIED && BOOL_NEW_TOKEN_VERIFIED) {
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
    oldAccAccessToken.style.borderColor = "#00cc00";
    oldTokenVerifyBtn.style.backgroundColor = "#00cc00";
    oldTokenVerifyBtn.disabled = true;
    oldTokenVerifyBtn.style.cursor = "default";
    oldTokenVerifyBtn.innerHTML = "Verified";

    document.getElementById("oldTokenVerifySuccessMessage").style.display =
      "block";
    document.getElementById("oldTokenVerifyFailMessage").style.display = "none";
    document.getElementById("oldAccountUserId").innerHTML =
      verifyOldToken.data.username;

    updateSubmitButtonState();
  } else {
    oldAccAccessToken.style.borderColor = "#ff0000";
    document.getElementById("oldTokenVerifyFailMessage").style.display =
      "block";
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
    newAccAccessToken.style.borderColor = "#00cc00";
    newTokenVerifyBtn.style.backgroundColor = "#00cc00";
    newTokenVerifyBtn.disabled = true;
    newTokenVerifyBtn.style.cursor = "default";
    newTokenVerifyBtn.innerHTML = "Verified";

    document.getElementById("newTokenVerifySuccessMessage").style.display =
      "block";
    document.getElementById("newTokenVerifyFailMessage").style.display = "none";
    document.getElementById("newAccountUserId").innerHTML =
      verifyNewToken.data.username;

    updateSubmitButtonState();
  } else {
    newAccAccessToken.style.borderColor = "#ff0000";
    document.getElementById("newTokenVerifyFailMessage").style.display =
      "block";
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

  const response = await fetch("/api/verify-cookie", {
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

// Initialize the page
document.addEventListener("DOMContentLoaded", () => {
  // Set default selections
  document.getElementById("subredditNone").checked = true;
  document.getElementById("postsNone").checked = true;
  document.getElementById("deleteSubredditsNo").checked = true;
  document.getElementById("deletePostsNo").checked = true;

  updateSubmitButtonState();

  // Initialize dark mode manager
  new DarkModeManager();
});
