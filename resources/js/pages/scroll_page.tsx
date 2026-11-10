import { createSignal, For, onMount, onCleanup } from 'solid-js'
import { router } from '@inertiajs/core'
import { usePage } from 'inertia-adapter-solid'
import Snapshot from '../components/page_snapshot'

export default function ScrollPage() {
  const page = usePage<any>()
  const [loading, setLoading] = createSignal(false)
  let alive = true
  const hasNext = () => page.scrollProps?.items?.nextPage != null
  function fetchNext() {
    if (loading() || !hasNext()) return
    const metadata = page.scrollProps!.items!
    const url = new URL(page.url, window.location.origin)
    url.searchParams.set(metadata.pageName, String(metadata.nextPage))
    setLoading(true)
    router.visit(url, { only: ['items'], preserveState: true, preserveScroll: true, onFinish: () => { if (alive) setLoading(false) } })
  }
  onMount(() => { window.__lab.scroll = { hasNext, fetchNext } })
  onCleanup(() => { alive = false; window.__lab.scroll = null })
  return <><Snapshot name="ScrollPage" title="Pagination with native merging" description="The button requests the next page using scrollProps and merges items.data. This is a manual application wrapper; the beta adapter's InfiniteScroll component is not used." /><section class="demo-section"><h2>Loaded items</h2><div id="scroll-items"><For each={page.props.items?.data ?? []}>{item => <p data-item={item.id}>{item.name}</p>}</For></div><div class="controls"><button id="scroll-next" disabled={loading() || !hasNext()} onClick={fetchNext}>{loading() ? 'Loading…' : hasNext() ? 'Load the next page' : 'All pages are loaded'}</button><button id="scroll-reset" class="secondary" disabled={loading()} onClick={() => router.visit('/scroll?page=1', { only: ['items'], reset: ['items'], preserveState: true, preserveScroll: true })}>Reset to page 1</button></div><output aria-live="polite">{page.props.items?.data?.length ?? 0} items loaded.</output><p>This wrapper does not include automatic loading on scroll, reverse loading, or the official InfiniteScroll component's URL synchronization.</p></section></>
}
