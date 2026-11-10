package application

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	bridge "github.com/andreitelteu/inertia-go-solid-template/app/inertia"
	upstream "github.com/inertia-go/inertia-go"
	"github.com/inertia-go/inertia-go/session"
)

//go:embed root.html
var views embed.FS

type RequestRecord struct {
	Path    string `json:"path"`
	Method  string `json:"method"`
	Purpose string `json:"purpose"`
	Only    string `json:"only"`
	Except  string `json:"except"`
}
type Stats struct {
	Requests    []RequestRecord `json:"requests"`
	Evaluations map[string]int  `json:"evaluations"`
}

// Application contains the dependencies shared by controller factories. Nothing
// is a package global, so tests and multiple server instances remain isolated.
type Application struct {
	Adapter   *upstream.Inertia
	Encrypted *upstream.Inertia
	Cleared   *upstream.Inertia
	Assets    Assets
	Session   *captureStore
	Demo      bool
	// The route table supplies these helpers after controller dependencies exist.
	RouteURL  func(string, map[string]string, url.Values) (string, error)
	RouteList func() any
	mu        sync.Mutex
	stats     Stats
}

func New(root, env, devServer string, demo bool) (*Application, error) {
	return NewWithAssets(root, env, devServer, demo, nil)
}

// NewWithAssets keeps deployment assets injectable without changing controller
// dependencies or the native Inertia protocol. assetFS is rooted at Vite build.
func NewWithAssets(root, env, devServer string, demo bool, assetFS fs.FS) (*Application, error) {
	assets, err := LoadAssetsFS(root, env, devServer, assetFS)
	if err != nil {
		return nil, err
	}
	key := make([]byte, 32)
	if configured := os.Getenv("APP_KEY"); configured != "" {
		key, err = hex.DecodeString(configured)
		if err != nil || len(key) != 32 {
			return nil, fmt.Errorf("APP_KEY must contain 64 hexadecimal characters (32 bytes)")
		}
	} else {
		if env == "production" {
			return nil, fmt.Errorf("APP_KEY is required in production; generate with openssl rand -hex 32")
		}
		if _, err = rand.Read(key); err != nil {
			return nil, fmt.Errorf("generate session key: %w", err)
		}
	}
	store, err := session.NewCookie(session.CookieOptions{Name: "solid_flash", Keys: [][]byte{key}, HTTPOnly: true, Secure: env == "production", SameSite: http.SameSiteLaxMode, MaxAge: 120})
	if err != nil {
		return nil, err
	}
	a := &Application{Assets: assets, Demo: demo, Session: &captureStore{store: store}}
	a.Reset()
	cfg := upstream.Config{RootView: "root.html", TemplateFS: views, Version: assets.Version, Session: a.Session, Vite: assets}
	create := func(config upstream.Config) (*upstream.Inertia, error) {
		i, err := upstream.New(config)
		if err == nil {
			i.ShareValue("shared", upstream.Always(map[string]any{"app": upstream.Always("compat-lab")}))
		}
		return i, err
	}
	if a.Adapter, err = create(cfg); err != nil {
		return nil, err
	}
	cfg.EncryptHistory = true
	if a.Encrypted, err = create(cfg); err != nil {
		return nil, err
	}
	cfg.EncryptHistory = false
	cfg.ClearHistory = true
	if a.Cleared, err = create(cfg); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *Application) Render(w http.ResponseWriter, r *http.Request, component string, props upstream.Props) {
	a.RenderWith(a.Adapter, w, r, component, props)
}
func (a *Application) RenderWith(adapter *upstream.Inertia, w http.ResponseWriter, r *http.Request, component string, props upstream.Props) {
	state := messagesFrom(r)
	if _, exists := props["errors"]; !exists {
		errors := map[string]any{}
		for key, value := range state.errors {
			errors[key] = value
		}
		if bag := upstream.FromRequest(r).ErrorBag; bag != "" && len(errors) > 0 {
			errors = map[string]any{bag: errors}
		}
		props["errors"] = upstream.Always(errors)
	}
	// The pinned native adapter consumes flash on stale-version responses.
	// Keep the messages for the client's subsequent full browser navigation.
	info := upstream.FromRequest(r)
	if info.IsInertia && r.Method == http.MethodGet && info.Version != "" && info.Version != a.Assets.Version {
		bag := info.ErrorBag
		if bag == "" {
			bag = "default"
		}
		_ = a.Session.FlashErrors(w, r, bag, state.errors)
		for key, value := range state.flash {
			_ = a.Session.FlashMessage(w, r, key, value)
		}
	}
	bridge.Render(adapter, a.Assets.Version, w, r, component, props)
}

// RedirectForm explicitly uses 303 for every mutation, including POST. Errors
// are persisted into the native encrypted flash store, not query parameters.
func (a *Application) RedirectForm(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	bag := r.Header.Get("X-Inertia-Error-Bag")
	if bag == "" {
		bag = "default"
	}
	if len(errors) > 0 {
		if err := a.Session.FlashErrors(w, r, bag, errors); err != nil {
			http.Error(w, "Cannot save validation errors", 500)
			return
		}
	} else {
		if err := a.Session.FlashMessage(w, r, "success", "Form accepted"); err != nil {
			http.Error(w, "Cannot save confirmation", 500)
			return
		}
	}
	http.Redirect(w, r, "/form", http.StatusSeeOther)
}

func (a *Application) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.stats = Stats{Requests: []RequestRecord{}, Evaluations: map[string]int{}}
	for _, key := range []string{"users", "companies", "optional", "slow", "second", "lookup", "items", "tick", "auth.notifications"} {
		a.stats.Evaluations[key] = 0
	}
}
func (a *Application) Snapshot() Stats {
	a.mu.Lock()
	defer a.mu.Unlock()
	s := Stats{Requests: append([]RequestRecord{}, a.stats.Requests...), Evaluations: map[string]int{}}
	for key, value := range a.stats.Evaluations {
		s.Evaluations[key] = value
	}
	return s
}
func (a *Application) Count(key string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.stats.Evaluations[key]++
	return a.stats.Evaluations[key]
}
func (a *Application) Callback(key string, value any) func() (any, error) {
	return func() (any, error) { a.Count(key); return value, nil }
}
func (a *Application) Record(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.Demo {
			a.mu.Lock()
			a.stats.Requests = append(a.stats.Requests, RequestRecord{r.URL.RequestURI(), r.Method, r.Header.Get("Purpose"), r.Header.Get("X-Inertia-Partial-Data"), r.Header.Get("X-Inertia-Partial-Except")})
			if len(a.stats.Requests) > 1000 {
				a.stats.Requests = a.stats.Requests[len(a.stats.Requests)-1000:]
			}
			a.mu.Unlock()
		}
		next.ServeHTTP(w, r)
	})
}
func WriteJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
func PageNumber(r *http.Request) int {
	n, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if n < 1 {
		n = 1
	}
	return n
}
func Rows(n int) []map[string]any {
	return []map[string]any{{"id": n, "name": fmt.Sprintf("item-%d", n)}}
}
func ReadName(w http.ResponseWriter, r *http.Request) (string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			return "", err
		}
		name, _ := payload["name"].(string)
		return name, nil
	}
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			return "", err
		}
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}
		return r.FormValue("name"), nil
	}
	if err := r.ParseForm(); err != nil {
		return "", err
	}
	return r.FormValue("name"), nil
}

func PageNumberFromQuery(page string) int {
	n, _ := strconv.Atoi(page)
	if n < 1 {
		n = 1
	}
	return n
}
