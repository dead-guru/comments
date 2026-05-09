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

function openConfirm(form, message, token) {
  var dialog = document.createElement("dialog");
  dialog.className = "confirm-dialog";
  dialog.innerHTML = [
    "<form method=\"dialog\" class=\"confirm-card\">",
    "<h2>Confirm action</h2>",
    "<p></p>",
    token ? "<label class=\"field\"><span>Type " + token + " to confirm</span><input data-confirm-input autocomplete=\"off\"></label>" : "",
    "<div class=\"actions\"><button value=\"cancel\">Cancel</button><button class=\"danger\" value=\"confirm\" data-confirm-final" + (token ? " disabled" : "") + ">Confirm</button></div>",
    "</form>"
  ].join("");
  dialog.querySelector("p").textContent = message || "Are you sure?";
  document.body.appendChild(dialog);
  var input = dialog.querySelector("[data-confirm-input]");
  var confirm = dialog.querySelector("[data-confirm-final]");
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
    var form = row.querySelector('form button[data-action="' + action + '"]');
    if (form) {
      rememberScrollPosition();
      form.closest("form").requestSubmit();
    }
  }

  function formFieldActive(target) {
    if (!target || typeof target.closest !== "function") return false;
    return target.closest("input, textarea, select, button, a, [contenteditable=true]");
  }

  function updateBulkState() {
    if (!bulkForm) return;
    var checked = table.querySelectorAll("[data-bulk-checkbox]:checked").length;
    var count = document.querySelector("[data-bulk-count]");
    var submit = document.querySelector("[data-bulk-submit]");
    if (count) count.textContent = checked + (checked === 1 ? " selected" : " selected");
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
