class AuthComponent {
  constructor(container) {
    this.container = container;
    this.currentMethod = "oauth";
    this.accounts = {
      source: { verified: false, data: null },
      destination: { verified: false, data: null },
    };
    this.API_BASE_URL = ""; // Can be configured if needed
    this.init();
  }

  init() {
    this.render();
    this.attachEventListeners();
  }

  render() {
    this.container.innerHTML = `
      <div class="auth-component glass-card rounded-2xl p-8 mb-6">
        <!-- Auth Method Toggle -->
        <div class="auth-method-selector mb-6">
          <div class="flex items-center justify-center space-x-4 mb-6">
            <label class="auth-method-option">
              <input type="radio" name="authMethod" value="oauth" ${
                this.currentMethod === "oauth" ? "checked" : ""
              }>
              <div class="method-card">
                <span class="material-icons text-2xl mb-2">verified_user</span>
                <span class="font-semibold">OAuth API</span>
                <small>Secure & Modern</small>
              </div>
            </label>
            <label class="auth-method-option">
              <input type="radio" name="authMethod" value="cookie" ${
                this.currentMethod === "cookie" ? "checked" : ""
              }>
              <div class="method-card">
                <span class="material-icons text-2xl mb-2">cookie</span>
                <span class="font-semibold">Cookie Auth</span>
                <small>Legacy Method</small>
              </div>
            </label>
          </div>
        </div>

        <!-- Account Sections -->
        <div class="accounts-grid grid grid-cols-1 lg:grid-cols-2 gap-6">
          ${this.renderAccountSection("source", "Source Account", "#FF4500")}
          ${this.renderAccountSection(
            "destination",
            "Destination Account",
            "#0079D3"
          )}
        </div>
      </div>
    `;
  }

  renderAccountSection(type, title, color) {
    const account = this.accounts[type];

    return `
      <div class="account-section glass-card rounded-xl p-6" data-account="${type}">
        <h3 class="text-lg font-semibold mb-4 flex items-center" style="color: ${color}">
          <span class="material-icons mr-2">account_circle</span>
          ${title}
        </h3>
        
        <div class="auth-content">
          ${
            this.currentMethod === "oauth"
              ? this.renderOAuthSection(type)
              : this.renderCookieSection(type)
          }
        </div>
        
        ${account.verified ? this.renderVerifiedStatus(account.data) : ""}
      </div>
    `;
  }

  renderOAuthSection(accountType) {
    return `
      <div class="oauth-section">
        <div class="oauth-config mb-4">
          <label class="block text-sm font-medium mb-2">Client ID</label>
          <input type="text" class="form-input w-full rounded-lg px-4 py-3" 
                 data-field="clientId" 
                 data-account="${accountType}"
                 placeholder="Reddit App Client ID">
          
          <label class="block text-sm font-medium mb-2 mt-3">Client Secret</label>
          <input type="password" class="form-input w-full rounded-lg px-4 py-3" 
                 data-field="clientSecret" 
                 data-account="${accountType}"
                 placeholder="Reddit App Client Secret">
        </div>
        
        <button class="btn-primary w-full rounded-lg py-3 px-4 oauth-login-btn flex items-center justify-center space-x-2" 
                data-account="${accountType}">
          <span class="material-icons">login</span>
          <span>Login with Reddit OAuth</span>
        </button>
        
        <div class="oauth-loading hidden mt-3 flex items-center justify-center space-x-2">
          <div class="loading-spinner"></div>
          <span class="text-sm text-slate-400">Authenticating...</span>
        </div>
      </div>
    `;
  }

  renderCookieSection(accountType) {
    return `
      <div class="cookie-section">
        <div class="flex items-center justify-between mb-2">
          <label class="block text-sm font-medium">Reddit Cookie</label>
          <button onclick="showCookieHelp()" 
                  class="text-sm text-slate-400 hover:text-orange-400 flex items-center space-x-1 transition-colors">
            <span class="material-icons text-base">help_outline</span>
            <span>How to get?</span>
          </button>
        </div>
        <textarea class="form-input w-full h-24 rounded-lg px-4 py-3 cookie-input" 
                  data-account="${accountType}" 
                  placeholder="Paste your Reddit cookie here..."></textarea>
        
        <button class="btn-primary w-full mt-3 rounded-lg py-3 px-4 verify-cookie-btn flex items-center justify-center space-x-2" 
                data-account="${accountType}">
          <span class="material-icons">security</span>
          <span>Verify Cookie</span>
        </button>
        
        <div class="cookie-loading hidden mt-3 flex items-center justify-center space-x-2">
          <div class="loading-spinner"></div>
          <span class="text-sm text-slate-400">Verifying...</span>
        </div>
      </div>
    `;
  }

  renderVerifiedStatus(accountData) {
    return `
      <div class="verified-status mt-4 p-3 bg-emerald-900/20 rounded-lg border border-emerald-500/20">
        <div class="flex items-center">
          <span class="material-icons text-emerald-400 mr-2">check_circle</span>
          <span class="text-emerald-400 font-medium">Verified</span>
          <span class="text-slate-400 mx-2">|</span>
          <span class="text-slate-200">${accountData.username}</span>
        </div>
      </div>
    `;
  }

  attachEventListeners() {
    // Auth method toggle
    this.container.addEventListener("change", (e) => {
      if (e.target.name === "authMethod") {
        this.currentMethod = e.target.value;
        // Reset verification status when switching methods
        this.accounts.source.verified = false;
        this.accounts.destination.verified = false;
        this.render();
        this.attachEventListeners();
        this.notifyAuthStatusChanged();
      }
    });

    // OAuth login buttons
    this.container.addEventListener("click", (e) => {
      if (e.target.closest(".oauth-login-btn")) {
        const accountType =
          e.target.closest(".oauth-login-btn").dataset.account;
        this.handleOAuthLogin(accountType);
      }
    });

    // Cookie verification buttons
    this.container.addEventListener("click", (e) => {
      if (e.target.closest(".verify-cookie-btn")) {
        const accountType =
          e.target.closest(".verify-cookie-btn").dataset.account;
        this.handleCookieVerification(accountType);
      }
    });
  }

  async handleOAuthLogin(accountType) {
    console.log(`Starting OAuth login for ${accountType} account`);

    const section = this.container.querySelector(
      `[data-account="${accountType}"]`
    );
    const clientIdInput = section.querySelector('[data-field="clientId"]');
    const clientSecretInput = section.querySelector(
      '[data-field="clientSecret"]'
    );
    const loginBtn = section.querySelector(".oauth-login-btn");
    const loadingEl = section.querySelector(".oauth-loading");

    const clientId = clientIdInput.value.trim();
    const clientSecret = clientSecretInput.value.trim();

    if (!clientId || !clientSecret) {
      alert("Please enter both Client ID and Client Secret");
      return;
    }

    // Show loading state
    loginBtn.classList.add("hidden");
    loadingEl.classList.remove("hidden");

    try {
      // Open OAuth login in popup window
      const authWindow = window.open(
        `${this.API_BASE_URL}/api/oauth/login`,
        "oauth-login",
        "width=600,height=700,scrollbars=yes,resizable=yes"
      );

      // Poll for window closure and token
      const pollTimer = setInterval(async () => {
        if (authWindow.closed) {
          clearInterval(pollTimer);

          // Check if we got the token from the callback
          const tokenData = localStorage.getItem(`oauth_token_${accountType}`);
          if (tokenData) {
            const parsedData = JSON.parse(tokenData);
            this.accounts[accountType] = {
              verified: true,
              data: {
                username: parsedData.username,
                accessToken: parsedData.access_token,
                refreshToken: parsedData.refresh_token,
              },
            };

            // Clean up stored token
            localStorage.removeItem(`oauth_token_${accountType}`);

            this.render();
            this.attachEventListeners();
            this.notifyAuthStatusChanged();
          } else {
            // OAuth was cancelled or failed
            loginBtn.classList.remove("hidden");
            loadingEl.classList.add("hidden");
          }
        }
      }, 1000);
    } catch (error) {
      console.error("OAuth login error:", error);
      alert("OAuth login failed. Please try again.");
      loginBtn.classList.remove("hidden");
      loadingEl.classList.add("hidden");
    }
  }

  async handleCookieVerification(accountType) {
    console.log(`Verifying cookie for ${accountType} account`);

    const section = this.container.querySelector(
      `[data-account="${accountType}"]`
    );
    const cookieInput = section.querySelector(".cookie-input");
    const verifyBtn = section.querySelector(".verify-cookie-btn");
    const loadingEl = section.querySelector(".cookie-loading");

    const cookie = cookieInput.value.trim();

    if (!cookie) {
      alert("Please paste your Reddit cookie");
      return;
    }

    // Show loading state
    verifyBtn.classList.add("hidden");
    loadingEl.classList.remove("hidden");

    try {
      const response = await fetch(`${this.API_BASE_URL}/api/verify-cookie`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ cookie: cookie }),
      });

      const result = await response.json();

      if (result.success) {
        this.accounts[accountType] = {
          verified: true,
          data: {
            username: result.data.username,
            cookie: cookie,
          },
        };

        this.render();
        this.attachEventListeners();
        this.notifyAuthStatusChanged();
      } else {
        throw new Error(result.message || "Cookie verification failed");
      }
    } catch (error) {
      console.error("Cookie verification error:", error);
      alert(`Cookie verification failed: ${error.message}`);

      // Reset loading state
      verifyBtn.classList.remove("hidden");
      loadingEl.classList.add("hidden");
    }
  }

  notifyAuthStatusChanged() {
    const event = new CustomEvent("authStatusChanged", {
      detail: {
        sourceReady: this.accounts.source.verified,
        destinationReady: this.accounts.destination.verified,
        method: this.currentMethod,
      },
    });
    document.dispatchEvent(event);
  }

  isReady() {
    return this.accounts.source.verified && this.accounts.destination.verified;
  }

  getAuthData() {
    return {
      method: this.currentMethod,
      source: this.accounts.source.data,
      destination: this.accounts.destination.data,
    };
  }
}
