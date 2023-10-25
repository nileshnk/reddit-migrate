let BOOL_OLD_TOKEN_VERIFIED = false;
let BOOL_NEW_TOKEN_VERIFIED = false;
let BOOL_MIGRATE_SUBREDDITS = false;
let BOOL_MIGRATE_SAVED_POSTS = false;
let BOOL_DELETE_SUBREDDITS = false;
let BOOL_DELETE_SAVED_POSTS = false;

let OLD_ACCESS_TOKEN = "";
let NEW_ACCESS_TOKEN = "";

const optionSubmit = document.getElementById("option-submit");
const loadingBtn = document.getElementById("loading-btn");
const migrateResponseBlock = document.getElementById("migrate-response-block");
const migrateResponseData = document.getElementById("migrate-response-data");
optionSubmit.addEventListener("click", async (e) => {
  e.preventDefault();
  optionSubmit.style.display = "none";
  loadingBtn.style.display = "block";
  // darken the input field and disable it
  const oldAccAccessToken = document.getElementById("oldAccessToken");
  const newAccAccessToken = document.getElementById("newAccessToken");

  migrateResponseBlock.style.display = "none";

  oldAccAccessToken.style.backgroundColor = "#e6e6e6";
  oldAccAccessToken.disabled = true;
  newAccAccessToken.style.backgroundColor = "#e6e6e6";
  newAccAccessToken.disabled = true;

  // get the access tokens
  OLD_ACCESS_TOKEN = oldAccAccessToken.value;
  NEW_ACCESS_TOKEN = newAccAccessToken.value;

  const subredditMigrate = document.getElementById("migrateSubredditYes");
  const deleteSubreddits = document.getElementById("deleteSubredditsYes");
  const savedPostsMigrate = document.getElementById("migrateSavedPostsYes");
  const deleteSavedPosts = document.getElementById("deleteSavedPostsYes");

  if (subredditMigrate.checked) {
    BOOL_MIGRATE_SUBREDDITS = true;
  }
  if (deleteSubreddits.checked) {
    BOOL_DELETE_SUBREDDITS = true;
  }
  if (savedPostsMigrate.checked) {
    BOOL_MIGRATE_SAVED_POSTS = true;
  }
  if (deleteSavedPosts.checked) {
    BOOL_DELETE_SAVED_POSTS = true;
  }

  // hide the options
  // const subredditOptions = document.getElementById("subreddit-options");
  // subredditOptions.style.display = "none";
  // const savedPostsOptions = document.getElementById("saved-posts-options");
  // savedPostsOptions.style.display = "none";

  // start the migration
  console.log("Starting migration...");

  const requestBody = {
    old_account_cookie: OLD_ACCESS_TOKEN,
    new_account_cookie: NEW_ACCESS_TOKEN,
    preferences: {
      migrate_subreddit_bool: BOOL_MIGRATE_SUBREDDITS,
      migrate_post_bool: BOOL_MIGRATE_SAVED_POSTS,
      delete_post_bool: BOOL_DELETE_SAVED_POSTS,
      delete_subreddit_bool: BOOL_DELETE_SUBREDDITS,
    },
  };
  const migrateResponse = await fetch("/api/migrate", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(requestBody),
  });
  console.log(migrateResponse);
  const response = await migrateResponse.json();
  console.log(response);

  if (migrateResponse.status == 200) {
    displayMigrationResponse(response);
  }
});

function displayMigrationResponse(response) {
  const SubscribeSubreddit = document
    .getElementById("subscribe-subreddit")
    .querySelector("span");
  SubscribeSubreddit.innerText = `Total subreddits successfully subscribed to new account: ${response.data.subscribeSubreddit.SuccessCount}`;

  const UnsubscribeSubreddit = document
    .getElementById("unsubscribe-subreddit")
    .querySelector("span");
  UnsubscribeSubreddit.innerHTML = `Total subreddits successfully unsubscribed from old account: ${response.data.unsubscribeSubreddit.SuccessCount}`;

  const SavePost = document.getElementById("save-post").querySelector("span");
  SavePost.innerHTML = `Total posts successfully saved in new account: ${response.data.savePost.SuccessCount}`;

  const UnsavePost = document
    .getElementById("unsave-post")
    .querySelector("span");
  UnsavePost.innerHTML = `Total posts successfully unsaved from old account: ${response.data.unsavePost.SuccessCount}`;

  optionSubmit.style.display = "block";
  loadingBtn.style.display = "none";
  migrateResponseBlock.style.display = "block";
  migrateResponseData.style.display = "block";
  migrateResponseBlock.scrollIntoView({
    behavior: "smooth",
  });
}

const oldTokenVerifyBtn = document.getElementById("oldTokenVerifyBtn");
const newTokenVerifyBtn = document.getElementById("newTokenVerifyBtn");

oldTokenVerifyBtn.addEventListener("click", async (e) => {
  e.preventDefault();
  const oldAccAccessToken = document.getElementById("oldAccessToken");
  const oldAccAccessTokenValue = oldAccAccessToken.value;
  const verifyOldToken = await verifyCookie(oldAccAccessTokenValue);
  console.log(verifyOldToken);
  if (verifyOldToken.success) {
    BOOL_OLD_TOKEN_VERIFIED = true;
    oldAccAccessToken.disabled = true;
    oldAccAccessToken.style.borderColor = "#00cc00";
    oldAccAccessToken.disabled = true;
    oldTokenVerifyBtn.style.backgroundColor = "#00cc00";
    oldTokenVerifyBtn.disabled = true;
    oldTokenVerifyBtn.style.cursor = "default";
    oldTokenVerifyBtn.innerHTML = "Verified";
    document.getElementById("oldTokenVerifySuccessMessage").style.display =
      "block";
    document.getElementById("oldTokenVerifyFailMessage").style.display = "none";
    document.getElementById("oldAccountUserId").innerHTML =
      verifyOldToken.data.username;
    // verifyOldToken.data.username;
    console.log("Old access token verified");
  } else {
    oldAccAccessToken.style.borderColor = "#ff0000";
    document.getElementById("oldTokenVerifyFailMessage").style.display =
      "block";
    console.log("Old access token verification failed");
  }
});

newTokenVerifyBtn.addEventListener("click", async (e) => {
  e.preventDefault();
  const newAccAccessToken = document.getElementById("newAccessToken");
  const newAccAccessTokenValue = newAccAccessToken.value;
  const verifynewToken = await verifyCookie(newAccAccessTokenValue);
  if (verifynewToken.success) {
    BOOL_NEW_TOKEN_VERIFIED = true;
    newAccAccessToken.disabled = true;
    newAccAccessToken.style.borderColor = "#00cc00";
    newAccAccessToken.disabled = true;
    newTokenVerifyBtn.style.backgroundColor = "#00cc00";
    newTokenVerifyBtn.disabled = true;
    newTokenVerifyBtn.style.cursor = "default";
    newTokenVerifyBtn.innerHTML = "Verified";
    document.getElementById("newTokenVerifySuccessMessage").style.display =
      "block";
    document.getElementById("newTokenVerifyFailMessage").style.display = "none";

    document.getElementById("newAccountUserId").innerHTML =
      verifynewToken.data.username;
    // verifynewToken.data.username;
    console.log("New access token verified");
  } else {
    newAccAccessToken.style.borderColor = "#ff0000";
    document.getElementById("newTokenVerifyFailMessage").style.display =
      "block";
    console.log("new access token verification failed");
  }
});

const cookieSubmitBtn = document.getElementById("cookie-parse-submit");
cookieSubmitBtn.addEventListener("click", (e) => {
  e.preventDefault();
  const cookieInput = document.getElementById("cookie-input");
  const cookieInputValue = cookieInput.value;
  const cookieObject = getCookieObject(cookieInputValue);
  console.log(cookieObject.token_v2);
  if (cookieObject.token_v2 !== undefined) {
    console.log("Cookie parsed successfully");
    document.getElementById("cookie-parse-success-message").style.display =
      "block";
    document.getElementById("cookie-parse-fail-message").style.display = "none";
    document.getElementById("parsedTokenBox").innerHTML =
      cookieObject["token_v2"];

    document.getElementById("parsedTokenBox").onselectionchange((e) => {
      e.preventDefault();
    });
  } else {
    document.getElementById("cookie-parse-fail-message").style.display =
      "block";
    document.getElementById("cookie-parse-success-message").style.display =
      "none";
  }
});
document.getElementById("copy-button").addEventListener("click", (e) => {
  document.getElementById("copy-button").innerHTML = "Copied";
  setTimeout(() => {
    document.getElementById("copy-button").innerHTML = "Copy";
  }, 2000);
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
    // mode: "no-cors",
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
  const cookieArray = cookie.split(";");
  const cookieObject = {};
  try {
    cookieArray.forEach((cookie) => {
      const cookieSplit = cookie.split("=");
      cookieObject[cookieSplit[0].trim()] = cookieSplit[1].trim();
    });
  } catch (err) {
    return {};
  }
  return cookieObject;
}
