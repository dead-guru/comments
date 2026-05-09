package public

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"deadcomments/internal/http/middleware"
)

func Routes(r chi.Router, h *Handlers, limiter *middleware.RateLimiter) {
	r.Get("/widget.js", h.WidgetJS)
	r.Get("/static/embed.css", h.EmbedCSS)
	r.Get("/embed/comments", h.EmbedComments)
	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/sites/{site_key}/pages/{page_key:.*}/comments", h.APIListComments)
		api.With(limiter.Middleware).Post("/sites/{site_key}/pages/{page_key:.*}/comments", h.APICreateComment)
	})
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
}
