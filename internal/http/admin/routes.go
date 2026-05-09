package admin

import (
	"github.com/go-chi/chi/v5"

	"deadcomments/internal/http/middleware"
	"deadcomments/internal/service"
)

func Routes(r chi.Router, h *Handlers, auth *service.AuthService, csrf *middleware.CSRF) {
	r.Get("/static/admin.css", h.AdminCSS)
	r.Get("/static/admin.js", h.AdminJS)

	r.Group(func(adminUI chi.Router) {
		adminUI.Use(middleware.DenyFrames)
		adminUI.Get("/admin/login", h.Login)
		adminUI.Get("/auth/github/start", h.GitHubStart)
		adminUI.Get("/auth/github/callback", h.GitHubCallback)

		adminUI.Group(func(admin chi.Router) {
			admin.Use(middleware.RequireAdmin(auth))
			admin.Use(csrf.Middleware)
			admin.Get("/admin", h.Dashboard)
			admin.Get("/admin/sites", h.Sites)
			admin.Get("/admin/sites/new", h.NewSite)
			admin.Post("/admin/sites", h.CreateSite)
			admin.Get("/admin/sites/{id}", h.SiteSettings)
			admin.Get("/admin/sites/{id}/settings", h.SiteSettings)
			admin.Post("/admin/sites/{id}/settings", h.UpdateSite)
			admin.Get("/admin/sites/{id}/pages", h.Pages)
			admin.Get("/admin/pages/{id}", h.PageDetail)
			admin.Post("/admin/pages/{id}/state", h.PageState)
			admin.Get("/admin/comments", h.Comments)
			admin.Get("/admin/comments/pending", h.PendingComments)
			admin.Get("/admin/comments/export", h.ExportComments)
			admin.Post("/admin/comments/bulk", h.BulkComments)
			admin.Get("/admin/comments/{id}", h.CommentDetail)
			admin.Post("/admin/comments/{id}/approve", h.ApproveComment)
			admin.Post("/admin/comments/{id}/reject", h.RejectComment)
			admin.Post("/admin/comments/{id}/spam", h.SpamComment)
			admin.Post("/admin/comments/{id}/delete", h.DeleteComment)
			admin.Post("/admin/comments/{id}/edit", h.EditComment)
			admin.Post("/admin/comments/{id}/ban-ip", h.BanIP)
			admin.Get("/admin/bans", h.Bans)
			admin.Post("/admin/bans/ip", h.CreateIPBan)
			admin.Post("/admin/bans/word", h.CreateWordBan)
			admin.Post("/admin/bans/{id}/delete", h.DeleteBan)
			admin.Get("/admin/identities", h.Identities)
			admin.Get("/admin/identities/new", h.NewIdentity)
			admin.Post("/admin/identities", h.CreateIdentity)
			admin.Get("/admin/identities/{id}", h.IdentityDetail)
			admin.Post("/admin/identities/{id}", h.UpdateIdentity)
			admin.Post("/admin/identities/{id}/delete", h.DeleteIdentity)
			admin.Post("/admin/identities/{id}/reset-secret", h.ResetIdentitySecret)
			admin.Get("/admin/events", h.Events)
			admin.Post("/admin/logout", h.Logout)
		})
	})
}
