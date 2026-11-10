package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gofiber/fiber/v3"
)

func fixtureBundle() fstest.MapFS {
	return fstest.MapFS{
		".vite/manifest.json": {Data: []byte(`{"resources/js/app.tsx":{"file":"assets/app.js","css":["assets/app.css"],"imports":["_shared.js"]},"_shared.js":{"file":"assets/shared.js","css":["assets/shared.css"]}}`)},
		"assets/app.js":       {Data: []byte("console.log('embedded frontend')")},
		"assets/shared.js":    {Data: []byte("export const shared = true")},
		"assets/app.css":      {Data: []byte("@font-face{font-family:test;src:url('/build/assets/test.woff2')}body{margin:0}")},
		"assets/shared.css":   {Data: []byte("html{color:#123}")},
		"assets/test.woff2":   {Data: []byte("test-font-bytes")},
	}
}

func fiberResponse(t *testing.T, app *fiber.App, method, path string, headers map[string]string) (*http.Response, []byte) {
	t.Helper()
	request := httptest.NewRequest(method, path, nil)
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response, body
}

func TestAssetFilesystemServesEntireBundleWithoutDiskFiles(t *testing.T) {
	t.Setenv("APP_KEY", strings.Repeat("12", 32))
	root := t.TempDir() // No public directory or manifest exists in this root.
	bundle := fixtureBundle()
	app, err := NewWithOptions(root, Options{Environment: "production", AssetFS: bundle})
	if err != nil {
		t.Fatal(err)
	}
	response, html := fiberResponse(t, app, "GET", "/", nil)
	if response.StatusCode != 200 {
		t.Fatalf("bootstrap: %d %s", response.StatusCode, html)
	}
	for _, marker := range []string{"/build/assets/app.js", "/build/assets/app.css", "/build/assets/shared.css", `data-page="app"`} {
		if !strings.Contains(string(html), marker) {
			t.Errorf("bootstrap missing %q", marker)
		}
	}
	for _, path := range []string{"assets/app.js", "assets/shared.js", "assets/app.css", "assets/shared.css", "assets/test.woff2"} {
		response, body := fiberResponse(t, app, "GET", "/build/"+path, nil)
		if response.StatusCode != 200 || string(body) != string(bundle[path].Data) {
			t.Fatalf("filesystem asset %s: %d %s", path, response.StatusCode, body)
		}
		if !strings.Contains(response.Header.Get("Cache-Control"), "immutable") {
			t.Errorf("asset %s missing immutable cache header", path)
		}
		mimeType := response.Header.Get("Content-Type")
		switch filepath.Ext(path) {
		case ".js":
			if !strings.HasPrefix(mimeType, "text/javascript") && !strings.HasPrefix(mimeType, "application/javascript") {
				t.Errorf("script MIME type: %s", mimeType)
			}
		case ".css":
			if !strings.HasPrefix(mimeType, "text/css") {
				t.Errorf("stylesheet MIME type: %s", mimeType)
			}
		case ".woff2":
			if !strings.HasPrefix(mimeType, "font/woff2") {
				t.Errorf("font MIME type: %s", mimeType)
			}
		}
	}
	response, body := fiberResponse(t, app, "GET", "/props", map[string]string{"X-Inertia": "true"})
	var page struct{ Version string }
	if response.StatusCode != 200 || json.Unmarshal(body, &page) != nil {
		t.Fatalf("page: %d %s", response.StatusCode, body)
	}
	hash := sha256.Sum256(bundle[".vite/manifest.json"].Data)
	if page.Version != hex.EncodeToString(hash[:]) {
		t.Fatalf("asset version not from selected filesystem manifest: %s", page.Version)
	}
	if _, err := os.Stat(filepath.Join(root, "public")); !os.IsNotExist(err) {
		t.Fatalf("server unexpectedly wrote or required public files: %v", err)
	}
}

func TestStaticAssetTypesIgnoreIncorrectSystemMIMERegistrations(t *testing.T) {
	bundle := fixtureBundle()
	resources := []struct{ extension, contentType string }{
		{".js", "text/javascript; charset=utf-8"},
		{".mjs", "text/javascript; charset=utf-8"},
		{".css", "text/css; charset=utf-8"},
		{".woff", "font/woff"},
		{".woff2", "font/woff2"},
		{".ttf", "font/ttf"},
		{".otf", "font/otf"},
		{".wasm", "application/wasm"},
	}
	for _, resource := range resources {
		original := mime.TypeByExtension(resource.extension)
		extension := resource.extension
		if err := mime.AddExtensionType(extension, "text/plain"); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			// Go's MIME table has no remove operation. An originally unknown
			// extension is restored to the neutral binary fallback in this test
			// process; every known registration is restored exactly.
			if original == "" {
				original = "application/octet-stream"
			}
			if err := mime.AddExtensionType(extension, original); err != nil {
				t.Error(err)
			}
		})
		bundle["assets/mime-test"+extension] = &fstest.MapFile{Data: []byte("ordinary ASCII bytes")}
	}
	app, err := NewWithOptions(t.TempDir(), Options{Environment: "demo", AssetFS: bundle})
	if err != nil {
		t.Fatal(err)
	}
	for _, resource := range resources {
		response, body := fiberResponse(t, app, "GET", "/build/assets/mime-test"+resource.extension, nil)
		if response.StatusCode != 200 || string(body) != "ordinary ASCII bytes" {
			t.Fatalf("asset body %s: %d %s", resource.extension, response.StatusCode, body)
		}
		if actual := response.Header.Get("Content-Type"); actual != resource.contentType {
			t.Errorf("asset %s trusted system MIME table: got %q want %q", resource.extension, actual, resource.contentType)
		}
		if response.Header.Get("X-Content-Type-Options") != "nosniff" {
			t.Error("asset lost nosniff header")
		}
	}
	response, _ := fiberResponse(t, app, "GET", "/build/assets/mime-test.%77off2", nil)
	if response.StatusCode != 200 || response.Header.Get("Content-Type") != "font/woff2" {
		t.Fatalf("encoded font extension: %d %s", response.StatusCode, response.Header.Get("Content-Type"))
	}
}

func TestPrivateAssetsAndTraversalRemainPrivate(t *testing.T) {
	app, err := NewWithOptions(t.TempDir(), Options{Environment: "demo", AssetFS: fixtureBundle()})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"/build", "/build/", "/build/.vite/manifest.json",
		"/build/%2evite/manifest.json", "/build/%252evite/manifest.json",
		"/build/%2evite%2fmanifest.json", "/build/assets/../.vite/manifest.json",
		"/build/assets/%2e%2e/.vite/manifest.json", "/build/assets/%252e%252e/.vite/manifest.json",
		"/build/assets/missing.js",
	} {
		t.Run(path, func(t *testing.T) {
			response, body := fiberResponse(t, app, "GET", path, nil)
			if response.StatusCode != 404 {
				t.Fatalf("private/missing path: %d %s", response.StatusCode, body)
			}
			if strings.Contains(string(body), "resources/js/app.tsx") {
				t.Fatal("private manifest leaked")
			}
		})
	}
	response, _ := fiberResponse(t, app, "POST", "/build/assets/app.js", nil)
	if response.StatusCode != 405 || response.Header.Get("Allow") != "GET, HEAD" {
		t.Fatalf("static method handling: %d %s", response.StatusCode, response.Header.Get("Allow"))
	}
}

func TestSelectedFilesystemNeverFallsBackToDiskOrDevServer(t *testing.T) {
	root := t.TempDir()
	for name, file := range fixtureBundle() {
		path := filepath.Join(root, "public", "build", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, file.Data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	_, err := NewWithOptions(root, Options{Environment: "development", DevServerURL: "http://localhost:5173", AssetFS: fstest.MapFS{}})
	if err == nil || !strings.Contains(err.Error(), "manifest") {
		t.Fatalf("selected empty filesystem silently used disk/dev server: %v", err)
	}
	broken := fixtureBundle()
	delete(broken, "assets/app.js")
	_, err = NewWithOptions(root, Options{Environment: "demo", AssetFS: broken})
	if err == nil || !strings.Contains(err.Error(), "asset is missing") {
		t.Fatalf("missing selected asset silently used disk: %v", err)
	}
}

func TestProductionSessionKeyRequiredAndCookieFlags(t *testing.T) {
	for _, key := range []string{"", "invalid", strings.Repeat("a", 62)} {
		t.Run("invalid-key-"+strconv.Itoa(len(key)), func(t *testing.T) {
			t.Setenv("APP_KEY", key)
			_, err := NewWithOptions(t.TempDir(), Options{Environment: "production", AssetFS: fixtureBundle()})
			if err == nil || !strings.Contains(err.Error(), "APP_KEY") {
				t.Fatalf("production accepted missing/invalid key: %v", err)
			}
		})
	}
	t.Setenv("APP_KEY", strings.Repeat("34", 32))
	app, err := NewWithOptions(t.TempDir(), Options{Environment: "production", AssetFS: fixtureBundle()})
	if err != nil {
		t.Fatal(err)
	}
	response, _ := fiberResponse(t, app, "POST", "/submit", map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if response.StatusCode != 303 {
		t.Fatalf("validation redirect: %d", response.StatusCode)
	}
	cookies := response.Cookies()
	if len(cookies) != 1 || !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode || cookies[0].MaxAge != 120 {
		t.Fatalf("production flash cookie flags: %+v", cookies)
	}
	response, _ = fiberResponse(t, app, "GET", "/stats", nil)
	if response.StatusCode != 404 {
		t.Fatalf("production diagnostics exposed: %d", response.StatusCode)
	}
}
