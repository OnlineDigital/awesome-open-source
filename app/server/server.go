package server

import (
	"fmt"
	"github.com/andreitelteu/inertia-go-solid-template/app/application"
	"github.com/andreitelteu/inertia-go-solid-template/public"
	"github.com/andreitelteu/inertia-go-solid-template/routes"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/static"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type Options struct {
	Demo         bool
	Environment  string
	DevServerURL string
	// AssetFS is rooted at Vite's build directory. A non-nil filesystem is
	// authoritative and useful for isolated tests or custom distributions.
	AssetFS fs.FS
}

// New creates the actual Fiber server. Production disables diagnostic routes.
func New(root string) (*fiber.App, error) {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = defaultEnvironment()
	}
	return NewWithOptions(root, Options{Demo: env != "production", Environment: env, DevServerURL: os.Getenv("VITE_DEV_SERVER_URL")})
}
func NewWithOptions(root string, options Options) (*fiber.App, error) {
	if root == "" {
		return nil, fmt.Errorf("application root is required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if options.Environment == "" {
		options.Environment = defaultEnvironment()
	}
	assetFS := options.AssetFS
	if assetFS == nil {
		assetFS = public.Assets()
	}
	// A shipped bundle or explicit filesystem always wins over dev-server URLs.
	// Production binaries never use a missing/modified on-disk build as fallback.
	if assetFS != nil {
		options.DevServerURL = ""
	}
	deps, err := application.NewWithAssets(root, options.Environment, options.DevServerURL, options.Demo, assetFS)
	if err != nil {
		return nil, err
	}
	app := fiber.New(fiber.Config{
		AppName:   "inertia-go + Solid template",
		Immutable: true, StrictRouting: true, CaseSensitive: true,
		// Keep encoded slashes inside one :parameter. The inertia boundary decodes
		// the parameter once; reverse URLs use PathEscape on each segment.
		UnescapePath: false, BodyLimit: 8 << 20,
		ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
	})
	app.Use(recover.New())
	app.Use(func(c fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Cache-Control", "no-cache, private")
		return c.Next()
	})
	// Vite assets are served natively by Fiber. No HTTP file server or ServeMux.
	if assetFS == nil {
		assetFS = os.DirFS(filepath.Join(root, "public", "build"))
	}
	assets := static.New(".", static.Config{
		FS: assetFS, Browse: false, MaxAge: 31536000,
		ModifyResponse: func(c fiber.Ctx) error {
			c.Set("Cache-Control", "public, max-age=31536000, immutable")
			// Windows MIME registrations can be absent or incorrect. In
			// particular, nosniff requires module scripts/styles to have a
			// browser-compatible type, and fonts must not depend on sniffing.
			if contentType := assetContentType(c.Path()); contentType != "" {
				c.Set("Content-Type", contentType)
			}
			return nil
		},
	})
	app.Use("/build", func(c fiber.Ctx) error {
		if !publicAssetPath(c.Path()) {
			return fiber.ErrNotFound
		}
		if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
			c.Set("Allow", "GET, HEAD")
			return fiber.ErrMethodNotAllowed
		}
		return assets(c)
	})
	routes.Web(deps, app)
	return app, nil
}

func defaultEnvironment() string {
	if public.Assets() != nil {
		return "production"
	}
	return "demo"
}

// Match Fiber static's repeated URL decoding before checking private paths.
// Hidden files and parent traversal remain private even through encoded slashes,
// percent-encoded names, nested encodings or backslash separators.
func publicAssetPath(assetPath string) bool {
	assetPath, valid := decodedAssetPath(assetPath)
	if !valid {
		return false
	}
	if assetPath == "/build" || assetPath == "/build/" {
		return false
	}
	for _, segment := range strings.Split(assetPath, "/") {
		if strings.HasPrefix(segment, ".") {
			return false
		}
	}
	return true
}

func decodedAssetPath(assetPath string) (string, bool) {
	for strings.Contains(assetPath, "%") {
		decoded, err := url.PathUnescape(assetPath)
		if err != nil {
			return "", false
		}
		if decoded == assetPath {
			break
		}
		assetPath = decoded
	}
	assetPath = strings.ReplaceAll(assetPath, "\\", "/")
	return assetPath, true
}

func assetContentType(assetPath string) string {
	assetPath, valid := decodedAssetPath(assetPath)
	if !valid {
		return ""
	}
	switch strings.ToLower(path.Ext(assetPath)) {
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".otf":
		return "font/otf"
	case ".wasm":
		return "application/wasm"
	default:
		return ""
	}
}

// HTTPHandler exists for net/http-based test tools and embedding only. It
// traverses the same Fiber app and native router; the standalone server Listen
// always runs Fiber/fasthttp, so this is not a second backend implementation.
func HTTPHandler(app *fiber.App) http.Handler { return adaptor.FiberApp(app) }
