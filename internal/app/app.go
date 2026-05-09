package app

import (
	"crypto/subtle"
	"database/sql"
	"encoding/json"
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
	"deadcomments/internal/i18n"
	"deadcomments/internal/markdown"
	"deadcomments/internal/observability"
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
	metrics := observability.NewMetrics("deadcomments")
	eventBus.Subscribe(observability.NewEventMetricsHandler(metrics))
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
		"json": func(v any) template.JS {
			data, err := json.Marshal(v)
			if err != nil {
				return template.JS("{}")
			}
			return template.JS(data)
		},
		"commentTime": func(t time.Time, locales ...string) string {
			if t.IsZero() {
				return ""
			}
			locale := i18n.LocaleEnglish
			if len(locales) > 0 {
				locale = i18n.Normalize(locales[0], "")
			}
			if locale == i18n.LocaleUkrainian {
				return t.Format("02.01.2006")
			}
			return t.Format("Jan 2, 2006")
		},
		"machineTime": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.UTC().Format(time.RFC3339)
		},
		"fullTime": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.UTC().Format("Jan 2, 2006, 15:04 UTC")
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
		"navActive": func(current, item string) bool {
			if item == "/admin" {
				return current == "/admin"
			}
			return strings.HasPrefix(current, item)
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
	if cfg.BehindTrustedProxy {
		r.Use(chimiddleware.RealIP)
	}
	r.Use(chimiddleware.Recoverer)
	r.Use(dcmiddleware.RequestID)
	r.Use(dcmiddleware.SecurityHeaders)
	r.Use(metrics.Middleware)

	health := observability.NewHealthHandler(database, metrics)
	r.Get("/livez", health.Livez)
	r.Get("/readyz", health.Readyz)
	r.Get("/healthz", health.Readyz)
	r.Get("/status", health.Status)
	r.Handle("/metrics", metricsHandler(metrics.Handler(), cfg.MetricsToken))

	publicHandlers := publichttp.NewHandlers(siteSvc, pageSvc, commentSvc, md, tmpl, cfg.ServerSecret)
	publichttp.Routes(r, publicHandlers, dcmiddleware.NewRateLimiter(120, time.Minute))

	adminHandlers := adminhttp.NewHandlers(siteSvc, pageSvc, commentSvc, moderationSvc, identitySvc, eventSvc, authSvc, tmpl, csrf, cfg.SecureCookies)
	adminhttp.Routes(r, adminHandlers, authSvc, csrf)

	return &App{Config: cfg, DB: database, Router: r}, nil
}

func metricsHandler(next http.Handler, token string) http.Handler {
	if strings.TrimSpace(token) == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(bearerToken(r.Header.Get("Authorization"))), []byte(token)) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="deadcomments metrics"`)
			http.Error(w, "metrics authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearerToken(header string) string {
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
}
