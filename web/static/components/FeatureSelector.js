class FeatureSelector {
  constructor(container) {
    this.container = container;
    this.currentFeature = "migration";
    this.features = {
      migration: new MigrationFeature(),
      cleanup: new CleanupFeature(),
    };
    this.init();
  }

  init() {
    this.render();
    this.attachEventListeners();
  }

  render() {
    this.container.innerHTML = `
      <div class="feature-selector glass-card rounded-2xl p-8">
        <!-- Feature Tabs -->
        <div class="feature-tabs flex space-x-4 mb-6">
          <button class="tab-btn ${
            this.currentFeature === "migration" ? "active" : ""
          }" 
                  data-feature="migration">
            <span class="material-icons">sync_alt</span>
            <span>Account Migration</span>
            <small>Move to New Account</small>
          </button>
          <button class="tab-btn ${
            this.currentFeature === "cleanup" ? "active" : ""
          }" 
                  data-feature="cleanup">
            <span class="material-icons">delete_sweep</span>
            <span>Content Deletion</span>
            <small>Clean Your Profile</small>
          </button>
        </div>

        <!-- Feature Content -->
        <div class="feature-content">
          ${this.renderFeatureContent()}
        </div>
      </div>
    `;
  }

  renderFeatureContent() {
    return this.features[this.currentFeature].render();
  }

  attachEventListeners() {
    this.container.addEventListener("click", (e) => {
      if (e.target.closest(".tab-btn")) {
        const feature = e.target.closest(".tab-btn").dataset.feature;
        this.switchFeature(feature);
      }
    });
  }

  switchFeature(feature) {
    this.currentFeature = feature;
    this.render();
    this.attachEventListeners();
  }
}
