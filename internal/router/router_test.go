package router

import (
	"github.com/gofiber/fiber/v3"
	"io"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func TestNestedGroupsAndEncodedParameters(t *testing.T) {
	r := New()
	var order []string
	middleware := func(label string) Middleware {
		return func(c fiber.Ctx) error {
			order = append(order, label+":before")
			err := c.Next()
			order = append(order, label+":after")
			return err
		}
	}
	r.Group("/admin", "admin.", []Middleware{middleware("outer")}, func(admin *Router) {
		admin.Group("/users", "users.", []Middleware{middleware("inner")}, func(users *Router) {
			users.Get("/{user}", func(c fiber.Ctx) error {
				value, err := url.PathUnescape(c.Params("user"))
				if err != nil {
					return err
				}
				return c.SendString(value)
			}).Name("show")
		})
	})
	target, err := r.URL("admin.users.show", map[string]string{"user": "a/b +?#"}, url.Values{"q": []string{"a b", "c&d"}})
	if err != nil {
		t.Fatal(err)
	}
	if target != "/admin/users/a%2Fb%20+%3F%23?q=a+b&q=c%26d" {
		t.Fatalf("URL: %q", target)
	}
	response, err := r.App.Test(httptest.NewRequest("GET", target, nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != 200 || string(body) != "a/b +?#" {
		t.Fatalf("parameter round trip: %d %s", response.StatusCode, body)
	}
	want := []string{"outer:before", "inner:before", "inner:after", "outer:after"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("middleware order: %v", order)
	}
	if native := r.App.GetRoute("admin.users.show"); native.Path != "/admin/users/:user" {
		t.Fatalf("named route missing from native Fiber registry: %+v", native)
	}
}

func TestMethodHandlingAndExactRoot(t *testing.T) {
	r := New()
	r.Get("/", func(c fiber.Ctx) error { return c.SendStatus(204) }).Name("home")
	r.Post("/users", func(c fiber.Ctx) error { return c.SendStatus(201) }).Name("users.store")
	for _, tt := range []struct {
		method, path string
		status       int
	}{{"GET", "/", 204}, {"HEAD", "/", 204}, {"GET", "/missing", 404}, {"POST", "/", 405}, {"POST", "/users", 201}, {"DELETE", "/users", 405}, {"GET", "/users", 405}} {
		t.Run(tt.method+tt.path, func(t *testing.T) {
			response, err := r.App.Test(httptest.NewRequest(tt.method, tt.path, nil))
			if err != nil {
				t.Fatal(err)
			}
			response.Body.Close()
			if response.StatusCode != tt.status {
				t.Fatalf("status %d want %d", response.StatusCode, tt.status)
			}
		})
	}
}
func TestNamedRouteValidation(t *testing.T) {
	r := New()
	handler := func(c fiber.Ctx) error { return nil }
	r.Get("/users/{user}", handler).Name("users.show")
	for _, tt := range []struct {
		name   string
		params map[string]string
	}{{"missing", nil}, {"users.show", nil}, {"users.show", map[string]string{"user": ""}}, {"users.show", map[string]string{"user": "42", "typo": "x"}}, {"users.show", map[string]string{"user": ".."}}} {
		if _, err := r.URL(tt.name, tt.params, nil); err == nil {
			t.Fatalf("expected invalid route arguments to fail: %+v", tt)
		}
	}
	for _, tt := range []struct {
		name     string
		register func()
	}{{"duplicate pattern", func() { r.Get("/users/:user", handler) }}, {"renamed duplicate parameter", func() { r.Get("/users/:id", handler) }}, {"duplicate name", func() { r.Get("/other", handler).Name("users.show") }}} {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected declaration panic")
				}
			}()
			tt.register()
		})
	}
}
func TestRegistrySnapshotCannotMutateRoutes(t *testing.T) {
	r := New()
	r.Get("/users/:user", func(c fiber.Ctx) error { return nil }).Name("users.show")
	registry := r.Routes()
	registry[0].Path = "/tampered"
	actual, err := r.URL("users.show", map[string]string{"user": "42"}, nil)
	if err != nil || actual != "/users/42" {
		t.Fatalf("registry mutated internal routes: %q %v", actual, err)
	}
}
