package navigation_controller

import (
	"github.com/andreitelteu/inertia-go-solid-template/app/application"
	"github.com/gofiber/fiber/v3"
	"net/http"
)

func Redirect(app *application.Application) fiber.Handler {
	return app.Handle(func(w http.ResponseWriter, r *http.Request) {
		app.Adapter.Redirect(w, r, "/other")
	})
}
