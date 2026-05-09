document.addEventListener("submit", function (event) {
  var form = event.target;
  if (form.matches("[data-confirm]") && !window.confirm(form.getAttribute("data-confirm"))) {
    event.preventDefault();
  }
  if (form.id === "comments-bulk-form") {
    var action = form.querySelector('[name="action"]');
    if (action && action.value === "delete" && !window.confirm("Delete selected comments?")) {
      event.preventDefault();
    }
  }
});

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
    currentRow.focus({ preventScroll: true });
    currentRow.scrollIntoView({ block: "nearest" });
  }

  function move(delta) {
    var list = rows();
    if (list.length === 0) return;
    var index = currentRow ? list.indexOf(currentRow) : -1;
    var next = Math.min(list.length - 1, Math.max(0, index + delta));
    if (index === -1) next = delta > 0 ? 0 : list.length - 1;
    setCurrentRow(list[next]);
  }

  function submitRowAction(action) {
    var row = currentRow || rows()[0];
    if (!row) return;
    var form = row.querySelector('form button[data-action="' + action + '"]');
    if (form) form.closest("form").requestSubmit();
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
