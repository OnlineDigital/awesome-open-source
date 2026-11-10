# Original functional requirements

These original acceptance criteria are goals rather than claims of implemented support. The current verified scope and limitations are in the project skill and `reports/template-verification.md`. The chosen stack is Fiber + inertia-go + SolidJS; Laravel/React is the behavioral reference. Use `./artisan test` for verification. Browser tests, protocol metadata tests, and passing diagnostics that reproduce an upstream limitation establish different levels of evidence.

## Highest-priority requirement: hover prefetch

The first acceptance criterion is a SolidJS Inertia link that prefetches its target after a short hover and serves the subsequent navigation from the prefetch cache.

Inertia v3's expected behavior is:

- `<Link prefetch>` starts prefetching after the pointer hovers for more than 75 ms by default;
- `prefetch.hoverDelay` can change the global delay;
- the default cache lifetime is 30 seconds and can be customized globally or per link;
- supported strategies include `hover`, `click`/mousedown and `mount`, including combinations such as `['mount', 'hover']`;
- programmatic `router.prefetch()` is available;
- `usePrefetch()` exposes prefetching state, cached state, last update time and cache flushing;
- cache tags support targeted invalidation, including invalidation after successful mutations.[1]

This feature matters most because it directly improves perceived navigation speed. It must receive both functional and timing-oriented browser tests.

### Planned prefetch scenarios

- hover shorter than the configured delay does not issue a request;
- hover longer than the delay issues exactly one request;
- clicking after a successful hover prefetch does not issue a duplicate navigation request;
- cached props and the correct component are rendered;
- expired cache causes a fresh request;
- `cacheFor` works globally and per link;
- `click` and `mount` strategies work;
- combined strategies deduplicate requests;
- programmatic prefetch and `usePrefetch()` work;
- `Purpose: prefetch` reaches the Go server;
- cache tags can be flushed explicitly and invalidated by a successful mutation;
- prefetch respects authentication, redirects, errors and asset-version mismatch behavior;
- rapid hover/unhover and concurrent navigation do not produce stale page state.

## Inertia v3 compatibility matrix to implement

Statuses to use later:

- `TODO` — not implemented;
- `PASS` — automated end-to-end test passes;
- `PARTIAL` — usable but not protocol-equivalent;
- `FAIL` — implementation exists but test fails;
- `UNSUPPORTED` — absent from the selected adapter.

### P0 — perceived speed and selective data loading

#### 1. Link hover prefetch and prefetch cache

Priority: **critical**.

Test all scenarios listed above, including cache duration, cache tags, invalidation and prefetch lifecycle events.[1]

#### 2. Partial prop reloads with lazy evaluation

Priority: **critical**.

This is the feature described in the project brief: the backend supplies props through callbacks, and the frontend asks the same page to refresh only selected props.

Expected behavior:

```ts
router.reload({ only: ['users'] })
router.reload({ except: ['companies'] })
```

The server should evaluate only the requested callback, return only the selected prop plus required shared/always data, and the client should merge it into the existing page state. Partial reloads only apply when visiting the same page component.[2]

Test:

- `only` and `except`;
- callback/lazy prop is not evaluated when excluded;
- optional prop appears only when explicitly requested;
- always prop remains present;
- nested prop paths such as `auth.notifications`;
- `router.reload()` preserves state and scroll;
- link-level `only`;
- `preserveErrors` behavior;
- concurrent partial requests cannot overwrite newer state.

#### 3. Deferred props

Priority: **high**.

The initial page should render before expensive data is ready. Deferred callbacks are fetched in follow-up partial requests, can be assigned to parallel groups, and can render fallback, reloading and rescued-error states.[3]

Test:

- one deferred prop;
- multiple props in one group;
- multiple groups requested in parallel;
- fallback → resolved content;
- stale content remains visible with `reloading` during a partial refresh;
- rescued failure UI and explicit retry;
- deferred + once;
- deferred + merge/deep merge.

#### 4. Once props

Priority: **high**.

Once props are resolved once, remembered across eligible navigations and skipped in subsequent responses. They may expire, use a shared custom key, be explicitly refreshed, or compose with optional/deferred/merge props.[5]

Test:

- server callback evaluation count;
- reuse across navigation;
- forgetting when leaving the eligible page section;
- explicit `only` refresh;
- server-forced `fresh` behavior;
- expiration;
- custom key shared by differently named props;
- authenticated shared data becoming `null` after logout;
- interaction with prefetched pages and cache expiry.

### P1 — navigation and server-driven UX

#### 5. Normal Inertia navigation

Test link interception, GET/POST/PUT/PATCH/DELETE visits, browser back/forward, redirects, replace-history, cancellation, progress events and external locations.

#### 6. State and scroll preservation

Test `preserveState`, `preserveScroll`, scroll regions, error-dependent preservation, form state and browser restoration after back/forward.

#### 7. Persistent layouts

Verify layouts are not remounted between compatible pages and that local layout state survives navigation. Include nested layouts and a deliberately stateful fixture such as an audio player or counter.

#### 8. Shared data and flash data

Verify request-scoped shared props, lazy shared callbacks, namespaced props, flash messages, authentication state and interaction with partial reloads and once props.

#### 9. Asset versioning

Verify a stale frontend asset version produces the expected conflict/location response and a full-page refresh, without reloading a non-GET request that the user did not initiate.

#### 10. Title and metadata

Verify `@solidjs/meta`, title callbacks, multiple head elements, deduplication, page replacement and SSR parity.

### P1 — data composition

#### 11. Merge and deep-merge props

On partial reloads, new values can append/prepend arrays, target nested paths, update matching records by ID and reset accumulated data when filters change.[4]

Test:

- append and prepend at root;
- nested path merge;
- `matchOn` update instead of duplicate append;
- multiple merge rules;
- deep merge;
- `reset` when changing filters;
- full visits replace instead of merge;
- combination with deferred and once props.

#### 12. Infinite scroll

Verify automatic bidirectional loading, merge semantics, buffer distance, URL/page synchronization, `preserveUrl`, reset after filtering, next-only/previous-only modes, reverse chat mode and manual controls.[8]

#### 13. Load when visible

Verify Intersection Observer loading, one or multiple props, pre-visibility buffer, configurable wrapper element, `always`, subsequent fetching state, error preservation and no duplicate in-flight request.[7]

#### 14. Polling

Verify automatic cleanup on unmount, dynamic request options, selected-prop polling, manual start/stop, reactive polling status, background-tab throttling and the `overlap`, `cancel` and `rest` concurrency modes.[6]

### P1 — forms, validation and direct HTTP

#### 15. `Form` and `useForm`

Test serialization, nested fields, defaults, reset, dirty state, processing state, progress, success/error lifecycle, recently-successful state, error bags, file uploads, method spoofing and cache-tag invalidation.[9]

#### 16. Validation redirects and errors

Verify the classic Inertia redirect-back flow, server-provided validation errors, multiple forms/error bags, scroll preservation and clearing/preserving errors during partial background requests.

#### 17. Precognition

Test field-level server validation before submission, touched/valid/invalid state and both form and direct HTTP integration. This may require explicit Go-side protocol work if no adapter offers a complete implementation.[9][10]

#### 18. `useHttp`

Verify non-page JSON requests without replacing the current Inertia page, processing/error state, history state and Precognition integration.[10]

### P2 — advanced resilience and security

#### 19. History encryption and clearing

Verify encrypted browser history, restoration, `clearHistory`, logout behavior and failure handling when history state cannot be decrypted.[11]

#### 20. Client-side visits and prop helpers

Test `router.push`, `router.replace`, local prop replacement, append/prepend helpers and browser history without server requests.

#### 21. Events and cancellation

Test before/start/progress/success/error/finish, navigate, prefetching/prefetched, cancellation tokens and `cancelAll` across synchronous, asynchronous and prefetch requests.

#### 22. Network and HTTP failure behavior

Test validation errors, authorization failures, 404/500 responses, malformed Inertia responses, offline/network errors, request cancellation and development error modal behavior.

#### 23. SSR and hydration

SSR is intentionally deferred until CSR compatibility is understood. Later tests should cover the `/render` contract, Solid `renderToString`, client hydration, head output, fallback to CSR, SSR-required failure, code splitting and production assets.

#### 24. Devtools observability

Determine whether Inertia v3 DevTools can classify Go-backed full visits, partial reloads, deferred loads, polls, prefetches and Precognition requests. Add stable request identifiers/headers if the server adapter requires them.


## Sources

[1] https://inertiajs.com/docs/v3/data-props/prefetching
[2] https://inertiajs.com/docs/v3/data-props/partial-reloads
[3] https://inertiajs.com/docs/v3/data-props/deferred-props
[4] https://inertiajs.com/docs/v3/data-props/merging-props
[5] https://inertiajs.com/docs/v3/data-props/once-props
[6] https://inertiajs.com/docs/v3/data-props/polling
[7] https://inertiajs.com/docs/v3/data-props/load-when-visible
[8] https://inertiajs.com/docs/v3/data-props/infinite-scroll
[9] https://inertiajs.com/docs/v3/the-basics/forms
[10] https://inertiajs.com/docs/v3/the-basics/http-requests
[11] https://inertiajs.com/docs/v3/security/history-encryption
[12] https://github.com/inertia-go/inertia-go
[13] https://github.com/iksaku/inertia-adapter-solid
[14] https://github.com/inertiajs/inertia/blob/3.x/CONTRIBUTING.md
