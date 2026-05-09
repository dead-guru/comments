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

  function loadingDocument(themeName) {
    var normalized = themeName === "light" || themeName === "dark" || themeName === "minimal" ? themeName : "auto";
    return "<!doctype html><html lang=\"en\" data-theme=\"" + normalized + "\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><style>" +
      ":root{--dc-bg:transparent;--dc-text:#202124;--dc-muted:#667085;--dc-border:#d0d7de;--dc-card:#f6f8fa;--dc-surface:#fff;--dc-radius:6px;--dc-font:ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif}" +
      "@media(prefers-color-scheme:dark){:root[data-theme=auto]{--dc-text:#e6edf3;--dc-muted:#8b949e;--dc-border:#30363d;--dc-card:#161b22;--dc-surface:#0d1117}}" +
      ":root[data-theme=dark]{--dc-text:#e6edf3;--dc-muted:#8b949e;--dc-border:#30363d;--dc-card:#161b22;--dc-surface:#0d1117}" +
      "*{box-sizing:border-box}body{margin:0;background:transparent;color:var(--dc-text);font:14px/1.5 var(--dc-font)}.dc-loading{display:grid;gap:12px;padding:2px}.dc-title{display:flex;align-items:center;gap:8px;font-size:18px;font-weight:700;margin:0 0 6px}.dc-line,.dc-card{background:linear-gradient(90deg,var(--dc-card),var(--dc-surface),var(--dc-card));background-size:200% 100%;animation:dc-pulse 1.1s ease-in-out infinite;border-radius:var(--dc-radius)}.dc-line{height:16px;width:160px}.dc-card{height:108px;border:1px solid var(--dc-border)}.dc-small{color:var(--dc-muted);font-size:12px}@keyframes dc-pulse{0%{background-position:200% 0}100%{background-position:-200% 0}}" +
      "</style></head><body><div class=\"dc-loading\" aria-live=\"polite\"><div class=\"dc-title\">Comments</div><div class=\"dc-small\">Loading comments...</div><div class=\"dc-card\"></div><div class=\"dc-line\"></div></div></body></html>";
  }

  var iframe = document.createElement("iframe");
  iframe.srcdoc = loadingDocument(theme);
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
  window.requestAnimationFrame(function () {
    iframe.src = src.toString();
  });
})();
