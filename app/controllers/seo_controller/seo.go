// Package seo_controller serves robots.txt and the sitemaps.
package seo_controller

import (
	"context"
	"time"

	"awesomeopensource.dev/app/application"
	"awesomeopensource.dev/app/seo"
	"awesomeopensource.dev/app/store"
	"github.com/gofiber/fiber/v3"
)

func xml(c fiber.Ctx, body []byte) error {
	c.Set("Content-Type", "application/xml; charset=utf-8")
	c.Set("Cache-Control", "public, max-age=900")
	return c.Send(body)
}

func Robots(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Set("Content-Type", "text/plain; charset=utf-8")
		c.Set("Cache-Control", "public, max-age=3600")
		return c.SendString(seo.Robots(app.Cfg.BaseURL))
	}
}

func Index(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		now := time.Now()
		var refs []seo.SitemapRef
		for _, name := range []string{"pages", "categories", "projects"} {
			refs = append(refs, seo.SitemapRef{Loc: app.Cfg.BaseURL + "/sitemap-" + name + ".xml", LastMod: now})
		}
		return xml(c, seo.Index(refs))
	}
}

// alternates returns one URL entry per locale for a canonical path.
func entries(app *application.Application, path string, mod time.Time) []seo.SitemapURL {
	var out []seo.SitemapURL
	alts := app.Alternates(path)
	for _, loc := range app.I18n.Locales {
		out = append(out, seo.SitemapURL{Loc: app.Abs(loc, path), LastMod: mod, Alternates: alts})
	}
	return out
}

func Pages(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error { return xml(c, seo.URLSet(entries(app, "/", time.Now()))) }
}

func Categories(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		cats, err := app.Store.ListCategories(c.Context(), app.Cfg.DefaultLocale)
		if err != nil {
			return err
		}
		var urls []seo.SitemapURL
		for _, cat := range cats {
			if cat.ProjectCount > 0 {
				urls = append(urls, entries(app, "/category/"+cat.Slug, time.Now())...)
			}
		}
		return xml(c, seo.URLSet(urls))
	}
}

func Projects(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx := c.Context()
		var urls []seo.SitemapURL
		for offset := 0; ; offset += 200 {
			ps, err := page(ctx, app, offset)
			if err != nil {
				return err
			}
			for _, p := range ps {
				path := "/project/" + p.Slug
				var imgs []string
				for _, s := range p.Screenshots {
					imgs = append(imgs, app.Cfg.BaseURL+s.Path)
				}
				for _, e := range entries(app, path, p.UpdatedAt) {
					e.Images = imgs
					if p.YouTubeID != "" {
						e.Video = &seo.SitemapVideo{Thumbnail: application.YouTubeThumb(p.YouTubeID), Title: p.FullName + " demo", Description: firstNonEmpty(p.Tagline, p.FullName),
							Player: application.YouTubeEmbed(p.YouTubeID), PublishedISO: p.CreatedAt.Format(time.RFC3339)}
					}
					urls = append(urls, e)
				}
			}
			if len(ps) < 200 {
				break
			}
		}
		return xml(c, seo.URLSet(urls))
	}
}

func page(ctx context.Context, app *application.Application, offset int) ([]store.Project, error) {
	ps, _, err := app.Store.ListProjects(ctx, store.ProjectFilter{Locale: app.Cfg.DefaultLocale, Status: "published", Sort: "name", Limit: 200, Offset: offset})
	if err != nil {
		return nil, err
	}
	for i := range ps {
		ps[i].Screenshots, _ = app.Store.Screenshots(ctx, ps[i].ID)
	}
	return ps, nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
