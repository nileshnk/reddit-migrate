// Migration submit handler and response display

import * as state from "./state.js";
import { isSourceAccountVerified, isDestAccountVerified } from "./auth.js";

export function initMigrationHandler() {
  const optionSubmit = document.getElementById("option-submit");
  const loadingBtn = document.getElementById("loading-btn");
  const migrateResponseBlock = document.getElementById("migrate-response-block");
  const migrateResponseData = document.getElementById("migrate-response-data");

  optionSubmit.addEventListener("click", async (e) => {
    e.preventDefault();

    if (!isSourceAccountVerified() || !isDestAccountVerified()) {
      alert("Please verify both accounts first");
      return;
    }

    optionSubmit.style.display = "none";
    loadingBtn.style.display = "block";

    migrateResponseBlock.style.display = "none";

    // Get tokens based on authentication method
    if (state.CURRENT_AUTH_METHOD === "cookie") {
      const sourceAccInput = document.getElementById("sourceAccessToken");
      const destAccInput = document.getElementById("destAccessToken");

      sourceAccInput.style.backgroundColor = "#e6e6e6";
      sourceAccInput.disabled = true;
      destAccInput.style.backgroundColor = "#e6e6e6";
      destAccInput.disabled = true;

      state.setSourceCookieToken(sourceAccInput.value);
      state.setDestCookieToken(destAccInput.value);
    } else if (state.CURRENT_AUTH_METHOD === "oauth") {
      state.setSourceCookieToken(state.SOURCE_ACCESS_TOKEN);
      state.setDestCookieToken(state.DEST_ACCESS_TOKEN);
    }

    const deleteSubreddits = document.getElementById(
      "deleteSubredditsYes"
    ).checked;
    const deletePosts = document.getElementById("deleteSavedPostsYes").checked;
    const deleteComments = document.getElementById("deleteSavedCommentsYes")?.checked || false;

    let requestBody;
    let endpoint;

    const hasCustomSelection = state.SUBREDDIT_SELECTION === "custom" || state.POSTS_SELECTION === "custom" || state.COMMENTS_SELECTION === "custom";
    if (hasCustomSelection) {
      endpoint = `${state.API_BASE_URL}/api/migrate-custom`;
      if (state.CURRENT_AUTH_METHOD === "oauth") {
        requestBody = {
          auth_method: "oauth",
          source_account_token: state.SOURCE_COOKIE_TOKEN,
          dest_account_token: state.DEST_COOKIE_TOKEN,
          source_account_username: state.SOURCE_USERNAME,
          dest_account_username: state.DEST_USERNAME,
          selected_subreddits:
            state.SUBREDDIT_SELECTION === "custom" ? state.SELECTED_SUBREDDITS : [],
          selected_posts: state.POSTS_SELECTION === "custom" ? state.SELECTED_POSTS : [],
          selected_comments: state.COMMENTS_SELECTION === "custom" ? state.SELECTED_COMMENTS : [],
          delete_source_subreddits: deleteSubreddits,
          delete_source_posts: deletePosts,
          delete_source_comments: deleteComments,
        };
      } else {
        requestBody = {
          auth_method: "cookie",
          source_account_cookie: state.SOURCE_COOKIE_TOKEN,
          dest_account_cookie: state.DEST_COOKIE_TOKEN,
          selected_subreddits:
            state.SUBREDDIT_SELECTION === "custom" ? state.SELECTED_SUBREDDITS : [],
          selected_posts: state.POSTS_SELECTION === "custom" ? state.SELECTED_POSTS : [],
          selected_comments: state.COMMENTS_SELECTION === "custom" ? state.SELECTED_COMMENTS : [],
          delete_source_subreddits: deleteSubreddits,
          delete_source_posts: deletePosts,
          delete_source_comments: deleteComments,
        };
      }
    } else {
      endpoint = `${state.API_BASE_URL}/api/migrate`;
      if (state.CURRENT_AUTH_METHOD === "oauth") {
        requestBody = {
          auth_method: "oauth",
          source_account_token: state.SOURCE_COOKIE_TOKEN,
          dest_account_token: state.DEST_COOKIE_TOKEN,
          source_account_username: state.SOURCE_USERNAME,
          dest_account_username: state.DEST_USERNAME,
          preferences: {
            migrate_subreddit_bool: state.SUBREDDIT_SELECTION === "all",
            migrate_post_bool: state.POSTS_SELECTION === "all",
            migrate_comment_bool: state.COMMENTS_SELECTION === "all",
            delete_post_bool: deletePosts,
            delete_comment_bool: deleteComments,
            delete_subreddit_bool: deleteSubreddits,
          },
        };
      } else {
        requestBody = {
          auth_method: "cookie",
          source_account_cookie: state.SOURCE_COOKIE_TOKEN,
          dest_account_cookie: state.DEST_COOKIE_TOKEN,
          preferences: {
            migrate_subreddit_bool: state.SUBREDDIT_SELECTION === "all",
            migrate_post_bool: state.POSTS_SELECTION === "all",
            migrate_comment_bool: state.COMMENTS_SELECTION === "all",
            delete_post_bool: deletePosts,
            delete_comment_bool: deleteComments,
            delete_subreddit_bool: deleteSubreddits,
          },
        };
      }
    }

    console.log("Starting migration with:", {
      endpoint,
      selections: { subreddits: state.SUBREDDIT_SELECTION, posts: state.POSTS_SELECTION, comments: state.COMMENTS_SELECTION },
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
        displayMigrationResponse(response, migrateResponseData, migrateResponseBlock, optionSubmit, loadingBtn);
      } else {
        throw new Error(response.message || "Migration failed");
      }
    } catch (error) {
      console.error("Migration error:", error);
      alert("Migration failed: " + error.message);

      optionSubmit.style.display = "block";
      loadingBtn.style.display = "none";

      if (state.CURRENT_AUTH_METHOD === "cookie") {
        const sourceAccInput = document.getElementById("sourceAccessToken");
        const destAccInput = document.getElementById("destAccessToken");
        sourceAccInput.disabled = false;
        destAccInput.disabled = false;
        sourceAccInput.style.backgroundColor = "";
        destAccInput.style.backgroundColor = "";
      }
    }
  });
}

function displayMigrationResponse(response, migrateResponseData, migrateResponseBlock, optionSubmit, loadingBtn) {
  migrateResponseData.innerHTML = "";

  const migratingSubreddits =
    state.SUBREDDIT_SELECTION === "all" ||
    (state.SUBREDDIT_SELECTION === "custom" && state.SELECTED_SUBREDDITS.length > 0);
  const migratingPosts =
    state.POSTS_SELECTION === "all" ||
    (state.POSTS_SELECTION === "custom" && state.SELECTED_POSTS.length > 0);
  const migratingComments =
    state.COMMENTS_SELECTION === "all" ||
    (state.COMMENTS_SELECTION === "custom" && state.SELECTED_COMMENTS.length > 0);

  if (migratingSubreddits && response.data.subscribeSubreddit) {
    const subredditStatusElement = document.createElement("li");
    subredditStatusElement.className =
      "flex items-center space-x-3 p-3 bg-emerald-900/20 rounded-lg border border-emerald-500/20";
    subredditStatusElement.innerHTML = `
      <span class="material-icons text-emerald-400">check_circle</span>
      <span class="text-sm font-medium text-slate-300">
        Total subreddits successfully subscribed to destination account:
        <span class="text-emerald-400 font-bold">${response.data.subscribeSubreddit.SuccessCount}</span>
      </span>
    `;
    migrateResponseData.appendChild(subredditStatusElement);
  }

  if (migratingPosts && response.data.savePost) {
    const postStatusElement = document.createElement("li");
    postStatusElement.className =
      "flex items-center space-x-3 p-3 bg-emerald-900/20 rounded-lg border border-emerald-500/20";
    postStatusElement.innerHTML = `
      <span class="material-icons text-emerald-400">check_circle</span>
      <span class="text-sm font-medium text-slate-300">
        Total posts successfully saved in destination account:
        <span class="text-emerald-400 font-bold">${response.data.savePost.SuccessCount}</span>
      </span>
    `;
    migrateResponseData.appendChild(postStatusElement);
  }

  if (migratingComments && response.data.saveComment) {
    const commentStatusElement = document.createElement("li");
    commentStatusElement.className =
      "flex items-center space-x-3 p-3 bg-emerald-900/20 rounded-lg border border-emerald-500/20";
    commentStatusElement.innerHTML = `
      <span class="material-icons text-emerald-400">check_circle</span>
      <span class="text-sm font-medium text-slate-300">
        Total comments successfully saved in destination account:
        <span class="text-emerald-400 font-bold">${response.data.saveComment.SuccessCount}</span>
      </span>
    `;
    migrateResponseData.appendChild(commentStatusElement);
  }

  if (!migratingSubreddits && !migratingPosts && !migratingComments) {
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
