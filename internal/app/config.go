package app

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	BaseURL             string
	DatabasePath        string
	ServerSecret        string
	GitHubClientID      string
	GitHubClientSecret  string
	GitHubAllowedLogins map[string]struct{}
	SessionSecret       string
	TripcodeSecret      string
	Port                string
	SessionTTL          time.Duration
	SecureCookies       bool
	BehindTrustedProxy  bool
	DevSeed             bool
}

func LoadConfig() (Config, error) {
	baseURL := env("BASE_URL", "http://localhost:8080")
	serverSecretSet := strings.TrimSpace(os.Getenv("SERVER_SECRET")) != ""
	sessionSecretSet := strings.TrimSpace(os.Getenv("SESSION_SECRET")) != ""
	tripcodeSecretSet := strings.TrimSpace(os.Getenv("TRIPCODE_SECRET")) != ""
	cfg := Config{
		BaseURL:            baseURL,
		DatabasePath:       env("DATABASE_PATH", "deadcomments.db"),
		ServerSecret:       os.Getenv("SERVER_SECRET"),
		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		SessionSecret:      os.Getenv("SESSION_SECRET"),
		TripcodeSecret:     os.Getenv("TRIPCODE_SECRET"),
		Port:               env("PORT", "8080"),
		SessionTTL:         30 * 24 * time.Hour,
		SecureCookies:      strings.HasPrefix(baseURL, "https://"),
		BehindTrustedProxy: truthy(os.Getenv("BEHIND_TRUSTED_PROXY")),
		DevSeed:            os.Getenv("DEADCOMMENTS_DEV_SEED") == "1",
	}
	cfg.GitHubAllowedLogins = parseAllowedLogins(os.Getenv("GITHUB_ALLOWED_LOGINS"))
	if cfg.GitHubClientID != "" && cfg.GitHubClientSecret != "" && len(cfg.GitHubAllowedLogins) == 0 {
		return Config{}, errors.New("GITHUB_ALLOWED_LOGINS must include at least one login when GitHub OAuth is configured")
	}
	if productionMode(cfg.BaseURL) {
		switch {
		case !serverSecretSet:
			return Config{}, errors.New("SERVER_SECRET must be set explicitly in production")
		case !sessionSecretSet:
			return Config{}, errors.New("SESSION_SECRET must be set explicitly in production")
		case !tripcodeSecretSet:
			return Config{}, errors.New("TRIPCODE_SECRET must be set explicitly in production")
		}
	}
	if cfg.ServerSecret == "" {
		cfg.ServerSecret = devSecret()
	}
	if cfg.SessionSecret == "" {
		cfg.SessionSecret = cfg.ServerSecret
	}
	if cfg.TripcodeSecret == "" {
		cfg.TripcodeSecret = cfg.ServerSecret
	}
	if ttl := os.Getenv("SESSION_TTL_HOURS"); ttl != "" {
		hours, err := strconv.Atoi(ttl)
		if err != nil || hours <= 0 {
			return Config{}, errors.New("SESSION_TTL_HOURS must be a positive integer")
		}
		cfg.SessionTTL = time.Duration(hours) * time.Hour
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func parseAllowedLogins(raw string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		login := strings.ToLower(strings.TrimSpace(part))
		if login != "" {
			out[login] = struct{}{}
		}
	}
	return out
}

func truthy(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func productionMode(baseURL string) bool {
	env := strings.ToLower(strings.TrimSpace(firstNonEmpty(os.Getenv("DEADCOMMENTS_ENV"), os.Getenv("APP_ENV"), os.Getenv("GO_ENV"))))
	return env == "production" || strings.HasPrefix(baseURL, "https://")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func devSecret() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return "deadcomments-dev-secret-change-me"
}
