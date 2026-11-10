package application

import (
	"context"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	upstream "github.com/inertia-go/inertia-go"
	"net/http"
	"net/url"
)

type paramsKey struct{}

// Handle adapts only the inertia-go protocol boundary. The server, router,
// route middleware and public controller handlers are native Fiber. Inertia-go
// v0.10.0 exposes net/http Render, so its request middleware must run inside this
// boundary before each page/controller action. No separate net/http server runs.
func (a *Application) Handle(action http.HandlerFunc) fiber.Handler {
	pipeline := a.SessionContext(a.Adapter.Middleware(a.Record(action)))
	return a.httpBoundary(pipeline)
}

// HandleHTTP is for demo responses that are intentionally not Inertia pages.
// It keeps request accounting without consuming session validation or flash.
func (a *Application) HandleHTTP(action http.HandlerFunc) fiber.Handler {
	return a.httpBoundary(a.Record(action))
}

func (a *Application) httpBoundary(handler http.Handler) fiber.Handler {
	converted := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// c.Context() supports values installed by native Fiber middleware. It is
		// not a net/http disconnect context; Fiber/fasthttp cancellation differs.
		if ctx, ok := adaptor.LocalContextFromHTTPRequest(r); ok {
			r = r.WithContext(ctx)
		}
		if params, ok := r.Context().Value(paramsKey{}).(map[string]string); ok {
			for key, value := range params {
				r.SetPathValue(key, value)
			}
		}
		handler.ServeHTTP(w, r)
	})
	adapted := adaptor.HTTPHandlerWithContext(converted)
	return func(c fiber.Ctx) error {
		params := map[string]string{}
		for _, key := range c.Route().Params {
			value, err := url.PathUnescape(c.Params(key))
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "Invalid route parameter")
			}
			params[key] = value
		}
		c.SetContext(context.WithValue(c.Context(), paramsKey{}, params))
		return adapted(c)
	}
}

func (a *Application) RecordFiber(c fiber.Ctx) {
	if !a.Demo {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.stats.Requests = append(a.stats.Requests, RequestRecord{c.OriginalURL(), c.Method(), c.Get("Purpose"), c.Get("X-Inertia-Partial-Data"), c.Get("X-Inertia-Partial-Except")})
	if len(a.stats.Requests) > 1000 {
		a.stats.Requests = a.stats.Requests[len(a.stats.Requests)-1000:]
	}
}

// RenderPage is the ergonomic entry for native Fiber controller actions.
func (a *Application) RenderPage(c fiber.Ctx, component string, props upstream.Props) error {
	return a.RenderPageWith(a.Adapter, c, component, props)
}
func (a *Application) RenderPageWith(adapter *upstream.Inertia, c fiber.Ctx, component string, props upstream.Props) error {
	return a.Handle(func(w http.ResponseWriter, r *http.Request) { a.RenderWith(adapter, w, r, component, props) })(c)
}

func RouteParameter(c fiber.Ctx, key string) string {
	value, err := url.PathUnescape(c.Params(key))
	if err != nil {
		return ""
	}
	return value
}
