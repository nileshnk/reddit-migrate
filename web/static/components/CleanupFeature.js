class CleanupFeature {
  constructor() {
    this.contentTypes = {
      comments: false,
      posts: false,
      savedItems: false,
      upvotes: false,
      downvotes: false,
      hiddenPosts: false,
    };
  }

  render() {
    return `
      <div class="cleanup-feature">
        <h3 class="text-xl font-semibold mb-6 flex items-center">
          <span class="material-icons mr-2 text-red-400">delete_sweep</span>
          Content Deletion
        </h3>

        <div class="content-types grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
          ${this.renderContentType("comments", "Comments", "comment")}
          ${this.renderContentType("posts", "Posts", "article")}
          ${this.renderContentType("savedItems", "Saved Items", "bookmark")}
          ${this.renderContentType("upvotes", "Upvotes", "thumb_up")}
          ${this.renderContentType("downvotes", "Downvotes", "thumb_down")}
          ${this.renderContentType(
            "hiddenPosts",
            "Hidden Posts",
            "visibility_off"
          )}
        </div>

        <div class="deletion-mode mb-6">
          <h4 class="text-lg font-semibold mb-4">Deletion Mode</h4>
          <div class="mode-options space-y-3">
            <label class="flex items-center">
              <input type="radio" name="deletionMode" value="editThenDelete" class="custom-radio" checked>
              <span class="ml-3">
                <span class="font-medium">Edit then Delete</span>
                <small class="block text-slate-400">Recommended for privacy</small>
              </span>
            </label>
            <label class="flex items-center">
              <input type="radio" name="deletionMode" value="deleteOnly" class="custom-radio">
              <span class="ml-3">
                <span class="font-medium">Delete Only</span>
                <small class="block text-slate-400">Direct deletion</small>
              </span>
            </label>
          </div>
        </div>

        <div class="advanced-options">
          <h4 class="text-lg font-semibold mb-4">Advanced Options</h4>
          <div class="options-grid grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label class="flex items-center">
                <input type="checkbox" class="mr-3">
                <span>Dry run mode</span>
              </label>
            </div>
            <div>
              <label class="flex items-center">
                <input type="checkbox" class="mr-3">
                <span>Custom replacement text</span>
              </label>
            </div>
          </div>
        </div>
      </div>
    `;
  }

  renderContentType(type, title, icon) {
    return `
      <label class="content-type-option flex items-center p-4 glass-card rounded-lg cursor-pointer hover:bg-slate-700/50">
        <input type="checkbox" name="contentTypes" value="${type}" class="custom-checkbox mr-3">
        <span class="material-icons mr-3 text-slate-400">${icon}</span>
        <span class="font-medium">${title}</span>
      </label>
    `;
  }
}
