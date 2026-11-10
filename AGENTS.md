# Project conventions

Use the project skill at `.agents/skills/inertia-go-solid/SKILL.md` when changing this Go/Inertia/Solid template. It documents the working helpers and pinned upstream limits.

- One Go module, Fiber + inertia-go backend, and common Solid frontend; add routes in `routes/web.go`. Keep net/http use inside the Inertia interoperability boundary.
- A controller is a folder/package in `app/controllers/`, with each action in a separate snake_case Go file.
- Keep dependencies pinned. Do not patch dependency caches or hide failed behavior behind fabricated Inertia metadata.
- Use `./artisan help` for development, production builds, scaffolds, route inspection, key generation, and tests. Root `main.go` is the server/CLI entrypoint; Node's bundled watcher handles development rebuilds without a mandatory Air dependency.
- `./artisan build` generates Vite assets and compiles with `-tags production`; `public.Assets()` serves the embedded bundle. `npm run build` builds only the frontend. Production runs as one executable without a separate `public/build/` directory.
- `./artisan test` validates TypeScript, production assets, Go race tests, and browser scenarios. Read `reports/template-verification.md` for the executed scope. Keep tested application helpers distinct from native upstream capabilities.
