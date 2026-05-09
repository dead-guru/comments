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
	DevSeed             bool
}

func LoadConfig() (Config, error) {
	cfg := Config{
		BaseURL:            env("BASE_URL", "http://localhost:8080"),
		DatabasePath:       env("DATABASE_PATH", "deadcomments.db"),
		ServerSecret:       os.Getenv("SERVER_SECRET"),
		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		SessionSecret:      os.Getenv("SESSION_SECRET"),
		TripcodeSecret:     os.Getenv("TRIPCODE_SECRET"),
		Port:               env("PORT", "8080"),
		SessionTTL:         30 * 24 * time.Hour,
		SecureCookies:      strings.HasPrefix(env("BASE_URL", "http://localhost:8080"), "https://"),
		DevSeed:            os.Getenv("DEADCOMMENTS_DEV_SEED") == "1",
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
	cfg.GitHubAllowedLogins = parseAllowedLogins(os.Getenv("GITHUB_ALLOWED_LOGINS"))
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

func devSecret() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return "deadcomments-dev-secret-change-me"
}
