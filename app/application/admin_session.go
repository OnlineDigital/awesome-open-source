package application

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"awesomeopensource.dev/app/seo"
	"awesomeopensource.dev/app/store"
	"github.com/gofiber/fiber/v3"
)

const (
	adminCookie = "aos_admin"
	adminTTL    = 12 * time.Hour
)

type adminKey struct{}

func (a *Application) sign(payload string) string {
	mac := hmac.New(sha256.New, a.Key)
	mac.Write([]byte("admin:" + payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (a *Application) LoginAdmin(c fiber.Ctx, id int64) {
	payload := fmt.Sprintf("%d.%d", id, time.Now().Add(adminTTL).Unix())
	c.Cookie(&fiber.Cookie{
		Name: adminCookie, Value: payload + "." + a.sign(payload), Path: "/", HTTPOnly: true,
		Secure: a.Env == "production", SameSite: "Lax", MaxAge: int(adminTTL.Seconds()),
	})
}

func (a *Application) LogoutAdmin(c fiber.Ctx) {
	c.Cookie(&fiber.Cookie{Name: adminCookie, Value: "", Path: "/", HTTPOnly: true, MaxAge: -1, SameSite: "Lax", Secure: a.Env == "production"})
}

// parseAdminCookie validates signature and expiry and returns the admin id.
func (a *Application) parseAdminCookie(v string) (int64, bool) {
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return 0, false
	}
	payload := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(a.sign(payload)), []byte(parts[2])) {
		return 0, false
	}
	exp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return 0, false
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	return id, err == nil
}

// Admin returns the signed-in admin for this request (set by RequireAdmin/LoadAdmin), or nil.
func (a *Application) Admin(c fiber.Ctx) *store.AdminUser {
	u, _ := c.Locals(adminKey{}).(*store.AdminUser)
	return u
}

// LoadAdmin resolves the admin cookie without enforcing it (used for the login page).
func (a *Application) LoadAdmin(c fiber.Ctx) error {
	if id, ok := a.parseAdminCookie(c.Cookies(adminCookie)); ok {
		if u, err := a.Store.AdminByID(c.Context(), id); err == nil {
			c.Locals(adminKey{}, u)
		}
	}
	return c.Next()
}

// RequireAdmin redirects anonymous visitors to the login page.
func (a *Application) RequireAdmin(c fiber.Ctx) error {
	if id, ok := a.parseAdminCookie(c.Cookies(adminCookie)); ok {
		if u, err := a.Store.AdminByID(c.Context(), id); err == nil {
			c.Locals(adminKey{}, u)
			c.Set("Cache-Control", "no-store")
			c.Set("X-Robots-Tag", "noindex, nofollow")
			return c.Next()
		}
	}
	if c.Get("X-Inertia") != "" && c.Method() == fiber.MethodGet {
		c.Set("X-Inertia-Location", "/admin/login")
		return c.SendStatus(fiber.StatusConflict)
	}
	return c.Redirect().Status(fiber.StatusSeeOther).To("/admin/login")
}

// SameOrigin rejects cross-site state-changing requests (CSRF defence on top of SameSite=Lax).
func (a *Application) SameOrigin(c fiber.Ctx) error {
	switch c.Method() {
	case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
		return c.Next()
	}
	if site := c.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
		return fiber.NewError(fiber.StatusForbidden, "cross-site request rejected")
	}
	if origin := c.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil {
			return fiber.NewError(fiber.StatusForbidden, "cross-origin request rejected")
		}
		// Accept the host the request was sent to, or the configured public URL (behind reverse proxies).
		public, _ := url.Parse(a.Cfg.BaseURL)
		if !strings.EqualFold(u.Host, c.Host()) && (public == nil || !strings.EqualFold(u.Host, public.Host)) {
			return fiber.NewError(fiber.StatusForbidden, "cross-origin request rejected")
		}
	}
	return c.Next()
}

func (a *Application) AllowLogin(c fiber.Ctx) bool { return a.limiter.Allow(c.IP()) }
func (a *Application) ResetLogin(c fiber.Ctx)      { a.limiter.Reset(c.IP()) }

// AdminPage renders an admin Inertia page (never indexed, never cached).
func (a *Application) AdminPage(c fiber.Ctx, component string, props map[string]any) error {
	meta := &seo.Meta{Title: "Admin | " + a.Cfg.SiteName, Robots: "noindex,nofollow", Locale: a.Cfg.DefaultLocale}
	if u := a.Admin(c); u != nil {
		props["auth"] = map[string]any{"email": u.Email}
	}
	props["adminLocales"] = a.I18n.Locales
	props["defaultLocale"] = a.Cfg.DefaultLocale
	props["capabilities"] = map[string]bool{"translate": a.Translator.Enabled(), "search": a.Search.Enabled(), "githubToken": a.Cfg.GitHubToken != "", "embeddings": a.Embedder.Enabled()}
	c.Set("Cache-Control", "no-store")
	c.Set("X-Robots-Tag", "noindex, nofollow")
	return a.Page(c, component, "/", props, meta)
}
