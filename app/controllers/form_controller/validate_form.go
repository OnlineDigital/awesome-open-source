package form_controller

import (
	"github.com/andreitelteu/inertia-go-solid-template/app/application"
	"github.com/gofiber/fiber/v3"
	inertia "github.com/inertia-go/inertia-go"
	"net/http"
	"strings"
)

func Validate(app *application.Application) fiber.Handler {
	return app.Handle(func(w http.ResponseWriter, r *http.Request) {
		name, err := application.ReadName(w, r)
		if err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}
		if app.Adapter.HandlePrecognition(w, r, func(r *http.Request) {
			if strings.TrimSpace(name) == "" {
				inertia.ValidationErrors(r).Add("name", "Name is required")
			}
		}) {
			return
		}
		application.WriteJSON(w, map[string]any{"ok": true})
	})
}
