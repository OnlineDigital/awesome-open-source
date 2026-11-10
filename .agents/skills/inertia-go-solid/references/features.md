# Demonstrated features and limitations

This project implements the Inertia protocol with community Go and Solid adapters. Official Laravel/React behavior is the functional reference; this repository does not execute a Laravel/React baseline alongside it. Passing one example does not certify every option in its feature family.

Pins: Fiber **v3.5.0**, inertia-go **v0.10.0**, Inertia core **3.8.0**, Solid adapter **1.0.0-beta.9**, Solid **1.9.15**. Current evidence comes from `tests/browser/`, `tests/protocol/`, and the [generated verification report](../../../../reports/template-verification.md).

## Feature coverage

| Family | Demonstrated behavior | Limit or application responsibility |
|---|---|---|
| 1. Prefetch | Hover >75 ms, short hover, click/mount/combined strategies, expiry, tags, cache hits | Native core + Link. The local hook uses the complete URL; the native Solid hook drops query strings. Prefetched error responses may remain cached. |
| 2. Partial reload / lazy | only/except skip companies; nested selectors and preserved state | Protocol filtering is native; ordinary lazy callbacks use an application helper. Bare upstream callbacks are unsupported. |
| 3. Deferred | Initial omission, loading fallback, independent groups | Native Defer + Solid Deferred. Rescue and builder compositions have separate protocol tests. |
| 4. Once | Reuse across eligible pages, explicit only, Fresh, forgetting after leaving the section | Navigation cache, not global Go memoization. Protocol tests check alias/TTL metadata in milliseconds rather than real browser expiration. |
| 5. Navigation | Initial HTML, SPA, back/forward, GET after POST/PUT/PATCH/DELETE, redirect/location | The application supplies 303 for submissions; do not assume native Redirect upgrades every method. |
| 6. State / scroll | preserveState, preserveScroll, local reset, useRemember | Scroll regions, error-dependent branches, and every device are not exhaustively tested. |
| 7. Persistent layout | The layout counter survives navigation | Shared Solid layout; nested layouts are not fully certified. |
| 8. Shared / flash | Namespaced shared data survives partial reload | Always wrappers on parent and leaf. Native flash uses props.flash rather than v3 page.flash; full flash compatibility remains limited. |
| 9. Asset version | Real manifest hash; stale versions produce 409 and a full reload | Local validation-error reflash has a tested roundtrip. Flash-message reflash is implemented without a separate application roundtrip test. Native stale409 loses flash/errors and omits the current version header; this remains a separate diagnostic. |
| 10. Head | Title and Meta update without duplicates | CSR is tested; SSR/hydration are not executed. |
| 11. Merge / deep merge | Append, matchOn deduplication, reset, full-visit replacement, nested arrays | Native Merge. Prepend and multiple nested rules have metadata tests without full browser reconciliation scenarios. No-argument `.Append()` loses the root marker; use `Merge(...)` for root append. |
| 12. Scroll | Next-page button, accumulation, end-of-list and reset | Manual helper over native Scroll. InfiniteScroll beta.9 has an initialization defect; automatic/reverse loading and full URL synchronization are not implemented. |
| 13. WhenVisible | Optional omission until viewport entry, then one load | Native Optional + Solid observer. Buffer/mode options and error recovery have limited coverage. |
| 14. Polling | Start/stop, selected-prop updates, cleanup on page exit | Native usePoll. Background throttling and every overlap mode are not certified. |
| 15. Forms | Form + useForm, dirty/reset, processing/success, upload serialization/progress | The application parses and validates requests, then redirects. The demonstration upload is not persistent file storage. |
| 16. Validation | Field errors after redirect, demonstrated named bag, preserveErrors | Application errors are Always props. Cookie roundtrips, session isolation, and named bags are protocol-tested; the browser checks the demonstration bag. Preservation of every field after errors has no dedicated test. Native named bags are flattened. |
| 17. Precognition | Empty/valid field validation without a submission | Native helper + application rules. The 422 body differs from Laravel's message + error arrays contract; there is no Laravel validator. |
| 18. Direct HTTP | JSON without page/history replacement, Promise result and reactive response | A local helper transfers the Promise result into a signal; native beta.9 useHttp.response remains a null snapshot. |
| 19. History | EncryptHistory, restoration, ClearHistory | Native APIs + browser cryptography; decryption failures and complete auth/logout flows are not certified. |
| 20. Client visits | push/replace, replaceProp/appendToProp/prependToProp without server requests | Local state changes do not persist data to the server. |
| 21. Events/cancellation | Lifecycle, before cancellation, cancelAll, avoiding delayed stale responses | Client hooks; the demonstration event log is not production telemetry. |
| 22. Failures | 403/404/500, offline, non-Inertia JSON, preserved page | Diagnostic examples rather than a complete logging/retry system. |
| 23. SSR/hydration | Untested | Initial HTML includes bootstrap data rather than server-rendered Solid markup. |
| 24. DevTools | Untested | Demonstration instrumentation is not a DevTools integration. |

## only / except and local Lazy

Pass expensive data as a callback rather than computing its result before Render:

```go
import (
    inertia "github.com/inertia-go/inertia-go"
    appinertia "github.com/andreitelteu/inertia-go-solid-template/app/inertia"
)

props := inertia.Props{
    "users": appinertia.Lazy(func() (any, error) { return loadUsers() }),
    "companies": appinertia.Lazy(func() (any, error) { return loadCompanies() }),
}
// Use the application's Render path, which resolves Lazy before upstream Render.
```

```ts
router.reload({ only: ['users'] })
router.reload({ except: ['companies'] })
```

Both requests skip companies while the client retains companies data already received. A full visit includes and evaluates ordinary Lazy values; Optional is omitted. Partial selection applies only when the request component matches the rendered page. A component change results in a full visit. On an asset-version conflict, upstream negotiation runs before callbacks.

Use local Lazy inside ordinary props maps. Native builders are opaque and are not modified. For deferred/once/merge, put the callback into the native constructor rather than placing a local Lazy wrapper inside it. Selecting a descendant of a callback that produces an entire object may require evaluating that callback; separate independent expensive queries into leaf callbacks to skip them independently. An excluded native Share callback may still execute.

The former `Merge(fn).Append()` workaround is not used. Ordinary lazy evaluation now belongs to `app/inertia.Lazy` and the application Render pipeline rather than depending on the upstream merge-marker defect.

## Local frontend boundaries

- `useCurrentPrefetch()` reads full `page.url` through core `getCached`, `getPrefetching`, and `flush`. Its event listeners and cache-expiry timer are cleaned up on unmount. This does not repair the adapter's native `usePrefetch` hook.
- `useHttpResponse(native)` retains the native request/validation API and exposes resolved request Promises through a reactive response getter. It cancels the native request on unmount. Rejections still propagate to callers; UI request handlers catch them. The native getter remains available for diagnostics.
- `ScrollPage` uses native `scrollProps.items.nextPage` and `pageName`, requests only items, and merges `items.data`. Its Load next and Reset controls replace the broken beta InfiniteScroll component. They do not provide automatic intersection loading, reverse loading, or viewport-driven URL updates.

## Native deficiencies that remain visible

- Bare prop callbacks produce HTTP 500; an ordinary Lazy constructor is absent upstream.
- No-path `.Append()` omits mergeProps in this pin; the application does not exploit it.
- Always on the parent alone loses nested leaves during partial reload; native Share callbacks are eager.
- Native automatic errors can be filtered out of partial reloads; native bags are flattened.
- Native flash uses props.flash; stale409 consumes flash/errors without reflashing.
- Native Scroll provides items.data and scrollProps rather than every Laravel paginator field.
- Form parsing, validation rules, 303 redirects, and client helpers are application code rather than Laravel capabilities supplied by the adapter.

`TestKnownUpstreamLimits` and `TestKnownUpstreamSessionAndValidationLimits` are passing diagnostics that assert observed deficiencies. Their passing status does not mean those capabilities are repaired. If an upgrade makes them fail, reassess the local helper instead of mechanically changing expected results.

## Uncertified subfeatures

These families are not an exhaustive certification. Unexecuted behavior includes SSR/hydration, DevTools, Once expiration in the browser, Once alias reuse across different browser components, a Deferred Rescue/retry UI, browser reconciliation of prepend/nested merge rules, automatic/reverse scroll and every scroll region, background polling throttling/overlap, file validation and persistent storage, the full set of Precognition modes, history decryption failures, and real authentication/logout flows. Deferred groups are requested and loaded separately without measuring their request overlap. No result certifies every builder composition or every official React adapter option.

## Differences from official Laravel + React

This starter has no ORM, migrations, authentication/authorization framework, policies, dependency container, automatic route model binding, resource-controller expansion, queues/jobs, complete CSRF system, or persistent file storage. Artisan generators produce files rather than a complete Laravel framework; routes and page resolvers still require explicit registration. The router is a declarative layer over Fiber and controllers are explicitly imported Go packages. Example data is illustrative.

The real server is Fiber/fasthttp. The inertia-go protocol and CookieStore use net/http internally through the official Fiber adaptor and local helpers. No parallel net/http server is started. Headers, cookies, bodies, and route parameters are tested through this pipeline; this does not establish zero overhead or net/http-equivalent disconnect cancellation. Use explicit deadlines for slow work; client cancellation does not prove a server query stopped.

Solid uses signals and reactive props rather than React hooks/lifecycle. Read page.props inside reactive expressions; initialization-time destructuring can create snapshots. Use the Inertia router/Link for navigation and fetch/useHttp when a non-page JSON request is intended. Native metadata and local helpers do not imply complete official-adapter compatibility.

Primary protocol references: [partial reloads](https://inertiajs.com/docs/v3/data-props/partial-reloads), [once](https://inertiajs.com/docs/v3/data-props/once-props), [merging](https://inertiajs.com/docs/v3/data-props/merging-props), [scroll](https://inertiajs.com/docs/v3/data-props/infinite-scroll), [inertia-go](https://github.com/inertia-go/inertia-go), [Solid adapter](https://github.com/iksaku/inertia-adapter-solid). Claims of working behavior come from this project's tests.
