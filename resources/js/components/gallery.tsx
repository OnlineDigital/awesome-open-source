import { createSignal, For, Show } from 'solid-js'
import type { Screenshot } from '../types'

export function Gallery(props: { shots: Screenshot[]; fallbackAlt: string }) {
  const [open, setOpen] = createSignal<Screenshot | null>(null)
  let dialog: HTMLDialogElement | undefined
  const show = (s: Screenshot) => { setOpen(s); dialog?.showModal() }
  return (
    <>
      <ul class="gallery">
        <For each={props.shots}>{s => (
          <li><button type="button" class="gallery-item" onClick={() => show(s)}><img src={s.path} alt={s.alt || props.fallbackAlt} loading="lazy" /></button></li>
        )}</For>
      </ul>
      <dialog ref={dialog} class="lightbox" onClick={e => { if (e.target === dialog) dialog?.close() }} onClose={() => setOpen(null)}>
        <Show when={open()}>{s => (
          <figure>
            <img src={s().path} alt={s().alt || props.fallbackAlt} />
            <figcaption>{s().alt}</figcaption>
            <button type="button" class="secondary" onClick={() => dialog?.close()} aria-label="Close">✕</button>
          </figure>
        )}</Show>
      </dialog>
    </>
  )
}
