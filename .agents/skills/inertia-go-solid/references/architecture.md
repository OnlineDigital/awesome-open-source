# Template architecture

## Page request flow

1. Root `main.go` runs the Fiber server without a command and dispatches Artisan commands otherwise. `./artisan` invokes that CLI entrypoint. SIGINT/SIGTERM stop the listener and allow active requests to finish within the shutdown deadline.
2. `routes/web.go` registers native Fiber methods and handlers through `internal/router`. A controller is a Go package in a snake_case folder; each action has a separate file, such as `form_controller/store_form.go`.
3. An action receives `*application.Application` and returns `fiber.Handler`. `app.RenderPage(c, component, props)` delegates to Inertia middleware through the official Fiber adaptor. `app.Handle(http.HandlerFunc(...))` adapts actions that need the upstream HTTP API. The inertia-go core receives standard HTTP requests and responses only inside this interoperability boundary.
4. The application resolves selected local Lazy values, supplies Always errors, and delegates native builders and protocol serialization to inertia-go. Initial HTML uses the native v3 JSON bootstrap script; subsequent Inertia visits receive a JSON page object.
5. `resources/js/app.tsx` resolves component names to Solid pages and applies the persistent layout. `@solidjs/meta` manages the browser head. SSR is not configured.

There is one Fiber/fasthttp listener. `server.HTTPHandler` is a testing and embedding bridge that executes the same Fiber application through the official adaptor; browser tests use a real Fiber listener.

## Artisan and scaffolding

Use `./artisan help` as the command inventory. The shell launcher is a convenience for POSIX environments; `go run -buildvcs=false . artisan <command>` invokes the same Go CLI on other platforms. Commands run from the project root and load literal `.env` values; existing process variables take precedence.

Development/build prerequisites are Go 1.26+, Node 22.12+, and npm. On Windows, use Git Bash for `./artisan`, or the native launchers:

```powershell
.\artisan.cmd install
.\artisan.cmd dev --port 8081 --vite-port 5174
# PowerShell alternative, when permitted by the local execution policy:
.\artisan.ps1 help
```

The Bash/CMD/PowerShell launchers enter the project directory before invoking Go. Direct `go run ... artisan ...` commands use the current directory, so run them from the repository root. `.gitattributes` keeps Bash files LF and Windows launchers CRLF.

```sh
./artisan install
./artisan key:generate
./artisan dev --host 127.0.0.1 --port 8080 --vite-port 5173
./artisan build
./artisan start --env demo --host 127.0.0.1 --port 8080
./artisan route:list
./artisan test
```

`install` runs `npm ci` and `go mod download`. Install the browser runtime with `npx playwright install chromium` when needed for tests. `route:list` prints the method/path/name registry without requiring a frontend build; `routes` is an alias. `about` reports runtime and project information. `key:generate` writes a random 32-byte hex APP_KEY to `.env` without printing it. It refuses to overwrite an existing key unless `--force` deliberately rotates it. Rotation invalidates existing encrypted session cookies.

For a new feature, start with a controller rather than adding handlers to `main.go`:

```sh
./artisan make:controller ProjectsController --action index_page,show_project,store_project
./artisan make:page Projects/Index
./artisan make:component ProjectList
```

The controller generator creates `app/controllers/projects_controller/`, one snake_case Go file per action, and exported action factories. `index_page` becomes `Index`; `show_project` becomes `ShowProject`; `store_project` becomes `StoreProject`. Generated actions are page-rendering stubs; replace a store action with parsing, validation, and redirects before using it as a mutation. The page generator creates `resources/js/pages/projects/index_page.tsx`; the component generator creates `resources/js/components/project_list.tsx`. Nested names are supported for pages and components. Generators refuse overwrites and do not automatically register routes or page resolvers.

Import the controller in `routes/web.go`, register its actions, then import the Solid page and add its exact server component name to `resources/js/app.tsx`. Existing pages receive the shared persistent layout through the resolver map.

```go
r.Group("/projects", "projects.", nil, func(projects *router.Router) {
    projects.Get("/:project", projects_controller.ShowProject(app)).Name("show")
    projects.Post("/", projects_controller.StoreProject(app)).Name("store")
})
```

Fiber `/:project` segments and Laravel-style `/{project}` segments are accepted; the router normalizes required Laravel segments to Fiber syntax. Optional parameters, wildcards, and constraints are outside the named-URL reversal contract. Validate parameters in the application; there is no automatic route model binding.

`r.URL(name, map[string]string, url.Values)` reverses a named route and is available as `app.RouteURL`. Unknown names, missing or extra parameters, empty values, and `.`/`..` segments return errors. Each parameter is escaped as one segment; query values are encoded separately. Read route parameters through `application.RouteParameter(c, "project")`, which decodes the matched segment once.

Middleware follows native Fiber semantics: `func(c fiber.Ctx) error`, followed by `c.Next()` to continue. Groups inherit URL prefixes, name prefixes, and ordered middleware. Put authorization before protected data loading when adding authentication.

An action file such as `projects_controller/index_page.go` contains:

```go
package projects_controller

import (
    "github.com/andreitelteu/inertia-go-solid-template/app/application"
    "github.com/gofiber/fiber/v3"
    inertia "github.com/inertia-go/inertia-go"
)

func Index(app *application.Application) fiber.Handler {
    return func(c fiber.Ctx) error {
        return app.RenderPage(c, "Projects/Index", inertia.Props{
            "projects": []string{"Example"},
        })
    }
}
```

## Sessions and validation

The inertia-go CookieStore encrypts flash/errors with a 32-byte key. The application's capture store retains consumed values in request context so Render can supply Always errors, named bags, and reflash after a stale-version 409. The validation-error reflash roundtrip is tested; flash-message reflash is implemented without a separate roundtrip test. This is a local correction, not an upstream patch. Static, JSON, health, and diagnostic requests do not consume session data intended for a page.

Form submissions parse JSON, URL-encoded, and multipart bodies, enforce the body limit, validate server rules, and redirect to the form with 303. The client preserves input; errors arrive through the next request's encrypted cookie. The `invalid`/`bag` query examples exist only in development/demo/test modes and are not a validation architecture for real data. Multipart temporary files are cleaned; the demo does not persist uploads.

Production requires a persistent APP_KEY of 64 hex characters. Its session cookie is HttpOnly, SameSite Lax, and Secure, so production deployment requires HTTPS for browser session roundtrips. Development/demo/test can generate an ephemeral key when none is supplied. Keep the production key consistent across restarts and instances and do not log it. Full CSRF protection, authentication, and an ORM are outside this starter's implemented scope.

## Development and production assets

`./artisan dev` runs the bundled Node watcher and Vite HMR. Go/HTML changes under `app/`, `cmd/`, `routes/`, and `internal/`, plus root `main.go`, `go.mod`, and `go.sum`, trigger a rebuild and restart after a successful build; an invalid edit leaves the existing server running. Changes to `.env` or Go files under `public/` require restarting the development command. Solid edits use Vite HMR. Air is not required. The development server reads `VITE_DEV_SERVER_URL`; Vite handles TSX and CSS entrypoints.

Defaults are `127.0.0.1:8080` for Fiber and port `5173` for Vite. Select free ports rather than stopping an unrelated process. For LAN development:

```sh
./artisan dev --host 0.0.0.0 --port 8081 --vite-port 5174
```

The watcher binds Vite to the application host unless `VITE_HOST` overrides it. With a wildcard host it advertises the first non-internal IPv4 address; with multiple interfaces, set `VITE_DEV_SERVER_URL` to the reachable Vite origin, such as `http://<LAN-IP>:5174`. It must be reachable from the browser, not just the Go process. Keep the selected application origin allowed by `vite.config.ts` CORS settings. Development Go binaries have separate generation filenames so Windows does not need to overwrite a running executable.

`npm run build` builds only the Vite frontend into `public/build/`. `./artisan build` runs that build and compiles root `main.go` with `-tags production`, `-trimpath`, and `CGO_ENABLED=0`. The default output is `build/inertia-go-solid-template` or the `.exe` equivalent. Use `--os`, `--arch`, and `--output` for another target; `--skip-frontend` reuses the existing generated bundle and requires it to be current.

`public/embedded_production.go` uses `//go:embed all:build`; `public.Assets()` returns an `fs.FS` rooted at the generated build. The bundle includes JavaScript, CSS, local fonts, and the private `.vite/manifest.json`. The application reads that manifest to resolve assets and hashes it with SHA256 for the Inertia asset version. Fiber serves public bundle files under `/build/` and rejects hidden paths and directory traversal. Manifest access stays private.

Static `ModifyResponse` supplies immutable caching and explicit MIME types for JS/MJS, CSS, WOFF/WOFF2, TTF/OTF, and WASM. Preserve `assetContentType` and `decodedAssetPath`: Windows MIME registrations can be missing or wrong, and `nosniff` makes script/style types significant. `TestStaticAssetTypesIgnoreIncorrectSystemMIMERegistrations` exercises HTTP responses with deliberately incorrect MIME registrations.

Without the production tag, `public.Assets()` returns nil, allowing compilation before Vite generates a bundle. Outside Vite development mode, that build reads disk assets and needs `public/build/`. A production-tagged executable uses its embedded filesystem, overrides dev-server URLs, and requires no external frontend directory, Node, Go, or npm at runtime. Explicit `server.Options.AssetFS` overrides support isolated tests or custom embedding.

`./artisan start` launches the built executable, defaults to production, and accepts `--host`, `--port`, `--env`, and an optional `--output` binary path. `serve` runs the current executable's server directly with host/port/environment options. A shipped executable with no command also serves directly and loads `.env`; process variables win. Production-tagged binaries default to production. Supply APP_KEY and runtime configuration through the environment or a private `.env`. No Docker runtime is required.

Cross-compile after generating the current frontend bundle:

```sh
./artisan build --os windows --arch amd64 --output build/template-windows.exe --skip-frontend
./artisan build --os darwin --arch arm64 --output build/template-macos --skip-frontend
```

These commands prove compilation only. A foreign executable must run on its target OS; `start --output` does not emulate that target. Native CI builds and executes the production smoke test on Linux, Windows, and macOS, and checks Bash or CMD/PowerShell launchers as appropriate.

## Verification

`./artisan test` runs the npm verification pipeline: TypeScript, production build, Go race tests, a standalone production smoke test, Chromium, and the generated report. `npm run test:portable` starts the built executable from an empty temporary directory and checks embedded JS/CSS/fonts, private manifest paths, secure session cookies, and absence of a Vite fallback. Native CI runs this check on Linux, Windows, and macOS. `npm run test:go` or `npm run test:browser` targets one layer. `tests/protocol/template_test.go` exercises Fiber and sessions; `native_features_test.go` checks native builder metadata; `known_limits_test.go` reproduces upstream deficiencies separately.

`tests/browser/compatibility.spec.ts` covers retained functional scenarios. `demo.spec.ts` checks working controls, route parameters, hook cleanup, and mobile layout. Browser tests are serial because demonstration counters are process-wide rather than per user. Diagnostic endpoints are disabled in production.

Chromium tests expect a compiled executable and use port `8102`. For a focused browser check after changing the app, run `./artisan build`, then `npm exec -- playwright test tests/browser/demo.spec.ts`. `npm run test:portable` also needs the current native binary. Full `./artisan test` rebuilds it automatically; its Go race stage requires a working native C compiler.

Run `go test -buildvcs=false -tags production ./...` after a frontend build when changing embedding or filesystem selection. Protocol asset-serving fixtures use `fstest.MapFS`, matching the embedded filesystem and avoiding Windows cleanup failures from Fiber's cached OS file handles. Use disk fixtures when testing disk-specific behavior; keep the missing-build test backed by a real empty `os.DirFS`. Regression assertions should still verify response bytes, MIME, private paths, and session behavior rather than skipping them on Windows.

## Fiber request lifetime

Fiber uses fasthttp and does not automatically reproduce net/http request-context cancellation when a browser disconnects. Do not retain `fiber.Ctx` or borrowed slices after a handler returns; copy values and use explicit deadlines for background work. Client cancellation and suppression of stale UI responses do not prove cancellation of a server-side query.

Primary references: [Fiber adaptor](https://docs.gofiber.io/middleware/adaptor/), [Fiber routing](https://docs.gofiber.io/guide/routing/). Local source and tests define the project's concrete API.
