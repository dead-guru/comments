package app

import (
	"database/sql"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	dcauth "deadcomments/internal/auth"
	dcevent "deadcomments/internal/event"
	adminhttp "deadcomments/internal/http/admin"
	dcmiddleware "deadcomments/internal/http/middleware"
	publichttp "deadcomments/internal/http/public"
	"deadcomments/internal/markdown"
	"deadcomments/internal/repository"
	"deadcomments/internal/service"
)

type App struct {
	Config Config
	DB     *sql.DB
	Router http.Handler
}

func New(cfg Config, database *sql.DB) (*App, error) {
	siteRepo := repository.NewSiteRepository(database)
	pageRepo := repository.NewPageRepository(database)
	commentRepo := repository.NewCommentRepository(database)
	identityRepo := repository.NewIdentityRepository(database)
	adminRepo := repository.NewAdminRepository(database)
	sessionRepo := repository.NewSessionRepository(database)
	moderationRepo := repository.NewModerationRepository(database)
	eventRepo := repository.NewEventRepository(database)
	eventBus := dcevent.NewBus(eventRepo)
	eventBus.Subscribe(dcevent.NewAuditHandler(moderationRepo))

	md := service.NewMarkdownService(markdown.NewRenderer())
	identitySvc := service.NewIdentityService(identityRepo, cfg.TripcodeSecret, eventBus)
	moderationSvc := service.NewModerationService(moderationRepo, commentRepo, eventBus)
	siteSvc := service.NewSiteService(siteRepo, eventBus)
	pageSvc := service.NewPageService(pageRepo, eventBus)
	commentSvc := service.NewCommentService(siteRepo, pageRepo, commentRepo, identitySvc, moderationSvc, md, cfg.ServerSecret, eventBus)
	eventSvc := service.NewEventService(eventRepo)
	oauth := dcauth.NewGitHubOAuth(cfg.GitHubClientID, cfg.GitHubClientSecret, cfg.BaseURL+"/auth/github/callback")
	authSvc := service.NewAuthService(adminRepo, sessionRepo, oauth, cfg.GitHubAllowedLogins, cfg.SessionSecret, cfg.SessionTTL, eventBus)

	funcs := template.FuncMap{
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
		"commentTime": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.Format("Jan 2, 2006")
		},
		"machineTime": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.UTC().Format(time.RFC3339)
		},
		"avatarInitial": func(values ...string) string {
			for _, value := range values {
				value = strings.TrimSpace(value)
				if value == "" {
					continue
				}
				for _, r := range value {
					return strings.ToUpper(string(r))
				}
			}
			return "?"
		},
		"join": func(v []string) string {
			out := ""
			for i, x := range v {
				if i > 0 {
					out += "\n"
				}
				out += x
			}
			return out
		},
		"dict": func(values ...any) map[string]any {
			out := map[string]any{}
			for i := 0; i+1 < len(values); i += 2 {
				key, _ := values[i].(string)
				out[key] = values[i+1]
			}
			return out
		},
		"siteSelected": func(selected *int64, id int64) bool {
			return selected != nil && *selected == id
		},
	}
	tmpl, err := template.New("").Funcs(funcs).ParseGlob(filepath.Join("internal", "templates", "*.html"))
	if err != nil {
		return nil, err
	}
	if _, err := tmpl.ParseGlob(filepath.Join("internal", "templates", "admin", "*.html")); err != nil {
		return nil, err
	}
	if _, err := tmpl.ParseGlob(filepath.Join("internal", "templates", "embed", "*.html")); err != nil {
		return nil, err
	}

	csrf := dcmiddleware.NewCSRF(cfg.SessionSecret, cfg.SecureCookies)
	r := chi.NewRouter()
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(dcmiddleware.RequestID)
	r.Use(dcmiddleware.SecurityHeaders)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := database.PingContext(r.Context()); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})

	publicHandlers := publichttp.NewHandlers(siteSvc, pageSvc, commentSvc, tmpl)
	publichttp.Routes(r, publicHandlers, dcmiddleware.NewRateLimiter(30, time.Minute))

	adminHandlers := adminhttp.NewHandlers(siteSvc, pageSvc, commentSvc, moderationSvc, identitySvc, eventSvc, authSvc, tmpl, csrf, cfg.SecureCookies)
	adminhttp.Routes(r, adminHandlers, authSvc, csrf)

	return &App{Config: cfg, DB: database, Router: r}, nil
}
