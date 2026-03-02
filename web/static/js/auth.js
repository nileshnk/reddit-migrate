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
    OLD_ACCESS_TOKEN: state.OLD_ACCESS_TOKEN
      ? state.OLD_ACCESS_TOKEN.substring(0, 10) + "..."
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
      cookie: state.OLD_ACCESS_TOKEN,
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
    return state.BOOL_OLD_TOKEN_VERIFIED;
  } else if (state.CURRENT_AUTH_METHOD === "oauth") {
    return state.OAUTH_SOURCE_VERIFIED;
  }
  return false;
}

export function isDestAccountVerified() {
  if (state.CURRENT_AUTH_METHOD === "cookie") {
    return state.BOOL_NEW_TOKEN_VERIFIED;
  } else if (state.CURRENT_AUTH_METHOD === "oauth") {
    return state.OAUTH_DEST_VERIFIED;
  }
  return false;
}

export function getSourceAccessToken() {
  if (state.CURRENT_AUTH_METHOD === "cookie") {
    return state.OLD_ACCESS_TOKEN;
  } else if (state.CURRENT_AUTH_METHOD === "oauth") {
    return state.SOURCE_ACCESS_TOKEN;
  }
  return "";
}

export function getDestAccessToken() {
  if (state.CURRENT_AUTH_METHOD === "cookie") {
    return state.NEW_ACCESS_TOKEN;
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
      state.setBoolOldTokenVerified(true);
      state.setOldAccessToken(oldAccAccessTokenValue);

      oldAccAccessToken.disabled = true;
      oldAccAccessToken.style.borderColor = "#10b981";

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
      document.getElementById("oldTokenVerifyFailMessage").style.display =
        "none";
      document.getElementById("oldAccountUserId").innerHTML =
        verifyOldToken.data.username;

      updateSubmitButtonState();
    } else {
      oldAccAccessToken.style.borderColor = "#ef4444";
      document.getElementById("oldTokenVerifyFailMessage").style.display =
        "flex";
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

    newTokenVerifyBtn.style.display = "block";
    verifyLoadBtn2.style.display = "none";

    if (verifyNewToken.success) {
      state.setBoolNewTokenVerified(true);
      state.setNewAccessToken(newAccAccessTokenValue);

      newAccAccessToken.disabled = true;
      newAccAccessToken.style.borderColor = "#10b981";

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
      document.getElementById("newTokenVerifyFailMessage").style.display =
        "none";
      document.getElementById("newAccountUserId").innerHTML =
        verifyNewToken.data.username;

      updateSubmitButtonState();
    } else {
      newAccAccessToken.style.borderColor = "#ef4444";
      document.getElementById("newTokenVerifyFailMessage").style.display =
        "flex";
      document.getElementById("newTokenVerifySuccessMessage").style.display =
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

    if (state.CURRENT_AUTH_METHOD !== "oauth") {
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
      document.getElementById("destOAuthUsername").textContent = username;
      document.getElementById("destOAuthStatus").classList.remove("hidden");
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
