// UI utilities: DarkModeManager, TabManager, tooltips, password toggle, help modals

import { setCurrentAuthMethod } from "./state.js";
import { updateSubmitButtonState } from "./auth.js";

// Dark Mode Management
export class DarkModeManager {
  constructor() {
    this.darkModeToggle = document.getElementById("darkModeToggle");
    this.themeIcon = document.getElementById("themeIcon");
    this.init();
  }

  init() {
    const isLightMode = localStorage.getItem("lightMode") === "true";

    if (isLightMode) {
      this.enableLightMode();
    } else {
      this.enableDarkMode();
    }

    this.darkModeToggle.addEventListener("click", () => {
      this.toggle();
    });
  }

  toggle() {
    const isCurrentlyLight = document.body.classList.contains("light-mode");

    if (isCurrentlyLight) {
      this.enableDarkMode();
      localStorage.setItem("lightMode", "false");
    } else {
      this.enableLightMode();
      localStorage.setItem("lightMode", "true");
    }
  }

  enableDarkMode() {
    document.body.classList.remove("light-mode");
    document.body.className =
      "bg-gradient-to-br from-zinc-900 via-neutral-900 to-black text-white min-h-screen transition-all duration-300";
    this.themeIcon.textContent = "light_mode";
  }

  enableLightMode() {
    document.body.classList.add("light-mode");
    document.body.className =
      "bg-gradient-to-br from-white via-gray-50 to-gray-100 text-gray-900 min-h-screen transition-all duration-300 light-mode";
    this.themeIcon.textContent = "dark_mode";
  }
}

// Tab Management
export class TabManager {
  constructor() {
    this.cookieTab = document.getElementById("cookieTab");
    this.oauthTab = document.getElementById("oauthTab");
    this.cookieContent = document.getElementById("cookieAuthContent");
    this.oauthContent = document.getElementById("oauthAuthContent");

    console.log("TabManager elements:", {
      cookieTab: this.cookieTab,
      oauthTab: this.oauthTab,
      cookieContent: this.cookieContent,
      oauthContent: this.oauthContent,
    });

    this.init();
  }

  init() {
    if (
      !this.cookieTab ||
      !this.oauthTab ||
      !this.cookieContent ||
      !this.oauthContent
    ) {
      console.error("Tab elements not found. Check HTML element IDs.");
      return;
    }

    this.cookieTab.addEventListener("click", (e) => {
      e.preventDefault();
      console.log("Cookie tab clicked");
      this.switchTab("cookie");
    });

    this.oauthTab.addEventListener("click", (e) => {
      e.preventDefault();
      console.log("OAuth tab clicked");
      this.switchTab("oauth");
    });
  }

  switchTab(method) {
    console.log("Switching to tab:", method);
    setCurrentAuthMethod(method);

    if (method === "cookie") {
      console.log("Switching to cookie tab");
      this.cookieTab.classList.add("active");
      this.oauthTab.classList.remove("active");
      this.cookieContent.classList.remove("hidden");
      this.oauthContent.classList.add("hidden");
    } else {
      console.log("Switching to oauth tab");
      this.oauthTab.classList.add("active");
      this.cookieTab.classList.remove("active");
      this.oauthContent.classList.remove("hidden");
      this.cookieContent.classList.add("hidden");
    }

    console.log("Tab classes after switch:", {
      cookieTab: this.cookieTab.className,
      oauthTab: this.oauthTab.className,
      cookieContent: this.cookieContent.className,
      oauthContent: this.oauthContent.className,
    });

    updateSubmitButtonState();
  }
}

// Tooltip
export function toggleTooltip(event) {
  event.stopPropagation();
  const tooltip = document.getElementById("tooltip");
  tooltip.classList.toggle("hidden");
}

// Password Visibility Toggle
export function togglePasswordVisibility(inputId, eyeIconId) {
  const input = document.getElementById(inputId);
  const eyeIcon = document.getElementById(eyeIconId);

  if (input.type === "password") {
    input.type = "text";
    eyeIcon.textContent = "visibility_off";
  } else {
    input.type = "password";
    eyeIcon.textContent = "visibility";
  }
}

// Cookie Help Modal
export function showCookieHelp() {
  const modal = document.getElementById("cookieHelpModal");
  modal.classList.remove("hidden");
  modal.classList.add("animate-slide-in");
}

export function hideCookieHelp() {
  const modal = document.getElementById("cookieHelpModal");
  modal.classList.add("hidden");
}

// OAuth Help Modal
export function showOAuthHelp() {
  const modal = document.getElementById("oauthHelpModal");
  modal.classList.remove("hidden");
  modal.classList.add("animate-slide-in");
}

export function hideOAuthHelp() {
  const modal = document.getElementById("oauthHelpModal");
  modal.classList.add("hidden");
}

// Initialize UI event listeners (called from app.js)
export function initUIListeners() {
  // Close tooltip if clicked outside
  document.addEventListener("click", function (event) {
    const tooltip = document.getElementById("tooltip");
    const tooltipButton = tooltip.previousElementSibling;
    if (
      !tooltip.classList.contains("hidden") &&
      !tooltip.contains(event.target) &&
      !tooltipButton.contains(event.target)
    ) {
      tooltip.classList.add("hidden");
    }
  });

  // Close help modals when clicking outside
  document
    .getElementById("cookieHelpModal")
    ?.addEventListener("click", function (event) {
      if (event.target === this) {
        hideCookieHelp();
      }
    });

  document
    .getElementById("oauthHelpModal")
    ?.addEventListener("click", function (event) {
      if (event.target === this) {
        hideOAuthHelp();
      }
    });
}

// Expose functions used in HTML onclick attributes to window
window.toggleTooltip = toggleTooltip;
window.togglePasswordVisibility = togglePasswordVisibility;
window.showCookieHelp = showCookieHelp;
window.hideCookieHelp = hideCookieHelp;
window.showOAuthHelp = showOAuthHelp;
window.hideOAuthHelp = hideOAuthHelp;
