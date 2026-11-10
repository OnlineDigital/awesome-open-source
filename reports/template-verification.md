# Inertia Go + Solid template verification

Browser run: **2026-10-05T18:45:48.900Z**. **71 passed / 0 non-passing**, across 71 scenarios. This tests this template, rather than a comparison with Laravel/React.

## Checks

| Check | Result |
|---|---|
| TypeScript | PASS |
| Embedded production build | PASS |
| Go tests with race detector | PASS |
| Portable embedded server | PASS |
| Chromium browser scenarios | PASS |

Additional smoke checks: 2026-10-05T18:37:22.147Z. Artisan development startup on the LAN, Vite bootstrap, Solid HMR, and Go rebuild/restart were exercised. Desktop/mobile inspection found no overflow or browser errors; the project skill was validated. [Desktop](template-ui-desktop.png), [mobile](template-ui-mobile.png).

Standalone embedded binary from a directory without project files: **PASS**. Cross-compilation: Linux amd64, Windows amd64, and macOS arm64. These local checks execute Linux only; native Windows/macOS checks are provided by CI.

## Reading the results

Tests execute the real Fiber application with production assets and Go endpoints. A pass means that the named scenario works with the pinned dependencies and local helpers. It does not certify the entire Inertia feature family or Laravel compatibility.

Ordinary lazy props use the local application helper, without Merge(fn).Append(). Scroll uses manual pagination over native metadata, avoiding the broken InfiniteScroll component in the Solid beta. Reactive JSON responses and query-aware prefetch inspection also use local helpers. See the [compatibility matrix](../.agents/skills/inertia-go-solid/references/features.md) for native limitations and coverage boundaries.

## Browser scenarios

| Scenario | Result |
|---|---|
| F05 initial HTML, production assets, SPA navigation and back/forward | PASSED |
| F01 hover under 75ms makes no request | PASSED |
| F01 hover prefetch exactly once, Purpose header, cached navigation | PASSED |
| F01 global hover delay | PASSED |
| F01 per-link cache expires and navigation fetches again | PASSED |
| F01 global cache lifetime | PASSED |
| F01 click strategy prefetches on mousedown | PASSED |
| F01 mount and combined strategies deduplicate | PASSED |
| F01 programmatic prefetch and explicit tag invalidation | PASSED |
| F01 mutation invalidates tagged prefetch cache | PASSED |
| F01 usePrefetch state and flush on current page | PASSED |
| F01 rapid hover/unhover deduplicates | PASSED |
| F02 only users avoids excluded callbacks and merges existing props | PASSED |
| F02 except companies avoids callback | PASSED |
| F02 optional initially absent and requested explicitly | PASSED |
| F02 nested only auth.notifications | PASSED |
| F02 link-level only selected callback | PASSED |
| F06 reload preserves local state and scroll | PASSED |
| F03 deferred fallback then content and parallel groups | PASSED |
| F03 deferred fallback is visible before delayed resolution | PASSED |
| F04 once callback reused across eligible navigation | PASSED |
| F04 explicit only refreshes once | PASSED |
| F04 server forced fresh once | PASSED |
| F04 once forgotten after leaving section | PASSED |
| F07 persistent layout retains counter | PASSED |
| F08 shared namespaced data survives partial reload | PASSED |
| F09 stale asset version refreshes full page | PASSED |
| F10 title and metadata replaced without duplicates | PASSED |
| F11 merge append and matchOn avoids duplicate | PASSED |
| F11 merge reset and full visit replace | PASSED |
| F11 deep merge nested arrays | PASSED |
| F12 local manual pagination uses native scroll metadata | PASSED |
| F13 WhenVisible waits then loads optional exactly once | PASSED |
| F14 polling start stop and cleanup on unmount | PASSED |
| F15 useForm dirty reset and successful lifecycle | PASSED |
| F15 Form component serializes and succeeds | PASSED |
| F16 validation redirect populates useForm errors | PASSED |
| F16 named error bag | PASSED |
| F17 useForm Precognition invalid and valid fields | PASSED |
| F18 useHttp JSON request preserves page and history | PASSED |
| F19 history encryption and restoration | PASSED |
| F19 server clearHistory flag | PASSED |
| F20 client push replace and prop helpers without server requests | PASSED |
| F21 successful visit event lifecycle | PASSED |
| F21 cancelAll prevents delayed page replacement | PASSED |
| F05 POST follows redirect as GET | PASSED |
| F05 PUT follows redirect as GET | PASSED |
| F05 PATCH follows redirect as GET | PASSED |
| F05 DELETE follows redirect as GET | PASSED |
| F05 server redirect and location full refresh | PASSED |
| F06 useRemember restores input through history | PASSED |
| F22 HTTP 403 reports exception and preserves page | PASSED |
| F22 HTTP 404 reports exception and preserves page | PASSED |
| F22 HTTP 500 reports exception and preserves page | PASSED |
| F22 offline reports networkError and preserves page | PASSED |
| F22 malformed non-Inertia JSON is surfaced | PASSED |
| F01 local prefetch wrapper includes query URL | PASSED |
| F01 concurrent prefetch and navigation cannot replace current page | PASSED |
| F01 prefetch HTTP failure preserves current page | PASSED |
| F02 concurrent navigation prevents stale delayed response | PASSED |
| F06 preserveState false resets same-component local state | PASSED |
| F15 useForm file upload serialization and progress | PASSED |
| F16 preserveErrors partial background reload | PASSED |
| F18 useHttp Promise resolves JSON without page replacement | PASSED |
| F20 client replace does not add history entry | PASSED |
| F21 before handler can cancel visit | PASSED |
| demonstration navigation reaches every page without runtime exceptions | PASSED |
| visible only and except controls demonstrate avoided companies work | PASSED |
| manual pagination reaches its end and reset replaces accumulated rows | PASSED |
| JSON wrapper keeps response reactive and releases page hooks on unmount | PASSED |
| every demonstration fits a mobile viewport | PASSED |

## Reproduce

Run `npx playwright install chromium` once, then `./artisan test`. Machine-readable browser results and check logs are generated locally in `reports/` and excluded from version control.
