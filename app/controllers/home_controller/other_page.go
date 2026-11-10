package home_controller

import (
	"github.com/andreitelteu/inertia-go-solid-template/app/application"
	"github.com/gofiber/fiber/v3"
	inertia "github.com/inertia-go/inertia-go"

	"strconv"
	"time"
)

func Other(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		ms, _ := strconv.Atoi(c.Query("delay"))
		if ms > 1500 {
			ms = 1500
		}
		if ms > 0 {
			timer := time.NewTimer(time.Duration(ms) * time.Millisecond)
			defer timer.Stop()
			select {
			case <-timer.C:
			case <-c.Context().Done():
				return nil
			}
		}
		return app.RenderPage(c, "Other", inertia.Props{"label": "other"})
	}
}
