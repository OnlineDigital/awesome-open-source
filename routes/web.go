// Package routes is the central route table. Controllers are packages/folders;
// each exported action lives in its own snake_case source file.
package routes

import (
	"github.com/andreitelteu/inertia-go-solid-template/app/application"
	"github.com/andreitelteu/inertia-go-solid-template/app/controllers/deferred_controller"
	"github.com/andreitelteu/inertia-go-solid-template/app/controllers/diagnostic_controller"
	"github.com/andreitelteu/inertia-go-solid-template/app/controllers/form_controller"
	"github.com/andreitelteu/inertia-go-solid-template/app/controllers/health_controller"
	"github.com/andreitelteu/inertia-go-solid-template/app/controllers/history_controller"
	"github.com/andreitelteu/inertia-go-solid-template/app/controllers/home_controller"
	"github.com/andreitelteu/inertia-go-solid-template/app/controllers/http_controller"
	"github.com/andreitelteu/inertia-go-solid-template/app/controllers/merge_controller"
	"github.com/andreitelteu/inertia-go-solid-template/app/controllers/navigation_controller"
	"github.com/andreitelteu/inertia-go-solid-template/app/controllers/once_controller"
	"github.com/andreitelteu/inertia-go-solid-template/app/controllers/poll_controller"
	"github.com/andreitelteu/inertia-go-solid-template/app/controllers/props_controller"
	"github.com/andreitelteu/inertia-go-solid-template/app/controllers/scroll_controller"
	"github.com/andreitelteu/inertia-go-solid-template/app/controllers/users_controller"
	"github.com/andreitelteu/inertia-go-solid-template/app/controllers/visible_controller"
	"github.com/andreitelteu/inertia-go-solid-template/internal/router"
	"github.com/gofiber/fiber/v3"
)

func Web(app *application.Application, fiberApp *fiber.App) *router.Router {
	r := router.New(fiberApp)
	app.RouteURL = r.URL
	app.RouteList = func() any { return r.Routes() }
	r.Get("/", home_controller.Index(app)).Name("home")
	r.Get("/home", home_controller.Index(app)).Name("home.index")
	r.Get("/other", home_controller.Other(app)).Name("home.other")
	r.Get("/health", health_controller.Index(app)).Name("health")
	r.Get("/props", props_controller.Index(app)).Name("props.index")
	r.Get("/deferred", deferred_controller.Index(app)).Name("deferred.index")
	r.Get("/once", once_controller.Index(app)).Name("once.index")
	r.Get("/once-other", once_controller.Index(app)).Name("once.other")
	r.Get("/merge", merge_controller.Index(app)).Name("merge.index")
	r.Get("/deep", merge_controller.Deep(app)).Name("merge.deep")
	r.Get("/scroll", scroll_controller.Index(app)).Name("scroll.index")
	r.Get("/visible", visible_controller.Index(app)).Name("visible.index")
	r.Get("/poll", poll_controller.Index(app)).Name("poll.index")
	r.Get("/form", form_controller.Index(app)).Name("form.index")
	r.Post("/submit", form_controller.Store(app)).Name("form.store")
	r.Put("/submit", form_controller.Store(app)).Name("form.update")
	r.Patch("/submit", form_controller.Store(app)).Name("form.patch")
	r.Delete("/submit", form_controller.Store(app)).Name("form.destroy")
	r.Post("/precognition", form_controller.Validate(app)).Name("form.validate")
	r.Get("/http", http_controller.Index(app)).Name("http.index")
	r.Get("/history", history_controller.Index(app)).Name("history.index")
	r.Get("/clear-history", history_controller.Clear(app)).Name("history.clear")
	r.Get("/location", navigation_controller.Location(app)).Name("navigation.location")
	r.Get("/redirect", navigation_controller.Redirect(app)).Name("navigation.redirect")
	// A Laravel-style group scopes both URL and name prefixes. Native Fiber
	// matches :user; application.RouteParameter(c, "user") decodes one segment.
	r.Group("/users", "users.", nil, func(users *router.Router) {
		users.Get("/:user", users_controller.Show(app)).Name("show")
	})
	if app.Demo {
		r.Get("/failure", diagnostic_controller.Failure(app)).Name("diagnostics.failure")
		r.Get("/malformed", diagnostic_controller.Malformed(app)).Name("diagnostics.malformed")
		r.Post("/reset", diagnostic_controller.Reset(app)).Name("diagnostics.reset")
		r.Get("/stats", diagnostic_controller.Stats(app)).Name("diagnostics.stats")
		r.Get("/routes", diagnostic_controller.Routes(app)).Name("diagnostics.routes")
	}
	return r
}
