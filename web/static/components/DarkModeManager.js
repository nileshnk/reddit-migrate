// Dark Mode Management
class DarkModeManager {
  constructor() {
    this.darkModeToggle = document.getElementById("darkModeToggle");
    this.themeIcon = document.getElementById("themeIcon");
    this.init();
  }

  init() {
    // Check for saved theme preference or default to dark mode
    const isLightMode = localStorage.getItem("lightMode") === "true";

    if (isLightMode) {
      this.enableLightMode();
    } else {
      this.enableDarkMode();
    }

    // Add event listener
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
