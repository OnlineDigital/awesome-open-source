// Package application is the dependency container shared by every controller.
package application

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"awesomeopensource.dev/app/config"
	"awesomeopensource.dev/app/db"
	"awesomeopensource.dev/app/deepwiki"
	vec "awesomeopensource.dev/app/embed"
	"awesomeopensource.dev/app/github"
	"awesomeopensource.dev/app/i18n"
	"awesomeopensource.dev/app/search"
	"awesomeopensource.dev/app/store"
	"awesomeopensource.dev/app/syncer"
	"awesomeopensource.dev/app/translate"
	upstream "github.com/inertia-go/inertia-go"
	"github.com/inertia-go/inertia-go/session"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

//go:embed root.html
var views embed.FS

type Application struct {
	Cfg     config.Config
	Env     string
	Key     []byte
	Adapter *upstream.Inertia
	Assets  Assets
	Session *captureStore

	Pool       *pgxpool.Pool
	Store      *store.Store
	I18n       *i18n.Bundle
	Search     *search.Client
	Indexer    *search.Indexer
	Translator *translate.Translator
	Embedder   *vec.Client
	GitHub     *github.Client
	DeepWiki   *deepwiki.Client
	Syncer     *syncer.Syncer

	RouteURL  func(string, map[string]string, url.Values) (string, error)
	RouteList func() any

	limiter *rateLimiter
}

// NewCore connects to PostgreSQL, applies migrations and wires every service.
// It needs neither the frontend build nor a session key, so CLI commands use it directly.
func NewCore(cfg config.Config) (*Application, error) {
	bundle, err := i18n.Load(cfg.DefaultLocale, cfg.Locales)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	applied, err := db.Migrate(ctx, pool)
	if err != nil {
		pool.Close()
		return nil, err
	}
	for _, m := range applied {
		log.Printf("applied migration %s", m)
	}
	a := &Application{Cfg: cfg, Env: cfg.Env, Pool: pool, I18n: bundle, limiter: newRateLimiter(8, 15*60)}
	a.Store = store.New(pool, cfg.DefaultLocale)
	a.Search = search.NewClient(cfg.TypesenseURL, cfg.TypesenseKey, cfg.TypesensePrefix)
	a.Indexer = &search.Indexer{Client: a.Search, Store: a.Store, Locales: bundle.Locales}
	a.Translator = translate.New(cfg.AnthropicKey, cfg.AnthropicModel)
	if cfg.EmbeddingsURL != "" {
		a.Embedder = vec.New(cfg.EmbeddingsURL, cfg.EmbeddingsKey, cfg.EmbeddingsModel)
	}
	a.GitHub = github.New(cfg.GitHubToken)
	a.DeepWiki = deepwiki.New()
	a.Syncer = &syncer.Syncer{Store: a.Store, GitHub: a.GitHub, Index: a.Indexer, Embed: a.Embedder, Delay: 300 * time.Millisecond}
	return a, nil
}

// New builds the web application: core services, session key, Vite assets and the Inertia adapter.
func New(root string, cfg config.Config, devServer string, assetFS fs.FS) (*Application, error) {
	assets, err := LoadAssetsFS(root, cfg.Env, devServer, assetFS)
	if err != nil {
		return nil, err
	}
	key := make([]byte, 32)
	if configured := os.Getenv("APP_KEY"); configured != "" {
		if key, err = hex.DecodeString(configured); err != nil || len(key) != 32 {
			return nil, fmt.Errorf("APP_KEY must contain 64 hexadecimal characters (32 bytes)")
		}
	} else {
		if cfg.Env == "production" {
			return nil, fmt.Errorf("APP_KEY is required in production; generate with ./artisan key:generate")
		}
		if _, err = rand.Read(key); err != nil {
			return nil, err
		}
	}
	a, err := NewCore(cfg)
	if err != nil {
		return nil, err
	}
	a.Key, a.Assets = key, assets
	flash, err := session.NewCookie(session.CookieOptions{Name: "aos_flash", Keys: [][]byte{key}, HTTPOnly: true, Secure: cfg.Env == "production", SameSite: http.SameSiteLaxMode, MaxAge: 120})
	if err != nil {
		return nil, err
	}
	a.Session = &captureStore{store: flash}
	if err := a.BootstrapAdmin(context.Background()); err != nil {
		return nil, err
	}
	a.Adapter, err = upstream.New(upstream.Config{RootView: "root.html", TemplateFS: views, Version: assets.Version, Session: a.Session, Vite: assets})
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (a *Application) Close() { a.Pool.Close() }

// BootstrapAdmin creates or updates the admin account from ADMIN_EMAIL / ADMIN_PASSWORD.
func (a *Application) BootstrapAdmin(ctx context.Context) error {
	email, pass := strings.TrimSpace(a.Cfg.AdminEmail), a.Cfg.AdminPassword
	if email == "" || pass == "" {
		n, err := a.Store.CountAdmins(ctx)
		if err != nil {
			return err
		}
		if n == 0 {
			log.Printf("no admin user yet: set ADMIN_EMAIL and ADMIN_PASSWORD (or run `go run . admin:create`) to enable /admin")
		}
		return nil
	}
	if a.Env == "production" && len(pass) < 12 {
		return fmt.Errorf("ADMIN_PASSWORD must have at least 12 characters in production")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return a.Store.UpsertAdmin(ctx, email, string(hash))
}
