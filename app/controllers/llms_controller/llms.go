// Package llms_controller serves the AI-agent documents (llms.txt tree).
package llms_controller

import (
	"errors"
	"strings"

	"awesomeopensource.dev/app/application"
	"awesomeopensource.dev/app/llms"
	"awesomeopensource.dev/app/store"
	"github.com/gofiber/fiber/v3"
)

func setHeaders(c fiber.Ctx, contentType string) {
	c.Set("Content-Type", contentType)
	c.Set("Cache-Control", "public, max-age=300")
	c.Set("X-Robots-Tag", "noindex") // for agents; keeps the .md/.txt twins out of web search results
	c.Set("Access-Control-Allow-Origin", "*")
}

func tFor(app *application.Application, loc string) llms.T {
	return func(key string, kv ...any) string { return app.I18n.T(loc, key, kv...) }
}

// Index serves /llms.txt (and /{locale}/llms.txt).
func Index(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		loc := app.LocaleOf(c)
		cats, err := app.Store.ListCategories(c.Context(), loc)
		if err != nil {
			return err
		}
		in := llms.IndexInput{T: tFor(app, loc), SiteName: app.Cfg.SiteName, SiteURL: app.Cfg.BaseURL, Description: app.I18n.T(loc, "seo.home_description"),
			SitemapURL: app.Cfg.BaseURL + "/sitemap.xml", HomeURL: app.Abs(loc, "/")}
		for _, cat := range cats {
			if cat.ProjectCount > 0 {
				in.Categories = append(in.Categories, llms.Cat{Slug: cat.Slug, Name: cat.Name, Description: cat.Description, Count: cat.ProjectCount})
			}
		}
		setHeaders(c, "text/plain; charset=utf-8")
		return c.SendString(llms.BuildIndex(in))
	}
}

// Category serves /llms/category/{slug}.txt.
func Category(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		loc := app.LocaleOf(c)
		slug, ok := strings.CutSuffix(application.RouteParameter(c, "file"), ".txt")
		if !ok {
			return fiber.ErrNotFound
		}
		cat, err := app.Store.CategoryBySlug(c.Context(), slug, loc)
		if errors.Is(err, store.ErrNotFound) {
			return fiber.ErrNotFound
		} else if err != nil {
			return err
		}
		ps, _, err := app.Store.ListProjects(c.Context(), store.ProjectFilter{Locale: loc, Status: "published", CategoryID: cat.ID, Sort: "score", Limit: 200})
		if err != nil {
			return err
		}
		t := tFor(app, loc)
		in := llms.CategoryInput{T: t, Name: cat.Name, Description: cat.Description, PageURL: app.Abs(loc, "/category/"+cat.Slug)}
		for i := range ps {
			in.Projects = append(in.Projects, projOf(t, &ps[i]))
		}
		setHeaders(c, "text/plain; charset=utf-8")
		return c.SendString(llms.BuildCategory(in))
	}
}

func projOf(t llms.T, p *store.Project) llms.Proj {
	out := llms.Proj{Slug: p.Slug, FullName: p.FullName, Tagline: p.Tagline, Language: p.Language, Score: p.Score, Stars: p.Stars}
	if act := application.ActivityOf(p); act != nil {
		out.TierLabel = t("tier." + act.Label)
	}
	return out
}

// Project serves /llms/{slug}.md.
func Project(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		loc := app.LocaleOf(c)
		slug, ok := strings.CutSuffix(application.RouteParameter(c, "file"), ".md")
		if !ok {
			return fiber.ErrNotFound
		}
		p, err := app.Store.GetProjectBySlug(c.Context(), slug, loc, true)
		if errors.Is(err, store.ErrNotFound) {
			return fiber.ErrNotFound
		} else if err != nil {
			return err
		}
		related, err := app.Store.Related(c.Context(), p, loc, 6)
		if err != nil {
			return err
		}
		t := tFor(app, loc)
		in := llms.ProjectInput{
			T: t, Slug: p.Slug, FullName: p.FullName, Tagline: p.Tagline, Description: p.Description, CategorySlug: p.CategorySlug, CategoryName: p.CategoryName,
			PageURL: app.Abs(loc, "/project/"+p.Slug), GitHubURL: "https://github.com/" + p.FullName, Homepage: p.Homepage, License: p.License, Language: p.Language,
			Topics: p.Topics, Stars: p.Stars, Archived: p.Archived, Score: p.Score, LastCommit: p.LastCommitAt, LastRelease: p.LastReleaseAt, LastReleaseTag: p.LastReleaseTag,
			Machine: p.Machine,
		}
		if act := application.ActivityOf(p); act != nil {
			in.TierLabel, in.Cap, in.Components, in.Commits90, in.Commits365 = t("tier."+act.Label), act.Cap, act.Components, act.Commits90, act.Commits365
		}
		if p.YouTubeID != "" {
			in.VideoURL = application.YouTubeWatch(p.YouTubeID)
		}
		for _, s := range p.Screenshots {
			u := s.Path
			if !strings.HasPrefix(u, "http") {
				u = app.Cfg.BaseURL + u
			}
			in.Screenshots = append(in.Screenshots, llms.Shot{URL: u, Alt: s.Alt})
		}
		for i := range related {
			in.Related = append(in.Related, projOf(t, &related[i]))
		}
		setHeaders(c, "text/markdown; charset=utf-8")
		return c.SendString(llms.BuildProject(in))
	}
}
