package poll_controller

import (
	"github.com/andreitelteu/inertia-go-solid-template/app/application"
	bridge "github.com/andreitelteu/inertia-go-solid-template/app/inertia"
	"github.com/gofiber/fiber/v3"
	inertia "github.com/inertia-go/inertia-go"
)

func Index(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		return app.RenderPage(c, "PollPage", inertia.Props{"tick": bridge.Lazy(func() (any, error) { return app.Count("tick"), nil })})
	}
}
