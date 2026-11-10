import { createSignal, onCleanup } from 'solid-js'
import { router } from '@inertiajs/core'
import { usePage } from 'inertia-adapter-solid'

// beta.9's usePrefetch checks pathname alone. Query strings identify distinct
// cache entries; read the complete current Inertia URL through the core API.
export function useCurrentPrefetch() {
  const page = usePage()
  const [revision, setRevision] = createSignal(0)
  const refresh = () => setRevision(value => value + 1)
  const cleanups = [router.on('prefetching', refresh), router.on('prefetched', refresh), router.on('navigate', refresh)]
  const timer = window.setInterval(refresh, 250)
  onCleanup(() => { cleanups.forEach(cleanup => cleanup()); window.clearInterval(timer) })
  const cached = () => { revision(); return router.getCached(page.url) }
  return {
    get isPrefetched() { return cached() !== null },
    get isPrefetching() { revision(); return router.getPrefetching(page.url) !== null },
    get lastUpdatedAt() { const entry = cached(); return entry && 'staleTimestamp' in entry ? entry.staleTimestamp : null },
    flush() { router.flush(page.url); refresh() },
  }
}
