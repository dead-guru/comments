package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoadConfigRequiresGitHubAllowlistWhenOAuthConfigured(t *testing.T) {
	t.Setenv("BASE_URL", "http://localhost:8080")
	t.Setenv("GITHUB_CLIENT_ID", "client")
	t.Setenv("GITHUB_CLIENT_SECRET", "secret")
	t.Setenv("GITHUB_ALLOWED_LOGINS", "")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected OAuth config without allowed logins to fail closed")
	}
}

func TestLoadConfigRequiresExplicitProductionSecrets(t *testing.T) {
	t.Setenv("BASE_URL", "https://comments.example.com")
	t.Setenv("SERVER_SECRET", "server-secret")
	t.Setenv("SESSION_SECRET", "")
	t.Setenv("TRIPCODE_SECRET", "tripcode-secret")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected production config without explicit SESSION_SECRET to fail")
	}
}

func TestLoadConfigAllowsDevelopmentSecretDefaults(t *testing.T) {
	t.Setenv("BASE_URL", "http://localhost:8080")
	t.Setenv("SERVER_SECRET", "")
	t.Setenv("SESSION_SECRET", "")
	t.Setenv("TRIPCODE_SECRET", "")
	t.Setenv("GITHUB_CLIENT_ID", "")
	t.Setenv("GITHUB_CLIENT_SECRET", "")
	t.Setenv("GITHUB_ALLOWED_LOGINS", "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServerSecret == "" || cfg.SessionSecret == "" || cfg.TripcodeSecret == "" {
		t.Fatal("expected development secrets to be populated")
	}
}

func TestLoadConfigTrustedProxyFlag(t *testing.T) {
	t.Setenv("BEHIND_TRUSTED_PROXY", "true")
	t.Setenv("GITHUB_CLIENT_ID", "")
	t.Setenv("GITHUB_CLIENT_SECRET", "")
	t.Setenv("GITHUB_ALLOWED_LOGINS", "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.BehindTrustedProxy {
		t.Fatal("expected trusted proxy flag to be enabled")
	}
}

func TestLoadConfigReadsMetricsToken(t *testing.T) {
	t.Setenv("METRICS_TOKEN", "metrics-secret")
	t.Setenv("GITHUB_CLIENT_ID", "")
	t.Setenv("GITHUB_CLIENT_SECRET", "")
	t.Setenv("GITHUB_ALLOWED_LOGINS", "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MetricsToken != "metrics-secret" {
		t.Fatalf("expected metrics token, got %q", cfg.MetricsToken)
	}
}

func TestLoadConfigReadsDatabasePoolLimits(t *testing.T) {
	t.Setenv("DATABASE_MAX_OPEN_CONNS", "12")
	t.Setenv("DATABASE_MAX_IDLE_CONNS", "6")
	t.Setenv("GITHUB_CLIENT_ID", "")
	t.Setenv("GITHUB_CLIENT_SECRET", "")
	t.Setenv("GITHUB_ALLOWED_LOGINS", "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseMaxOpenConns != 12 {
		t.Fatalf("expected max open conns 12, got %d", cfg.DatabaseMaxOpenConns)
	}
	if cfg.DatabaseMaxIdleConns != 6 {
		t.Fatalf("expected max idle conns 6, got %d", cfg.DatabaseMaxIdleConns)
	}
}

func TestLoadConfigRejectsInvalidDatabasePoolLimit(t *testing.T) {
	t.Setenv("DATABASE_MAX_OPEN_CONNS", "0")
	t.Setenv("GITHUB_CLIENT_ID", "")
	t.Setenv("GITHUB_CLIENT_SECRET", "")
	t.Setenv("GITHUB_ALLOWED_LOGINS", "")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected invalid database pool limit to fail")
	}
}

func TestMetricsHandlerRequiresBearerTokenWhenConfigured(t *testing.T) {
	handler := metricsHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), "metrics-secret")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized metrics request, got %d", rec.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer metrics-secret")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected authorized metrics request, got %d", rec.Code)
	}
}
