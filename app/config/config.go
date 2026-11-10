// Package config loads runtime settings from environment variables.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env         string
	Addr        string
	BaseURL     string // public origin, no trailing slash, e.g. https://awesomeopensource.dev
	SiteName    string
	DatabaseURL string
	UploadDir   string

	DefaultLocale string
	Locales       []string

	AdminEmail    string
	AdminPassword string

	GitHubToken string

	TypesenseURL    string
	TypesenseKey    string
	TypesensePrefix string

	AnthropicKey   string
	AnthropicModel string

	EmbeddingsURL   string
	EmbeddingsKey   string
	EmbeddingsModel string

	SyncInterval  time.Duration // 0 disables the background worker
	SyncBatchSize int
	SyncStaleAge  time.Duration
}

func Load() Config {
	c := Config{
		Env:             get("APP_ENV", ""),
		Addr:            get("APP_ADDR", "127.0.0.1:8080"),
		BaseURL:         strings.TrimRight(get("APP_URL", "http://127.0.0.1:8080"), "/"),
		SiteName:        get("SITE_NAME", "Awesome Open Source"),
		DatabaseURL:     get("DATABASE_URL", "postgres://aos:aos@127.0.0.1:5432/aos?sslmode=disable"),
		UploadDir:       get("UPLOAD_DIR", "data/uploads"),
		DefaultLocale:   get("DEFAULT_LOCALE", "en"),
		Locales:         splitList(get("LOCALES", "en,ro,es,fr,de")),
		AdminEmail:      get("ADMIN_EMAIL", ""),
		AdminPassword:   get("ADMIN_PASSWORD", ""),
		GitHubToken:     get("GITHUB_TOKEN", ""),
		TypesenseURL:    strings.TrimRight(get("TYPESENSE_URL", "http://127.0.0.1:8108"), "/"),
		TypesenseKey:    get("TYPESENSE_API_KEY", ""),
		TypesensePrefix: get("TYPESENSE_PREFIX", "aos"),
		AnthropicKey:    get("ANTHROPIC_API_KEY", ""),
		AnthropicModel:  get("ANTHROPIC_MODEL", "claude-haiku-5-5"),
		EmbeddingsURL:   strings.TrimRight(get("EMBEDDINGS_URL", ""), "/"),
		EmbeddingsKey:   get("EMBEDDINGS_API_KEY", ""),
		EmbeddingsModel: get("EMBEDDINGS_MODEL", "text-embedding-3-small"),
		SyncInterval:    time.Duration(getInt("SYNC_INTERVAL_MINUTES", 0)) * time.Minute,
		SyncBatchSize:   getInt("SYNC_BATCH_SIZE", 50),
		SyncStaleAge:    time.Duration(getInt("SYNC_STALE_HOURS", 24)) * time.Hour,
	}
	if len(c.Locales) == 0 {
		c.Locales = []string{c.DefaultLocale}
	}
	return c
}

func (c Config) UploadsEnabled() bool { return c.UploadDir != "" }

func get(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return v
	}
	return fallback
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
