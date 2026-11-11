# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users
Developers choosing open source software to adopt, and AI agents that consume the same catalog programmatically (llms.txt, WebMCP tools, JSON API). Developers judge a project by whether it is alive and what it looks like in use; agents need structured, relative-linked, machine-readable content.

## Product Purpose
A catalog of open source software from GitHub: categories, demo videos (YouTube), screenshots, an automatic 1–5 maintenance score, and multi-locale content (default locale at `/`, others at `/ro/...`). Success is a developer quickly telling a live project from an abandoned one and seeing it in action before adopting it.

## Positioning
The computed maintenance score (recency of commits on any branch, commit volume, release cadence, bug-fix share) combined with media-rich entries (demo videos, screenshots). Neighboring awesome lists and star-count catalogs show popularity or bare links, not liveness or the tool in use.

## Operating Context
Projects are imported from GitHub URLs as drafts, synced for GitHub data and score, indexed in Typesense, optionally translated and embedded, then published by an admin who adds video, screenshots and texts. Score tiers: thriving ≥4.5, active ≥3.5, maintained ≥2.5, slow ≥1.7, abandoned.

## Capabilities and Constraints
- Stack: Go (Fiber v3 + inertia-go), SolidJS, PostgreSQL + pgvector, Typesense; routes in `routes/web.go`.
- Server-rendered head tags, schema.org JSON-LD, `<noscript>` fallback content, sitemaps, hreflang only for translated pages.
- Public pages: Home, Category, Project, Search, Error. Admin exists but is out of scope for the current redesign.
- Search, categories and project data are exposed via `/api/*` and WebMCP; redesign must not break them.

## Evidence on Hand
Real catalog data lives in the database (no fixtures committed). No testimonials, user counts or press are recorded; do not fabricate them.

## Product Principles
1. Liveness over popularity: the maintenance score is the primary signal and must be legible at a glance.
2. Show the tool in use: video and screenshots are first-class, not decoration.
3. Humans and agents are equal audiences; every visual surface has a structured, linkable counterpart.
4. Honest data: show missing or uncertain data (no releases, default-branch-only sync) rather than imply precision.

## Accessibility & Inclusion
Multi-locale content (English default, Romanian); the design must tolerate longer translated strings. No further standard established.
