(function () {
  "use strict";
  function loadComments() {
    document.querySelectorAll("[data-deadcomments-page]").forEach(function (node, index) {
      if (node.dataset.deadcommentsLoaded === "1") return;
      node.dataset.deadcommentsLoaded = "1";
      if (!node.id) node.id = "deadcomments-" + index;
      var script = document.createElement("script");
      script.src = "http://localhost:8080/widget.js";
      script.setAttribute("data-site", "docs-demo");
      script.setAttribute("data-page", node.getAttribute("data-deadcomments-page"));
      script.setAttribute("data-target", "#" + node.id);
      script.setAttribute("data-theme", node.getAttribute("data-theme") || "auto");
      node.after(script);
    });
  }
  document.addEventListener("DOMContentLoaded", loadComments);
  document.addEventListener("docusaurus:routeDidUpdate", loadComments);
})();
