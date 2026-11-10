package application

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
)

type paramsKey struct{}

// Handle adapts the inertia-go protocol boundary. The server, router and
// controllers are native Fiber; inertia-go v0.10 exposes net/http, so its
// middleware runs inside this boundary before each page action.
func (a *Application) Handle(action http.HandlerFunc) fiber.Handler {
	return a.httpBoundary(a.SessionContext(a.Adapter.Middleware(action)))
}

func (a *Application) httpBoundary(handler http.Handler) fiber.Handler {
	converted := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

// RouteParameter returns one URL-decoded path segment.
func RouteParameter(c fiber.Ctx, key string) string {
	value, err := url.PathUnescape(c.Params(key))
	if err != nil {
		return ""
	}
	return value
}
