//go:build production

package server

import (
	"bytes"
	"github.com/andreitelteu/inertia-go-solid-template/public"
	"io/fs"
	"strings"
	"testing"
)

func TestProductionBuildUsesEmbeddedAssetsFromEmptyRoot(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("APP_KEY", strings.Repeat("56", 32))
	t.Setenv("VITE_DEV_SERVER_URL", "http://invalid.example:5173")
	root := t.TempDir()
	app, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	response, body := fiberResponse(t, app, "GET", "/", nil)
	if response.StatusCode != 200 || !strings.Contains(string(body), "/build/assets/") || strings.Contains(string(body), "invalid.example") {
		t.Fatalf("production bootstrap did not use embedded bundle: %d %s", response.StatusCode, body)
	}
	assets := public.Assets()
	if assets == nil {
		t.Fatal("production build has no embedded assets")
	}
	if _, err := fs.ReadFile(assets, ".vite/manifest.json"); err != nil {
		t.Fatalf("hidden manifest was not embedded: %v", err)
	}
	if err := fs.WalkDir(assets, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || strings.HasPrefix(path, ".") {
			return nil
		}
		response, body := fiberResponse(t, app, "GET", "/build/"+path, nil)
		if response.StatusCode != 200 || len(body) == 0 {
			t.Errorf("embedded asset %s: %d", path, response.StatusCode)
		}
		expected, err := fs.ReadFile(assets, path)
		if err != nil {
			return err
		}
		if !bytes.Equal(body, expected) {
			t.Errorf("embedded asset %s body changed", path)
		}
		if contentType := assetContentType(path); contentType != "" && response.Header.Get("Content-Type") != contentType {
			t.Errorf("embedded asset %s MIME: %q want %q", path, response.Header.Get("Content-Type"), contentType)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	response, _ = fiberResponse(t, app, "GET", "/stats", nil)
	if response.StatusCode != 404 {
		t.Fatalf("production-tag default environment exposed diagnostics: %d", response.StatusCode)
	}
}
