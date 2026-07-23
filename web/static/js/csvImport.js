// CSV import: parses Reddit's official data-export saved_posts.csv / saved_comments.csv
// client-side and feeds the parsed items into the existing selection-modal state,
// so a user can migrate saved items without a live source Reddit account.

import * as state from "./state.js";
import { verifyCookie, updateSubmitButtonState } from "./auth.js";
import { updateSelectionSummary } from "./modal.js";

// Splits one CSV line into fields, tolerating simple double-quoted fields.
function splitCsvLine(line) {
  const fields = [];
  let current = "";
  let inQuotes = false;

  for (let i = 0; i < line.length; i++) {
    const char = line[i];
    if (inQuotes) {
      if (char === '"') {
        if (line[i + 1] === '"') {
          current += '"';
          i++;
        } else {
          inQuotes = false;
        }
      } else {
        current += char;
      }
    } else if (char === '"') {
      inQuotes = true;
    } else if (char === ",") {
      fields.push(current.trim());
      current = "";
    } else {
      current += char;
    }
  }
  fields.push(current.trim());
  return fields;
}

function parseCsv(text) {
  const lines = text
    .replace(/^﻿/, "") // strip BOM if present
    .split(/\r?\n/)
    .filter((line) => line.trim().length > 0);

  if (lines.length === 0) {
    return { header: [], rows: [] };
  }

  const header = splitCsvLine(lines[0]).map((h) => h.toLowerCase());
  const rows = lines.slice(1).map(splitCsvLine);
  return { header, rows };
}

// Builds a SavedPostInfo/SavedCommentInfo-shaped object with every rich field
// defaulted, since existing UI code (search filtering, rendering) reads fields
// like title/subreddit/author unconditionally.
function buildCsvItem(fullName, permalink) {
  const id = fullName.replace(/^t[13]_/, "");
  return {
    id,
    full_name: fullName,
    permalink,
    is_csv_import: true,
    title: "",
    subreddit: "",
    author: "",
    url: "",
    created_utc: 0,
    score: 0,
    num_comments: 0,
    post_hint: "",
    domain: "",
    selftext: "",
    is_video: false,
    is_self: false,
    over_18: false,
    spoiler: false,
    image_data: { media_type: "", thumbnail_url: "", preview_url: "", high_res_url: "", width: 0, height: 0 },
    body: "",
    body_text: "",
    link_title: "",
    link_url: "",
  };
}

function normalizePermalink(rawPermalink) {
  if (!rawPermalink) return "";
  return rawPermalink.startsWith("http")
    ? rawPermalink
    : `https://reddit.com${rawPermalink}`;
}

// Parses a saved_posts.csv / saved_comments.csv File into CSV-import-shaped items.
// kindPrefix is "t3_" for posts, "t1_" for comments (Reddit fullname kind).
async function parseSavedItemsFile(file, kindPrefix) {
  const text = await file.text();
  const { header, rows } = parseCsv(text);

  const idIdx = header.indexOf("id");
  const permalinkIdx = header.indexOf("permalink");

  if (idIdx === -1 || permalinkIdx === -1) {
    throw new Error(
      `"${file.name}" is missing the expected "id"/"permalink" columns.`
    );
  }

  const items = [];
  const seen = new Set();

  for (const row of rows) {
    const rawId = (row[idIdx] || "").trim();
    if (!rawId) continue;

    const fullName =
      rawId.startsWith("t1_") || rawId.startsWith("t3_")
        ? rawId
        : `${kindPrefix}${rawId}`;

    if (seen.has(fullName)) continue;
    seen.add(fullName);

    const permalink = normalizePermalink((row[permalinkIdx] || "").trim());
    items.push(buildCsvItem(fullName, permalink));
  }

  if (items.length === 0) {
    throw new Error(`"${file.name}" doesn't contain any rows to import.`);
  }

  return items;
}

function showParseSummary(postsCount, commentsCount) {
  const summaryEl = document.getElementById("csvParseSummary");
  const summaryTextEl = document.getElementById("csvParseSummaryText");
  const errorEl = document.getElementById("csvParseError");

  const parts = [];
  if (postsCount > 0) parts.push(`${postsCount} post${postsCount === 1 ? "" : "s"}`);
  if (commentsCount > 0) parts.push(`${commentsCount} comment${commentsCount === 1 ? "" : "s"}`);

  summaryTextEl.textContent = `Parsed ${parts.join(" and ")}.`;
  summaryEl.classList.remove("hidden");
  errorEl.classList.add("hidden");
}

function showParseError(message) {
  const errorEl = document.getElementById("csvParseError");
  const errorTextEl = document.getElementById("csvParseErrorText");
  const summaryEl = document.getElementById("csvParseSummary");

  errorTextEl.textContent = message;
  errorEl.classList.remove("hidden");
  summaryEl.classList.add("hidden");
}

async function handlePostsFileChange(e) {
  const file = e.target.files[0];
  if (!file) return;

  try {
    const posts = await parseSavedItemsFile(file, "t3_");
    state.setAllPosts(posts);
    state.setSelectedPosts(posts.map((p) => p.full_name));
    state.setPostsSelection("custom");
    updateSelectionSummary("posts", "custom", posts.length);
    showParseSummary(posts.length, state.ALL_COMMENTS.length);
  } catch (error) {
    console.error("Error parsing saved posts CSV:", error);
    showParseError(error.message);
  }
}

async function handleCommentsFileChange(e) {
  const file = e.target.files[0];
  if (!file) return;

  try {
    const comments = await parseSavedItemsFile(file, "t1_");
    state.setAllComments(comments);
    state.setSelectedComments(comments.map((c) => c.full_name));
    state.setCommentsSelection("custom");
    updateSelectionSummary("comments", "custom", comments.length);
    showParseSummary(state.ALL_POSTS.length, comments.length);
  } catch (error) {
    console.error("Error parsing saved comments CSV:", error);
    showParseError(error.message);
  }
}

// Nested destination cookie/oauth sub-tab (only relevant in CSV-import mode,
// since the destination always needs a live account regardless of source method).
function initDestSubTabs() {
  const cookieSubTab = document.getElementById("csvDestCookieSubTab");
  const oauthSubTab = document.getElementById("csvDestOAuthSubTab");
  const cookieContent = document.getElementById("csvDestCookieContent");
  const oauthContent = document.getElementById("csvDestOAuthContent");

  cookieSubTab.addEventListener("click", (e) => {
    e.preventDefault();
    state.setDestAuthSubmethod("cookie");
    cookieSubTab.classList.add("active");
    oauthSubTab.classList.remove("active");
    cookieContent.classList.remove("hidden");
    oauthContent.classList.add("hidden");
  });

  oauthSubTab.addEventListener("click", (e) => {
    e.preventDefault();
    state.setDestAuthSubmethod("oauth");
    oauthSubTab.classList.add("active");
    cookieSubTab.classList.remove("active");
    oauthContent.classList.remove("hidden");
    cookieContent.classList.add("hidden");
  });
}

// Destination cookie verify button scoped to the CSV-import panel — mirrors
// initCookieAuthListeners' destTokenVerifyBtn handler in auth.js.
function initDestCookieVerify() {
  const destTokenVerifyBtn = document.getElementById("destTokenVerifyBtnCsvMode");
  const destVerifyLoadBtn = document.getElementById("destVerifyLoadBtnCsvMode");

  destTokenVerifyBtn.addEventListener("click", async (e) => {
    e.preventDefault();
    destTokenVerifyBtn.style.display = "none";
    destVerifyLoadBtn.style.display = "block";

    const destAccessTokenInput = document.getElementById("destAccessTokenCsvMode");
    const destAccessTokenValue = destAccessTokenInput.value;
    const verifyDestToken = await verifyCookie(destAccessTokenValue);

    destTokenVerifyBtn.style.display = "block";
    destVerifyLoadBtn.style.display = "none";

    if (verifyDestToken.success) {
      state.setDestAuthSubmethod("cookie");
      state.setBoolDestTokenVerified(true);
      state.setDestCookieToken(destAccessTokenValue);

      destAccessTokenInput.disabled = true;
      destAccessTokenInput.style.borderColor = "#10b981";

      destTokenVerifyBtn.className =
        "btn-verified px-6 py-3 text-white font-semibold rounded-xl flex items-center space-x-2";
      destTokenVerifyBtn.disabled = true;
      destTokenVerifyBtn.style.cursor = "default";
      destTokenVerifyBtn.innerHTML = `
        <span class="material-icons text-lg">verified</span>
        <span>Verified</span>
      `;

      document.getElementById("destTokenVerifySuccessMessageCsvMode").style.display = "flex";
      document.getElementById("destTokenVerifyFailMessageCsvMode").style.display = "none";
      document.getElementById("destAccountUserIdCsvMode").innerHTML = verifyDestToken.data.username;

      updateSubmitButtonState();
    } else {
      destAccessTokenInput.style.borderColor = "#ef4444";
      document.getElementById("destTokenVerifyFailMessageCsvMode").style.display = "flex";
      document.getElementById("destTokenVerifySuccessMessageCsvMode").style.display = "none";
    }
  });
}

export function initCsvImportListeners() {
  document
    .getElementById("csvPostsFile")
    .addEventListener("change", handlePostsFileChange);
  document
    .getElementById("csvCommentsFile")
    .addEventListener("change", handleCommentsFileChange);

  initDestSubTabs();
  initDestCookieVerify();
}
