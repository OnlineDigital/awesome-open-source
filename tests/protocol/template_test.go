package protocol_test

import (
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/andreitelteu/inertia-go-solid-template/app/server"
)

func templateAssets() fstest.MapFS {
	// Match embedded production assets without leaving cached OS file handles
	// open while testing.TempDir removes fixtures on Windows.
	return fstest.MapFS{
		".vite/manifest.json": {Data: []byte(`{"resources/js/app.tsx":{"file":"assets/app-test.js","css":["assets/app-test.css"]}}`)},
		"assets/app-test.js":  {Data: []byte("console.log('template fixture')")},
		"assets/app-test.css": {Data: []byte("body{margin:0}")},
	}
}

func templateHandler(t *testing.T) http.Handler {
	t.Helper()
	root := t.TempDir()
	h, err := server.NewWithOptions(root, server.Options{
		Demo: true, Environment: "demo",
		AssetFS: templateAssets(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return server.HTTPHandler(h)
}

func request(t *testing.T, h http.Handler, method, path, body string, headers map[string]string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	for key, value := range headers {
		r.Header.Set(key, value)
	}
	for _, cookie := range cookies {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func page(t *testing.T, h http.Handler, path, component, only, except string) map[string]any {
	t.Helper()
	headers := map[string]string{"X-Inertia": "true"}
	if component != "" {
		headers["X-Inertia-Partial-Component"] = component
	}
	if only != "" {
		headers["X-Inertia-Partial-Data"] = only
	}
	if except != "" {
		headers["X-Inertia-Partial-Except"] = except
	}
	w := request(t, h, "GET", path, "", headers)
	if w.Code != 200 || w.Header().Get("X-Inertia") != "true" {
		t.Fatalf("GET %s: status=%d header=%q body=%s", path, w.Code, w.Header().Get("X-Inertia"), w.Body.String())
	}
	var p map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func evaluations(t *testing.T, h http.Handler) map[string]int {
	t.Helper()
	w := request(t, h, "GET", "/stats", "", nil)
	var data struct {
		Evaluations map[string]int `json:"evaluations"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	return data.Evaluations
}

func requireEqual(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v; want %#v", got, want)
	}
}

func TestTemplateHTMLAssetsAndVersion(t *testing.T) {
	h := templateHandler(t)
	w := request(t, h, "GET", "/home", "", nil)
	if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("invalid bootstrap: %d %s", w.Code, w.Body.String())
	}
	for _, marker := range []string{`id="app"`, `name="viewport"`, `/build/assets/app-test.js`, `/build/assets/app-test.css`} {
		if !strings.Contains(w.Body.String(), marker) {
			t.Errorf("bootstrap missing %s", marker)
		}
	}
	// The template must embed parseable initial JSON, independent of the current
	// upstream format (v3 JSON script or escaped legacy data-page attribute).
	var bootstrap map[string]any
	if match := regexp.MustCompile(`(?s)<script[^>]*data-page=["']app["'][^>]*>(.*?)</script>`).FindStringSubmatch(w.Body.String()); len(match) == 2 {
		if err := json.Unmarshal([]byte(match[1]), &bootstrap); err != nil {
			t.Fatal(err)
		}
	} else if match := regexp.MustCompile(`data-page="([^"]*)"`).FindStringSubmatch(w.Body.String()); len(match) == 2 {
		if err := json.Unmarshal([]byte(html.UnescapeString(match[1])), &bootstrap); err != nil {
			t.Fatal(err)
		}
	} else {
		t.Fatal("missing initial page JSON")
	}
	requireEqual(t, bootstrap["component"], "Home")
	asset := request(t, h, "GET", "/build/assets/app-test.js", "", nil)
	requireEqual(t, asset.Code, 200)
	requireEqual(t, asset.Body.String(), "console.log('template fixture')")
	current := page(t, h, "/home", "", "", "")["version"].(string)
	if len(current) != 64 {
		t.Fatalf("version must be manifest SHA256, got %q", current)
	}
	w = request(t, h, "GET", "/other?x=1", "", map[string]string{"X-Inertia": "true", "X-Inertia-Version": "outdated"})
	requireEqual(t, w.Code, 409)
	requireEqual(t, w.Header().Get("X-Inertia-Location"), "/other?x=1")
}

func TestTemplateSelectiveLazyEvaluation(t *testing.T) {
	for _, test := range []struct {
		name, component, only, except string
		users, companies, nested      int
	}{
		{"only users", "Props", "users", "", 1, 0, 0},
		{"except companies", "Props", "", "companies", 1, 0, 1},
		{"nested only", "Props", "auth.notifications", "", 0, 0, 1},
		{"nested except", "Props", "", "auth.notifications", 1, 1, 0},
		{"mismatched component", "Other", "users", "companies", 1, 1, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := templateHandler(t)
			p := page(t, h, "/props", test.component, test.only, test.except)
			counts := evaluations(t, h)
			requireEqual(t, counts["users"], test.users)
			requireEqual(t, counts["companies"], test.companies)
			requireEqual(t, counts["auth.notifications"], test.nested)
			requireEqual(t, nativeProp(p, "props.always"), "always")
			if nativeProp(p, "props.shared.app") == nil || nativeProp(p, "props.errors") == nil {
				t.Fatalf("partial lost always shared/errors: %v", p)
			}
			if p["mergeProps"] != nil {
				t.Fatalf("ordinary Lazy must not set merge metadata: %v", p)
			}
		})
	}
}

func TestTemplateOptionalDeferredAndOnce(t *testing.T) {
	h := templateHandler(t)
	p := page(t, h, "/props", "", "", "")
	requireEqual(t, nativeProp(p, "props.optional"), nil)
	requireEqual(t, evaluations(t, h)["optional"], 0)
	p = page(t, h, "/props", "Props", "optional", "")
	requireEqual(t, nativeProp(p, "props.optional"), "optional-value")
	p = page(t, h, "/deferred", "", "", "")
	requireEqual(t, nativeProp(p, "props.slow"), nil)
	requireEqual(t, p["deferredProps"], map[string]any{"default": []any{"slow"}, "extra": []any{"second"}})
	p = page(t, h, "/deferred", "DeferredPage", "second", "")
	requireEqual(t, nativeProp(p, "props.second"), "second-value")
	requireEqual(t, evaluations(t, h)["slow"], 0)
	p = page(t, h, "/once", "", "", "")
	requireEqual(t, nativeProp(p, "onceProps.lookup.prop"), "lookup")
	w := request(t, h, "GET", "/once-other", "", map[string]string{"X-Inertia": "true", "X-Inertia-Except-Once-Props": "lookup"})
	requireEqual(t, w.Code, 200)
	requireEqual(t, evaluations(t, h)["lookup"], 1)
	p = page(t, h, "/once", "OncePage", "lookup", "")
	requireEqual(t, nativeProp(p, "props.lookup"), []any{"US", "RO"})
	requireEqual(t, evaluations(t, h)["lookup"], 2)
}

func TestTemplateMergeScrollAndHistoryMetadata(t *testing.T) {
	h := templateHandler(t)
	p := page(t, h, "/merge?page=2", "MergePage", "items", "")
	requireEqual(t, p["mergeProps"], []any{"items"})
	requireEqual(t, p["matchPropsOn"], []any{"items.id"})
	p = page(t, h, "/deep?page=2", "DeepPage", "tree", "")
	requireEqual(t, p["deepMergeProps"], []any{"tree"})
	p = page(t, h, "/scroll?page=2", "ScrollPage", "items", "")
	requireEqual(t, nativeProp(p, "scrollProps.items.currentPage"), float64(2))
	requireEqual(t, nativeProp(p, "scrollProps.items.nextPage"), float64(3))
	p = page(t, h, "/scroll?page=3", "ScrollPage", "items", "")
	requireEqual(t, nativeProp(p, "scrollProps.items.nextPage"), nil)
	requireEqual(t, page(t, h, "/history", "", "", "")["encryptHistory"], true)
	requireEqual(t, page(t, h, "/clear-history", "", "", "")["clearHistory"], true)
}

func TestTemplateMethodsAndRedirects(t *testing.T) {
	h := templateHandler(t)
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			w := request(t, h, method, "/submit", `{"name":"Ada"}`, map[string]string{"Content-Type": "application/json", "X-Inertia": "true"})
			requireEqual(t, w.Code, 303)
			if !strings.HasPrefix(w.Header().Get("Location"), "/form") {
				t.Fatalf("invalid redirect %q", w.Header().Get("Location"))
			}
		})
	}
	requireEqual(t, request(t, h, "GET", "/submit", "", nil).Code, 405)
	requireEqual(t, request(t, h, "POST", "/other", "", nil).Code, 405)
	requireEqual(t, request(t, h, "GET", "/does-not-exist", "", nil).Code, 404)
	w := request(t, h, "GET", "/location", "", map[string]string{"X-Inertia": "true"})
	requireEqual(t, w.Code, 409)
	requireEqual(t, w.Header().Get("X-Inertia-Location"), "/other")
}

func TestTemplateDirectJSONAndValidation(t *testing.T) {
	h := templateHandler(t)
	w := request(t, h, "GET", "/http", "", nil)
	requireEqual(t, w.Code, 200)
	requireEqual(t, w.Header().Get("X-Inertia"), "")
	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	requireEqual(t, response, map[string]any{"ok": true, "value": float64(42)})
	w = request(t, h, "POST", "/precognition", `{"name":""}`, map[string]string{"Content-Type": "application/json", "Precognition": "true"})
	requireEqual(t, w.Code, 422)
	w = request(t, h, "POST", "/precognition", `{"name":"Ada"}`, map[string]string{"Content-Type": "application/json", "Precognition": "true"})
	requireEqual(t, w.Code, 204)
	requireEqual(t, w.Header().Get("Precognition-Success"), "true")
}

func TestMissingBuildHasActionableError(t *testing.T) {
	root := t.TempDir()
	_, err := server.NewWithOptions(root, server.Options{
		Environment: "demo",
		AssetFS:     os.DirFS(filepath.Join(root, "public", "build")),
	})
	if err == nil {
		t.Fatal("missing production manifest must fail startup")
	}
	if !strings.Contains(err.Error(), "manifest") || !strings.Contains(err.Error(), "npm run build") {
		t.Fatalf("missing build error must explain recovery: %v", err)
	}
}

func TestTemplateValidationSessionIsolationAndNamedBags(t *testing.T) {
	for _, bag := range []string{"", "profile"} {
		t.Run("bag="+bag, func(t *testing.T) {
			h := templateHandler(t)
			headers := map[string]string{"Content-Type": "application/json", "X-Inertia": "true"}
			if bag != "" {
				headers["X-Inertia-Error-Bag"] = bag
			}
			w := request(t, h, "POST", "/submit", `{"name":""}`, headers)
			requireEqual(t, w.Code, 303)
			requireEqual(t, w.Header().Get("Location"), "/form")
			cookies := w.Result().Cookies()
			if len(cookies) == 0 {
				t.Fatal("validation must persist a session cookie")
			}
			// A browser with no submission cookies must never receive another
			// browser's errors.
			clean := page(t, h, "/form", "", "", "")
			requireEqual(t, nativeProp(clean, "props.errors"), map[string]any{})
			w = request(t, h, "GET", "/form", "", headers, cookies...)
			requireEqual(t, w.Code, 200)
			var p map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
				t.Fatal(err)
			}
			path := "props.errors.name"
			if bag != "" {
				path = "props.errors." + bag + ".name"
			}
			requireEqual(t, nativeProp(p, path), "Name is required")
			// Honor replacement Set-Cookie values, as a real cookie jar does.
			w = request(t, h, "GET", "/form", "", headers, w.Result().Cookies()...)
			if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
				t.Fatal(err)
			}
			requireEqual(t, nativeProp(p, "props.errors"), map[string]any{})
		})
	}
}

func TestTemplateStaleVersionPreservesValidationForRefresh(t *testing.T) {
	h := templateHandler(t)
	w := request(t, h, "POST", "/submit", `{"name":""}`, map[string]string{"Content-Type": "application/json", "X-Inertia": "true"})
	cookies := w.Result().Cookies()
	w = request(t, h, "GET", "/form", "", map[string]string{"X-Inertia": "true", "X-Inertia-Version": "old"}, cookies...)
	requireEqual(t, w.Code, 409)
	refreshed := request(t, h, "GET", "/form", "", map[string]string{"X-Inertia": "true"}, w.Result().Cookies()...)
	requireEqual(t, refreshed.Code, 200)
	var p map[string]any
	if err := json.Unmarshal(refreshed.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	requireEqual(t, nativeProp(p, "props.errors.name"), "Name is required")
}

func TestTemplateRouteParameterAndRegistry(t *testing.T) {
	h := templateHandler(t)
	p := page(t, h, "/users/37", "", "", "")
	requireEqual(t, p["component"], "Users/Show")
	requireEqual(t, nativeProp(p, "props.user.id"), "37")
	w := request(t, h, "GET", "/routes", "", nil)
	requireEqual(t, w.Code, 200)
	if !strings.Contains(w.Body.String(), "users.show") || !strings.Contains(w.Body.String(), "/users/:user") {
		t.Fatalf("named route missing from registry: %s", w.Body.String())
	}
}

func TestTemplateProductionDisablesDemoDiagnostics(t *testing.T) {
	t.Setenv("APP_KEY", strings.Repeat("01", 32))
	root := t.TempDir()
	app, err := server.NewWithOptions(root, server.Options{
		Environment: "production",
		AssetFS:     templateAssets(),
	})
	if err != nil {
		t.Fatal(err)
	}
	h := server.HTTPHandler(app)
	for _, path := range []string{"/stats", "/reset", "/routes", "/failure", "/malformed"} {
		method := "GET"
		if path == "/reset" {
			method = "POST"
		}
		requireEqual(t, request(t, h, method, path, "", nil).Code, 404)
	}
	requireEqual(t, request(t, h, "GET", "/health", "", nil).Code, 200)
	requireEqual(t, page(t, h, "/home", "", "", "")["component"], "Home")
}

func TestTemplateManifestIsPrivateAndAssetsDoNotConsumeValidation(t *testing.T) {
	h := templateHandler(t)
	requireEqual(t, request(t, h, "GET", "/build/.vite/manifest.json", "", nil).Code, 404)
	w := request(t, h, "POST", "/submit", `{"name":""}`, map[string]string{"X-Inertia": "true", "Content-Type": "application/json"})
	cookies := w.Result().Cookies()
	for _, path := range []string{"/stats", "/health", "/http", "/build/assets/app-test.js"} {
		asset := request(t, h, "GET", path, "", nil, cookies...)
		requireEqual(t, asset.Code, 200)
		if len(asset.Result().Cookies()) != 0 {
			t.Errorf("%s unexpectedly consumes/changes flash cookie", path)
		}
	}
	w = request(t, h, "GET", "/form", "", map[string]string{"X-Inertia": "true"}, cookies...)
	var p map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	requireEqual(t, nativeProp(p, "props.errors.name"), "Name is required")
}
