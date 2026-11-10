package application

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	upstream "github.com/inertia-go/inertia-go"
	"github.com/inertia-go/inertia-go/session"
)

func TestFiberBoundaryPreservesMiddlewareContextAndDecodedParameters(t *testing.T) {
	i, err := upstream.New(upstream.Config{Session: session.NewNoop(), Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	a := &Application{Adapter: i}
	app := fiber.New(fiber.Config{Immutable: true, UnescapePath: false})
	type userKey struct{}
	app.Use(func(c fiber.Ctx) error {
		c.SetContext(context.WithValue(c.Context(), userKey{}, "middleware-user"))
		return c.Next()
	})
	app.Get("/probe/:user", a.Handle(func(w http.ResponseWriter, r *http.Request) {
		info := upstream.FromRequest(r)
		WriteJSON(w, map[string]any{"user": r.PathValue("user"), "context": r.Context().Value(userKey{}), "inertia": info.IsInertia, "only": info.PartialData})
	}))
	req := httptest.NewRequest("GET", "/probe/a%2Fb%20%2B%3F%23", nil)
	req.Header.Set("X-Inertia", "true")
	req.Header.Set("X-Inertia-Partial-Data", "users,auth.notifications")
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var values struct {
		User    string
		Context string
		Inertia bool
		Only    []string
	}
	if err := json.NewDecoder(response.Body).Decode(&values); err != nil {
		t.Fatal(err)
	}
	if values.User != "a/b +?#" || values.Context != "middleware-user" || !values.Inertia || len(values.Only) != 2 || values.Only[1] != "auth.notifications" {
		t.Fatalf("lost Fiber context/parameters or native request middleware: %+v", values)
	}
}
