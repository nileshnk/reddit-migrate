class MigrationFeature {
  constructor() {
    this.selections = {
      subreddits: "none",
      posts: "none",
      follows: "none",
    };
    this.selectedSubreddits = [];
    this.selectedPosts = [];
    this.selectedFollows = [];
  }

  render() {
    const html = `
      <div class="migration-feature">
        <h3 class="text-xl font-semibold mb-6 flex items-center">
          <span class="material-icons mr-2 text-orange-400">sync_alt</span>
          Migration Types
        </h3>

        <div class="migration-types space-y-6">
          ${this.renderMigrationType(
            "subreddits",
            "Subreddit Subscriptions",
            "group",
            "#FF4500"
          )}
          ${this.renderMigrationType(
            "posts",
            "Saved Posts",
            "bookmark",
            "#0079D3"
          )}
          ${this.renderMigrationType(
            "follows",
            "User Follows",
            "person_add",
            "#10b981"
          )}
        </div>

        <div class="batch-settings mt-8">
          <h4 class="text-lg font-semibold mb-4">Batch Settings</h4>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium mb-2">Batch Size</label>
              <select class="form-input w-full rounded-lg px-4 py-3" id="batchSize">
                <option value="10">10</option>
                <option value="25">25</option>
                <option value="50" selected>50</option>
                <option value="100">100</option>
              </select>
            </div>
            <div>
              <label class="block text-sm font-medium mb-2">Rate Limiting</label>
              <div class="flex items-center space-x-2">
                <input type="checkbox" id="rateLimiting" class="custom-checkbox" checked>
                <label for="rateLimiting" class="text-sm">Enable Rate Limiting</label>
              </div>
            </div>
          </div>
        </div>
      </div>
    `;

    // After rendering, attach event listeners
    setTimeout(() => {
      this.attachEventListeners();
    }, 0);

    return html;
  }

  renderMigrationType(type, title, icon, color) {
    const isSelected = this.selections[type];
    const selectedCount = this.getSelectedCount(type);

    return `
      <fieldset class="migration-type glass-card rounded-xl p-6">
        <legend class="text-lg font-semibold mb-4 flex items-center" style="color: ${color}">
          <span class="material-icons mr-2">${icon}</span>
          ${title}
        </legend>
        
        <div class="selection-options flex items-center space-x-8 mb-4">
          <label class="flex items-center">
            <input type="radio" name="${type}Selection" value="all" class="custom-radio" 
                   ${isSelected === "all" ? "checked" : ""}>
            <span class="ml-3 text-sm font-medium">All</span>
          </label>
          <label class="flex items-center">
            <input type="radio" name="${type}Selection" value="custom" class="custom-radio"
                   ${isSelected === "custom" ? "checked" : ""}>
            <span class="ml-3 text-sm font-medium">Custom</span>
          </label>
          <label class="flex items-center">
            <input type="radio" name="${type}Selection" value="none" class="custom-radio" 
                   ${isSelected === "none" ? "checked" : ""}>
            <span class="ml-3 text-sm font-medium">None</span>
          </label>
        </div>

        <div class="selection-summary ${
          isSelected !== "none" ? "" : "hidden"
        }" id="${type}Summary">
          <div class="bg-slate-700/30 rounded-lg p-3 flex items-center justify-between">
            <span class="text-sm text-slate-400 flex items-center">
              <span class="material-icons mr-2 text-sm">info</span>
              ${isSelected === "all" ? "All" : selectedCount} ${type} selected
            </span>
            ${
              isSelected === "custom"
                ? `
              <button class="btn-secondary px-4 py-2 text-white text-sm font-semibold rounded-lg edit-selection-btn" 
                      data-type="${type}">
                Edit Selection
              </button>
            `
                : ""
            }
          </div>
        </div>

        <div class="deletion-options mt-4 ${
          isSelected !== "none" ? "" : "hidden"
        }" id="${type}Deletion">
          <label class="block text-sm font-semibold mb-3">Delete from Source Account:</label>
          <div class="flex items-center space-x-6">
            <label class="flex items-center">
              <input type="radio" name="delete${type}" value="yes" class="custom-radio">
              <span class="ml-3 text-sm font-medium">Yes</span>
            </label>
            <label class="flex items-center">
              <input type="radio" name="delete${type}" value="no" class="custom-radio" checked>
              <span class="ml-3 text-sm font-medium">No</span>
            </label>
          </div>
        </div>
      </fieldset>
    `;
  }

  attachEventListeners() {
    // Listen for selection changes
    document.addEventListener("change", (e) => {
      if (e.target.name && e.target.name.endsWith("Selection")) {
        const type = e.target.name.replace("Selection", "");
        this.handleSelectionChange(type, e.target.value);
      }
    });

    // Listen for edit selection button clicks
    document.addEventListener("click", (e) => {
      if (e.target.classList.contains("edit-selection-btn")) {
        const type = e.target.dataset.type;
        this.openSelectionModal(type);
      }
    });
  }

  handleSelectionChange(type, value) {
    this.selections[type] = value;

    // Update UI to show/hide summary and deletion options
    const summary = document.getElementById(`${type}Summary`);
    const deletion = document.getElementById(`${type}Deletion`);

    if (value === "none") {
      summary.classList.add("hidden");
      deletion.classList.add("hidden");
    } else {
      summary.classList.remove("hidden");
      deletion.classList.remove("hidden");

      // Update summary content
      this.updateSummary(type);
    }

    // If custom is selected, open selection modal
    if (value === "custom") {
      this.openSelectionModal(type);
    }
  }

  updateSummary(type) {
    const summary = document.getElementById(`${type}Summary`);
    const isSelected = this.selections[type];
    const selectedCount = this.getSelectedCount(type);

    summary.innerHTML = `
      <div class="bg-slate-700/30 rounded-lg p-3 flex items-center justify-between">
        <span class="text-sm text-slate-400 flex items-center">
          <span class="material-icons mr-2 text-sm">info</span>
          ${isSelected === "all" ? "All" : selectedCount} ${type} selected
        </span>
        ${
          isSelected === "custom"
            ? `
          <button class="btn-secondary px-4 py-2 text-white text-sm font-semibold rounded-lg edit-selection-btn" 
                  data-type="${type}">
            Edit Selection
          </button>
        `
            : ""
        }
      </div>
    `;
  }

  getSelectedCount(type) {
    switch (type) {
      case "subreddits":
        return this.selectedSubreddits.length;
      case "posts":
        return this.selectedPosts.length;
      case "follows":
        return this.selectedFollows.length;
      default:
        return 0;
    }
  }

  async openSelectionModal(type) {
    // This would open a modal for custom selection
    // For now, we'll simulate the selection process
    console.log(`Opening selection modal for ${type}`);

    // Simulate opening the existing selection modal
    // The actual implementation would integrate with the existing SelectionModal class
    alert(
      `Custom ${type} selection feature would open here. This would integrate with your existing SelectionModal component.`
    );

    // For demo purposes, set some dummy selected items
    switch (type) {
      case "subreddits":
        this.selectedSubreddits = ["javascript", "programming", "webdev"];
        break;
      case "posts":
        this.selectedPosts = ["post1", "post2", "post3"];
        break;
      case "follows":
        this.selectedFollows = ["user1", "user2"];
        break;
    }

    this.updateSummary(type);
  }

  getBatchSettings() {
    return {
      batchSize: parseInt(document.getElementById("batchSize")?.value || "50"),
      rateLimiting: document.getElementById("rateLimiting")?.checked || true,
    };
  }

  getSelectionData() {
    return {
      selections: this.selections,
      selectedSubreddits: this.selectedSubreddits,
      selectedPosts: this.selectedPosts,
      selectedFollows: this.selectedFollows,
      batchSettings: this.getBatchSettings(),
    };
  }
}
