// Package router adds Laravel-style groups and named URLs to the native Fiber router.
package router

import (
	"fmt"
	"github.com/gofiber/fiber/v3"
	"net/url"
	"sort"
	"strings"
)

type Middleware = fiber.Handler

type Route struct {
	Method       string `json:"method"`
	Path         string `json:"path"`
	RouteName    string `json:"name,omitempty"`
	root         *Router
	namePrefix   string
	named        bool
	nativeRoutes []*fiber.Route
}

type Router struct {
	App        *fiber.App
	root       *Router
	prefix     string
	namePrefix string
	middleware []fiber.Handler
	routes     []*Route
	named      map[string]*Route
	registered map[string]bool
}

// New wraps an existing Fiber app; without one it creates a small test app.
func New(app ...*fiber.App) *Router {
	var f *fiber.App
	if len(app) > 0 {
		f = app[0]
	} else {
		f = fiber.New(fiber.Config{Immutable: true, StrictRouting: true, CaseSensitive: true})
	}
	r := &Router{App: f, named: map[string]*Route{}, registered: map[string]bool{}}
	r.root = r
	return r
}

// Group scopes URL and route-name prefixes and ordered native Fiber middleware.
// Middleware must call c.Next() to continue to the next middleware or controller.
func (r *Router) Group(prefix, namePrefix string, middleware []Middleware, define func(*Router)) {
	group := &Router{App: r.App, root: r.root, prefix: joinPath(r.prefix, prefix), namePrefix: r.namePrefix + namePrefix}
	group.middleware = append(append([]fiber.Handler{}, r.middleware...), middleware...)
	define(group)
}
func joinPath(prefix, path string) string {
	joined := strings.TrimRight(prefix, "/") + "/" + strings.TrimLeft(path, "/")
	if joined != "/" {
		joined = strings.TrimRight(joined, "/")
	}
	return joined
}

// Normalize Laravel {parameter} spelling to native Fiber :parameter spelling.
// Fiber syntax is also accepted, so middleware and routing remain native.
func fiberPath(path string) string {
	segments := strings.Split(path, "/")
	for n, segment := range segments {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			param := strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}")
			if param == "" || strings.ContainsAny(param, ".*?{}") {
				panic("router: only required named segment parameters are supported")
			}
			segments[n] = ":" + param
		}
	}
	return strings.Join(segments, "/")
}

func routeShape(path string) string {
	segments := strings.Split(path, "/")
	for n, segment := range segments {
		if strings.HasPrefix(segment, ":") && !strings.ContainsAny(segment, "?*+<>") {
			segments[n] = ":parameter"
		}
	}
	return strings.Join(segments, "/")
}

func (r *Router) Handle(method, path string, handler fiber.Handler) *Route {
	method = strings.ToUpper(method)
	if handler == nil {
		panic("router: nil handler")
	}
	path = fiberPath(joinPath(r.prefix, path))
	key := method + " " + routeShape(path)
	if r.root.registered[key] {
		panic("router: duplicate route " + key)
	}
	handlers := make([]any, 0, len(r.middleware)+1)
	for _, middleware := range r.middleware {
		handlers = append(handlers, middleware)
	}
	handlers = append(handlers, handler)
	// App.Add/Get register automatic HEAD for GET. Routing and method handling are Fiber's.
	r.App.Add([]string{method}, path, handlers[0], handlers[1:]...)
	r.root.registered[key] = true
	route := &Route{Method: method, Path: path, root: r.root, namePrefix: r.namePrefix}
	// Bind stable native route pointers at declaration time. This also names
	// Fiber's automatic HEAD route and supports Name called after registration.
	for _, stack := range r.App.Stack() {
		for _, native := range stack {
			if native.Path == path && (native.Method == method || method == fiber.MethodGet && native.Method == fiber.MethodHead) {
				route.nativeRoutes = append(route.nativeRoutes, native)
			}
		}
	}
	r.root.routes = append(r.root.routes, route)
	return route
}
func (r *Router) Get(path string, h fiber.Handler) *Route { return r.Handle(fiber.MethodGet, path, h) }
func (r *Router) Post(path string, h fiber.Handler) *Route {
	return r.Handle(fiber.MethodPost, path, h)
}
func (r *Router) Put(path string, h fiber.Handler) *Route { return r.Handle(fiber.MethodPut, path, h) }
func (r *Router) Patch(path string, h fiber.Handler) *Route {
	return r.Handle(fiber.MethodPatch, path, h)
}
func (r *Router) Delete(path string, h fiber.Handler) *Route {
	return r.Handle(fiber.MethodDelete, path, h)
}

func (route *Route) Name(name string) *Route {
	if name == "" {
		panic("router: empty route name")
	}
	if route.named {
		panic("router: route already named")
	}
	fullName := route.namePrefix + name
	if _, exists := route.root.named[fullName]; exists {
		panic("router: duplicate route name " + fullName)
	}
	route.RouteName = fullName
	route.named = true
	for _, native := range route.nativeRoutes {
		native.Name = fullName
	}
	route.root.named[fullName] = route
	return route
}

// URL reverses a named route with individually escaped segment parameters and
// encoded query values. It rejects typos (unknown, missing and extra parameters).
func (r *Router) URL(name string, params map[string]string, query url.Values) (string, error) {
	route, ok := r.root.named[name]
	if !ok {
		return "", fmt.Errorf("router: unknown route %q", name)
	}
	segments := strings.Split(route.Path, "/")
	used := map[string]bool{}
	for n, segment := range segments {
		if !strings.HasPrefix(segment, ":") {
			continue
		}
		key := strings.TrimPrefix(segment, ":")
		if strings.ContainsAny(key, "?*+<>") {
			return "", fmt.Errorf("router: only required parameters can be reversed: %s", name)
		}
		value, exists := params[key]
		if !exists || value == "" {
			return "", fmt.Errorf("router: missing parameter %q for %s", key, name)
		}
		if value == "." || value == ".." {
			return "", fmt.Errorf("router: invalid dot segment %q", key)
		}
		segments[n] = url.PathEscape(value)
		used[key] = true
	}
	for key := range params {
		if !used[key] {
			return "", fmt.Errorf("router: unexpected parameter %q for %s", key, name)
		}
	}
	result := strings.Join(segments, "/")
	if q := query.Encode(); q != "" {
		result += "?" + q
	}
	return result, nil
}
func (r *Router) Routes() []Route {
	result := make([]Route, 0, len(r.root.routes))
	for _, route := range r.root.routes {
		result = append(result, Route{Method: route.Method, Path: route.Path, RouteName: route.RouteName})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Path == result[j].Path {
			return result[i].Method < result[j].Method
		}
		return result[i].Path < result[j].Path
	})
	return result
}
