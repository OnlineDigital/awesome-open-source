package once_controller

import (
	"github.com/andreitelteu/inertia-go-solid-template/app/application"
	"github.com/gofiber/fiber/v3"
	inertia "github.com/inertia-go/inertia-go"

	"strings"
)

func Index(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		lookup := inertia.Once(app.Callback("lookup", []string{"US", "RO"})).As("lookup")
		if c.Query("fresh") == "1" {
			lookup.Fresh()
		}
		return app.RenderPage(c, "OncePage", inertia.Props{"lookup": lookup, "label": strings.TrimPrefix(c.Path(), "/")})
	}
}
