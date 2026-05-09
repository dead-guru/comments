package middleware

import (
	"context"
	"net/http"

	"deadcomments/internal/actionctx"
	"deadcomments/internal/domain"
	"deadcomments/internal/service"
)

type adminKey struct{}

const SessionCookieName = "dc_admin_session"

func RequireAdmin(auth *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil || cookie.Value == "" {
				http.Redirect(w, r, "/admin/login", http.StatusFound)
				return
			}
			admin, err := auth.AdminForToken(r.Context(), cookie.Value)
			if err != nil || admin == nil {
				http.Redirect(w, r, "/admin/login", http.StatusFound)
				return
			}
			ctx := context.WithValue(r.Context(), adminKey{}, admin)
			ctx = actionctx.WithAdminID(ctx, admin.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AdminFromContext(ctx context.Context) *domain.Admin {
	admin, _ := ctx.Value(adminKey{}).(*domain.Admin)
	return admin
}
