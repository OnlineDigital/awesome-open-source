package form_controller

import (
	"github.com/andreitelteu/inertia-go-solid-template/app/application"
	"github.com/gofiber/fiber/v3"
	inertia "github.com/inertia-go/inertia-go"
)

func Index(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		props := inertia.Props{}
		// Query fixtures are demo-only. Normal submissions use encrypted flash cookies.
		if app.Demo && c.Query("invalid") == "1" {
			props["errors"] = inertia.Always(map[string]any{"name": "Name is required"})
		}
		if app.Demo && c.Query("bag") == "1" {
			props["errors"] = inertia.Always(map[string]any{"profile": map[string]any{"name": "Name is required"}})
		}
		return app.RenderPage(c, "FormPage", props)
	}
}
