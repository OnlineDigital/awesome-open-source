// Package seo_controller serves robots.txt and the sitemaps.
package seo_controller

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"awesomeopensource.dev/app/application"
	"awesomeopensource.dev/app/seo"
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

// ProjectsPerSitemap keeps every file far below the 50,000 URL limit (each project has one URL per locale).
const ProjectsPerSitemap = 5000

// Index lists sitemap-general.xml plus one sitemap-project-NNN.xml per page of projects.
func Index(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx := c.Context()
		total, err := app.Store.CountPublished(ctx)
		if err != nil {
			return err
		}
		refs := []seo.SitemapRef{{Loc: app.Cfg.BaseURL + "/sitemap-general.xml", LastMod: time.Now()}}
		for page := 1; (page-1)*ProjectsPerSitemap < total; page++ {
			mod, err := app.Store.SitemapPageLastMod(ctx, (page-1)*ProjectsPerSitemap, ProjectsPerSitemap)
			if err != nil {
				return err
			}
			refs = append(refs, seo.SitemapRef{Loc: fmt.Sprintf("%s/sitemap-project-%03d.xml", app.Cfg.BaseURL, page), LastMod: mod})
		}
		return xml(c, seo.Index(refs))
	}
}

// entries returns one URL entry per translated locale for a canonical path.
// A nil translated list means every locale (pages that only use UI strings).
func entries(app *application.Application, path string, mod time.Time, translated []string, all bool) []seo.SitemapURL {
	var out []seo.SitemapURL
	alts := app.Alternates(path)
	locales := app.I18n.Locales
	if !all {
		alts = app.AlternatesIn(path, translated)
		locales = locales[:0:0]
		for _, a := range alts {
			if a.Locale != "x-default" {
				locales = append(locales, a.Locale)
			}
		}
	}
	for _, loc := range locales {
		out = append(out, seo.SitemapURL{Loc: app.Abs(loc, path), LastMod: mod, Alternates: alts})
	}
	return out
}

// General holds every URL except project pages: home and categories in all locales.
func General(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		cats, err := app.Store.ListCategories(c.Context(), app.Cfg.DefaultLocale)
		if err != nil {
			return err
		}
		locs, err := app.Store.CategoryLocales(c.Context())
		if err != nil {
			return err
		}
		urls := entries(app, "/", time.Now(), nil, true)
		for _, cat := range cats {
			if cat.ProjectCount > 0 {
				urls = append(urls, entries(app, "/category/"+cat.Slug, time.Now(), locs[cat.ID], false)...)
			}
		}
		topics, err := app.Store.ListTopics(c.Context(), app.Cfg.DefaultLocale, true)
		if err != nil {
			return err
		}
		tlocs, err := app.Store.TopicLocales(c.Context())
		if err != nil {
			return err
		}
		for _, tp := range topics {
			if tp.ProjectCount > 0 {
				urls = append(urls, entries(app, "/topic/"+tp.Slug, time.Now(), tlocs[tp.ID], false)...)
			}
		}
		return xml(c, seo.URLSet(urls))
	}
}

// Projects serves sitemap-project-NNN.xml: page NNN (1-based) of projects ordered by publication date ascending.
func Projects(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		num, ok := strings.CutSuffix(application.RouteParameter(c, "file"), ".xml")
		page, err := strconv.Atoi(num)
		if !ok || err != nil || page < 1 || num != fmt.Sprintf("%03d", page) {
			return fiber.ErrNotFound
		}
		ctx := c.Context()
		ps, err := app.Store.SitemapProjects(ctx, (page-1)*ProjectsPerSitemap, ProjectsPerSitemap)
		if err != nil {
			return err
		}
		if len(ps) == 0 {
			return fiber.ErrNotFound
		}
		locs, err := app.Store.ProjectLocales(ctx)
		if err != nil {
			return err
		}
		var urls []seo.SitemapURL
		for _, p := range ps {
			var imgs []string
			for _, s := range p.Screenshots {
				imgs = append(imgs, app.Cfg.BaseURL+s)
			}
			for _, e := range entries(app, "/project/"+p.Slug, p.UpdatedAt, locs[p.ID], false) {
				e.Images = imgs
				if p.YouTubeID != "" {
					e.Video = &seo.SitemapVideo{Thumbnail: application.YouTubeThumb(p.YouTubeID), Title: p.FullName + " demo", Description: firstNonEmpty(p.Tagline, p.FullName),
						Player: application.YouTubeEmbed(p.YouTubeID), PublishedISO: p.CreatedAt.Format(time.RFC3339)}
				}
				urls = append(urls, e)
			}
		}
		return xml(c, seo.URLSet(urls))
	}
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
