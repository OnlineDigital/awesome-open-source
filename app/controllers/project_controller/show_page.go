package project_controller

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"awesomeopensource.dev/app/application"
	"awesomeopensource.dev/app/seo"
	"awesomeopensource.dev/app/store"
	"github.com/gofiber/fiber/v3"
	inertia "github.com/inertia-go/inertia-go"
)

func Show(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx, loc := c.Context(), app.LocaleOf(c)
		t := func(k string, kv ...any) string { return app.I18n.T(loc, k, kv...) }
		slug := application.RouteParameter(c, "slug")
		p, err := app.Store.GetProjectBySlug(ctx, slug, loc, true)
		if errors.Is(err, store.ErrNotFound) {
			return app.NotFound(c)
		} else if err != nil {
			return err
		}
		related, err := app.Store.Related(ctx, p, loc, 6)
		if err != nil {
			return err
		}
		path := "/project/" + p.Slug
		base := app.Cfg.BaseURL
		desc := p.Description
		if desc == "" {
			desc = p.Tagline
		}
		title := seo.Truncate(p.Repo+" — "+p.Tagline, 58) + " | " + app.Cfg.SiteName
		if p.Tagline == "" {
			title = p.FullName + " | " + app.Cfg.SiteName
		}
		meta := app.NewMeta(loc, path, title, seo.Truncate(desc, 160))
		meta.OGType = "article"
		meta.Markdown = base + app.Path(loc, "/llms/"+p.Slug+".md")

		var shots []string
		for _, s := range p.Screenshots {
			shots = append(shots, absURL(base, s.Path))
		}
		switch {
		case len(shots) > 0:
			meta.Image = shots[0]
		case p.YouTubeID != "":
			meta.Image = application.YouTubeThumb(p.YouTubeID)
		case p.AvatarURL != "":
			meta.Image = p.AvatarURL
		}
		in := seo.SoftwareInput{
			Name: p.Repo, Description: desc, URL: meta.Canonical, Lang: loc, Category: p.CategoryName, Language: p.Language, License: p.License,
			RepoURL: "https://github.com/" + p.FullName, Homepage: p.Homepage, Version: strings.TrimPrefix(p.LastReleaseTag, "v"),
			Image: meta.Image, Screenshots: shots, Topics: p.Topics,
			PublishedISO: p.CreatedAt.Format(time.RFC3339), ModifiedISO: p.UpdatedAt.Format(time.RFC3339),
		}
		meta.JSONLD = []any{seo.Software(in)}
		if p.YouTubeID != "" {
			meta.JSONLD = append(meta.JSONLD, seo.Video(seo.VideoInput{
				Name: t("project.demo") + ": " + p.Repo, Description: seo.Truncate(desc, 300), EmbedURL: application.YouTubeEmbed(p.YouTubeID),
				ContentURL: application.YouTubeWatch(p.YouTubeID), Thumbnail: application.YouTubeThumb(p.YouTubeID), UploadDate: p.CreatedAt.Format(time.RFC3339),
			}))
		}
		crumbs := []seo.Crumb{{Name: app.Cfg.SiteName, URL: app.Abs(loc, "/")}}
		if p.CategorySlug != "" {
			crumbs = append(crumbs, seo.Crumb{Name: p.CategoryName, URL: app.Abs(loc, "/category/"+p.CategorySlug)})
		}
		crumbs = append(crumbs, seo.Crumb{Name: p.Repo, URL: meta.Canonical})
		meta.JSONLD = append(meta.JSONLD, seo.Breadcrumbs(crumbs))

		act := application.ActivityOf(p)
		var facts []seo.Link
		if act != nil {
			facts = append(facts, seo.Link{Text: t("score.title"), URL: meta.Canonical, Note: formatScore(act.Score) + " / 5 — " + t("tier."+act.Label)})
		}
		meta.Fallback = seo.Fallback(p.FullName, desc, []seo.Section{
			{Heading: t("project.activity"), Links: facts},
			{Links: []seo.Link{{Text: t("project.github"), URL: "https://github.com/" + p.FullName}, {Text: t("project.read_md"), URL: app.Path(loc, "/llms/"+p.Slug+".md")}}},
		})

		return app.Page(c, "Project", path, inertia.Props{
			"project": p, "activity": act, "related": app.Cards(related, loc), "translations": map[string]any{"machine": p.Machine},
			"video":        map[string]string{"id": p.YouTubeID, "thumb": application.YouTubeThumb(p.YouTubeID), "embed": application.YouTubeEmbed(p.YouTubeID)},
			"markdownPath": app.Path(loc, "/llms/"+p.Slug+".md"),
		}, meta)
	}
}

func absURL(base, p string) string {
	if strings.HasPrefix(p, "http") {
		return p
	}
	return base + p
}

func formatScore(f float64) string { return strconv.FormatFloat(f, 'f', 1, 64) }
