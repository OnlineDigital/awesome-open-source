import { createSignal, For, Show } from 'solid-js'
import type { Screenshot } from '../types'
import { IconClose } from './icons'

export function Gallery(props: { shots: Screenshot[]; fallbackAlt: string; closeLabel?: string }) {
  const [open, setOpen] = createSignal<Screenshot | null>(null)
  let dialog: HTMLDialogElement | undefined
  const show = (s: Screenshot) => { setOpen(s); dialog?.showModal() }
  return (
    <>
      <ul class="gallery">
        <For each={props.shots}>{(s, i) => (
          <li class={i() === 0 ? 'gallery-lead' : ''}>
            <button type="button" class="gallery-item" onClick={() => show(s)}>
              <img src={s.path} alt={s.alt || props.fallbackAlt} loading="lazy" />
              <Show when={s.alt}><span class="gallery-cap">{s.alt}</span></Show>
            </button>
          </li>
        )}</For>
      </ul>
      <dialog ref={dialog} class="lightbox" onClick={e => { if (e.target === dialog) dialog?.close() }} onClose={() => setOpen(null)}>
        <Show when={open()}>{s => (
          <figure>
            <img src={s().path} alt={s().alt || props.fallbackAlt} />
            <figcaption>{s().alt}</figcaption>
            <button type="button" class="lightbox-close" onClick={() => dialog?.close()} aria-label={props.closeLabel ?? 'Close'}><IconClose width={18} height={18} /></button>
          </figure>
        )}</Show>
      </dialog>
    </>
  )
}
