package visible_controller

import (
	"github.com/andreitelteu/inertia-go-solid-template/app/application"
	"github.com/gofiber/fiber/v3"
	inertia "github.com/inertia-go/inertia-go"
)

func Index(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		return app.RenderPage(c, "VisiblePage", inertia.Props{"optional": inertia.Optional(app.Callback("optional", "optional-value"))})
	}
}
