import { router } from '@inertiajs/core'
import { Link } from 'inertia-adapter-solid'
import Snapshot from '../components/page_snapshot'
import { useCurrentPrefetch } from '../lib/use_current_prefetch'

export default function Other() {
  const cache = useCurrentPrefetch()
  return <><Snapshot name="Other" title="Navigate without an HTML reload" description="This page receives new props and preserves the layout. For URLs with a query string, the cache uses the complete URL." /><section class="demo-section"><h2>Cache and history</h2><output aria-live="polite">{cache.isPrefetched ? 'A prefetched response is available for this URL.' : 'This URL is not cached.'}</output><div class="controls"><Link href="/home">Back to demonstrations</Link><button class="secondary" onClick={() => cache.flush()}>Clear this URL from the cache</button><button class="secondary" onClick={() => router.reload()}>Reload props</button></div></section></>
}
