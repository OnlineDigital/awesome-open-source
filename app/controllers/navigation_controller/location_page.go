package navigation_controller

import (
	"github.com/andreitelteu/inertia-go-solid-template/app/application"
	"github.com/gofiber/fiber/v3"
	"net/http"
)

func Location(app *application.Application) fiber.Handler {
	return app.Handle(func(w http.ResponseWriter, r *http.Request) {
		app.Adapter.Location(w, r, "/other")
	})
}
