package diagnostic_controller

import (
	"fmt"
	"github.com/andreitelteu/inertia-go-solid-template/app/application"
	"github.com/gofiber/fiber/v3"
	"net/http"
	"strconv"
)

func Failure(app *application.Application) fiber.Handler {
	return app.HandleHTTP(func(w http.ResponseWriter, r *http.Request) {
		status, _ := strconv.Atoi(r.URL.Query().Get("status"))
		if status != 403 && status != 404 && status != 500 {
			status = 500
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		fmt.Fprintf(w, "<!doctype html><title>Failure</title><h1>%d %s</h1>", status, http.StatusText(status))
	})
}
