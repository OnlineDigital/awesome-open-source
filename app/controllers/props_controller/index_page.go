package props_controller

import (
	"github.com/andreitelteu/inertia-go-solid-template/app/application"
	bridge "github.com/andreitelteu/inertia-go-solid-template/app/inertia"
	"github.com/gofiber/fiber/v3"
	inertia "github.com/inertia-go/inertia-go"
)

func Index(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		return app.RenderPage(c, "Props", inertia.Props{
			"users":     bridge.Lazy(app.Callback("users", []map[string]any{{"id": 1, "name": "Ada"}})),
			"companies": bridge.Lazy(app.Callback("companies", []string{"Acme"})),
			"optional":  inertia.Optional(app.Callback("optional", "optional-value")),
			"always":    inertia.Always("always"),
			"auth":      map[string]any{"notifications": bridge.Lazy(app.Callback("auth.notifications", []string{"hello"}))},
		})
	}
}
