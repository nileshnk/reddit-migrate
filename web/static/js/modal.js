// SelectionModal class and selection event listeners

import * as state from "./state.js";
import { formatNumber, formatTimeAgo, getPostImageUrl, getMediaTypeIcon } from "./utils.js";
import { getAuthRequestBody, isSourceAccountVerified, getSourceAccessToken } from "./auth.js";

// Selection UI helpers
export function updateSelectionSummary(type, selection, count = 0) {
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
  } else if (type === "comments") {
    const summaryEl = document.getElementById("commentsSelectionSummary");
    const countEl = document.getElementById("selectedCommentsCount");
    const editBtn = document.getElementById("editCommentsSelection");

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
  } else if (type === "multireddits") {
    const summaryEl = document.getElementById("multiredditSelectionSummary");
    const countEl = document.getElementById("selectedMultiredditCount");
    const editBtn = document.getElementById("editMultiredditSelection");

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

export function toggleDeleteOptions(type, show) {
  let deleteElId;
  if (type === "subreddits") {
    deleteElId = "deleteSubreddits";
  } else if (type === "posts") {
    deleteElId = "deletePosts";
  } else {
    deleteElId = "deleteComments";
  }
  const deleteEl = document.getElementById(deleteElId);
  if (show) {
    deleteEl.classList.remove("hidden");
  } else {
    deleteEl.classList.add("hidden");
    const noRadio = deleteEl.querySelector('input[value="no"]');
    if (noRadio) noRadio.checked = true;
  }
}

// SelectionModal class
export class SelectionModal {
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

    this.modal.addEventListener("click", (e) => {
      if (e.target === this.modal) this.close();
    });
  }

  async open(type, token) {
    state.setCurrentModalType(type);
    this.modal.classList.remove("hidden");

    if (type === "subreddits") {
      this.modalTitle.textContent = "Select Subreddits";
      await this.loadSubreddits(token);
    } else if (type === "posts") {
      this.modalTitle.textContent = "Select Saved Posts";
      await this.loadPosts(token);
    } else if (type === "comments") {
      this.modalTitle.textContent = "Select Saved Comments";
      await this.loadComments(token);
    } else if (type === "multireddits") {
      this.modalTitle.textContent = "Select Multireddits";
      await this.loadMultireddits(token);
    }
  }

  close() {
    this.modal.classList.add("hidden");
    state.setCurrentModalType(null);
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
      const response = await fetch(`${state.API_BASE_URL}/api/subreddits`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(getAuthRequestBody()),
      });

      if (!response.ok) {
        const errorText = await response.text();

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
        state.setAllSubreddits(data.subreddits || []);
        state.setFilteredItems([...state.ALL_SUBREDDITS]);
        this.renderSubreddits();
      } else {
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
      const response = await fetch(`${state.API_BASE_URL}/api/saved-posts`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(getAuthRequestBody()),
      });

      if (!response.ok) {
        const errorText = await response.text();

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
        state.setAllPosts(data.posts || []);
        state.setFilteredItems([...state.ALL_POSTS]);
        this.renderPosts();
      } else {
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

  async loadComments(token) {
    this.showLoading();

    try {
      const response = await fetch(`${state.API_BASE_URL}/api/saved-comments`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(getAuthRequestBody()),
      });

      if (!response.ok) {
        const errorText = await response.text();
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
        state.setAllComments(data.comments || []);
        state.setFilteredItems([...state.ALL_COMMENTS]);
        this.renderComments();
      } else {
        if (
          data.message &&
          (data.message.toLowerCase().includes("token") ||
            data.message.toLowerCase().includes("expired") ||
            data.message.toLowerCase().includes("invalid"))
        ) {
          throw new Error("COOKIE_EXPIRED");
        }
        throw new Error(data.message || "Failed to load comments");
      }
    } catch (error) {
      console.error("Error loading comments:", error);

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
            <p class="text-red-400 font-semibold mb-2">Error Loading Comments</p>
            <p class="text-slate-400 text-sm">${error.message}</p>
            <p class="text-slate-500 text-xs mt-4">Please verify your cookie is valid and try again.</p>
          </div>
        `;
      }
    }

    this.hideLoading();
  }

  renderComments() {
    this.totalCount.textContent = state.filteredItems.length;

    if (state.filteredItems.length === 0) {
      const searchTerm = this.searchInput?.value?.trim() || "";

      if (searchTerm) {
        this.itemsList.innerHTML = `
          <div class="p-8 text-center">
            <span class="material-icons text-5xl text-red-400 mb-4 block">search</span>
            <p class="text-red-400 font-semibold mb-2">No Search Results</p>
            <p class="text-slate-400 text-sm mb-4">No saved comments found for "<span class="font-semibold">${searchTerm}</span>".</p>
            <button onclick="window._selectionModal.clearSearch()" class="btn-primary px-6 py-2 text-white font-semibold rounded-lg flex items-center space-x-2 mx-auto">
              <span class="material-icons">clear</span>
              <span>Clear Search</span>
            </button>
          </div>
        `;
      } else {
        this.itemsList.innerHTML = `
          <div class="p-8 text-center">
            <span class="material-icons text-5xl text-red-400 mb-4 block">comment</span>
            <p class="text-red-400 font-semibold mb-2">No Saved Comments Found</p>
            <p class="text-slate-400 text-sm mb-4">You don't seem to have any saved comments.</p>
          </div>
        `;
      }
    } else {
      const html = state.filteredItems
        .map((comment) => {
          const isSelected = state.SELECTED_COMMENTS.includes(comment.full_name);
          const timeAgo = formatTimeAgo(comment.created_utc);

          let commentUrl;
          if (comment.permalink && comment.permalink.startsWith("http")) {
            commentUrl = comment.permalink;
          } else if (comment.permalink) {
            commentUrl = `https://reddit.com${comment.permalink}`;
          } else {
            commentUrl = "#";
          }

          const bodySnippet = comment.body_text
            ? comment.body_text.substring(0, 200) +
              (comment.body_text.length > 200 ? "..." : "")
            : "[deleted]";

          return `
                <div class="group p-4 border-b border-gray-200 dark:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer item-row transition-colors duration-150" data-id="${comment.full_name}">
                    <div class="flex items-start space-x-4">
                        <div class="flex-shrink-0 flex items-center">
                            <label class="relative inline-flex items-center cursor-pointer">
                                <input type="checkbox" class="sr-only item-checkbox" data-id="${comment.full_name}" ${isSelected ? "checked" : ""}>
                                <div class="checkbox-visual w-5 h-5 bg-white border-2 border-gray-300 rounded flex items-center justify-center transition-all duration-200 group-hover:border-red-400">
                                    <svg class="checkmark w-3 h-3 text-white hidden" fill="currentColor" viewBox="0 0 20 20">
                                        <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"></path>
                                    </svg>
                                </div>
                            </label>
                        </div>

                        <div class="flex-shrink-0">
                            <div class="w-10 h-10 bg-gray-200 dark:bg-gray-700 rounded-lg border-2 border-gray-200 dark:border-gray-600 flex items-center justify-center text-gray-400 dark:text-gray-500">
                                <span class="material-icons text-sm">comment</span>
                            </div>
                        </div>

                        <div class="flex-1 min-w-0">
                            ${
                              comment.link_title
                                ? `<div class="text-xs text-slate-400 mb-1 truncate">
                                     on: <span class="font-medium">${comment.link_title}</span>
                                   </div>`
                                : ""
                            }

                            <div class="bg-gray-50 dark:bg-gray-700 rounded-lg p-3 mb-2">
                                <a href="${commentUrl}"
                                   target="_blank"
                                   rel="noopener noreferrer"
                                   class="text-sm text-gray-700 dark:text-gray-200 hover:text-red-600 dark:hover:text-red-400 transition-colors duration-150 line-clamp-3"
                                   onclick="event.stopPropagation()">
                                    ${bodySnippet}
                                </a>
                            </div>

                            <div class="flex items-center space-x-2 text-xs text-gray-500 dark:text-gray-400">
                                <a href="https://reddit.com/r/${comment.subreddit}"
                                   target="_blank"
                                   rel="noopener noreferrer"
                                   class="hover:text-red-600 dark:hover:text-red-400 transition-colors duration-150 font-medium"
                                   onclick="event.stopPropagation()">
                                    r/${comment.subreddit}
                                </a>
                                <span>&bull;</span>
                                <a href="https://reddit.com/u/${comment.author}"
                                   target="_blank"
                                   rel="noopener noreferrer"
                                   class="hover:text-red-600 dark:hover:text-red-400 transition-colors duration-150"
                                   onclick="event.stopPropagation()">
                                    u/${comment.author}
                                </a>
                                <span>&bull;</span>
                                <span>${timeAgo}</span>
                                <span>&bull;</span>
                                <span class="inline-flex items-center">
                                    <svg class="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
                                        <path d="M2 10.5a1.5 1.5 0 113 0v6a1.5 1.5 0 01-3 0v-6zM6 10.333v5.43a2 2 0 001.106 1.79l.05.025A4 4 0 008.943 18h5.416a2 2 0 001.962-1.608l1.2-6A2 2 0 0015.56 8H12V4a2 2 0 00-2-2 1 1 0 00-1 1v.667a4 4 0 01-.8 2.4L6.8 7.933a4 4 0 00-.8 2.4z"></path>
                                    </svg>
                                    ${formatNumber(comment.score)}
                                </span>
                                ${
                                  comment.over_18
                                    ? '<span class="px-2 py-1 text-xs bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200 rounded-full">NSFW</span>'
                                    : ""
                                }
                            </div>
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

  async loadMultireddits(token) {
    this.showLoading();

    try {
      const response = await fetch(`${state.API_BASE_URL}/api/multireddits`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(getAuthRequestBody()),
      });

      if (!response.ok) {
        const errorText = await response.text();
        if (
          response.status === 401 || response.status === 403 ||
          errorText.toLowerCase().includes("token") ||
          errorText.toLowerCase().includes("expired")
        ) {
          throw new Error("COOKIE_EXPIRED");
        }
        throw new Error(`Server error: ${errorText}`);
      }

      const data = await response.json();

      if (data.success) {
        state.setAllMultireddits(data.multireddits || []);
        state.setFilteredItems([...state.ALL_MULTIREDDITS]);
        this.renderMultireddits();
      } else {
        throw new Error(data.message || "Failed to load multireddits");
      }
    } catch (error) {
      console.error("Error loading multireddits:", error);

      if (error.message === "COOKIE_EXPIRED") {
        this.itemsList.innerHTML = `
          <div class="p-8 text-center">
            <span class="material-icons text-6xl text-amber-400 mb-4 block">cookie</span>
            <p class="text-amber-400 font-semibold mb-2 text-lg">Cookie Expired or Invalid</p>
            <p class="text-slate-400 text-sm mb-4">Your Reddit authentication cookie has expired or is invalid.</p>
          </div>
        `;
      } else {
        this.itemsList.innerHTML = `
          <div class="p-8 text-center">
            <span class="material-icons text-5xl text-red-400 mb-4 block">error_outline</span>
            <p class="text-red-400 font-semibold mb-2">Error Loading Multireddits</p>
            <p class="text-slate-400 text-sm">${error.message}</p>
          </div>
        `;
      }
    }

    this.hideLoading();
  }

  renderMultireddits() {
    this.totalCount.textContent = state.filteredItems.length;

    if (state.filteredItems.length === 0) {
      const searchTerm = this.searchInput?.value?.trim() || "";

      if (searchTerm) {
        this.itemsList.innerHTML = `
          <div class="p-8 text-center">
            <span class="material-icons text-5xl text-red-400 mb-4 block">search</span>
            <p class="text-red-400 font-semibold mb-2">No Search Results</p>
            <p class="text-slate-400 text-sm mb-4">No multireddits found for "<span class="font-semibold">${searchTerm}</span>".</p>
            <button onclick="window._selectionModal.clearSearch()" class="btn-primary px-6 py-2 text-white font-semibold rounded-lg flex items-center space-x-2 mx-auto">
              <span class="material-icons">clear</span>
              <span>Clear Search</span>
            </button>
          </div>
        `;
      } else {
        this.itemsList.innerHTML = `
          <div class="p-8 text-center">
            <span class="material-icons text-5xl text-red-400 mb-4 block">playlist_add</span>
            <p class="text-red-400 font-semibold mb-2">No Multireddits Found</p>
            <p class="text-slate-400 text-sm mb-4">You don't have any custom feeds / multireddits.</p>
          </div>
        `;
      }
    } else {
      const html = state.filteredItems
        .map((multi) => {
          const isSelected = state.SELECTED_MULTIREDDITS.includes(multi.name);
          const subCount = multi.subreddits ? multi.subreddits.length : 0;
          const subList = multi.subreddits
            ? multi.subreddits.slice(0, 5).join(", ") + (multi.subreddits.length > 5 ? ` +${multi.subreddits.length - 5} more` : "")
            : "No subreddits";
          const description = multi.description_md
            ? multi.description_md.substring(0, 100) + (multi.description_md.length > 100 ? "..." : "")
            : "";

          return `
                <div class="group p-4 border-b border-gray-200 dark:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer item-row transition-colors duration-150" data-id="${multi.name}">
                    <div class="flex items-start space-x-4">
                        <div class="flex-shrink-0 flex items-center">
                            <label class="relative inline-flex items-center cursor-pointer">
                                <input type="checkbox" class="sr-only item-checkbox" data-id="${multi.name}" ${isSelected ? "checked" : ""}>
                                <div class="checkbox-visual w-5 h-5 bg-white border-2 border-gray-300 rounded flex items-center justify-center transition-all duration-200 group-hover:border-red-400">
                                    <svg class="checkmark w-3 h-3 text-white hidden" fill="currentColor" viewBox="0 0 20 20">
                                        <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"></path>
                                    </svg>
                                </div>
                            </label>
                        </div>

                        <div class="flex-shrink-0">
                            <div class="w-10 h-10 bg-blue-100 dark:bg-blue-900/30 rounded-lg border-2 border-blue-200 dark:border-blue-700 flex items-center justify-center text-blue-500 dark:text-blue-400">
                                <span class="material-icons text-sm">playlist_play</span>
                            </div>
                        </div>

                        <div class="flex-1 min-w-0">
                            <h3 class="font-semibold text-sm text-gray-900 dark:text-white truncate">
                                ${multi.display_name || multi.name}
                            </h3>
                            ${description ? `<p class="text-xs text-slate-400 mt-1 line-clamp-2">${description}</p>` : ""}
                            <div class="flex items-center space-x-2 text-xs text-gray-500 dark:text-gray-400 mt-2">
                                <span class="inline-flex items-center">
                                    <span class="material-icons text-xs mr-1">forum</span>
                                    ${subCount} subreddit${subCount !== 1 ? "s" : ""}
                                </span>
                                <span>&bull;</span>
                                <span class="truncate">${subList}</span>
                                ${multi.visibility ? `<span>&bull;</span><span>${multi.visibility}</span>` : ""}
                            </div>
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

  renderSubreddits() {
    this.totalCount.textContent = state.filteredItems.length;

    if (state.filteredItems.length === 0) {
      const searchTerm = this.searchInput?.value?.trim() || "";

      if (searchTerm) {
        this.itemsList.innerHTML = `
          <div class="p-8 text-center">
            <span class="material-icons text-5xl text-red-400 mb-4 block">search</span>
            <p class="text-red-400 font-semibold mb-2">No Search Results</p>
            <p class="text-slate-400 text-sm mb-4">No subreddits found for "<span class="font-semibold">${searchTerm}</span>".</p>
            <button onclick="window._selectionModal.clearSearch()" class="btn-primary px-6 py-2 text-white font-semibold rounded-lg flex items-center space-x-2 mx-auto">
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
      const html = state.filteredItems
        .map((subreddit) => {
          const isSelected = state.SELECTED_SUBREDDITS.includes(
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
    this.totalCount.textContent = state.filteredItems.length;

    if (state.filteredItems.length === 0) {
      const searchTerm = this.searchInput?.value?.trim() || "";

      if (searchTerm) {
        this.itemsList.innerHTML = `
          <div class="p-8 text-center">
            <span class="material-icons text-5xl text-red-400 mb-4 block">search</span>
            <p class="text-red-400 font-semibold mb-2">No Search Results</p>
            <p class="text-slate-400 text-sm mb-4">No saved posts found for "<span class="font-semibold">${searchTerm}</span>".</p>
            <button onclick="window._selectionModal.clearSearch()" class="btn-primary px-6 py-2 text-white font-semibold rounded-lg flex items-center space-x-2 mx-auto">
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
      const html = state.filteredItems
        .map((post) => {
          const isSelected = state.SELECTED_POSTS.includes(post.full_name);
          const imageUrl = getPostImageUrl(post);
          const mediaTypeIcon = getMediaTypeIcon(
            post.image_data.media_type
          );
          const timeAgo = formatTimeAgo(post.created_utc);

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

  attachCheckboxListeners() {
    const checkboxes = this.itemsList.querySelectorAll(".item-checkbox");
    checkboxes.forEach((checkbox) => {
      checkbox.addEventListener("change", (e) => {
        e.stopPropagation();
        const id = e.target.dataset.id;
        this.toggleSelection(id, e.target.checked);
        this.updateCheckboxVisuals();
      });
    });
  }

  toggleSelection(id, isChecked) {
    if (state.currentModalType === "subreddits") {
      if (isChecked) {
        if (!state.SELECTED_SUBREDDITS.includes(id)) {
          state.SELECTED_SUBREDDITS.push(id);
        }
      } else {
        state.setSelectedSubreddits(state.SELECTED_SUBREDDITS.filter((item) => item !== id));
      }
    } else if (state.currentModalType === "posts") {
      if (isChecked) {
        if (!state.SELECTED_POSTS.includes(id)) {
          state.SELECTED_POSTS.push(id);
        }
      } else {
        state.setSelectedPosts(state.SELECTED_POSTS.filter((item) => item !== id));
      }
    } else if (state.currentModalType === "comments") {
      if (isChecked) {
        if (!state.SELECTED_COMMENTS.includes(id)) {
          state.SELECTED_COMMENTS.push(id);
        }
      } else {
        state.setSelectedComments(state.SELECTED_COMMENTS.filter((item) => item !== id));
      }
    } else if (state.currentModalType === "multireddits") {
      if (isChecked) {
        if (!state.SELECTED_MULTIREDDITS.includes(id)) {
          state.SELECTED_MULTIREDDITS.push(id);
        }
      } else {
        state.setSelectedMultireddits(state.SELECTED_MULTIREDDITS.filter((item) => item !== id));
      }
    }
    this.updateSelectedCount();
  }

  updateSelectedCount() {
    let currentSelection;
    if (state.currentModalType === "subreddits") {
      currentSelection = state.SELECTED_SUBREDDITS;
    } else if (state.currentModalType === "posts") {
      currentSelection = state.SELECTED_POSTS;
    } else if (state.currentModalType === "comments") {
      currentSelection = state.SELECTED_COMMENTS;
    } else {
      currentSelection = state.SELECTED_MULTIREDDITS;
    }
    this.selectedCount.textContent = currentSelection.length;
  }

  filterItems(searchTerm) {
    const term = searchTerm.toLowerCase();

    if (state.currentModalType === "subreddits") {
      state.setFilteredItems(state.ALL_SUBREDDITS.filter(
        (subreddit) =>
          subreddit.display_name.toLowerCase().includes(term) ||
          (subreddit.title && subreddit.title.toLowerCase().includes(term)) ||
          (subreddit.public_description &&
            subreddit.public_description.toLowerCase().includes(term))
      ));
      this.renderSubreddits();
    } else if (state.currentModalType === "posts") {
      state.setFilteredItems(state.ALL_POSTS.filter(
        (post) =>
          post.title.toLowerCase().includes(term) ||
          post.subreddit.toLowerCase().includes(term) ||
          post.author.toLowerCase().includes(term) ||
          (post.selftext && post.selftext.toLowerCase().includes(term))
      ));
      this.renderPosts();
    } else if (state.currentModalType === "comments") {
      state.setFilteredItems(state.ALL_COMMENTS.filter(
        (comment) =>
          (comment.body_text && comment.body_text.toLowerCase().includes(term)) ||
          comment.subreddit.toLowerCase().includes(term) ||
          comment.author.toLowerCase().includes(term) ||
          (comment.link_title && comment.link_title.toLowerCase().includes(term))
      ));
      this.renderComments();
    } else if (state.currentModalType === "multireddits") {
      state.setFilteredItems(state.ALL_MULTIREDDITS.filter(
        (multi) =>
          (multi.name && multi.name.toLowerCase().includes(term)) ||
          (multi.display_name && multi.display_name.toLowerCase().includes(term)) ||
          (multi.description_md && multi.description_md.toLowerCase().includes(term)) ||
          (multi.subreddits && multi.subreddits.some(s => s.toLowerCase().includes(term)))
      ));
      this.renderMultireddits();
    }
  }

  selectAll() {
    if (state.currentModalType === "subreddits") {
      state.filteredItems.forEach((subreddit) => {
        if (!state.SELECTED_SUBREDDITS.includes(subreddit.display_name)) {
          state.SELECTED_SUBREDDITS.push(subreddit.display_name);
        }
      });
      this.renderSubreddits();
    } else if (state.currentModalType === "posts") {
      state.filteredItems.forEach((post) => {
        if (!state.SELECTED_POSTS.includes(post.full_name)) {
          state.SELECTED_POSTS.push(post.full_name);
        }
      });
      this.renderPosts();
    } else if (state.currentModalType === "comments") {
      state.filteredItems.forEach((comment) => {
        if (!state.SELECTED_COMMENTS.includes(comment.full_name)) {
          state.SELECTED_COMMENTS.push(comment.full_name);
        }
      });
      this.renderComments();
    } else if (state.currentModalType === "multireddits") {
      state.filteredItems.forEach((multi) => {
        if (!state.SELECTED_MULTIREDDITS.includes(multi.name)) {
          state.SELECTED_MULTIREDDITS.push(multi.name);
        }
      });
      this.renderMultireddits();
    }
  }

  selectNone() {
    if (state.currentModalType === "subreddits") {
      state.filteredItems.forEach((subreddit) => {
        state.setSelectedSubreddits(state.SELECTED_SUBREDDITS.filter(
          (item) => item !== subreddit.display_name
        ));
      });
      this.renderSubreddits();
    } else if (state.currentModalType === "posts") {
      state.filteredItems.forEach((post) => {
        state.setSelectedPosts(state.SELECTED_POSTS.filter(
          (item) => item !== post.full_name
        ));
      });
      this.renderPosts();
    } else if (state.currentModalType === "comments") {
      state.filteredItems.forEach((comment) => {
        state.setSelectedComments(state.SELECTED_COMMENTS.filter(
          (item) => item !== comment.full_name
        ));
      });
      this.renderComments();
    } else if (state.currentModalType === "multireddits") {
      state.filteredItems.forEach((multi) => {
        state.setSelectedMultireddits(state.SELECTED_MULTIREDDITS.filter(
          (item) => item !== multi.name
        ));
      });
      this.renderMultireddits();
    }
  }

  confirmSelection() {
    if (state.currentModalType === "subreddits") {
      state.setSubredditSelection("custom");
      updateSelectionSummary(
        "subreddits",
        "custom",
        state.SELECTED_SUBREDDITS.length
      );
    } else if (state.currentModalType === "posts") {
      state.setPostsSelection("custom");
      updateSelectionSummary("posts", "custom", state.SELECTED_POSTS.length);
    } else if (state.currentModalType === "comments") {
      state.setCommentsSelection("custom");
      updateSelectionSummary("comments", "custom", state.SELECTED_COMMENTS.length);
    } else if (state.currentModalType === "multireddits") {
      state.setMultiredditSelection("custom");
      updateSelectionSummary("multireddits", "custom", state.SELECTED_MULTIREDDITS.length);
    }
    this.close();
  }

  attachRowClickListeners() {
    const rows = this.itemsList.querySelectorAll(".item-row");
    rows.forEach((row) => {
      row.addEventListener("click", (e) => {
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
          this.updateCheckboxVisuals();
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

      if (isChecked) {
        checkboxVisual.classList.add("bg-red-500", "border-red-500");
        checkboxVisual.classList.remove("bg-white", "border-gray-300");
        checkmark.classList.remove("hidden");
      } else {
        checkboxVisual.classList.remove("bg-red-500", "border-red-500");
        checkboxVisual.classList.add("bg-white", "border-gray-300");
        checkmark.classList.add("hidden");
      }

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

// Initialize selection event listeners (called from app.js)
export function initSelectionListeners(selectionModal) {
  // Subreddit selection radio buttons
  document
    .querySelectorAll('input[name="subredditSelection"]')
    .forEach((radio) => {
      radio.addEventListener("change", async (e) => {
        state.setSubredditSelection(e.target.value);

        if (e.target.value === "all") {
          updateSelectionSummary("subreddits", "all");
          toggleDeleteOptions("subreddits", true);
        } else if (e.target.value === "custom") {
          if (!isSourceAccountVerified()) {
            alert("Please verify your source account first");
            document.getElementById("subredditNone").checked = true;
            state.setSubredditSelection("none");
            return;
          }

          await selectionModal.open("subreddits", getSourceAccessToken());
          toggleDeleteOptions("subreddits", true);
        } else {
          updateSelectionSummary("subreddits", "none");
          toggleDeleteOptions("subreddits", false);
          state.setSelectedSubreddits([]);
        }
      });
    });

  // Posts selection radio buttons
  document.querySelectorAll('input[name="postsSelection"]').forEach((radio) => {
    radio.addEventListener("change", async (e) => {
      state.setPostsSelection(e.target.value);
      console.log("Posts selection changed to:", e.target.value);
      console.log("Current authentication state:", {
        CURRENT_AUTH_METHOD: state.CURRENT_AUTH_METHOD,
        isSourceAccountVerified: isSourceAccountVerified(),
        OAUTH_SOURCE_VERIFIED: state.OAUTH_SOURCE_VERIFIED,
        SOURCE_USERNAME: state.SOURCE_USERNAME,
        SOURCE_ACCESS_TOKEN: state.SOURCE_ACCESS_TOKEN
          ? state.SOURCE_ACCESS_TOKEN.substring(0, 10) + "..."
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
          state.setPostsSelection("none");
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
        state.setSelectedPosts([]);
      }
    });
  });

  // Comments selection radio buttons
  document.querySelectorAll('input[name="commentsSelection"]').forEach((radio) => {
    radio.addEventListener("change", async (e) => {
      state.setCommentsSelection(e.target.value);
      console.log("Comments selection changed to:", e.target.value);

      if (e.target.value === "all") {
        updateSelectionSummary("comments", "all");
        toggleDeleteOptions("comments", true);
      } else if (e.target.value === "custom") {
        if (!isSourceAccountVerified()) {
          console.log(
            "Source account not verified, blocking custom comments selection"
          );
          alert("Please verify your source account first");
          document.getElementById("commentsNone").checked = true;
          state.setCommentsSelection("none");
          return;
        }

        await selectionModal.open("comments", getSourceAccessToken());
        toggleDeleteOptions("comments", true);
      } else {
        updateSelectionSummary("comments", "none");
        toggleDeleteOptions("comments", false);
        state.setSelectedComments([]);
      }
    });
  });

  // Edit selection buttons
  document
    .getElementById("editSubredditSelection")
    .addEventListener("click", async () => {
      if (!isSourceAccountVerified()) {
        alert("Please verify your source account first");
        return;
      }
      document.getElementById("subredditCustom").checked = true;
      state.setSubredditSelection("custom");
      await selectionModal.open("subreddits", getSourceAccessToken());
    });

  document
    .getElementById("editPostsSelection")
    .addEventListener("click", async () => {
      if (!isSourceAccountVerified()) {
        alert("Please verify your source account first");
        return;
      }
      document.getElementById("postsCustom").checked = true;
      state.setPostsSelection("custom");
      await selectionModal.open("posts", getSourceAccessToken());
    });

  document
    .getElementById("editCommentsSelection")
    .addEventListener("click", async () => {
      if (!isSourceAccountVerified()) {
        alert("Please verify your source account first");
        return;
      }
      document.getElementById("commentsCustom").checked = true;
      state.setCommentsSelection("custom");
      await selectionModal.open("comments", getSourceAccessToken());
    });

  // Multireddit selection radio buttons
  document.querySelectorAll('input[name="multiredditSelection"]').forEach((radio) => {
    radio.addEventListener("change", async (e) => {
      state.setMultiredditSelection(e.target.value);

      if (e.target.value === "all") {
        updateSelectionSummary("multireddits", "all");
      } else if (e.target.value === "custom") {
        if (!isSourceAccountVerified()) {
          alert("Please verify your source account first");
          document.getElementById("multiredditNone").checked = true;
          state.setMultiredditSelection("none");
          return;
        }

        await selectionModal.open("multireddits", getSourceAccessToken());
      } else {
        updateSelectionSummary("multireddits", "none");
        state.setSelectedMultireddits([]);
      }
    });
  });

  document
    .getElementById("editMultiredditSelection")
    .addEventListener("click", async () => {
      if (!isSourceAccountVerified()) {
        alert("Please verify your source account first");
        return;
      }
      document.getElementById("multiredditCustom").checked = true;
      state.setMultiredditSelection("custom");
      await selectionModal.open("multireddits", getSourceAccessToken());
    });
}
