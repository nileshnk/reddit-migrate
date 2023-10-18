let BOOL_OLD_TOKEN_VERIFIED = false;
let BOOL_NEW_TOKEN_VERIFIED = false;
let BOOL_MIGRATE_SUBREDDITS = false;
let BOOL_MIGRATE_SAVED_POSTS = false;
let BOOL_DELETE_SUBREDDITS = false;
let BOOL_DELETE_SAVED_POSTS = false;

let OLD_ACCESS_TOKEN = "";
let NEW_ACCESS_TOKEN = "";

const optionSubmit = document.getElementById("option-submit");

optionSubmit.addEventListener("click", (e) => {
  e.preventDefault();

  // darken the input field and disable it
  const oldAccAccessToken = document.getElementById("oldAccessToken");
  const newAccAccessToken = document.getElementById("newAccessToken");

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
  const subredditOptions = document.getElementById("subreddit-options");
  subredditOptions.style.display = "none";
  const savedPostsOptions = document.getElementById("saved-posts-options");
  savedPostsOptions.style.display = "none";

  console.log(OLD_ACCESS_TOKEN, NEW_ACCESS_TOKEN);
  console.log(BOOL_MIGRATE_SUBREDDITS, BOOL_DELETE_SUBREDDITS);
  console.log(BOOL_MIGRATE_SAVED_POSTS, BOOL_DELETE_SAVED_POSTS);

  // show the progress bar

  // start the migration
  console.log("Starting migration...");
});

const oldTokenVerifyBtn = document.getElementById("oldTokenVerifyBtn");
const newTokenVerifyBtn = document.getElementById("newTokenVerifyBtn");

oldTokenVerifyBtn.addEventListener("click", async (e) => {
  e.preventDefault();
  const oldAccAccessToken = document.getElementById("oldAccessToken");
  const oldAccAccessTokenValue = oldAccAccessToken.value;
  const verifyOldToken = await verifyToken(oldAccAccessTokenValue);
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
  const verifynewToken = await verifyToken(newAccAccessTokenValue);
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
async function verifyToken(token) {
  const response = await fetch("/api/verify-token", {
    body: JSON.stringify({ access_token: token }),
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
      message: "Invalid access token",
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
