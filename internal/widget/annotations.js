(function () {
  "use strict";

  var script = document.currentScript;
  if (!script) return;

  var site = script.getAttribute("data-site");
  var page = script.getAttribute("data-page");
  var contentSelector = script.getAttribute("data-content-selector") || script.getAttribute("data-selector") || "article, main";
  var locale = normalizeLocale(script.getAttribute("data-locale") || document.documentElement.lang || navigator.language || "en");
  var minSelectionLength = clampNumber(script.getAttribute("data-min-selection-length"), 2, 1, 200);
  var maxSelectionLength = clampNumber(script.getAttribute("data-max-selection-length"), 2000, 100, 5000);
  var apiURL;
  var annotations = [];
  var groups = Object.create(null);
  var activeRange = null;
  var activeSelection = null;
  var popover = null;
  var panel = null;
  var messageTimer = null;

  if (!site) return;
  if (!page || page === "auto") page = window.location.pathname + window.location.search;

  apiURL = new URL("/api/v1/sites/" + encodeURIComponent(site) + "/pages/" + encodeURIComponent(page) + "/annotations", script.src);

  injectStyles();
  assignRootAnchors();
  loadAnnotations();

  document.addEventListener("mouseup", handleSelection);
  document.addEventListener("keyup", function (event) {
    if (event.key === "Escape") {
      closePopover();
      closePanel();
      return;
    }
    handleSelection();
  });
  document.addEventListener("selectionchange", function () {
    if (!window.getSelection || !window.getSelection().toString().trim()) {
      return;
    }
    window.clearTimeout(messageTimer);
    messageTimer = window.setTimeout(handleSelection, 80);
  });
  window.addEventListener("message", function (event) {
    if (event.origin !== window.location.origin || !event.data) return;
    if (event.data.type === "deadcomments:annotationFocus") {
      focusAnnotation(event.data.annotation_id);
    }
  });
  window.addEventListener("resize", closePopover);
  window.setInterval(loadAnnotations, 30000);

  function t(key) {
    var dict = {
      en: {
        add: "Comment on selection",
        name: "Name",
        email: "Email (optional)",
        website: "Website (optional)",
        comment: "Comment",
        submit: "Submit",
        submitting: "Submitting...",
        cancel: "Cancel",
        selected: "Selected text",
        posted: "Comment posted.",
        pending: "Comment submitted and waiting for moderation.",
        failed: "Could not save the annotation.",
        tooLong: "Selection is too long.",
        tooShort: "Select text to comment.",
        thread: "Selection comments",
        close: "Close",
        empty: "No comments on this selection yet.",
        tripcode: "Use Name##secret for a tripcode",
        avatar: "Used only for avatar"
      },
      uk: {
        add: "Коментар до виділення",
        name: "Ім'я",
        email: "Email (не показується)",
        website: "Сайт (необов'язково)",
        comment: "Коментар",
        submit: "Надіслати",
        submitting: "Надсилаємо...",
        cancel: "Скасувати",
        selected: "Виділений текст",
        posted: "Коментар опубліковано.",
        pending: "Коментар надіслано і чекає модерації.",
        failed: "Не вдалося зберегти коментар.",
        tooLong: "Виділення завелике.",
        tooShort: "Виділіть текст для коментаря.",
        thread: "Коментарі до виділення",
        close: "Закрити",
        empty: "До цього виділення ще немає коментарів.",
        tripcode: "Name##secret створить tripcode",
        avatar: "Використовується лише для аватара"
      }
    };
    return (dict[locale] && dict[locale][key]) || dict.en[key] || key;
  }

  function normalizeLocale(value) {
    value = String(value || "").toLowerCase().replace(/_/g, "-");
    return value.indexOf("uk") === 0 ? "uk" : "en";
  }

  function clampNumber(value, fallback, min, max) {
    var n = Number(value);
    if (!Number.isFinite(n)) return fallback;
    return Math.max(min, Math.min(max, Math.floor(n)));
  }

  function cssEscape(value) {
    if (window.CSS && typeof window.CSS.escape === "function") return window.CSS.escape(value);
    return String(value).replace(/["\\]/g, "\\$&");
  }

  function assignRootAnchors() {
    roots().forEach(function (root, index) {
      if (!root.getAttribute("data-dc-annotation-root")) {
        root.setAttribute("data-dc-annotation-root", String(index));
      }
    });
  }

  function roots() {
    return Array.prototype.slice.call(document.querySelectorAll(contentSelector)).filter(function (node) {
      return node && node.nodeType === 1;
    });
  }

  function rootSelector(root) {
    if (root.id) return "#" + cssEscape(root.id);
    if (root.getAttribute("data-dc-anchor")) return '[data-dc-anchor="' + cssEscape(root.getAttribute("data-dc-anchor")) + '"]';
    return '[data-dc-annotation-root="' + cssEscape(root.getAttribute("data-dc-annotation-root") || "0") + '"]';
  }

  function allowedRootForRange(range) {
    var allRoots = roots();
    for (var i = 0; i < allRoots.length; i += 1) {
      if (allRoots[i].contains(range.commonAncestorContainer)) return allRoots[i];
      if (range.commonAncestorContainer === allRoots[i]) return allRoots[i];
    }
    return null;
  }

  function handleSelection() {
    if (!window.getSelection) return;
    var selection = window.getSelection();
    if (!selection || selection.rangeCount === 0 || selection.isCollapsed) return;
    var text = selection.toString().trim();
    if (text.length < minSelectionLength) return;
    if (text.length > maxSelectionLength) {
      showDocumentMessage(t("tooLong"), "warning");
      return;
    }
    var range = selection.getRangeAt(0).cloneRange();
    var root = allowedRootForRange(range);
    if (!root) return;
    var offsets = rangeOffsets(root, range);
    activeRange = range;
    activeSelection = {
      root: root,
      selector: rootSelector(root),
      text: text,
      textStart: offsets.start,
      textEnd: offsets.end,
      prefix: contextBefore(root.textContent || "", offsets.start),
      suffix: contextAfter(root.textContent || "", offsets.end)
    };
    openPopover(range);
  }

  function rangeOffsets(root, range) {
    var before = document.createRange();
    before.selectNodeContents(root);
    before.setEnd(range.startContainer, range.startOffset);
    var start = before.toString().length;
    before.detach && before.detach();
    return { start: start, end: start + range.toString().length };
  }

  function contextBefore(text, offset) {
    return text.slice(Math.max(0, offset - 160), offset).trim();
  }

  function contextAfter(text, offset) {
    return text.slice(offset, Math.min(text.length, offset + 160)).trim();
  }

  function openPopover(range) {
    closePopover();
    popover = document.createElement("form");
    popover.className = "dc-annotation-popover";
    popover.innerHTML = [
      '<div class="dc-annotation-title"></div>',
      '<blockquote class="dc-annotation-quote"></blockquote>',
      '<div class="dc-annotation-grid">',
      '<label><span></span><input name="author_name" autocomplete="name" required></label>',
      '<label><span></span><input name="author_email" type="email" autocomplete="email"></label>',
      '</div>',
      '<div class="dc-annotation-hints"><span></span><span></span></div>',
      '<label><span></span><input name="author_website" type="url" autocomplete="url"></label>',
      '<label><span></span><textarea name="body_markdown" rows="4" required></textarea></label>',
      '<input name="honeypot" class="dc-annotation-honeypot" tabindex="-1" autocomplete="off">',
      '<div class="dc-annotation-message" role="alert" aria-live="polite"></div>',
      '<div class="dc-annotation-actions"><button class="dc-annotation-submit" type="submit"></button><button class="dc-annotation-cancel" type="button"></button></div>'
    ].join("");
    fillPopoverText(popover);
    restoreProfile(popover);
    popover.addEventListener("submit", submitAnnotation);
    popover.querySelector(".dc-annotation-cancel").addEventListener("click", closePopover);
    document.body.appendChild(popover);
    positionPopover(popover, range);
    popover.querySelector('textarea[name="body_markdown"]').focus();
  }

  function fillPopoverText(form) {
    var labels = form.querySelectorAll("label > span");
    labels[0].textContent = t("name");
    labels[1].textContent = t("email");
    labels[2].textContent = t("website");
    labels[3].textContent = t("comment");
    var hints = form.querySelectorAll(".dc-annotation-hints span");
    hints[0].textContent = t("tripcode");
    hints[1].textContent = t("avatar");
    form.querySelector(".dc-annotation-title").textContent = t("add");
    form.querySelector(".dc-annotation-quote").textContent = activeSelection ? activeSelection.text : "";
    form.querySelector(".dc-annotation-submit").textContent = t("submit");
    form.querySelector(".dc-annotation-cancel").textContent = t("cancel");
  }

  function positionPopover(node, range) {
    var rect = range.getBoundingClientRect();
    var width = Math.min(520, Math.max(320, window.innerWidth - 32));
    node.style.width = width + "px";
    var left = Math.min(window.innerWidth - width - 16, Math.max(16, rect.left + rect.width / 2 - width / 2));
    var top = Math.max(16, rect.bottom + window.scrollY + 12);
    node.style.left = left + window.scrollX + "px";
    node.style.top = top + "px";
  }

  function closePopover() {
    if (popover && popover.parentNode) popover.parentNode.removeChild(popover);
    popover = null;
  }

  function profileKey() {
    return "deadcomments:annotation-profile:" + site;
  }

  function restoreProfile(form) {
    try {
      var saved = JSON.parse(localStorage.getItem(profileKey()) || "{}");
      ["author_name", "author_email", "author_website"].forEach(function (name) {
        if (saved[name]) form.elements[name].value = saved[name];
      });
    } catch (_) {}
  }

  function saveProfile(form) {
    try {
      localStorage.setItem(profileKey(), JSON.stringify({
        author_name: form.elements.author_name.value,
        author_email: form.elements.author_email.value,
        author_website: form.elements.author_website.value
      }));
    } catch (_) {}
  }

  function submitAnnotation(event) {
    event.preventDefault();
    if (!activeSelection) return;
    var form = event.currentTarget;
    var button = form.querySelector(".dc-annotation-submit");
    var oldText = button.textContent;
    button.disabled = true;
    button.textContent = t("submitting");
    setFormMessage(form, "", "");
    saveProfile(form);
    fetch(apiURL.toString(), {
      method: "POST",
      headers: {"Content-Type": "application/json"},
      body: JSON.stringify({
        author_name: form.elements.author_name.value,
        author_email: form.elements.author_email.value,
        author_website: form.elements.author_website.value,
        body_markdown: form.elements.body_markdown.value,
        honeypot: form.elements.honeypot.value,
        page_title: document.title || "",
        page_url: window.location.href,
        locale: locale,
        selector: activeSelection.selector,
        selected_text: activeSelection.text,
        selection_prefix: activeSelection.prefix,
        selection_suffix: activeSelection.suffix,
        text_start: activeSelection.textStart,
        text_end: activeSelection.textEnd
      })
    }).then(function (response) {
      return response.json().then(function (data) {
        if (!response.ok && !data.annotation) throw new Error(data.error || data.message || t("failed"));
        return data;
      });
    }).then(function (data) {
      var annotation = data.annotation;
      if (annotation) {
        annotation._localPending = data.status === "pending";
        addAnnotations([annotation]);
        openPanel(groupKey(annotation));
      }
      showDocumentMessage(data.message || (data.status === "pending" ? t("pending") : t("posted")), data.status === "pending" ? "warning" : "success");
      closePopover();
      if (window.getSelection) window.getSelection().removeAllRanges();
    }).catch(function (error) {
      setFormMessage(form, error.message || t("failed"), "error");
    }).finally(function () {
      button.disabled = false;
      button.textContent = oldText;
    });
  }

  function setFormMessage(form, text, kind) {
    var node = form.querySelector(".dc-annotation-message");
    node.textContent = text || "";
    node.className = "dc-annotation-message" + (kind ? " is-" + kind : "");
  }

  function showDocumentMessage(text, kind) {
    var node = document.querySelector(".dc-annotation-toast");
    if (!node) {
      node = document.createElement("div");
      node.className = "dc-annotation-toast";
      node.setAttribute("role", "status");
      node.setAttribute("aria-live", "polite");
      document.body.appendChild(node);
    }
    node.textContent = text;
    node.className = "dc-annotation-toast is-visible is-" + (kind || "info");
    window.clearTimeout(messageTimer);
    messageTimer = window.setTimeout(function () {
      node.classList.remove("is-visible");
    }, 3600);
  }

  function loadAnnotations() {
    return fetch(apiURL.toString(), {headers: {"Accept": "application/json"}})
      .then(function (response) {
        if (!response.ok) throw new Error("annotations unavailable");
        return response.json();
      })
      .then(function (data) {
        addAnnotations(data.annotations || []);
      })
      .catch(function () {});
  }

  function addAnnotations(items) {
    var known = Object.create(null);
    annotations.forEach(function (annotation) {
      known[annotation.id] = true;
    });
    var fresh = [];
    items.forEach(function (annotation) {
      if (!annotation || !annotation.id || known[annotation.id]) return;
      known[annotation.id] = true;
      annotations.push(annotation);
      fresh.push(annotation);
    });
    if (fresh.length === 0) return;
    renderHighlights();
  }

  function groupKey(annotation) {
    return [annotation.selector, annotation.text_start, annotation.text_end, annotation.text_hash || annotation.selected_text].join("|");
  }

  function renderHighlights() {
    groups = Object.create(null);
    annotations.forEach(function (annotation) {
      var key = groupKey(annotation);
      if (!groups[key]) groups[key] = [];
      groups[key].push(annotation);
    });
    Object.keys(groups).map(function (key) {
      return { key: key, item: groups[key][0] };
    }).sort(function (a, b) {
      return Number(b.item.text_start || 0) - Number(a.item.text_start || 0);
    }).forEach(function (entry) {
      highlightGroup(entry.key, groups[entry.key]);
    });
  }

  function highlightGroup(key, group) {
    var annotation = group[0];
    if (document.querySelector('[data-dc-annotation-key="' + cssEscape(key) + '"]')) return;
    var root = document.querySelector(annotation.selector);
    if (!root) return;
    var range = rangeForAnnotation(root, annotation);
    if (!range) return;
    var mark = document.createElement("mark");
    mark.className = "dc-annotation-mark" + (annotation._localPending ? " is-pending" : "");
    mark.setAttribute("data-dc-annotation-key", key);
    mark.setAttribute("tabindex", "0");
    mark.setAttribute("role", "button");
    mark.setAttribute("aria-label", t("thread") + ": " + annotation.selected_text);
    if (group.length > 1) mark.setAttribute("data-count", String(group.length));
    try {
      var fragment = range.extractContents();
      mark.appendChild(fragment);
      range.insertNode(mark);
    } catch (_) {
      return;
    }
    mark.addEventListener("click", function () { openPanel(key, {focusComment: true}); });
    mark.addEventListener("keydown", function (event) {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        openPanel(key, {focusComment: true});
      }
    });
  }

  function rangeForAnnotation(root, annotation) {
    var text = root.textContent || "";
    var selected = annotation.selected_text || "";
    var start = Number(annotation.text_start);
    if (!Number.isFinite(start) || text.slice(start, start + selected.length) !== selected) {
      start = text.indexOf(selected);
    }
    if (start < 0) return null;
    return rangeFromTextOffsets(root, start, start + selected.length);
  }

  function textNodes(root) {
    var nodes = [];
    var walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
      acceptNode: function (node) {
        var parent = node.parentElement;
        if (!parent) return NodeFilter.FILTER_REJECT;
        if (/^(SCRIPT|STYLE|TEXTAREA|INPUT)$/i.test(parent.tagName)) return NodeFilter.FILTER_REJECT;
        return NodeFilter.FILTER_ACCEPT;
      }
    });
    var offset = 0;
    var node;
    while ((node = walker.nextNode())) {
      nodes.push({node: node, start: offset, end: offset + node.nodeValue.length});
      offset += node.nodeValue.length;
    }
    return nodes;
  }

  function rangeFromTextOffsets(root, start, end) {
    var nodes = textNodes(root);
    var startNode = null;
    var endNode = null;
    for (var i = 0; i < nodes.length; i += 1) {
      if (!startNode && start >= nodes[i].start && start <= nodes[i].end) startNode = nodes[i];
      if (!endNode && end >= nodes[i].start && end <= nodes[i].end) endNode = nodes[i];
    }
    if (!startNode || !endNode) return null;
    var range = document.createRange();
    range.setStart(startNode.node, Math.max(0, start - startNode.start));
    range.setEnd(endNode.node, Math.max(0, end - endNode.start));
    return range;
  }

  function focusAnnotation(id) {
    if (!id) return false;
    var annotation = annotations.find(function (item) { return item.id === id; });
    if (!annotation) {
      loadAnnotations().then(function () {
        window.setTimeout(function () { focusAnnotation(id); }, 80);
      });
      return false;
    }
    var key = groupKey(annotation);
    var mark = document.querySelector('[data-dc-annotation-key="' + cssEscape(key) + '"]');
    if (mark) {
      mark.scrollIntoView({block: "center", behavior: "smooth"});
      mark.classList.add("is-focused");
      window.setTimeout(function () { mark.classList.remove("is-focused"); }, 1800);
    }
    openPanel(key);
    return true;
  }

  function focusCommentForGroup(group) {
    var first = group && group[0];
    var comment = first && first.comment;
    if (!comment || !comment.id) return;
    window.postMessage({
      type: "deadcomments:commentFocus",
      annotation_id: first.id,
      comment_id: comment.id
    }, window.location.origin);
  }

  function openPanel(key, options) {
    options = options || {};
    closePanel();
    var group = groups[key] || [];
    if (options.focusComment) focusCommentForGroup(group);
    panel = document.createElement("aside");
    panel.className = "dc-annotation-panel";
    panel.innerHTML = '<button class="dc-annotation-panel-close" type="button"></button><h2></h2><blockquote></blockquote><div class="dc-annotation-panel-list"></div>';
    panel.querySelector(".dc-annotation-panel-close").textContent = "×";
    panel.querySelector(".dc-annotation-panel-close").setAttribute("aria-label", t("close"));
    panel.querySelector("h2").textContent = t("thread");
    panel.querySelector("blockquote").textContent = group[0] ? group[0].selected_text : "";
    var list = panel.querySelector(".dc-annotation-panel-list");
    if (group.length === 0) {
      var empty = document.createElement("p");
      empty.className = "dc-annotation-empty";
      empty.textContent = t("empty");
      list.appendChild(empty);
    }
    group.forEach(function (annotation) {
      list.appendChild(commentCard(annotation));
    });
    panel.querySelector(".dc-annotation-panel-close").addEventListener("click", closePanel);
    document.body.appendChild(panel);
    requestAnimationFrame(function () { panel.classList.add("is-open"); });
  }

  function closePanel() {
    if (panel && panel.parentNode) panel.parentNode.removeChild(panel);
    panel = null;
  }

  function commentCard(annotation) {
    var card = document.createElement("article");
    var comment = annotation.comment || {};
    card.className = "dc-annotation-comment";
    var header = document.createElement("header");
    var author = document.createElement("strong");
    author.textContent = comment.author_display_name || comment.author_name || "Anonymous";
    var time = document.createElement("time");
    time.dateTime = comment.created_at || annotation.created_at || "";
    time.textContent = relativeTime(time.dateTime);
    header.appendChild(author);
    header.appendChild(time);
    if (annotation._localPending || comment.status === "pending") {
      var badge = document.createElement("span");
      badge.className = "dc-annotation-pending";
      badge.textContent = "pending";
      header.appendChild(badge);
    }
    var body = document.createElement("div");
    body.className = "dc-annotation-body";
    body.innerHTML = comment.body_html || "";
    card.appendChild(header);
    card.appendChild(body);
    return card;
  }

  function relativeTime(value) {
    var date = new Date(value || "");
    if (Number.isNaN(date.getTime())) return "";
    var seconds = Math.floor((Date.now() - date.getTime()) / 1000);
    if (seconds < 45) return locale === "uk" ? "щойно" : "just now";
    var units = [["year", 31536000], ["month", 2592000], ["day", 86400], ["hour", 3600], ["minute", 60]];
    var formatter = new Intl.RelativeTimeFormat(locale, {numeric: "auto"});
    for (var i = 0; i < units.length; i += 1) {
      var count = Math.floor(seconds / units[i][1]);
      if (count >= 1) return formatter.format(-count, units[i][0]);
    }
    return "";
  }

  function injectStyles() {
    if (document.getElementById("dc-annotation-styles")) return;
    var style = document.createElement("style");
    style.id = "dc-annotation-styles";
    style.textContent = [
      ".dc-annotation-mark{background:rgba(255,212,59,.42);color:inherit;border-radius:3px;box-shadow:0 0 0 2px rgba(255,212,59,.2);cursor:pointer;padding:0 .04em}",
      ".dc-annotation-mark:hover,.dc-annotation-mark:focus{background:rgba(255,212,59,.62);outline:2px solid rgba(9,105,218,.45);outline-offset:2px}",
      ".dc-annotation-mark.is-focused{background:rgba(88,166,255,.42);box-shadow:0 0 0 3px rgba(88,166,255,.28)}",
      ".dc-annotation-mark.is-pending{background:rgba(210,153,34,.32)}",
      ".dc-annotation-mark[data-count]::after{content:attr(data-count);display:inline-flex;align-items:center;justify-content:center;min-width:1.25em;height:1.25em;margin-left:.25em;border-radius:999px;background:#0969da;color:#fff;font:700 .68em/1 system-ui}",
      ".dc-annotation-popover{position:absolute;z-index:2147483000;box-sizing:border-box;background:#0d1117;color:#e6edf3;border:1px solid #30363d;border-radius:8px;box-shadow:0 16px 48px rgba(0,0,0,.38);padding:14px;font:14px/1.45 ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif}",
      ".dc-annotation-title{font-weight:700;font-size:15px;margin-bottom:8px}.dc-annotation-quote{margin:0 0 12px;padding:8px 10px;border-left:3px solid #58a6ff;background:rgba(88,166,255,.08);color:#c9d1d9;max-height:88px;overflow:auto}",
      ".dc-annotation-grid{display:grid;grid-template-columns:1fr 1fr;gap:10px}.dc-annotation-popover label{display:grid;gap:5px;margin-top:10px;color:#8b949e;font-weight:650}.dc-annotation-popover input,.dc-annotation-popover textarea{width:100%;box-sizing:border-box;border:1px solid #30363d;border-radius:6px;background:#010409;color:#e6edf3;padding:9px 10px;font:inherit}.dc-annotation-popover textarea{resize:vertical}",
      ".dc-annotation-hints{display:grid;grid-template-columns:1fr 1fr;gap:10px;margin-top:4px;color:#8b949e;font-size:12px}.dc-annotation-actions{display:flex;align-items:center;gap:10px;margin-top:12px}.dc-annotation-submit{border:0;border-radius:6px;background:#238636;color:#fff;font-weight:700;padding:9px 14px}.dc-annotation-submit:disabled{opacity:.65}.dc-annotation-cancel{border:0;background:transparent;color:#8b949e;font-weight:700;padding:9px 10px}.dc-annotation-message{min-height:18px;margin-top:8px;font-size:13px}.dc-annotation-message.is-error{color:#ff7b72}.dc-annotation-honeypot{position:absolute!important;left:-10000px!important}",
      ".dc-annotation-panel{position:fixed;z-index:2147483001;top:0;right:0;width:min(420px,calc(100vw - 24px));height:100vh;box-sizing:border-box;background:#0d1117;color:#e6edf3;border-left:1px solid #30363d;box-shadow:-16px 0 48px rgba(0,0,0,.35);padding:20px;overflow:auto;transform:translateX(105%);transition:transform .18s ease;font:14px/1.5 ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif}.dc-annotation-panel.is-open{transform:translateX(0)}",
      ".dc-annotation-panel-close{position:absolute;top:12px;right:12px;border:1px solid #30363d;border-radius:6px;background:#161b22;color:#e6edf3;font-size:22px;line-height:1;width:34px;height:34px}.dc-annotation-panel h2{font-size:18px;margin:0 42px 12px 0}.dc-annotation-panel blockquote{margin:0 0 16px;padding:10px 12px;border-left:3px solid #58a6ff;background:#161b22;color:#c9d1d9}",
      ".dc-annotation-comment{border:1px solid #30363d;border-radius:8px;background:#010409;margin:12px 0;overflow:hidden}.dc-annotation-comment header{display:flex;align-items:center;gap:8px;border-bottom:1px solid #30363d;background:#161b22;padding:9px 11px}.dc-annotation-comment time{color:#8b949e}.dc-annotation-body{padding:11px}.dc-annotation-body img{max-width:100%;height:auto;border-radius:6px}.dc-annotation-body a{color:#58a6ff;text-decoration:underline;text-underline-offset:2px}.dc-annotation-pending{border:1px solid rgba(210,153,34,.55);border-radius:999px;color:#d29922;padding:1px 7px;font-size:12px;font-weight:700}",
      ".dc-annotation-toast{position:fixed;z-index:2147483002;left:50%;bottom:24px;transform:translate(-50%,16px);opacity:0;pointer-events:none;max-width:min(520px,calc(100vw - 32px));border-radius:8px;padding:11px 14px;background:#161b22;color:#e6edf3;border:1px solid #30363d;box-shadow:0 12px 32px rgba(0,0,0,.3);transition:opacity .16s ease,transform .16s ease;font:14px/1.45 ui-sans-serif,system-ui}.dc-annotation-toast.is-visible{opacity:1;transform:translate(-50%,0)}.dc-annotation-toast.is-success{border-color:#2ea043;color:#3fb950}.dc-annotation-toast.is-warning{border-color:#bb8009;color:#d29922}",
      "@media(max-width:640px){.dc-annotation-grid,.dc-annotation-hints{grid-template-columns:1fr}.dc-annotation-popover{position:fixed!important;left:12px!important;right:12px!important;top:auto!important;bottom:12px!important;width:auto!important;max-height:calc(100vh - 24px);overflow:auto}.dc-annotation-panel{width:100vw}}",
      "@media(prefers-color-scheme:light){.dc-annotation-popover,.dc-annotation-panel{background:#fff;color:#24292f;border-color:#d0d7de}.dc-annotation-popover input,.dc-annotation-popover textarea,.dc-annotation-comment{background:#fff;color:#24292f;border-color:#d0d7de}.dc-annotation-comment header,.dc-annotation-panel blockquote{background:#f6f8fa;border-color:#d0d7de}.dc-annotation-popover label,.dc-annotation-hints,.dc-annotation-comment time{color:#57606a}.dc-annotation-submit{background:#1f883d}.dc-annotation-panel-close{background:#f6f8fa;color:#24292f;border-color:#d0d7de}.dc-annotation-body a{color:#0969da}}"
    ].join("");
    document.head.appendChild(style);
  }
})();
