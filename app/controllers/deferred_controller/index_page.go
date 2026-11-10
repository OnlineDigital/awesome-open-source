package deferred_controller

import (
	"github.com/andreitelteu/inertia-go-solid-template/app/application"
	"github.com/gofiber/fiber/v3"
	inertia "github.com/inertia-go/inertia-go"

	"time"
)

func Index(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		return app.RenderPage(c, "DeferredPage", inertia.Props{
			"slow":   inertia.Defer(func() (any, error) { app.Count("slow"); time.Sleep(150 * time.Millisecond); return "slow-value", nil }),
			"second": inertia.Defer(app.Callback("second", "second-value"), "extra"),
		})
	}
}
