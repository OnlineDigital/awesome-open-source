//go:build !production

// Package public exposes the frontend bundle only in production builds.
package public

import "io/fs"

// Assets returns nil in development so a fresh clone compiles before Vite runs.
func Assets() fs.FS { return nil }
