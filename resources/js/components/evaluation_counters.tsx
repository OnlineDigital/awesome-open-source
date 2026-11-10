import { createSignal, onMount, onCleanup, For, Show } from 'solid-js'
import { router } from '@inertiajs/core'

export default function EvaluationCounters() {
  const [counts, setCounts] = createSignal<Record<string, number>>({})
  const [error, setError] = createSignal('')
  let pending: AbortController | undefined
  async function refresh() {
    pending?.abort()
    const request = new AbortController()
    pending = request
    try {
      const response = await fetch('/stats', { signal: request.signal, headers: { Accept: 'application/json' } })
      if (!response.ok) throw new Error(`HTTP ${response.status}`)
      const data = await response.json()
      setCounts(data.evaluations)
      setError('')
    } catch (failure) { if (!request.signal.aborted) setError('The counters could not be loaded. Try again.') }
  }
  const cleanup = router.on('finish', () => void refresh())
  onMount(() => void refresh())
  onCleanup(() => { cleanup(); pending?.abort() })
  return <section class="demo-section">
    <div class="section-heading"><h2>Server callback calls</h2><button class="secondary" onClick={() => void refresh()}>Refresh counters</button></div>
    <p>Each value increases when the server executes its callback. These demonstration counters are shared by the demo process.</p>
    <Show when={error()}><p role="alert">{error()}</p></Show>
    <dl class="counter-list"><For each={Object.entries(counts())}>{([name, count]) => <div><dt>{name}</dt><dd>{count}</dd></div>}</For></dl>
    <details><summary>Raw data</summary><pre id="evaluation-counters">{JSON.stringify(counts(), null, 2)}</pre></details>
  </section>
}
