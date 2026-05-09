(function () {
  "use strict";

  var MIN_SCROLL_MARGIN_PX = 24;
  // Defensive cap for postMessage-driven resizing to avoid runaway layout
  // from malformed or maliciously large height values.
  var MAX_IFRAME_HEIGHT_PX = 20000;
  // sRGB relative luminance coefficients from WCAG contrast calculation.
  var SRGB_LUMINANCE_R_COEFFICIENT = 0.2126;
  var SRGB_LUMINANCE_G_COEFFICIENT = 0.7152;
  var SRGB_LUMINANCE_B_COEFFICIENT = 0.0722;
  // Keep in sync with internal/i18n/embed.go.
  var SUPPORTED_LOCALES = ["en", "uk"];

  var script = document.currentScript;
  if (!script) return;

  var site = script.getAttribute("data-site");
  var page = script.getAttribute("data-page");
  var targetSelector = script.getAttribute("data-target") || "#comments";
  var theme = script.getAttribute("data-theme") || "auto";
  var sort = normalizeSort(script.getAttribute("data-sort") || "oldest");
  var inputPosition = normalizeInputPosition(script.getAttribute("data-input-position") || "bottom");
  var rawLocale = script.getAttribute("data-locale") || document.documentElement.lang || navigator.language || "en";
  var locale = normalizeLocale(rawLocale);
  var target = document.querySelector(targetSelector) || script.parentElement;
  var minHeight = 260;

  if (!site || !target) return;
  if (!page || page === "auto") page = window.location.pathname + window.location.search;

  var src = new URL("/embed/comments", script.src);
  src.searchParams.set("site", site);
  src.searchParams.set("page", page);
  src.searchParams.set("theme", theme);
  src.searchParams.set("sort", sort);
  src.searchParams.set("input_position", inputPosition);
  src.searchParams.set("locale", locale);
  src.searchParams.set("parent_origin", window.location.origin);
  src.searchParams.set("title", document.title || "");
  src.searchParams.set("url", window.location.href);

  function safeCSSValue(value) {
    value = String(value || "").trim();
    // Defensive sanitization for untrusted CSS values:
    // - 160 chars is enough for normal colors/font stacks while rejecting abnormal payloads.
    // - < > { } ; could break out of a single CSS value into markup, blocks, or declarations.
    if (!value || value.length > 160 || /[<>{};]/.test(value)) return "";
    return value;
  }
  function parseRGB(value) {
    var match = String(value || "").match(/rgba?\((\d+),\s*(\d+),\s*(\d+)(?:,\s*(\d*\.?\d+))?\)/i);
    if (!match) return null;
    return {
      r: Number(match[1]),
      g: Number(match[2]),
      b: Number(match[3]),
      a: match[4] === undefined ? 1 : Number(match[4])
    };
  }
  function isTransparent(value) {
    var rgb = parseRGB(value);
    return !value || value === "transparent" || (rgb && rgb.a === 0);
  }
  function hostBackground(element) {
    var node = element;
    while (node && node.nodeType === 1) {
      var bg = window.getComputedStyle(node).backgroundColor;
      if (!isTransparent(bg)) return bg;
      node = node.parentElement;
    }
    var bodyBg = window.getComputedStyle(document.body).backgroundColor;
    if (!isTransparent(bodyBg)) return bodyBg;
    return "#ffffff";
  }
  function linearizeSRGBChannel(channel) {
    return channel <= 0.03928 ? channel / 12.92 : Math.pow((channel + 0.055) / 1.055, 2.4);
  }
  function isDarkColor(value) {
    var rgb = parseRGB(value);
    if (!rgb) return false;
    var r = linearizeSRGBChannel(rgb.r / 255);
    var g = linearizeSRGBChannel(rgb.g / 255);
    var b = linearizeSRGBChannel(rgb.b / 255);
    var luminance = SRGB_LUMINANCE_R_COEFFICIENT * r + SRGB_LUMINANCE_G_COEFFICIENT * g + SRGB_LUMINANCE_B_COEFFICIENT * b;
    return luminance < 0.5;
  }
  function inheritedTheme() {
    var computed = window.getComputedStyle(target);
    var bg = hostBackground(target);
    var dark = isDarkColor(bg);
    return {
      text: safeCSSValue(computed.color) || (dark ? "#e6edf3" : "#202124"),
      muted: dark ? "rgba(230, 237, 243, .66)" : "rgba(32, 33, 36, .62)",
      border: dark ? "rgba(139, 148, 158, .34)" : "rgba(31, 35, 40, .18)",
      card: dark ? "rgba(22, 27, 34, .88)" : "rgba(246, 248, 250, .92)",
      surface: dark ? "rgba(13, 17, 23, .88)" : "rgba(255, 255, 255, .92)",
      accent: dark ? "#58a6ff" : "#0969da",
      danger: dark ? "#ff7b72" : "#b42318",
      success: dark ? "#3fb950" : "#1a7f37",
      successBg: dark ? "rgba(46, 160, 67, .14)" : "#dafbe1",
      warning: dark ? "#d29922" : "#9a6700",
      warningBg: dark ? "rgba(187, 128, 9, .18)" : "#fff8c5",
      font: safeCSSValue(computed.fontFamily) || "ui-sans-serif, system-ui, sans-serif"
    };
  }
  function themeVars(themeData) {
    if (!themeData) return "";
    var names = {
      text: "--dc-text",
      muted: "--dc-muted",
      border: "--dc-border",
      card: "--dc-card",
      surface: "--dc-surface",
      accent: "--dc-accent",
      danger: "--dc-danger",
      success: "--dc-success",
      successBg: "--dc-success-bg",
      warning: "--dc-warning",
      warningBg: "--dc-warning-bg",
      font: "--dc-font"
    };
    return Object.keys(names).map(function (key) {
      var value = safeCSSValue(themeData[key]);
      return value ? names[key] + ":" + value : "";
    }).filter(Boolean).join(";");
  }

  function normalizeLocale(value) {
    var normalized = String(value || "").toLowerCase().replace(/_/g, "-");
    var language = normalized.split("-")[0];
    return SUPPORTED_LOCALES.indexOf(language) !== -1 ? language : "en";
  }
  function normalizeSort(value) {
    value = String(value || "").toLowerCase();
    return value === "newest" || value === "best" ? value : "oldest";
  }
  function normalizeInputPosition(value) {
    return String(value || "").toLowerCase() === "top" ? "top" : "bottom";
  }
  function loadingText(value, key) {
    var catalog = {
      en: { comments: "Comments", loading: "Loading comments..." },
      uk: { comments: "Коментарі", loading: "Завантажуємо коментарі..." }
    };
    var normalized = normalizeLocale(value);
    return catalog[normalized][key] || catalog.en[key];
  }
  function loadingDocument(themeName, inheritedData) {
    var normalized = themeName === "light" || themeName === "dark" || themeName === "minimal" ? themeName : "auto";
    if (themeName === "inherit") normalized = "inherit";
    var inheritedVars = normalized === "inherit" ? themeVars(inheritedData) : "";
    return "<!doctype html><html lang=\"en\" data-theme=\"" + normalized + "\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><style>" +
      ":root{--dc-bg:transparent;--dc-text:#202124;--dc-muted:#667085;--dc-border:#d0d7de;--dc-card:#f6f8fa;--dc-surface:#fff;--dc-radius:6px;--dc-font:ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif}" +
      (inheritedVars ? ":root[data-theme=inherit]{" + inheritedVars + "}" : "") +
      "@media(prefers-color-scheme:dark){:root[data-theme=auto]{--dc-text:#e6edf3;--dc-muted:#8b949e;--dc-border:#30363d;--dc-card:#161b22;--dc-surface:#0d1117}}" +
      ":root[data-theme=dark]{--dc-text:#e6edf3;--dc-muted:#8b949e;--dc-border:#30363d;--dc-card:#161b22;--dc-surface:#0d1117}" +
      "*{box-sizing:border-box}body{margin:0;background:transparent;color:var(--dc-text);font:14px/1.5 var(--dc-font);overflow:hidden}.dc-loading{display:grid;gap:14px;padding:2px 2px 18px}.dc-title{display:flex;align-items:center;gap:8px;font-size:18px;font-weight:700;margin:0 0 6px}.dc-line,.dc-card{background:linear-gradient(90deg,var(--dc-card),var(--dc-surface),var(--dc-card));background-size:200% 100%;animation:dc-pulse 1.1s ease-in-out infinite;border-radius:var(--dc-radius)}.dc-line{height:16px;width:160px}.dc-card{height:96px;border:1px solid var(--dc-border)}.dc-small{color:var(--dc-muted);font-size:12px}@keyframes dc-pulse{0%{background-position:200% 0}100%{background-position:-200% 0}}" +
      "</style></head><body><div class=\"dc-loading\" aria-live=\"polite\"><div class=\"dc-title\">" + loadingText(locale, "comments") + "</div><div class=\"dc-small\">" + loadingText(locale, "loading") + "</div><div class=\"dc-card\"></div><div class=\"dc-line\"></div></div></body></html>";
  }

  var cachedInheritedTheme = null;
  var cachedInheritedThemeSignature = null;
  function inheritedThemeSignature() {
    var computed = window.getComputedStyle(target);
    return [
      computed.color,
      hostBackground(target),
      computed.fontFamily
    ].join("|");
  }
  function currentInheritedTheme() {
    var signature = inheritedThemeSignature();
    if (cachedInheritedThemeSignature !== signature) {
      cachedInheritedThemeSignature = signature;
      cachedInheritedTheme = inheritedTheme();
    }
    return cachedInheritedTheme;
  }
  var inherited = theme === "inherit" ? currentInheritedTheme() : null;

  var iframe = document.createElement("iframe");
  iframe.srcdoc = loadingDocument(theme, inherited);
  iframe.title = loadingText(locale, "comments");
  iframe.loading = "lazy";
  iframe.style.width = "100%";
  iframe.style.border = "0";
  iframe.style.display = "block";
  iframe.style.overflow = "hidden";
  iframe.style.minHeight = minHeight + "px";
  iframe.style.height = minHeight + "px";
  iframe.setAttribute("scrolling", "no");

  function sendInheritedTheme() {
    if (theme !== "inherit" || !iframe.contentWindow) return;
    var inheritedData = currentInheritedTheme();
    if (!inheritedData) return;
    iframe.contentWindow.postMessage({type: "deadcomments:theme", theme: inheritedData}, src.origin);
  }
  var themeTimer = null;
  iframe.addEventListener("load", sendInheritedTheme);
  window.addEventListener("resize", function () {
    window.clearTimeout(themeTimer);
    themeTimer = window.setTimeout(sendInheritedTheme, 100);
  });

  window.addEventListener("message", function (event) {
    if (!event.data) return;
    if (event.origin === window.location.origin && event.data.type === "deadcomments:commentFocus") {
      if (iframe.contentWindow) iframe.contentWindow.postMessage(event.data, src.origin);
      return;
    }
    if (event.origin !== src.origin) return;
    if (event.data.type === "deadcomments:height") {
      var height = Number(event.data.height);
      if (height > 0 && height < MAX_IFRAME_HEIGHT_PX) iframe.style.height = Math.max(minHeight, height) + "px";
      return;
    }
    if (event.data.type === "deadcomments:scrollIntoView") {
      var top = Number(event.data.top);
      var itemHeight = Number(event.data.height) || 0;
      if (Number.isFinite(top) && top >= 0) {
        var iframeRect = iframe.getBoundingClientRect();
        var scrollTargetY = window.pageYOffset + iframeRect.top + top - Math.max(MIN_SCROLL_MARGIN_PX, (window.innerHeight - itemHeight) / 2);
        window.scrollTo({top: Math.max(0, scrollTargetY), behavior: "smooth"});
      } else {
        iframe.scrollIntoView({block: "center", behavior: "smooth"});
      }
      return;
    }
    if (event.data.type === "deadcomments:annotationFocus") {
      window.postMessage(event.data, window.location.origin);
    }
  });

  target.appendChild(iframe);
  window.requestAnimationFrame(function () {
    iframe.removeAttribute("srcdoc");
    iframe.src = src.toString();
  });
})();
