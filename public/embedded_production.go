//go:build production

package public

import (
	"embed"
	"io/fs"
)

// Include the private Vite manifest and every generated JS/CSS/font asset. The
// server reads the manifest internally and prevents HTTP access to hidden files.
//
//go:embed all:build
var bundle embed.FS

// Assets is rooted at build, so .vite/manifest.json and assets/... use the same
// paths as the manifest. Production has no dependency on the working directory.
func Assets() fs.FS {
	assets, err := fs.Sub(bundle, "build")
	if err != nil {
		panic(err)
	} // A compile-time embed guarantees this directory.
	return assets
}
