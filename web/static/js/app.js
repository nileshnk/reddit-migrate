// Application entry point — imports all modules and initializes the app

import { updateSubmitButtonState, initCookieAuthListeners, OAuthModalManager } from "./auth.js";
import { DarkModeManager, TabManager, initUIListeners } from "./ui.js";
import { SelectionModal, initSelectionListeners } from "./modal.js";
import { initMigrationHandler } from "./migration.js";
import { initExportHandler } from "./export.js";

document.addEventListener("DOMContentLoaded", () => {
  console.log("DOM Content Loaded - Initializing application");

  // Set default selections
  document.getElementById("subredditNone").checked = true;
  document.getElementById("postsNone").checked = true;
  document.getElementById("commentsNone").checked = true;
  document.getElementById("multiredditNone").checked = true;
  document.getElementById("deleteSubredditsNo").checked = true;
  document.getElementById("deletePostsNo").checked = true;
  document.getElementById("deleteCommentsNo").checked = true;

  updateSubmitButtonState();

  try {
    // Initialize managers
    new DarkModeManager();
    console.log("DarkModeManager initialized");

    new TabManager();
    console.log("TabManager initialized");

    new OAuthModalManager();
    console.log("OAuthModalManager initialized");

    // Initialize selection modal and listeners
    const selectionModal = new SelectionModal();
    window._selectionModal = selectionModal; // Expose for inline onclick in rendered HTML
    initSelectionListeners(selectionModal);
    console.log("SelectionModal initialized");

    // Initialize cookie auth listeners
    initCookieAuthListeners();
    console.log("Cookie auth listeners initialized");

    // Initialize UI listeners (tooltips, help modals)
    initUIListeners();
    console.log("UI listeners initialized");

    // Initialize migration handler
    initMigrationHandler();
    console.log("Migration handler initialized");

    // Initialize export handler
    initExportHandler();
    console.log("Export handler initialized");
  } catch (error) {
    console.error("Error initializing application:", error);
  }
});
