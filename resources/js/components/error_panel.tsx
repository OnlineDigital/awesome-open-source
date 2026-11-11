import { createSignal, onCleanup, onMount, Show, type JSX } from 'solid-js'

/** Shows the error and its stack in the page instead of leaving it in the console. */
export function ErrorPanel(props: { error: unknown; reset?: () => void }) {
  const err = () => props.error as { message?: string; stack?: string } | string
  const message = () => (typeof err() === 'string' ? (err() as string) : (err() as { message?: string }).message || String(props.error))
  const stack = () => (typeof err() === 'string' ? '' : (err() as { stack?: string }).stack ?? '')
  return (
    <div class="notice notice-err error-panel" role="alert">
      <h2>Something went wrong on this page</h2>
      <p><strong>{message()}</strong></p>
      <Show when={stack()}><details open><summary>Stack trace</summary><pre class="snippet">{stack()}</pre></details></Show>
      <div class="actions">
        <Show when={props.reset}><button type="button" onClick={() => props.reset!()}>Try again</button></Show>
        <button type="button" class="secondary" onClick={() => navigator.clipboard?.writeText(`${message()}\n${stack()}`)}>Copy error</button>
        <button type="button" class="secondary" onClick={() => location.reload()}>Reload page</button>
      </div>
    </div>
  )
}

/** Catches errors that happen outside rendering (rejected promises, event handlers) and shows them as a dismissible banner. */
export function AsyncErrorBanner(): JSX.Element {
  const [error, setError] = createSignal<unknown>(null)
  const onError = (e: ErrorEvent) => setError(e.error ?? e.message)
  const onRejection = (e: PromiseRejectionEvent) => setError(e.reason)
  onMount(() => { window.addEventListener('error', onError); window.addEventListener('unhandledrejection', onRejection) })
  onCleanup(() => { window.removeEventListener('error', onError); window.removeEventListener('unhandledrejection', onRejection) })
  return (
    <Show when={error()}>
      <div class="error-banner">
        <ErrorPanel error={error()} />
        <button type="button" class="secondary" onClick={() => setError(null)}>Dismiss</button>
      </div>
    </Show>
  )
}
