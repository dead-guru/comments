(function () {
  "use strict";

  var script = document.currentScript;
  if (!script) return;

  var site = script.getAttribute("data-site");
  var page = script.getAttribute("data-page");
  var targetSelector = script.getAttribute("data-target") || "#comments";
  var theme = script.getAttribute("data-theme") || "auto";
  var target = document.querySelector(targetSelector) || script.parentElement;

  if (!site || !target) return;
  if (!page || page === "auto") page = window.location.pathname + window.location.search;

  var src = new URL("/embed/comments", script.src);
  src.searchParams.set("site", site);
  src.searchParams.set("page", page);
  src.searchParams.set("theme", theme);
  src.searchParams.set("parent_origin", window.location.origin);
  src.searchParams.set("title", document.title || "");
  src.searchParams.set("url", window.location.href);

  var iframe = document.createElement("iframe");
  iframe.src = src.toString();
  iframe.title = "Comments";
  iframe.loading = "lazy";
  iframe.style.width = "100%";
  iframe.style.border = "0";
  iframe.style.display = "block";
  iframe.style.overflow = "hidden";
  iframe.style.minHeight = "180px";
  iframe.setAttribute("scrolling", "no");

  window.addEventListener("message", function (event) {
    if (event.origin !== src.origin) return;
    if (!event.data) return;
    if (event.data.type === "deadcomments:height") {
      var height = Number(event.data.height);
      if (height > 0 && height < 20000) iframe.style.height = Math.max(180, height) + "px";
      return;
    }
    if (event.data.type === "deadcomments:scrollIntoView") {
      iframe.scrollIntoView({block: "center", behavior: "smooth"});
    }
  });

  target.appendChild(iframe);
})();
