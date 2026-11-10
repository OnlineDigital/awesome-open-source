package protocol_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	inertia "github.com/inertia-go/inertia-go"
	"github.com/inertia-go/inertia-go/session"
)

// These green diagnostics assert defects of the pinned upstream adapter. They
// deliberately use upstream directly, bypassing the application's fixes. An
// upstream change fails the assertion and prompts review of the local fix.
func nativeAdapter(t *testing.T) *inertia.Inertia {
	t.Helper()
	i, err := inertia.New(inertia.Config{Version: "native-test", Session: session.NewMemory(),
		RootView: "root.html", TemplateFS: fstest.MapFS{"root.html": &fstest.MapFile{Data: []byte(`<!doctype html><html><body>{{ .InertiaBody }}</body></html>`)}}})
	if err != nil {
		t.Fatal(err)
	}
	return i
}

func nativeRender(i *inertia.Inertia, p func() inertia.Props) http.Handler {
	return i.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { i.Render(w, r, "Probe", p()) }))
}

func nativeRequest(t *testing.T, h http.Handler, only string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	r := httptest.NewRequest("GET", "/probe", nil)
	r.Header.Set("X-Inertia", "true")
	r.Header.Set("X-Inertia-Version", "native-test")
	if only != "" {
		r.Header.Set("X-Inertia-Partial-Component", "Probe")
		r.Header.Set("X-Inertia-Partial-Data", only)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var p map[string]any
	if w.Code == 200 {
		if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
			t.Fatal(err)
		}
	}
	return w, p
}

func nativeProp(p map[string]any, path string) any {
	var value any = p
	for _, part := range strings.Split(path, ".") {
		obj, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		value = obj[part]
	}
	return value
}

func TestKnownUpstreamLimits(t *testing.T) {
	t.Run("bare callback produces 500", func(t *testing.T) {
		i := nativeAdapter(t)
		calls := 0
		h := nativeRender(i, func() inertia.Props {
			return inertia.Props{"users": func() (any, error) { calls++; return []string{"Ada"}, nil }}
		})
		w, _ := nativeRequest(t, h, "")
		if w.Code != 500 || calls != 0 {
			t.Fatalf("upstream behavior changed: status=%d callback calls=%d", w.Code, calls)
		}
	})
	t.Run("argumentless Append loses merge metadata", func(t *testing.T) {
		i := nativeAdapter(t)
		h := nativeRender(i, func() inertia.Props { return inertia.Props{"items": inertia.Merge([]int{1}).Append()} })
		w, p := nativeRequest(t, h, "items")
		if w.Code != 200 || p["mergeProps"] != nil {
			t.Fatalf("upstream behavior changed: status=%d mergeProps=%v", w.Code, p["mergeProps"])
		}
	})
	t.Run("excluded Share callback is eager", func(t *testing.T) {
		i := nativeAdapter(t)
		calls := 0
		i.Share("companies", func(*http.Request) any { calls++; return []string{"Acme"} })
		h := nativeRender(i, func() inertia.Props { return inertia.Props{"users": []string{"Ada"}} })
		w, p := nativeRequest(t, h, "users")
		if w.Code != 200 || nativeProp(p, "props.companies") != nil || calls != 1 {
			t.Fatalf("upstream behavior changed: status=%d calls=%d page=%v", w.Code, calls, p)
		}
	})
	t.Run("Always parent does not retain nested leaves", func(t *testing.T) {
		i := nativeAdapter(t)
		i.ShareValue("shared", inertia.Always(map[string]any{"app": "demo"}))
		h := nativeRender(i, func() inertia.Props { return inertia.Props{"users": []string{"Ada"}} })
		_, initial := nativeRequest(t, h, "")
		w, partial := nativeRequest(t, h, "users")
		if w.Code != 200 || nativeProp(initial, "props.shared.app") != "demo" || nativeProp(partial, "props.shared.app") != nil {
			t.Fatalf("upstream behavior changed: initial=%v partial=%v", initial, partial)
		}
	})
	t.Run("native errors disappear on partial", func(t *testing.T) {
		i := nativeAdapter(t)
		h := nativeRender(i, func() inertia.Props { return inertia.Props{"users": []string{"Ada"}} })
		_, initial := nativeRequest(t, h, "")
		_, partial := nativeRequest(t, h, "users")
		if nativeProp(initial, "props.errors") == nil || nativeProp(partial, "props.errors") != nil {
			t.Fatalf("upstream behavior changed: initial=%v partial=%v", initial, partial)
		}
	})
	t.Run("native stale409 omits current version header", func(t *testing.T) {
		i := nativeAdapter(t)
		h := nativeRender(i, func() inertia.Props { return inertia.Props{"users": []string{"Ada"}} })
		r := httptest.NewRequest("GET", "/probe", nil)
		r.Header.Set("X-Inertia", "true")
		r.Header.Set("X-Inertia-Version", "old")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 409 || w.Header().Get("X-Inertia-Version") != "" {
			t.Fatalf("upstream behavior changed: status=%d version=%q", w.Code, w.Header().Get("X-Inertia-Version"))
		}
	})
}
