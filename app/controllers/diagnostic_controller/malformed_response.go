package diagnostic_controller

import (
	"github.com/andreitelteu/inertia-go-solid-template/app/application"
	"github.com/gofiber/fiber/v3"
)

func Malformed(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error { app.RecordFiber(c); return c.JSON(fiber.Map{"not": "inertia"}) }
}
