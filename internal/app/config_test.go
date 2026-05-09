package app

import "testing"

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
