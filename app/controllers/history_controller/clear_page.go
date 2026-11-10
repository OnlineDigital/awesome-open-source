package history_controller

import (
	"github.com/andreitelteu/inertia-go-solid-template/app/application"
	"github.com/gofiber/fiber/v3"
	inertia "github.com/inertia-go/inertia-go"
)

func Clear(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		return app.RenderPageWith(app.Cleared, c, "HistoryPage", inertia.Props{})
	}
}
