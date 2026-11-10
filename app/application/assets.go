package application

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Assets struct {
	Script    string
	Styles    []string
	Version   string
	DevServer string
}

func LoadAssets(root, env, devServer string) (Assets, error) {
	return LoadAssetsFS(root, env, devServer, nil)
}

// LoadAssetsFS uses a build-root filesystem for all manifest and asset reads.
// A provided filesystem is authoritative: missing files never fall back to disk.
func LoadAssetsFS(root, env, devServer string, assetFS fs.FS) (Assets, error) {
	if assetFS == nil && env == "development" && devServer != "" {
		u, err := url.Parse(devServer)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return Assets{}, fmt.Errorf("invalid VITE_DEV_SERVER_URL")
		}
		devServer = strings.TrimRight(devServer, "/")
		return Assets{Script: devServer + "/resources/js/app.tsx", Version: "development", DevServer: devServer}, nil
	}
	if assetFS == nil {
		assetFS = os.DirFS(filepath.Join(root, "public", "build"))
	}
	data, err := fs.ReadFile(assetFS, ".vite/manifest.json")
	if err != nil {
		return Assets{}, fmt.Errorf("read Vite manifest (run npm run build first): %w", err)
	}
	var entries map[string]struct {
		File    string   `json:"file"`
		CSS     []string `json:"css"`
		Imports []string `json:"imports"`
	}
	if err = json.Unmarshal(data, &entries); err != nil {
		return Assets{}, fmt.Errorf("decode Vite manifest: %w", err)
	}
	entry, ok := entries["resources/js/app.tsx"]
	if !ok || entry.File == "" {
		return Assets{}, fmt.Errorf("Vite manifest has no resources/js/app.tsx entry")
	}
	assetURL := func(file string) (string, error) {
		if file == "" || path.IsAbs(file) || path.Clean(file) != file || file == ".." || strings.HasPrefix(file, "../") || strings.Contains(file, "\\") {
			return "", fmt.Errorf("invalid manifest asset path %q", file)
		}
		info, err := fs.Stat(assetFS, file)
		if err != nil || info.IsDir() {
			return "", fmt.Errorf("Vite asset is missing: %s", file)
		}
		return "/build/" + file, nil
	}
	script, err := assetURL(entry.File)
	if err != nil {
		return Assets{}, err
	}
	a := Assets{Script: script}
	hash := sha256.Sum256(data)
	a.Version = hex.EncodeToString(hash[:])
	seenCSS := map[string]bool{}
	seenEntries := map[string]bool{}
	var collect func(string) error
	collect = func(key string) error {
		if seenEntries[key] {
			return nil
		}
		seenEntries[key] = true
		item, ok := entries[key]
		if !ok {
			return fmt.Errorf("manifest import %q is missing", key)
		}
		if _, err := assetURL(item.File); err != nil {
			return err
		}
		for _, importKey := range item.Imports {
			if err := collect(importKey); err != nil {
				return err
			}
		}
		for _, file := range item.CSS {
			if seenCSS[file] {
				continue
			}
			href, err := assetURL(file)
			if err != nil {
				return err
			}
			seenCSS[file] = true
			a.Styles = append(a.Styles, href)
		}
		return nil
	}
	if err := collect("resources/js/app.tsx"); err != nil {
		return Assets{}, err
	}
	return a, nil
}

func (a Assets) Tag(string) template.HTML {
	var b strings.Builder
	if a.DevServer != "" {
		fmt.Fprintf(&b, `<script type="module" src="%s/@vite/client"></script>`, template.HTMLEscapeString(a.DevServer))
	}
	fmt.Fprintf(&b, `<script type="module" src="%s"></script>`, template.HTMLEscapeString(a.Script))
	for _, css := range a.Styles {
		fmt.Fprintf(&b, `<link rel="stylesheet" href="%s">`, template.HTMLEscapeString(css))
	}
	return template.HTML(b.String())
}
func (a Assets) Asset(string) string         { return a.Script }
func (a Assets) CSS(string) template.HTML    { return "" }
func (a Assets) ReactRefresh() template.HTML { return "" }
