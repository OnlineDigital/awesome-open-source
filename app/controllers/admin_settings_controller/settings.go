// Package admin_settings_controller manages MCP access links (tokens) for AI agents.
package admin_settings_controller

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"

	"awesomeopensource.dev/app/application"
	"awesomeopensource.dev/app/store"
	"github.com/gofiber/fiber/v3"
)

type link struct {
	store.MCPToken
	SSEURL  string `json:"sseUrl"`
	HTTPURL string `json:"httpUrl"`
	Active  bool   `json:"active"`
}

func newSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func tokenID(c fiber.Ctx) (int64, error) {
	n, err := strconv.ParseInt(application.RouteParameter(c, "id"), 10, 64)
	if err != nil || n < 1 {
		return 0, fiber.ErrNotFound
	}
	return n, nil
}

func Index(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		toks, err := app.Store.ListMCPTokens(c.Context())
		if err != nil {
			return err
		}
		links := make([]link, 0, len(toks))
		for _, t := range toks {
			links = append(links, link{MCPToken: t, SSEURL: app.Cfg.BaseURL + "/mcp/" + t.Token + "/sse", HTTPURL: app.Cfg.BaseURL + "/mcp/" + t.Token, Active: t.RevokedAt == nil})
		}
		return app.AdminPage(c, "Admin/Settings", map[string]any{"links": links})
	}
}

func Create(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req struct {
			Name string `json:"name" form:"name"`
		}
		if err := c.Bind().Body(&req); err != nil {
			return fiber.ErrBadRequest
		}
		name := strings.TrimSpace(req.Name)
		if name == "" {
			name = "Agent link"
		}
		if len(name) > 80 {
			name = name[:80]
		}
		if _, err := app.Store.CreateMCPToken(c.Context(), name, newSecret()); err != nil {
			return err
		}
		return app.Redirect303(c, "/admin/settings", nil, "MCP link created. Copy it into your agent's MCP configuration.")
	}
}

func Rename(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		tid, err := tokenID(c)
		if err != nil {
			return err
		}
		var req struct {
			Name string `json:"name" form:"name"`
		}
		if err := c.Bind().Body(&req); err != nil {
			return fiber.ErrBadRequest
		}
		name := strings.TrimSpace(req.Name)
		if name == "" || len(name) > 80 {
			return app.Redirect303(c, "/admin/settings", map[string]string{"name": "Enter a name up to 80 characters."}, "")
		}
		if err := app.Store.RenameMCPToken(c.Context(), tid, name); err != nil {
			return err
		}
		return app.Redirect303(c, "/admin/settings", nil, "Link renamed.")
	}
}

func Rotate(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		tid, err := tokenID(c)
		if err != nil {
			return err
		}
		if err := app.Store.RotateMCPToken(c.Context(), tid, newSecret()); err != nil {
			return err
		}
		return app.Redirect303(c, "/admin/settings", nil, "Token rotated. The previous URL stopped working; update your agent with the new one.")
	}
}

func Revoke(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		tid, err := tokenID(c)
		if err != nil {
			return err
		}
		if err := app.Store.RevokeMCPToken(c.Context(), tid); err != nil {
			return err
		}
		return app.Redirect303(c, "/admin/settings", nil, "Link revoked. Rotate it to reactivate with a new secret.")
	}
}

func Destroy(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		tid, err := tokenID(c)
		if err != nil {
			return err
		}
		if err := app.Store.DeleteMCPToken(c.Context(), tid); err != nil {
			return err
		}
		return app.Redirect303(c, "/admin/settings", nil, "Link deleted.")
	}
}
