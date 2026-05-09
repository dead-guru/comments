var scrollKey = "deadcomments:admin-scroll:" + window.location.pathname;

function rememberScrollPosition() {
  try {
    sessionStorage.setItem(scrollKey, String(window.scrollY || window.pageYOffset || 0));
  } catch (_) {}
}

function restoreScrollPosition() {
  var raw = "";
  try {
    raw = sessionStorage.getItem(scrollKey) || "";
    sessionStorage.removeItem(scrollKey);
  } catch (_) {}
  var y = Number(raw);
  if (Number.isFinite(y) && y > 0) {
    window.requestAnimationFrame(function () {
      window.scrollTo({ top: y, behavior: "auto" });
    });
  }
}

restoreScrollPosition();

document.addEventListener("submit", function (event) {
  var form = event.target;
  if (form.dataset.confirmed === "true") {
    delete form.dataset.confirmed;
    rememberScrollPosition();
    return;
  }
  if (!validateOriginList(form)) {
    event.preventDefault();
    return;
  }
  if (form.matches("[data-confirm]")) {
    event.preventDefault();
    openConfirm(form, form.getAttribute("data-confirm"), form.getAttribute("data-confirm-token") || "");
    return;
  }
  if (form.id === "comments-bulk-form") {
    var action = form.querySelector('[name="action"]');
    if (action && action.value === "delete") {
      event.preventDefault();
      openConfirm(form, "Delete selected comments?", "DELETE");
      return;
    }
  }
  if (form.querySelector('[name="redirect_to"]')) {
    rememberScrollPosition();
  }
});

function validateOriginList(form) {
  var input = form.querySelector("[data-origin-list]");
  if (!input) return true;
  var rawLines = input.value.split(/\n+/);
  var trimmedLines = rawLines.map(function (line) { return line.trim(); });
  var lines = trimmedLines.filter(Boolean);
  for (var i = 0; i < lines.length; i += 1) {
    try {
      var url = new URL(lines[i]);
      if ((url.protocol !== "http:" && url.protocol !== "https:") || url.pathname !== "/" || url.search || url.hash) {
        throw new Error("invalid origin");
      }
    } catch (_) {
      input.setCustomValidity("Use full origins like https://example.com, one per line.");
      input.reportValidity();
      return false;
    }
  }
  input.setCustomValidity("");
  return true;
}

document.addEventListener("input", function (event) {
  if (event.target.matches("[data-origin-list]")) {
    event.target.setCustomValidity("");
  }
});

function openConfirm(form, message, token) {
  var dialog = document.createElement("dialog");
  dialog.className = "confirm-dialog";
  var dialogForm = document.createElement("form");
  dialogForm.method = "dialog";
  dialogForm.className = "confirm-card";
  var title = document.createElement("h2");
  title.textContent = "Confirm action";
  var copy = document.createElement("p");
  copy.textContent = message || "Are you sure?";
  dialogForm.append(title, copy);
  var input = null;
  if (token) {
    var label = document.createElement("label");
    label.className = "field";
    var labelText = document.createElement("span");
    labelText.textContent = "Type " + token + " to confirm";
    input = document.createElement("input");
    input.dataset.confirmInput = "";
    input.autocomplete = "off";
    label.append(labelText, input);
    dialogForm.appendChild(label);
  }
  var actions = document.createElement("div");
  actions.className = "actions";
  var cancel = document.createElement("button");
  cancel.value = "cancel";
  cancel.textContent = "Cancel";
  var confirm = document.createElement("button");
  confirm.className = "danger";
  confirm.value = "confirm";
  confirm.dataset.confirmFinal = "";
  confirm.textContent = "Confirm";
  if (token) confirm.disabled = true;
  actions.append(cancel, confirm);
  dialogForm.appendChild(actions);
  dialog.appendChild(dialogForm);
  document.body.appendChild(dialog);
  if (input) {
    input.addEventListener("input", function () {
      confirm.disabled = input.value !== token;
    });
  }
  dialog.addEventListener("close", function () {
    var ok = dialog.returnValue === "confirm";
    dialog.remove();
    if (ok) {
      form.dataset.confirmed = "true";
      form.requestSubmit();
    }
  });
  dialog.showModal();
  if (input) input.focus();
}

(function () {
  var table = document.querySelector("[data-moderation-table]");
  if (!table) return;

  var currentRow = null;
  var bulkForm = document.getElementById("comments-bulk-form");

  function rows() {
    return Array.prototype.slice.call(table.querySelectorAll("[data-moderation-row]"));
  }

  function setCurrentRow(row) {
    if (!row) return;
    if (currentRow) currentRow.classList.remove("is-current");
    currentRow = row;
    currentRow.classList.add("is-current");
    try {
      currentRow.focus({ preventScroll: true });
    } catch (_) {
      currentRow.focus();
    }
  }

  function move(delta) {
    var list = rows();
    if (list.length === 0) return;
    var index = currentRow ? list.indexOf(currentRow) : -1;
    var next = Math.min(list.length - 1, Math.max(0, index + delta));
    if (index === -1) next = delta > 0 ? 0 : list.length - 1;
    setCurrentRow(list[next]);
    list[next].scrollIntoView({ block: "nearest" });
  }

  function submitRowAction(action) {
    var row = currentRow || rows()[0];
    if (!row) return;
    var actionButton = row.querySelector('form button[data-action="' + action + '"]');
    if (actionButton) {
      rememberScrollPosition();
      actionButton.closest("form").requestSubmit();
    }
  }

  function formFieldActive(target) {
    if (!target || typeof target.closest !== "function") return false;
    return !!target.closest("input, textarea, select, button, a, [contenteditable=true]");
  }

  function updateBulkState() {
    if (!bulkForm) return;
    var checked = table.querySelectorAll("[data-bulk-checkbox]:checked").length;
    var count = document.querySelector("[data-bulk-count]");
    var submit = document.querySelector("[data-bulk-submit]");
    if (count) count.textContent = checked + (checked === 1 ? " item selected" : " items selected");
    if (submit) submit.disabled = checked === 0;
  }

  table.addEventListener("focusin", function (event) {
    var row = event.target.closest("[data-moderation-row]");
    if (row) setCurrentRow(row);
  });

  table.addEventListener("click", function (event) {
    var row = event.target.closest("[data-moderation-row]");
    if (row) setCurrentRow(row);
  });

  document.addEventListener("change", function (event) {
    if (event.target.matches("[data-bulk-toggle]")) {
      table.querySelectorAll("[data-bulk-checkbox]").forEach(function (checkbox) {
        checkbox.checked = event.target.checked;
      });
      updateBulkState();
    }
    if (event.target.matches("[data-bulk-checkbox]")) {
      updateBulkState();
    }
  });

  document.addEventListener("keydown", function (event) {
    if (event.metaKey || event.ctrlKey || event.altKey || formFieldActive(event.target)) return;
    if (event.key === "j") {
      event.preventDefault();
      move(1);
    } else if (event.key === "k") {
      event.preventDefault();
      move(-1);
    } else if (event.key === "a") {
      event.preventDefault();
      submitRowAction("approve");
    } else if (event.key === "s") {
      event.preventDefault();
      submitRowAction("spam");
    } else if (event.key === "r") {
      event.preventDefault();
      submitRowAction("reject");
    }
  });

  updateBulkState();
})();
