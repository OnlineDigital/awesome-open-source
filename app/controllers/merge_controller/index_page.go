package merge_controller

import (
	"github.com/andreitelteu/inertia-go-solid-template/app/application"
	"github.com/gofiber/fiber/v3"
	inertia "github.com/inertia-go/inertia-go"
)

func Index(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		return app.RenderPage(c, "MergePage", inertia.Props{"items": inertia.Merge(app.Callback("items", application.Rows(application.PageNumberFromQuery(c.Query("page"))))).MatchOn(map[string]string{"": "id"})})
	}
}
