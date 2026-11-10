package server

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"awesomeopensource.dev/app/application"
	"awesomeopensource.dev/app/config"
	"awesomeopensource.dev/public"
	"awesomeopensource.dev/routes"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/static"
)

type Options struct {
	Config       config.Config
	DevServerURL string
	// AssetFS is rooted at Vite's build directory. A non-nil filesystem is
	// authoritative and useful for isolated tests or custom distributions.
	AssetFS fs.FS
}

// New creates the Fiber server and the application behind it.
func New(root string) (*fiber.App, *application.Application, error) {
	cfg := config.Load()
	if cfg.Env == "" {
		cfg.Env = defaultEnvironment()
	}
	return NewWithOptions(root, Options{Config: cfg, DevServerURL: os.Getenv("VITE_DEV_SERVER_URL")})
}

func NewWithOptions(root string, options Options) (*fiber.App, *application.Application, error) {
	if root == "" {
		return nil, nil, fmt.Errorf("application root is required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, nil, err
	}
	cfg := options.Config
	if cfg.Env == "" {
		cfg.Env = defaultEnvironment()
	}
	assetFS := options.AssetFS
	if assetFS == nil {
		assetFS = public.Assets()
	}
	// A shipped bundle or explicit filesystem always wins over dev-server URLs.
	if assetFS != nil {
		options.DevServerURL = ""
	}
	deps, err := application.New(root, cfg, options.DevServerURL, assetFS)
	if err != nil {
		return nil, nil, err
	}
	if cfg.UploadDir != "" {
		if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
			return nil, nil, fmt.Errorf("create upload dir: %w", err)
		}
	}
	app := fiber.New(fiber.Config{
		AppName:   cfg.SiteName,
		Immutable: true, StrictRouting: true, CaseSensitive: true,
		UnescapePath: false, BodyLimit: 12 << 20,
		ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 60 * time.Second,
		ErrorHandler: errorHandler(deps),
	})
	app.Use(recover.New())
	app.Use(compress.New())
	app.Use(func(c fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "SAMEORIGIN")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Cache-Control", "no-cache, private")
		return c.Next()
	})
	if assetFS == nil {
		assetFS = os.DirFS(filepath.Join(root, "public", "build"))
	}
	assets := static.New(".", static.Config{
		FS: assetFS, Browse: false, MaxAge: 31536000,
		ModifyResponse: func(c fiber.Ctx) error {
			c.Set("Cache-Control", "public, max-age=31536000, immutable")
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
	if cfg.UploadDir != "" {
		uploads := static.New(cfg.UploadDir, static.Config{Browse: false, MaxAge: 31536000, ModifyResponse: func(c fiber.Ctx) error {
			c.Set("Cache-Control", "public, max-age=31536000, immutable")
			c.Set("Content-Security-Policy", "default-src 'none'; img-src 'self'; sandbox")
			return nil
		}})
		app.Use("/uploads", func(c fiber.Ctx) error {
			if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
				return fiber.ErrMethodNotAllowed
			}
			if strings.Contains(c.Path(), "/.") {
				return fiber.ErrNotFound
			}
			return uploads(c)
		})
	}
	routes.Web(deps, app)
	return app, deps, nil
}

// errorHandler renders HTML error pages for browsers and plain text/JSON for machines.
func errorHandler(deps *application.Application) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		var fe *fiber.Error
		if errors.As(err, &fe) {
			code = fe.Code
		}
		if code >= 500 {
			log.Printf("%s %s: %v", c.Method(), c.OriginalURL(), err)
		}
		p := c.Path()
		machine := strings.HasPrefix(p, "/api/") || strings.Contains(p, "/llms") || strings.HasSuffix(p, ".xml") || strings.HasSuffix(p, ".txt")
		if machine {
			if strings.HasPrefix(p, "/api/") {
				return c.Status(code).JSON(fiber.Map{"error": http.StatusText(code)})
			}
			return c.Status(code).SendString(http.StatusText(code))
		}
		if code < 400 || (c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead && c.Get("X-Inertia") == "") {
			return c.Status(code).SendString(http.StatusText(code))
		}
		if renderErr := deps.ErrorPage(c, code); renderErr != nil {
			return c.Status(code).SendString(http.StatusText(code))
		}
		return nil
	}
}

func defaultEnvironment() string {
	if public.Assets() != nil {
		return "production"
	}
	return "development"
}

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

// HTTPHandler exists for net/http-based test tools and embedding only.
func HTTPHandler(app *fiber.App) http.Handler { return adaptor.FiberApp(app) }
