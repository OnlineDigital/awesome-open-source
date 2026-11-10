import { createSignal, Show } from 'solid-js'
import { router } from '@inertiajs/core'
import { Link, useRemember } from 'inertia-adapter-solid'
import Snapshot from '../components/page_snapshot'

export default function Home() {
  const [state, setState] = useRemember('initial', 'home-state')
  const [mount, setMount] = createSignal(false)
  return <>
    <Snapshot name="Home" title="Go on the server. Solid in the browser." description="Try SPA navigation, prefetching, props evaluated on demand, and forms. Each page has working controls and shows the data received from the server." />
    <section class="demo-section"><h2>Navigation and remembered state</h2><p>Enter a value, visit the secondary page, then return with Back. The layout and remembered state remain available.</p><label for="remembered">Value remembered in history</label><input id="remembered" value={state()} onInput={event => setState(event.currentTarget.value)} /><div class="controls"><Link href="/other">Open the secondary page</Link></div></section>
    <section class="demo-section"><h2>Prefetch before navigation</h2><p>Hover over a link. Its response is cached, so clicking can reuse it without a second request.</p>
      <div class="controls"><Link id="hover" href="/other?case=hover" prefetch>Default hover</Link><Link id="expiry" href="/other?case=expiry" prefetch cacheFor={250}>Short cache: 250 ms</Link><Link id="tagged" href="/other?case=tagged" prefetch cacheTags="users">Cache tagged users</Link><Link id="click-prefetch" href="/other?case=click" prefetch="click">Prefetch on press</Link><Link id="global-cache" href="/other?case=global" prefetch>Global cache</Link></div>
      <div class="controls"><button id="mount-links" onClick={() => setMount(true)} disabled={mount()}>Mount prefetch links</button><button class="secondary" onClick={() => router.flushByCacheTags('users')}>Invalidate the users tag</button></div>
      <Show when={mount()}><div class="controls"><Link id="mount" href="/other?case=mount" prefetch="mount">Prefetch on mount</Link><Link id="combined" href="/other?case=combined" prefetch={['mount', 'hover']}>Mount + hover</Link></div></Show>
      <p>The cache state in the props panel uses a local wrapper that includes the query string.</p>
    </section>
    <section class="demo-section"><h2>Responses that cannot replace the page</h2><p>An HTTP error or JSON without the Inertia protocol displays a message and preserves the current page.</p><div class="controls"><button class="secondary" onClick={() => router.visit('/failure?status=403')}>Simulate HTTP 403</button><button class="secondary" onClick={() => router.visit('/failure?status=404')}>Simulate HTTP 404</button><button class="secondary" onClick={() => router.visit('/failure?status=500')}>Simulate HTTP 500</button><button class="secondary" onClick={() => router.visit('/malformed')}>Simulate invalid Inertia JSON</button></div></section>
  </>
}
