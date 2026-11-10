package scroll_controller

import (
	"github.com/andreitelteu/inertia-go-solid-template/app/application"
	"github.com/gofiber/fiber/v3"
	inertia "github.com/inertia-go/inertia-go"
)

func Index(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		n := application.PageNumberFromQuery(c.Query("page"))
		meta := inertia.ScrollConfig{PageName: "page", CurrentPage: n}
		if n > 1 {
			previous := n - 1
			meta.PreviousPage = &previous
		}
		if n < 3 {
			next := n + 1
			meta.NextPage = &next
		}
		// Native Scroll has a data wrapper, not a complete Laravel paginator.
		return app.RenderPage(c, "ScrollPage", inertia.Props{"items": inertia.Scroll(meta, func() any { app.Count("items"); return application.Rows(n) })})
	}
}
