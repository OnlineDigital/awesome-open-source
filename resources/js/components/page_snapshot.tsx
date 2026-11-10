import { onMount } from 'solid-js'
import { Title, Meta } from '@solidjs/meta'
import { usePage } from 'inertia-adapter-solid'
import { useCurrentPrefetch } from '../lib/use_current_prefetch'

export default function PageSnapshot(props: { name: string; title?: string; description?: string }) {
  const page = usePage()
  const prefetched = useCurrentPrefetch()
  onMount(() => { window.__lab.page = () => JSON.parse(JSON.stringify(page)); window.__lab.prefetch = prefetched })
  return <>
    <Title>{props.name} — inertia-go · Solid template</Title>
    <Meta name="description" content={props.name} />
    <h1>{props.title ?? props.name}</h1>
    <span id="component" class="component-name">{props.name}</span>
    <p class="intro">{props.description}</p>
    <details class="snapshot">
      <summary>Page props and prefetch cache</summary>
      <pre id="props">{JSON.stringify(page.props, null, 2)}</pre>
      <output id="prefetch-state">{JSON.stringify({ cached: prefetched.isPrefetched, pending: prefetched.isPrefetching, updated: prefetched.lastUpdatedAt })}</output>
    </details>
  </>
}
