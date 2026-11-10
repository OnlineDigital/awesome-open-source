package form_controller

import (
	"github.com/andreitelteu/inertia-go-solid-template/app/application"
	"github.com/gofiber/fiber/v3"
	"net/http"
	"strings"
)

func Store(app *application.Application) fiber.Handler {
	return app.Handle(func(w http.ResponseWriter, r *http.Request) {
		name, err := application.ReadName(w, r)
		if err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}
		errors := map[string]string{}
		if strings.TrimSpace(name) == "" {
			errors["name"] = "Name is required"
		}
		app.RedirectForm(w, r, errors)
	})
}
