package admin

import "net/http"

func (h *Handlers) Events(w http.ResponseWriter, r *http.Request) {
	events, _ := h.events.Recent(r.Context(), 200)
	h.render(w, r, "admin/events.html", map[string]any{"Events": events})
}
