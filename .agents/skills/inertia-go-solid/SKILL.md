---
name: inertia-go-solid
description: Build, extend, and test this project's Fiber, inertia-go, and SolidJS template using its Artisan CLI, routes, controllers, page props, forms, navigation, and tested compatibility helpers. This is a community Go/Solid integration rather than Laravel or React.
---

# inertia-go + SolidJS

Work from the repository root containing `go.mod` and `package.json`. There is one Fiber backend and one Solid frontend. Exact dependency pins and lockfiles define the behavior: Fiber v3.5.0, inertia-go v0.10.0, Inertia core v3.8.0, inertia-adapter-solid 1.0.0-beta.9, Solid 1.9.15. Server/router/controllers use Fiber; net/http remains inside the official Fiber adaptor boundary because inertia-go core consumes standard HTTP requests.

## Read what the task needs

- [architecture.md](references/architecture.md): adding routes/controllers/pages, bootstrapping assets, sessions and development commands.
- [features.md](references/features.md): demonstrated behaviors, native versus application helpers, and differences from official Inertia/Laravel/React. Read this before changing lazy/partial props, shared errors, Once, scroll, forms, cache hooks or dependency pins.
- Root `docs/requirements.md` preserves the original 24-family acceptance targets; `reports/template-verification.md` records the latest executed scenarios. Treat targets and PASS on a subscenario separately.

## Local development

Use Go 1.26+, Node 22.12+, and npm. Start with `./artisan install`, then `./artisan dev`; choose free application and Vite ports with `--port` and `--vite-port`. Use Bash on Linux/macOS or Git Bash on Windows; native Windows alternatives are `artisan.cmd` and `artisan.ps1`. No Docker or Air installation is required.

Read the architecture reference for LAN addresses, Windows launchers, cross-compilation, and asset-serving tests. Source commands need the development toolchain; a production executable serves its embedded frontend without Go or Node installed.

## Project conventions

1. Declare routes in `routes/web.go`; use the project's method helpers, `.Name(...)`, groups and middleware. Avoid a central path switch. A controller is a Go package in a snake_case folder under `app/controllers/`; each action has its own snake_case.go file, such as `props_controller/index_page.go`. Inject the application dependencies; Go does not discover controllers from filenames.
2. Resolve page names to Solid components through `resources/js/app.tsx`; keep each page in `resources/js/pages/`. Preserve the persistent layout and use the Inertia router/Link rather than a second client router.
3. For an ordinary expensive prop use **the local** `app/inertia.Lazy` together with the application's Render path. A bare callback is not supported by the pinned upstream. Do not revive `Merge(fn).Append()` as a lazy workaround. Native Optional/Defer/Once/Merge remain native builders; do not wrap local Lazy inside opaque native builders.
4. Keep shared nested parent and leaf Always wrappers where required. Validation errors are Always props in the app. Native Share callbacks are eager, so placing an expensive query in Share is not a lazy optimization.
5. Use the local full-URL cache hook, the Promise-to-signal JSON helper, and the manual scroll demo where applicable. These are application fixes, not proof that upstream usePrefetch/useHttp/InfiniteScroll are fixed. Do not edit node_modules, module caches or protocol metadata to make tests pass.
6. Keep demo instrumentation and query-driven examples scoped to development/demo/test modes. Actual product handlers must validate input and use request-scoped session data; query-string errors are demonstrations, not a validation architecture. The template is not a complete Laravel framework: auth, ORM and full CSRF infrastructure require explicit implementation.

## Verification

Use `./artisan` as the project command interface; run `help` for its available commands. `dev` starts the bundled Go watcher and Vite HMR without requiring Air. `build` runs Vite and compiles root `main.go` with `-tags production`, embedding the complete frontend through `public.Assets()` into a single executable. `npm run build` builds the frontend only. `test` runs TypeScript, the production build, Go race tests, a portable production smoke test, Chromium, and the generated report. Browser tests are serial because demo counters are process-wide. `npm run test:go` or `npm run test:browser` can target one layer. Read the architecture reference for scaffolds, environment loading, key generation, and production runtime details.

Rebuild the executable before testing changed frontend or server behavior in Chromium or `test:portable`; those checks run the compiled production server. Cross-compilation alone does not verify execution on the target operating system. Keep the explicit asset MIME handling and in-memory asset fixtures described in the architecture reference when changing static serving.

For changed protocol behavior, test response shape **and evaluation counts**. For browser behavior, test actual navigation/cache/state/form effects, not just JSON keys. Known upstream-limit tests assert the observed deficiency directly; their passing status means the limitation is still reproduced, not that the feature works.

After changing dependency versions, rerun the existing suite and reassess every local helper against the known-limit tests. Mark unexercised subfeatures as untested; SSR/hydration and DevTools have no certification here. Avoid claiming Laravel/React parity from metadata-only tests or a green manual pagination demo.
