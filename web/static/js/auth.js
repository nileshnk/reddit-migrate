// Authentication: cookie verification, OAuth modal management, auth helpers

import * as state from "./state.js";

// Auth request body builder
export function getAuthRequestBody() {
  console.log(
    "getAuthRequestBody called - Current auth method:",
    state.CURRENT_AUTH_METHOD
  );
  console.log("Authentication state:", {
    OAUTH_SOURCE_VERIFIED: state.OAUTH_SOURCE_VERIFIED,
    SOURCE_ACCESS_TOKEN: state.SOURCE_ACCESS_TOKEN
      ? state.SOURCE_ACCESS_TOKEN.substring(0, 10) + "..."
      : "none",
    SOURCE_USERNAME: state.SOURCE_USERNAME,
    SOURCE_COOKIE_TOKEN: state.SOURCE_COOKIE_TOKEN
      ? state.SOURCE_COOKIE_TOKEN.substring(0, 10) + "..."
      : "none",
  });

  if (state.CURRENT_AUTH_METHOD === "oauth") {
    const body = {
      auth_method: "oauth",
      access_token: state.SOURCE_ACCESS_TOKEN,
      username: state.SOURCE_USERNAME,
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
      cookie: state.SOURCE_COOKIE_TOKEN,
    };
    console.log("Cookie auth request body:", {
      auth_method: body.auth_method,
      cookie: body.cookie ? body.cookie.substring(0, 10) + "..." : "none",
    });
    return body;
  }
}

// Auth check helpers
export function isSourceAccountVerified() {
  if (state.CURRENT_AUTH_METHOD === "cookie") {
    return state.BOOL_SOURCE_TOKEN_VERIFIED;
  } else if (state.CURRENT_AUTH_METHOD === "oauth") {
    return state.OAUTH_SOURCE_VERIFIED;
  } else if (state.CURRENT_AUTH_METHOD === "csv_import") {
    // No live source account — "verified" once at least one CSV has been parsed.
    return state.ALL_POSTS.length > 0 || state.ALL_COMMENTS.length > 0;
  }
  return false;
}

export function isDestAccountVerified() {
  if (state.CURRENT_AUTH_METHOD === "csv_import") {
    // Destination still needs a live account; which flag to check depends on the
    // nested cookie/oauth choice made for the destination in CSV-import mode.
    return state.DEST_AUTH_SUBMETHOD === "oauth"
      ? state.OAUTH_DEST_VERIFIED
      : state.BOOL_DEST_TOKEN_VERIFIED;
  } else if (state.CURRENT_AUTH_METHOD === "cookie") {
    return state.BOOL_DEST_TOKEN_VERIFIED;
  } else if (state.CURRENT_AUTH_METHOD === "oauth") {
    return state.OAUTH_DEST_VERIFIED;
  }
  return false;
}

export function getSourceAccessToken() {
  if (state.CURRENT_AUTH_METHOD === "cookie") {
    return state.SOURCE_COOKIE_TOKEN;
  } else if (state.CURRENT_AUTH_METHOD === "oauth") {
    return state.SOURCE_ACCESS_TOKEN;
  }
  return "";
}

export function getDestAccessToken() {
  if (state.CURRENT_AUTH_METHOD === "cookie") {
    return state.DEST_COOKIE_TOKEN;
  } else if (state.CURRENT_AUTH_METHOD === "oauth") {
    return state.DEST_ACCESS_TOKEN;
  }
  return "";
}

// Submit button state
export function updateSubmitButtonState() {
  const optionSubmit = document.getElementById("option-submit");
  const isVerified = isSourceAccountVerified() && isDestAccountVerified();

  if (isVerified) {
    optionSubmit.disabled = false;
    optionSubmit.classList.remove("cursor-not-allowed", "opacity-50");
  } else {
    optionSubmit.disabled = true;
    optionSubmit.classList.add("cursor-not-allowed", "opacity-50");
  }
}

// OAuth Modal Management Functions
export function showOAuthModal(type) {
  const modal = document.getElementById(`${type}OAuthModal`);
  modal.classList.remove("hidden");
  resetOAuthModal(type);
}

export function hideOAuthModal(type) {
  const modal = document.getElementById(`${type}OAuthModal`);
  modal.classList.add("hidden");
}

export function resetOAuthModal(type) {
  document.getElementById(`${type}ModalClientId`).value = "";
  document.getElementById(`${type}ModalClientSecret`).value = "";
  document.getElementById(`${type}ModalUsername`).value = "";
  document.getElementById(`${type}ModalPassword`).value = "";

  document.getElementById(`${type}ModalSuccessMessage`).classList.add("hidden");
  document.getElementById(`${type}ModalErrorMessage`).classList.add("hidden");
  document.getElementById(`${type}ModalVerifyBtn`).style.display = "block";
  document.getElementById(`${type}ModalLoadBtn`).style.display = "none";

  switchModalAuthMethod(type, "oauth");
}

export function switchModalAuthMethod(type, method) {
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

  if (type === "source") {
    state.setSourceAuthMethod(method);
  } else {
    state.setDestAuthMethod(method);
  }
}

// Cookie verification
export async function verifyCookie(cookie) {
  const cookieData = getCookieObject(cookie);
  if (cookieData.token_v2 === undefined) {
    return {
      success: false,
      message: "Invalid Cookie. Please get a new one.",
      data: {},
    };
  }

  const response = await fetch(`${state.API_BASE_URL}/api/verify-cookie`, {
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

export function getCookieObject(cookie) {
  const pairs = cookie.split(";");
  const cookieObject = {};
  for (const pair of pairs) {
    const [name, value] = pair.trim().split("=");
    cookieObject[name] = value;
  }
  return cookieObject;
}

// Initialize cookie auth event listeners
export function initCookieAuthListeners() {
  const sourceTokenVerifyBtn = document.getElementById("sourceTokenVerifyBtn");
  const sourceVerifyLoadBtn = document.getElementById("sourceVerifyLoadBtn");
  const destTokenVerifyBtn = document.getElementById("destTokenVerifyBtn");
  const destVerifyLoadBtn = document.getElementById("destVerifyLoadBtn");

  sourceTokenVerifyBtn.addEventListener("click", async (e) => {
    e.preventDefault();
    sourceTokenVerifyBtn.style.display = "none";
    sourceVerifyLoadBtn.style.display = "block";

    const sourceAccessTokenInput = document.getElementById("sourceAccessToken");
    const sourceAccessTokenValue = sourceAccessTokenInput.value;
    const verifySourceToken = await verifyCookie(sourceAccessTokenValue);

    sourceTokenVerifyBtn.style.display = "block";
    sourceVerifyLoadBtn.style.display = "none";

    if (verifySourceToken.success) {
      state.setBoolSourceTokenVerified(true);
      state.setSourceCookieToken(sourceAccessTokenValue);

      sourceAccessTokenInput.disabled = true;
      sourceAccessTokenInput.style.borderColor = "#10b981";

      sourceTokenVerifyBtn.className =
        "btn-verified px-6 py-3 text-white font-semibold rounded-xl flex items-center space-x-2";
      sourceTokenVerifyBtn.disabled = true;
      sourceTokenVerifyBtn.style.cursor = "default";
      sourceTokenVerifyBtn.innerHTML = `
        <span class="material-icons text-lg">verified</span>
        <span>Verified</span>
      `;

      document.getElementById("sourceTokenVerifySuccessMessage").style.display =
        "flex";
      document.getElementById("sourceTokenVerifyFailMessage").style.display =
        "none";
      document.getElementById("sourceAccountUserId").innerHTML =
        verifySourceToken.data.username;

      updateSubmitButtonState();
    } else {
      sourceAccessTokenInput.style.borderColor = "#ef4444";
      document.getElementById("sourceTokenVerifyFailMessage").style.display =
        "flex";
      document.getElementById("sourceTokenVerifySuccessMessage").style.display =
        "none";
    }
  });

  destTokenVerifyBtn.addEventListener("click", async (e) => {
    e.preventDefault();
    destTokenVerifyBtn.style.display = "none";
    destVerifyLoadBtn.style.display = "block";

    const destAccessTokenInput = document.getElementById("destAccessToken");
    const destAccessTokenValue = destAccessTokenInput.value;
    const verifyDestToken = await verifyCookie(destAccessTokenValue);

    destTokenVerifyBtn.style.display = "block";
    destVerifyLoadBtn.style.display = "none";

    if (verifyDestToken.success) {
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

      document.getElementById("destTokenVerifySuccessMessage").style.display =
        "flex";
      document.getElementById("destTokenVerifyFailMessage").style.display =
        "none";
      document.getElementById("destAccountUserId").innerHTML =
        verifyDestToken.data.username;

      updateSubmitButtonState();
    } else {
      destAccessTokenInput.style.borderColor = "#ef4444";
      document.getElementById("destTokenVerifyFailMessage").style.display =
        "flex";
      document.getElementById("destTokenVerifySuccessMessage").style.display =
        "none";
    }
  });
}

// OAuth Modal Manager class
export class OAuthModalManager {
  constructor() {
    this.init();
  }

  init() {
    document
      .getElementById("sourceOAuthModalBtn")
      ?.addEventListener("click", () => {
        showOAuthModal("source");
      });

    document
      .getElementById("destOAuthModalBtnCsvMode")
      ?.addEventListener("click", () => {
        showOAuthModal("dest");
      });

    document
      .getElementById("destOAuthModalBtn")
      ?.addEventListener("click", () => {
        showOAuthModal("dest");
      });

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
      type === "source" ? state.SOURCE_AUTH_METHOD : state.DEST_AUTH_METHOD;

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
      const initResponse = await fetch(
        `${state.API_BASE_URL}/api/oauth/init`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            client_id: clientId,
            client_secret: clientSecret,
            account_type: type,
          }),
        }
      );

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
              `${state.API_BASE_URL}/api/oauth/status?account_type=${type}`
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
      const response = await fetch(
        `${state.API_BASE_URL}/api/oauth/direct`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            client_id: clientId,
            client_secret: clientSecret,
            username: username,
            password: password,
            account_type: type,
          }),
        }
      );

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

    if (
      state.CURRENT_AUTH_METHOD !== "oauth" &&
      state.CURRENT_AUTH_METHOD !== "csv_import"
    ) {
      console.log("Switching to OAuth mode after successful authentication");
      state.setCurrentAuthMethod("oauth");

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

    // In CSV-import mode, the destination's cookie/oauth choice is tracked
    // independently via DEST_AUTH_SUBMETHOD so this successful OAuth verification
    // isn't misread as a cookie-mode verification for the destination.
    if (state.CURRENT_AUTH_METHOD === "csv_import" && type === "dest") {
      state.setDestAuthSubmethod("oauth");
    }

    if (type === "source") {
      state.setOAuthSourceVerified(true);
      state.setSourceAccessToken(accessToken);
      state.setSourceUsername(username);
      console.log("Set SOURCE_USERNAME to:", username);
      document.getElementById("sourceOAuthUsername").textContent = username;
      document.getElementById("sourceOAuthStatus").classList.remove("hidden");
    } else {
      state.setOAuthDestVerified(true);
      state.setDestAccessToken(accessToken);
      state.setDestUsername(username);
      console.log("Set DEST_USERNAME to:", username);
      const usernameElId =
        state.CURRENT_AUTH_METHOD === "csv_import"
          ? "destOAuthUsernameCsvMode"
          : "destOAuthUsername";
      const statusElId =
        state.CURRENT_AUTH_METHOD === "csv_import"
          ? "destOAuthStatusCsvMode"
          : "destOAuthStatus";
      document.getElementById(usernameElId).textContent = username;
      document.getElementById(statusElId).classList.remove("hidden");
    }

    document.getElementById(`${type}ModalSuccessUsername`).textContent =
      username;
    document
      .getElementById(`${type}ModalSuccessMessage`)
      .classList.remove("hidden");
    document
      .getElementById(`${type}ModalErrorMessage`)
      .classList.add("hidden");

    updateSubmitButtonState();

    console.log("Updated authentication state:", {
      CURRENT_AUTH_METHOD: state.CURRENT_AUTH_METHOD,
      OAUTH_SOURCE_VERIFIED: state.OAUTH_SOURCE_VERIFIED,
      OAUTH_DEST_VERIFIED: state.OAUTH_DEST_VERIFIED,
      SOURCE_USERNAME: state.SOURCE_USERNAME,
      DEST_USERNAME: state.DEST_USERNAME,
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

// Expose functions used in HTML onclick attributes to window
window.showOAuthModal = showOAuthModal;
window.hideOAuthModal = hideOAuthModal;
