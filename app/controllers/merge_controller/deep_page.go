package merge_controller

import (
	"github.com/andreitelteu/inertia-go-solid-template/app/application"
	"github.com/gofiber/fiber/v3"
	inertia "github.com/inertia-go/inertia-go"
)

func Deep(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		return app.RenderPage(c, "DeepPage", inertia.Props{"tree": inertia.DeepMerge(app.Callback("items", map[string]any{"items": application.Rows(application.PageNumberFromQuery(c.Query("page")))}))})
	}
}
