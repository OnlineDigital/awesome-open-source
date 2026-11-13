package application

import (
	"bytes"
	"html"
	"net/http"
	"strconv"
	"strings"

	"awesomeopensource.dev/app/i18n"
	"awesomeopensource.dev/app/seo"
	"github.com/gofiber/fiber/v3"
	upstream "github.com/inertia-go/inertia-go"
)

type localeKey struct{}

// LocaleMiddleware pins the locale for every route registered in its group.
func (a *Application) LocaleMiddleware(locale string) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Locals(localeKey{}, locale)
		return c.Next()
	}
}

func (a *Application) LocaleOf(c fiber.Ctx) string {
	if l, ok := c.Locals(localeKey{}).(string); ok && l != "" {
		return l
	}
	return a.Cfg.DefaultLocale
}

// Path returns the site-relative URL of a canonical path ("/", "/category/x") in a locale.
func (a *Application) Path(locale, path string) string {
	if path == "" {
		path = "/"
	}
	if locale == a.Cfg.DefaultLocale {
		return path
	}
	if path == "/" {
		return "/" + locale
	}
	return "/" + locale + path
}

func (a *Application) Abs(locale, path string) string { return a.Cfg.BaseURL + a.Path(locale, path) }

// Alternates lists the hreflang links for a canonical path, including x-default.
func (a *Application) Alternates(path string) []seo.Alternate {
	out := make([]seo.Alternate, 0, len(a.I18n.Locales)+1)
	for _, loc := range a.I18n.Locales {
		out = append(out, seo.Alternate{Locale: loc, URL: a.Abs(loc, path)})
	}
	return append(out, seo.Alternate{Locale: "x-default", URL: a.Abs(a.Cfg.DefaultLocale, path)})
}

// AlternatesIn lists hreflang links for the default locale plus the given translated locales.
func (a *Application) AlternatesIn(path string, translated []string) []seo.Alternate {
	has := map[string]bool{a.Cfg.DefaultLocale: true}
	for _, l := range translated {
		has[l] = true
	}
	out := make([]seo.Alternate, 0, len(has)+1)
	for _, loc := range a.I18n.Locales {
		if has[loc] {
			out = append(out, seo.Alternate{Locale: loc, URL: a.Abs(loc, path)})
		}
	}
	return append(out, seo.Alternate{Locale: "x-default", URL: a.Abs(a.Cfg.DefaultLocale, path)})
}

// Untranslated points a fallback-language page at the default locale and keeps it out of the index.
func (a *Application) Untranslated(m *seo.Meta, path string) {
	m.Canonical = a.Abs(a.Cfg.DefaultLocale, path)
	m.Robots = "noindex,follow"
}

// NewMeta returns SEO metadata prefilled with canonical, alternates and site defaults.
func (a *Application) NewMeta(locale, path, title, description string) *seo.Meta {
	return &seo.Meta{
		Title: title, Description: description, Locale: locale, OGLocale: i18n.OGLocales[locale], SiteName: a.Cfg.SiteName,
		Canonical: a.Abs(locale, path), Alternates: a.Alternates(path),
	}
}

type localeInfo struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Path string `json:"path"`
}

func (a *Application) localeList(path string) []localeInfo {
	out := make([]localeInfo, 0, len(a.I18n.Locales))
	for _, loc := range a.I18n.Locales {
		name := i18n.NativeNames[loc]
		if name == "" {
			name = loc
		}
		out = append(out, localeInfo{Code: loc, Name: name, Path: a.Path(loc, path)})
	}
	return out
}

// Page renders an Inertia page with shared props (locale, UI strings, SEO).
// path is the canonical (locale-less) path, used for the language switcher.
func (a *Application) Page(c fiber.Ctx, component, path string, props upstream.Props, meta *seo.Meta) error {
	loc := a.LocaleOf(c)
	props["locale"] = loc
	props["locales"] = a.localeList(path)
	props["i18n"] = a.I18n.Strings(loc)
	props["site"] = map[string]any{"name": a.Cfg.SiteName, "baseUrl": a.Cfg.BaseURL, "defaultLocale": a.Cfg.DefaultLocale, "homePath": a.Path(loc, "/")}
	if meta != nil {
		props["seo"] = meta.Build()
	}
	status := c.Response().StatusCode()
	err := a.RenderPageWith(c, component, props)
	if err != nil {
		return err
	}
	c.Vary("X-Inertia")
	if status != 0 && status != 200 && c.Response().StatusCode() == 200 {
		c.Status(status)
	}
	if meta != nil {
		injectSEO(c, loc, meta)
	}
	return nil
}

// analyticsTag is the traffic analytics snippet rendered from root.html.
const analyticsTag = `<script defer src="https://stats.onlinedigital.ro/script.js" data-website-id="16550856-70f6-4c1b-b3d6-f517940482f3"></script>`

// injectSEO puts head tags, <html lang> and the no-JS fallback into full HTML responses.
func injectSEO(c fiber.Ctx, locale string, meta *seo.Meta) {
	if !strings.HasPrefix(string(c.Response().Header.ContentType()), "text/html") {
		return
	}
	body := c.Response().Body()
	if strings.HasPrefix(c.Path(), "/admin") {
		body = bytes.Replace(body, []byte(analyticsTag), nil, 1)
	}
	body = bytes.Replace(body, []byte(`<meta name="x-seo-head">`), []byte(meta.HeadHTML()), 1)
	body = bytes.Replace(body, []byte(`<html lang="en" data-locale-placeholder>`), []byte(`<html lang="`+html.EscapeString(locale)+`">`), 1)
	body = bytes.Replace(body, []byte(`<template id="x-seo-body"></template>`), []byte(`<noscript>`+string(meta.Fallback)+`</noscript>`), 1)
	c.Response().ResetBody()
	c.Response().SetBodyRaw(body)
}

// RenderPageWith renders via the native Inertia adapter.
func (a *Application) RenderPageWith(c fiber.Ctx, component string, props upstream.Props) error {
	return a.Handle(func(w http.ResponseWriter, r *http.Request) { a.renderWith(w, r, component, props) })(c)
}

func (a *Application) renderWith(w http.ResponseWriter, r *http.Request, component string, props upstream.Props) {
	state := messagesFrom(r)
	if _, exists := props["errors"]; !exists {
		errors := map[string]any{}
		for key, value := range state.errors {
			errors[key] = value
		}
		if bag := upstream.FromRequest(r).ErrorBag; bag != "" && len(errors) > 0 {
			errors = map[string]any{bag: errors}
		}
		props["errors"] = upstream.Always(errors)
	}
	flash := map[string]any{}
	for k, v := range state.flash {
		flash[k] = v
	}
	props["flash"] = upstream.Always(flash)

	// The upstream adapter consumes flash on stale-version responses; keep it for the follow-up full navigation.
	info := upstream.FromRequest(r)
	if info.IsInertia && r.Method == http.MethodGet && info.Version != "" && info.Version != a.Assets.Version {
		bag := info.ErrorBag
		if bag == "" {
			bag = "default"
		}
		_ = a.Session.FlashErrors(w, r, bag, state.errors)
		for key, value := range state.flash {
			_ = a.Session.FlashMessage(w, r, key, value)
		}
	}
	a.Adapter.Render(w, r, component, props)
}

// Redirect303 flashes validation errors or a success message and redirects with 303 (required after non-GET Inertia requests).
func (a *Application) Redirect303(c fiber.Ctx, to string, errors map[string]string, success string) error {
	return a.Handle(func(w http.ResponseWriter, r *http.Request) {
		bag := r.Header.Get("X-Inertia-Error-Bag")
		if bag == "" {
			bag = "default"
		}
		if len(errors) > 0 {
			_ = a.Session.FlashErrors(w, r, bag, errors)
		} else if success != "" {
			_ = a.Session.FlashMessage(w, r, "success", success)
		}
		http.Redirect(w, r, to, http.StatusSeeOther)
	})(c)
}

// QueryInt reads an integer query parameter with a default.
func QueryInt(c fiber.Ctx, key string, def int) int {
	if v, err := strconv.Atoi(c.Query(key)); err == nil {
		return v
	}
	return def
}

// NotFound renders the 404 page for the current locale.
func (a *Application) NotFound(c fiber.Ctx) error { return a.ErrorPage(c, fiber.StatusNotFound) }

// LocaleFromPath resolves the locale from a URL path prefix (used when no route matched).
func (a *Application) LocaleFromPath(path string) string {
	first, _, _ := strings.Cut(strings.TrimPrefix(path, "/"), "/")
	if first != a.Cfg.DefaultLocale && a.I18n.Has(first) {
		return first
	}
	return a.Cfg.DefaultLocale
}

// ErrorPage renders the Inertia error page with the given HTTP status.
func (a *Application) ErrorPage(c fiber.Ctx, status int) error {
	loc := a.LocaleFromPath(c.Path())
	c.Locals(localeKey{}, loc)
	key := "error.not_found"
	if status != fiber.StatusNotFound {
		key = "error.server"
	}
	c.Set("Cache-Control", "no-store")
	meta := a.NewMeta(loc, "/", a.I18n.T(loc, key)+" | "+a.Cfg.SiteName, "")
	meta.Robots = "noindex,follow"
	meta.Canonical, meta.Alternates = "", nil
	meta.Fallback = seo.Fallback(a.I18n.T(loc, key), "", []seo.Section{{Links: []seo.Link{{Text: a.I18n.T(loc, "error.back_home"), URL: a.Path(loc, "/")}}}})
	c.Status(status)
	return a.Page(c, "Error", "/", upstream.Props{"status": status}, meta)
}
