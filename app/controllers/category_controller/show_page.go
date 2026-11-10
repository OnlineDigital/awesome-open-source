package category_controller

import (
	"errors"
	"fmt"
	"strconv"

	"awesomeopensource.dev/app/application"
	"awesomeopensource.dev/app/seo"
	"awesomeopensource.dev/app/store"
	"github.com/gofiber/fiber/v3"
	inertia "github.com/inertia-go/inertia-go"
)

const perPage = 24

func Show(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx, loc := c.Context(), app.LocaleOf(c)
		t := func(k string, kv ...any) string { return app.I18n.T(loc, k, kv...) }
		slug := application.RouteParameter(c, "slug")
		cat, err := app.Store.CategoryBySlug(ctx, slug, loc)
		if errors.Is(err, store.ErrNotFound) {
			return app.NotFound(c)
		} else if err != nil {
			return err
		}
		sort := c.Query("sort")
		if sort != "stars" && sort != "new" {
			sort = "score"
		}
		page := max(1, application.QueryInt(c, "page", 1))
		minScore := 0.0
		if v, err := strconv.ParseFloat(c.Query("min"), 64); err == nil && v >= 1 && v <= 5 {
			minScore = v
		}
		ps, total, err := app.Store.ListProjects(ctx, store.ProjectFilter{Locale: loc, Status: "published", CategoryID: cat.ID, Sort: sort, MinScore: minScore, Limit: perPage, Offset: (page - 1) * perPage})
		if err != nil {
			return err
		}
		pages := max(1, (total+perPage-1)/perPage)
		if page > pages {
			return app.NotFound(c)
		}
		cats, err := app.Store.ListCategories(ctx, loc)
		if err != nil {
			return err
		}

		path := "/category/" + cat.Slug
		title := t("category.title", "name", cat.Name)
		desc := cat.Description
		if desc == "" {
			desc = fmt.Sprintf("%s — %s", title, t("common.projects", "n", total))
		}
		metaTitle := seo.Truncate(title, 55) + " | " + app.Cfg.SiteName
		if page > 1 {
			metaTitle = seo.Truncate(title, 45) + " – " + t("common.page", "n", page) + " | " + app.Cfg.SiteName
		}
		meta := app.NewMeta(loc, path, metaTitle, seo.Truncate(desc, 160))
		if page > 1 {
			meta.Canonical += "?page=" + strconv.Itoa(page)
		}
		if sort != "score" || minScore > 0 {
			meta.Robots = "noindex,follow" // filtered/sorted views duplicate the canonical list
			meta.Canonical = app.Abs(loc, path)
		}
		if locs, err := app.Store.CategoryLocales(ctx); err == nil {
			meta.Alternates = app.AlternatesIn(path, locs[cat.ID])
		}
		if loc != app.Cfg.DefaultLocale && !cat.Translated {
			app.Untranslated(meta, path)
		}
		meta.Markdown = app.Cfg.BaseURL + app.Path(loc, "/llms/category/"+cat.Slug+".txt")
		cards := app.Cards(ps, loc)
		items := make([]seo.ListItem, 0, len(cards))
		links := make([]seo.Link, 0, len(cards))
		for _, p := range cards {
			items = append(items, seo.ListItem{Name: p.FullName, URL: p.URL})
			links = append(links, seo.Link{Text: p.FullName, URL: p.Path, Note: p.Tagline})
		}
		cp := seo.CollectionPage(title, desc, meta.Canonical, loc, app.Cfg.BaseURL, items)
		meta.JSONLD = []any{cp, seo.Breadcrumbs([]seo.Crumb{{Name: app.Cfg.SiteName, URL: app.Abs(loc, "/")}, {Name: cat.Name, URL: app.Abs(loc, path)}})}
		meta.Fallback = seo.Fallback(title, desc, []seo.Section{{Links: links}})

		return app.Page(c, "Category", path, inertia.Props{
			"category":   app.CategoryCards([]store.Category{*cat}, loc)[0],
			"categories": app.CategoryCards(cats, loc),
			"projects":   cards, "total": total, "page": page, "pages": pages, "sort": sort, "min": minScore,
		}, meta)
	}
}
