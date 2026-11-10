package users_controller

import (
	"github.com/andreitelteu/inertia-go-solid-template/app/application"
	"github.com/gofiber/fiber/v3"
	inertia "github.com/inertia-go/inertia-go"
)

func Show(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		return app.RenderPage(c, "Users/Show", inertia.Props{"user": map[string]any{"id": application.RouteParameter(c, "user"), "name": "Route parameter example"}})
	}
}
