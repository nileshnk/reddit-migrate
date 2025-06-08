// Main Application
class RedditMigrationApp {
  constructor() {
    this.authComponent = null;
    this.featureSelector = null;
    this.API_BASE_URL = ""; // Can be configured if needed
    this.init();
  }

  init() {
    // Initialize components
    this.authComponent = new AuthComponent(
      document.getElementById("authComponent")
    );
    this.featureSelector = new FeatureSelector(
      document.getElementById("featureSelector")
    );

    // Initialize dark mode
    new DarkModeManager();

    // Set up global event listeners
    this.setupGlobalEvents();

    // Initialize process actions
    this.initializeProcessActions();
  }

  setupGlobalEvents() {
    // Listen for auth status changes
    document.addEventListener("authStatusChanged", (e) => {
      this.updateSubmitButtonState();
    });

    // Listen for feature changes
    document.addEventListener("featureChanged", (e) => {
      this.updateSubmitButtonState();
    });
  }

  initializeProcessActions() {
    const processActionsContainer = document.getElementById("processActions");
    processActionsContainer.innerHTML = `
      <div class="glass-card rounded-2xl p-8">
        <div class="flex flex-col items-center space-y-6">
          <!-- Progress Section (hidden by default) -->
          <div id="progressSection" class="w-full hidden">
            <div class="mb-4">
              <div class="flex justify-between items-center mb-2">
                <span class="text-sm font-medium text-slate-300">Progress</span>
                <span class="text-sm font-medium text-slate-300" id="progressText">0%</span>
              </div>
              <div class="progress-container w-full">
                <div class="progress-bar" id="progressBar" style="width: 0%"></div>
              </div>
            </div>
            <p class="text-center text-slate-400" id="progressMessage">Initializing...</p>
          </div>

          <!-- Submit Button -->
          <button id="submitBtn" 
                  class="btn-primary px-8 py-4 text-white font-bold rounded-xl text-lg flex items-center space-x-3 disabled:opacity-50 disabled:cursor-not-allowed"
                  disabled>
            <span class="material-icons text-xl">play_arrow</span>
            <span>Start Process</span>
          </button>
        </div>
      </div>
    `;

    // Add submit button event listener
    document.getElementById("submitBtn").addEventListener("click", (e) => {
      this.handleSubmission();
    });

    // Initialize results container
    this.initializeResultsContainer();
  }

  initializeResultsContainer() {
    const resultsContainer = document.getElementById("resultsContainer");
    resultsContainer.innerHTML = `
      <div class="glass-card rounded-2xl p-8">
        <h2 class="text-xl font-semibold text-slate-200 mb-4 flex items-center">
          <span class="material-icons mr-2 text-emerald-400">check_circle_outline</span>
          Process Results
        </h2>
        <div id="resultsContent">
          <!-- Results will be displayed here -->
        </div>
      </div>
    `;
  }

  updateSubmitButtonState() {
    const submitBtn = document.getElementById("submitBtn");
    if (this.authComponent.isReady()) {
      submitBtn.disabled = false;
      submitBtn.classList.remove("opacity-50", "cursor-not-allowed");
    } else {
      submitBtn.disabled = true;
      submitBtn.classList.add("opacity-50", "cursor-not-allowed");
    }
  }

  async handleSubmission() {
    if (!this.authComponent.isReady()) {
      alert("Please verify both accounts first");
      return;
    }

    const authData = this.authComponent.getAuthData();
    const currentFeature = this.featureSelector.currentFeature;

    // Show progress
    this.showProgress();

    try {
      if (currentFeature === "migration") {
        await this.handleMigration(authData);
      } else if (currentFeature === "cleanup") {
        await this.handleCleanup(authData);
      }
    } catch (error) {
      console.error("Process failed:", error);
      this.showError(error.message);
    } finally {
      this.hideProgress();
    }
  }

  async handleMigration(authData) {
    const migrationFeature = this.featureSelector.features.migration;
    const selections = migrationFeature.selections;

    // Determine if we need custom migration or regular migration
    const hasCustomSelections =
      selections.subreddits === "custom" ||
      selections.posts === "custom" ||
      selections.follows === "custom";

    let endpoint, requestBody;

    if (hasCustomSelections) {
      // Use custom migration endpoint
      endpoint = `${this.API_BASE_URL}/api/migrate-custom`;
      requestBody = {
        old_account_cookie: this.getAuthToken(authData.source, authData.method),
        new_account_cookie: this.getAuthToken(
          authData.destination,
          authData.method
        ),
        selected_subreddits:
          selections.subreddits === "custom"
            ? migrationFeature.selectedSubreddits || []
            : [],
        selected_posts:
          selections.posts === "custom"
            ? migrationFeature.selectedPosts || []
            : [],
        selected_follows:
          selections.follows === "custom"
            ? migrationFeature.selectedFollows || []
            : [],
        delete_old_subreddits: this.getDeleteOption("subreddits"),
        delete_old_posts: this.getDeleteOption("posts"),
        delete_old_follows: this.getDeleteOption("follows"),
      };
    } else {
      // Use traditional migration endpoint
      endpoint = `${this.API_BASE_URL}/api/migrate`;
      requestBody = {
        old_account_cookie: this.getAuthToken(authData.source, authData.method),
        new_account_cookie: this.getAuthToken(
          authData.destination,
          authData.method
        ),
        preferences: {
          migrate_subreddit_bool: selections.subreddits === "all",
          migrate_post_bool: selections.posts === "all",
          migrate_follows_bool: selections.follows === "all",
          delete_post_bool: this.getDeleteOption("posts"),
          delete_subreddit_bool: this.getDeleteOption("subreddits"),
          delete_follows_bool: this.getDeleteOption("follows"),
        },
      };
    }

    // Set up progress tracking
    let progress = 0;
    const progressInterval = setInterval(() => {
      progress += Math.random() * 10;
      if (progress > 90) progress = 90;
      this.updateProgress(progress, "Processing migration...");
    }, 1000);

    try {
      const response = await fetch(endpoint, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(requestBody),
      });

      clearInterval(progressInterval);

      const result = await response.json();

      if (response.ok) {
        this.updateProgress(100, "Migration completed!");
        setTimeout(() => {
          this.showResults(result);
        }, 1000);
      } else {
        throw new Error(result.message || "Migration failed");
      }
    } catch (error) {
      clearInterval(progressInterval);
      throw error;
    }
  }

  async handleCleanup(authData) {
    const cleanupFeature = this.featureSelector.features.cleanup;

    // This would need a new endpoint for cleanup operations
    // For now, we'll show a placeholder
    this.updateProgress(50, "Content cleanup not yet implemented...");

    setTimeout(() => {
      this.updateProgress(100, "Cleanup feature coming soon!");
      setTimeout(() => {
        this.showResults({
          success: true,
          message: "Cleanup feature is under development",
          data: {},
        });
      }, 1000);
    }, 2000);
  }

  getAuthToken(accountData, method) {
    if (method === "oauth") {
      return accountData.accessToken;
    } else {
      return accountData.cookie;
    }
  }

  getDeleteOption(type) {
    const deleteRadio = document.querySelector(
      `input[name="delete${type}"]:checked`
    );
    return deleteRadio ? deleteRadio.value === "yes" : false;
  }

  showProgress() {
    const progressSection = document.getElementById("progressSection");
    const submitBtn = document.getElementById("submitBtn");

    progressSection.classList.remove("hidden");
    submitBtn.disabled = true;
    submitBtn.innerHTML = `
      <div class="loading-spinner"></div>
      <span>Processing...</span>
    `;

    // Scroll to progress section
    progressSection.scrollIntoView({ behavior: "smooth" });
  }

  hideProgress() {
    const progressSection = document.getElementById("progressSection");
    const submitBtn = document.getElementById("submitBtn");

    progressSection.classList.add("hidden");
    submitBtn.disabled = false;
    submitBtn.innerHTML = `
      <span class="material-icons text-xl">play_arrow</span>
      <span>Start Process</span>
    `;
  }

  updateProgress(percentage, text) {
    const progressBar = document.getElementById("progressBar");
    const progressText = document.getElementById("progressText");
    const progressMessage = document.getElementById("progressMessage");

    if (progressBar) {
      progressBar.style.width = `${percentage}%`;
    }

    if (progressText) {
      progressText.textContent = `${Math.round(percentage)}%`;
    }

    if (progressMessage) {
      progressMessage.textContent = text;
    }
  }

  showResults(result) {
    const resultsContainer = document.getElementById("resultsContainer");
    const resultsContent = document.getElementById("resultsContent");
    const progressSection = document.getElementById("progressSection");

    // Hide progress and show results
    progressSection.classList.add("hidden");
    resultsContainer.classList.remove("hidden");

    let resultsHTML = "";

    if (result.success) {
      resultsHTML = `<div class="space-y-4">`;

      if (result.data.subscribeSubreddit) {
        resultsHTML += `
          <div class="flex items-center space-x-3 p-3 bg-emerald-900/20 rounded-lg border border-emerald-500/20">
            <span class="material-icons text-emerald-400">check_circle</span>
            <span class="text-sm font-medium text-slate-300">
              Subreddits migrated: 
              <span class="text-emerald-400 font-bold">${result.data.subscribeSubreddit.SuccessCount}</span>
            </span>
          </div>
        `;
      }

      if (result.data.savePost) {
        resultsHTML += `
          <div class="flex items-center space-x-3 p-3 bg-emerald-900/20 rounded-lg border border-emerald-500/20">
            <span class="material-icons text-emerald-400">check_circle</span>
            <span class="text-sm font-medium text-slate-300">
              Posts migrated: 
              <span class="text-emerald-400 font-bold">${result.data.savePost.SuccessCount}</span>
            </span>
          </div>
        `;
      }

      resultsHTML += `</div>`;
    } else {
      resultsHTML = `
        <div class="flex items-center space-x-3 p-3 bg-red-900/20 rounded-lg border border-red-500/20">
          <span class="material-icons text-red-400">error</span>
          <span class="text-sm font-medium text-slate-300">
            ${result.message || "Process failed"}
          </span>
        </div>
      `;
    }

    resultsContent.innerHTML = resultsHTML;

    // Scroll to results
    resultsContainer.scrollIntoView({ behavior: "smooth" });
  }

  showError(message) {
    const resultsContainer = document.getElementById("resultsContainer");
    const resultsContent = document.getElementById("resultsContent");
    const progressSection = document.getElementById("progressSection");

    // Hide progress and show error
    progressSection.classList.add("hidden");
    resultsContainer.classList.remove("hidden");

    resultsContent.innerHTML = `
      <div class="flex items-center space-x-3 p-3 bg-red-900/20 rounded-lg border border-red-500/20">
        <span class="material-icons text-red-400">error</span>
        <span class="text-sm font-medium text-slate-300">${message}</span>
      </div>
    `;

    // Scroll to results
    resultsContainer.scrollIntoView({ behavior: "smooth" });
  }
}

// Initialize app when DOM is loaded
document.addEventListener("DOMContentLoaded", () => {
  new RedditMigrationApp();
});
