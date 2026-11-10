package health_controller

import (
	"github.com/andreitelteu/inertia-go-solid-template/app/application"
	"github.com/gofiber/fiber/v3"
)

func Index(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error { return c.JSON(fiber.Map{"ok": true, "adapter": "inertia-go"}) }
}
