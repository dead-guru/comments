(function () {
  "use strict";

  var attempts = 0;
  var maxAttempts = 40;

  function loadComments() {
    document.querySelectorAll("[data-deadcomments-page]").forEach(function (node, index) {
      if (node.dataset.deadcommentsLoaded === "1") return;
      node.dataset.deadcommentsLoaded = "1";
      if (!node.id) node.id = "deadcomments-" + index + "-" + Math.random().toString(36).slice(2);
      var script = document.createElement("script");
      script.src = "http://localhost:8080/widget.js";
      script.async = true;
      script.setAttribute("data-site", "docs-demo");
      script.setAttribute("data-page", node.getAttribute("data-deadcomments-page"));
      script.setAttribute("data-target", "#" + node.id);
      script.setAttribute("data-theme", node.getAttribute("data-theme") || "auto");
      script.setAttribute("data-sort", node.getAttribute("data-sort") || "oldest");
      script.setAttribute("data-input-position", node.getAttribute("data-input-position") || "bottom");
      node.after(script);
    });
  }

  function scheduleLoad() {
    loadComments();
    if (attempts >= maxAttempts) return;
    attempts += 1;
    window.setTimeout(scheduleLoad, 250);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", scheduleLoad, { once: true });
  } else {
    scheduleLoad();
  }

  document.addEventListener("docusaurus:routeDidUpdate", loadComments);
  new MutationObserver(loadComments).observe(document.documentElement, { childList: true, subtree: true });
})();
