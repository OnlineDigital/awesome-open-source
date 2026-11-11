# Awesome Open Source (awesomeopensource.dev)

Catalog of open source software from GitHub: categories, demo videos (YouTube), screenshots, an automatic **1–5 maintenance score**, multi-locale content, and an AI-agent friendly surface.

Stack: Go (Fiber v3 + inertia-go) · SolidJS · PostgreSQL + pgvector · Typesense. Based on `inertia-go-solid-template`.

## Quick start

```bash
cp .env.example .env            # set ADMIN_EMAIL / ADMIN_PASSWORD, GITHUB_TOKEN (recommended)
podman compose up -d            # or: docker compose up -d   (postgres+pgvector, typesense)
./artisan install
./artisan dev                   # http://127.0.0.1:8080 ; admin at /admin
```

Bulk import your list, then score and index it:

```bash
go run . import repos.txt --category databases   # one GitHub URL or owner/repo per line (drafts)
go run . sync --all                              # GitHub data + activity score
go run . search:reindex                          # Typesense
go run . translate --all                         # optional, needs ANTHROPIC_API_KEY
go run . embed                                   # optional, EMBEDDINGS_URL (pgvector "similar projects")
```
Publish projects from `/admin/projects` (add video, screenshots, texts).

## Activity score (`app/scoring`)
Single number 1.0–5.0 (tiers: thriving ≥4.5, active ≥3.5, maintained ≥2.5, slow ≥1.7, abandoned).
Weights: recency of the newest commit on **any branch** 40% (half-life 90 days) · commit volume 90d/365d 25% · release cadence 25% · bug-fix share (fix commits + patch releases) 10%.
Missing data (e.g. no GitHub releases) is dropped and re-weighted (no-release projects get a 7% discount). Archived = 1.0; >1y without commits capped at 2.4; >2y at 1.4.
With `GITHUB_TOKEN` GraphQL inspects all branches and exact commit counts; without it REST sees the default branch only (commit counts cap at 100).

## SEO / AI surface
- Locale routes: default locale at `/`, others at `/ro/...`; hreflang only for translated pages, untranslated ones are `noindex` + canonical to default.
- Head tags + schema.org JSON-LD (WebSite+SearchAction, CollectionPage, SoftwareApplication, VideoObject, BreadcrumbList) rendered server-side; `<noscript>` fallback content.
- `/sitemap.xml` is only an index (generated on request): `sitemap-general.xml` (home + categories, all locales) and `sitemap-project-001.xml`, `-002`, … (5000 projects per page, ordered by publication date ascending, so new projects append and only the last page changes). Page count comes from the number of published projects. Hreflang, image and video extensions included; `/robots.txt`.
- `/llms.txt` → `/llms/category/{slug}.txt` → `/llms/{project-slug}.md`, all relative links, also under `/{locale}/`.
- WebMCP tools (`search_projects`, `list_categories`, `get_project`) registered via `navigator.modelContext`; backed by `/api/search`, `/api/categories`, `/api/projects/{slug}`.

## Topic pages (feature comparison)
Admin > Topics: a topic page (slug = a GitHub topic such as `laravel-admin-panel`) has translated name, summary and SEO text, a list of features (table rows) and, for every project that carries the topic, a value per feature: Yes / No / Partial / text-only, plus an optional note per language (e.g. "paid first-party plugin"). Projects can also be added by hand (Project > Extra topics, or the topic editor); these curated topics are stored in `projects.extra_topics` and survive GitHub re-syncs.
Public: `/topic/{slug}` (hreflang, JSON-LD, no-JS fallback), listed on the homepage, in `sitemap-general.xml` and `/llms.txt` (`/llms/topic/{slug}.md` contains the comparison as a markdown table). JSON: `/api/topics`, `/api/topics/{slug}`.

## MCP server for AI agents
Admin > Settings > MCP access generates links such as `https://host/mcp/<secret>/sse` (HTTP+SSE) and `https://host/mcp/<secret>` (streamable HTTP). The secret in the URL is the only credential, so no env configuration is needed. Links can be named, renamed, rotated (old URL stops working at once), revoked and deleted; the full URL stays visible to admins. Tools cover projects, translations, screenshots, sync, search reindex, categories, topics, features, comparison values and curated membership (`tools/list` for the full list). Example: `claude mcp add --transport sse aos-admin <sse url>`.

## Tests
`./artisan test` (Go + typecheck). Store integration test needs `TEST_DATABASE_URL` (uses a throwaway schema).
