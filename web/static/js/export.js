// Export/Backup to JSON handler

import * as state from "./state.js";
import { getAuthRequestBody, isSourceAccountVerified } from "./auth.js";

export function initExportHandler() {
  const exportBtn = document.getElementById("export-btn");
  const exportLoadingBtn = document.getElementById("export-loading-btn");

  exportBtn.addEventListener("click", async () => {
    if (!isSourceAccountVerified()) {
      alert("Please verify the source account first");
      return;
    }

    // Gather selected sections
    const sections = [];
    document.querySelectorAll("#export-section input[type='checkbox']:checked").forEach((cb) => {
      sections.push(cb.value);
    });

    if (sections.length === 0) {
      alert("Please select at least one data type to export");
      return;
    }

    exportBtn.style.display = "none";
    exportLoadingBtn.style.display = "flex";

    try {
      const authBody = getAuthRequestBody();
      const requestBody = {
        ...authBody,
        sections: sections,
      };

      const response = await fetch(`${state.API_BASE_URL}/api/export`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(requestBody),
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(errorText);
      }

      // Get the filename from Content-Disposition header or generate one
      const contentDisposition = response.headers.get("Content-Disposition");
      let filename = "reddit-export.json";
      if (contentDisposition) {
        const match = contentDisposition.match(/filename="(.+)"/);
        if (match) {
          filename = match[1];
        }
      }

      // Download the file
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      window.URL.revokeObjectURL(url);
    } catch (error) {
      console.error("Export error:", error);
      alert("Export failed: " + error.message);
    } finally {
      exportBtn.style.display = "flex";
      exportLoadingBtn.style.display = "none";
    }
  });
}
