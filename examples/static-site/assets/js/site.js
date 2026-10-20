// The static-site package's script: a light and dark theme toggle, kept in
// the reader's browser. It loads nothing and sends nothing; without it, the
// pages read the same in the light theme.
(function () {
  "use strict";
  var key = "static-site-theme";
  var root = document.documentElement;

  function stored() {
    try {
      return window.localStorage.getItem(key);
    } catch (e) {
      return null;
    }
  }

  function apply(theme, button) {
    root.setAttribute("data-theme", theme);
    if (button) {
      button.textContent = theme === "dark" ? "Light theme" : "Dark theme";
    }
  }

  var initial = stored();
  if (initial !== "dark" && initial !== "light") {
    initial = window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  }
  apply(initial, null);

  document.addEventListener("DOMContentLoaded", function () {
    var button = document.querySelector(".theme-toggle");
    if (!button) {
      return;
    }
    apply(root.getAttribute("data-theme"), button);
    button.hidden = false;
    button.addEventListener("click", function () {
      var next = root.getAttribute("data-theme") === "dark" ? "light" : "dark";
      apply(next, button);
      try {
        window.localStorage.setItem(key, next);
      } catch (e) {
        // A browser that keeps nothing still switches the theme for this page.
      }
    });
  });
})();
