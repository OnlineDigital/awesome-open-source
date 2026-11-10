package home_controller

import (
	"awesomeopensource.dev/app/application"
	"awesomeopensource.dev/app/seo"
	"awesomeopensource.dev/app/store"
	"github.com/gofiber/fiber/v3"
	inertia "github.com/inertia-go/inertia-go"
)

func Index(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx, loc := c.Context(), app.LocaleOf(c)
		t := func(k string, kv ...any) string { return app.I18n.T(loc, k, kv...) }
		cats, err := app.Store.ListCategories(ctx, loc)
		if err != nil {
			return err
		}
		top, _, err := app.Store.ListProjects(ctx, store.ProjectFilter{Locale: loc, Status: "published", Sort: "score", Limit: 8, MinScore: 1})
		if err != nil {
			return err
		}
		latest, _, err := app.Store.ListProjects(ctx, store.ProjectFilter{Locale: loc, Status: "published", Sort: "new", Limit: 8})
		if err != nil {
			return err
		}
		base := app.Cfg.BaseURL
		meta := app.NewMeta(loc, "/", t("seo.home_title"), t("seo.home_description"))
		meta.Markdown = base + app.Path(loc, "/llms.txt")
		meta.JSONLD = []any{
			seo.WebSite(app.Cfg.SiteName, base, base+app.Path(loc, "/search")+"?q={search_term_string}", loc),
			seo.Org(app.Cfg.SiteName, base),
		}
		var catLinks, projLinks []seo.Link
		for _, cc := range app.CategoryCards(cats, loc) {
			catLinks = append(catLinks, seo.Link{Text: cc.Name, URL: cc.Path, Note: cc.Description})
		}
		for _, p := range app.Cards(top, loc) {
			projLinks = append(projLinks, seo.Link{Text: p.FullName, URL: p.Path, Note: p.Tagline})
		}
		meta.Fallback = seo.Fallback(t("home.title"), t("home.subtitle"), []seo.Section{
			{Heading: t("home.browse"), Links: catLinks}, {Heading: t("home.top_active"), Links: projLinks},
			{Links: []seo.Link{{Text: t("footer.ai"), URL: app.Path(loc, "/llms.txt")}, {Text: t("footer.sitemap"), URL: "/sitemap.xml"}}},
		})
		return app.Page(c, "Home", "/", inertia.Props{
			"categories": app.CategoryCards(cats, loc), "top": app.Cards(top, loc), "latest": app.Cards(latest, loc),
		}, meta)
	}
}
