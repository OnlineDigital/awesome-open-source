package health_controller

import (
	"context"
	"time"

	"awesomeopensource.dev/app/application"
	"github.com/gofiber/fiber/v3"
)

func Index(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()
		status := fiber.Map{"status": "ok", "database": "ok"}
		if err := app.Pool.Ping(ctx); err != nil {
			status["status"], status["database"] = "degraded", "unreachable"
			return c.Status(fiber.StatusServiceUnavailable).JSON(status)
		}
		return c.JSON(status)
	}
}
