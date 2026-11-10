package search_controller

import (
	"awesomeopensource.dev/app/application"
	"github.com/gofiber/fiber/v3"
	inertia "github.com/inertia-go/inertia-go"
)

func Index(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		loc := app.LocaleOf(c)
		params := application.SearchParamsFrom(c)
		result, err := app.RunSearch(c.Context(), loc, params)
		unavailable := err != nil
		meta := app.NewMeta(loc, "/search", app.I18n.T(loc, "search.title")+" | "+app.Cfg.SiteName, app.I18n.T(loc, "seo.home_description"))
		meta.Robots = "noindex,follow"
		meta.Canonical = app.Abs(loc, "/search")
		props := inertia.Props{
			"query": params.Query, "category": params.Category, "min": params.MinScore, "sort": params.Sort, "page": params.Page,
			"unavailable": unavailable, "result": result,
		}
		return app.Page(c, "Search", "/search", props, meta)
	}
}
