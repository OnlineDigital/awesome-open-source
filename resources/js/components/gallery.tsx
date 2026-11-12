import { createSignal, For, Show } from 'solid-js'
import type { Screenshot } from '../types'
import { IconClose } from './icons'

const SWIPE_PX = 56
const MAX_ZOOM = 5

type Pt = { x: number; y: number }
const dist = (a: Pt, b: Pt) => Math.hypot(a.x - b.x, a.y - b.y)
const clamp = (v: number, lo: number, hi: number) => Math.min(hi, Math.max(lo, v))

// A stage showing one capture at a time (never cropped) and a strip of step
// keys underneath. Drag the stage or the lightbox image sideways to change
// capture; in the lightbox, pinch (or double-click) zooms and, while zoomed,
// one finger pans instead of swiping.
export function Gallery(props: { shots: Screenshot[]; fallbackAlt: string; closeLabel?: string }) {
  const [cur, setCur] = createSignal(0)
  let dialog: HTMLDialogElement | undefined
  const last = () => props.shots.length - 1
  const idx = () => Math.min(cur(), last())
  const shot = () => props.shots[idx()]
  const alt = (s: Screenshot) => s.alt || props.fallbackAlt
    // Dragging left goes to the next capture, right to the previous one; the
  // track slides so the old capture leaves and the new one enters.
  const damp = (dx: number) => (dx > 0 && idx() === 0) || (dx < 0 && idx() === last()) ? dx * .25 : dx
  const trackX = (dx: number) => `translate3d(calc(${-idx() * 100}% + ${dx}px), 0, 0)`
  const step = (dir: 1 | -1) => setCur(i => clamp(Math.min(i, last()) + dir, 0, last()))

  // ---- stage: single-pointer horizontal drag ----
  const [stageDx, setStageDx] = createSignal(0)
  const [stageDrag, setStageDrag] = createSignal(false)
  let sx = 0, sy = 0, stageMoved = false, stageId = -1
  const stageDown = (e: PointerEvent) => {
    if (e.button > 0 || stageId !== -1) return
    stageId = e.pointerId; sx = e.clientX; sy = e.clientY; stageMoved = false
    ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
    setStageDrag(true)
  }
  const stageMove = (e: PointerEvent) => {
    if (e.pointerId !== stageId) return
    const dx = e.clientX - sx
    if (Math.abs(dx) > 6) stageMoved = true
    setStageDx(props.shots.length > 1 ? damp(dx) : 0)
  }
  const stageUp = (e: PointerEvent) => {
    if (e.pointerId !== stageId) return
    stageId = -1
    const dx = e.clientX - sx, dy = e.clientY - sy
    if (e.type === 'pointerup' && Math.abs(dx) > SWIPE_PX && Math.abs(dx) > Math.abs(dy) * 1.2) step(dx < 0 ? 1 : -1)
    setStageDx(0); setStageDrag(false)
  }
  const stageClick = () => { if (stageMoved) { stageMoved = false; return } dialog?.showModal() }

  // ---- lightbox: swipe, pinch zoom, pan ----
  const [scale, setScale] = createSignal(1)
  const [tx, setTx] = createSignal(0)
  const [ty, setTy] = createSignal(0)
  const [lbDx, setLbDx] = createSignal(0)
  const [lbDrag, setLbDrag] = createSignal(false)
  const pts = new Map<number, Pt>()
  let view: HTMLDivElement | undefined
  const curImg = () => view?.querySelectorAll('img')[idx()]
  let lbMoved = false
  let start: Pt = { x: 0, y: 0 }, base = { tx: 0, ty: 0 }
  let d0 = 1, s0 = 1, multi = false, lastTap = 0
  const resetZoom = () => { setScale(1); setTx(0); setTy(0); setLbDx(0) }
  const pan = (nx: number, ny: number, s = scale()) => {
    const el = curImg(), w = el?.clientWidth ?? 0, h = el?.clientHeight ?? 0
    setTx(clamp(nx, -(s - 1) * w / 2, (s - 1) * w / 2)); setTy(clamp(ny, -(s - 1) * h / 2, (s - 1) * h / 2))
  }
  const go = (dir: 1 | -1) => { resetZoom(); step(dir) }
  const lbDown = (e: PointerEvent) => {
    if (e.button > 0) return
    ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
    pts.set(e.pointerId, { x: e.clientX, y: e.clientY })
    setLbDrag(true)
    if (pts.size === 2) {
      const [a, b] = [...pts.values()]
      d0 = dist(a, b) || 1; s0 = scale(); multi = true; setLbDx(0)
    } else if (pts.size === 1) {
      start = { x: e.clientX, y: e.clientY }; base = { tx: tx(), ty: ty() }; multi = false; lbMoved = false
    }
  }
  const lbMove = (e: PointerEvent) => {
    if (!pts.has(e.pointerId)) return
    pts.set(e.pointerId, { x: e.clientX, y: e.clientY })
    if (pts.size >= 2) {
      const [a, b] = [...pts.values()]
      const s = clamp(s0 * dist(a, b) / d0, 1, MAX_ZOOM)
      setScale(s); pan(tx(), ty(), s)
    } else if (scale() > 1.02) {
      pan(base.tx + e.clientX - start.x, base.ty + e.clientY - start.y)
    } else if (!multi) {
      if (Math.abs(e.clientX - start.x) > 6) lbMoved = true
      setLbDx(damp(e.clientX - start.x))
    }
  }
  const lbUp = (e: PointerEvent) => {
    if (!pts.has(e.pointerId)) return
    const dx = e.clientX - start.x, dy = e.clientY - start.y
    pts.delete(e.pointerId)
    if (pts.size === 1) { // pinch ended with one finger left: keep panning from here
      const [p] = [...pts.values()]; start = { ...p }; base = { tx: tx(), ty: ty() }
      return
    }
    if (pts.size > 0) return
    setLbDrag(false); setLbDx(0)
    if (e.pointerType === 'touch' && e.type === 'pointerup' && !multi && Math.abs(dx) < 10 && Math.abs(dy) < 10) { // touch has no dblclick on a captured target
      const now = Date.now(); if (now - lastTap < 300) { lbDouble(); lastTap = 0; return } lastTap = now
    }
    if (scale() <= 1.05) {
      setScale(1); setTx(0); setTy(0)
      if (!multi && e.type === 'pointerup' && Math.abs(dx) > SWIPE_PX && Math.abs(dx) > Math.abs(dy) * 1.2) go(dx < 0 ? 1 : -1)
    }
    multi = false
  }
  const lbDouble = () => { if (scale() > 1) resetZoom(); else { setScale(2.5); setTx(0); setTy(0) } }
  // Close on a tap outside the capture (backdrop or empty slide area), never on the capture itself.
  const outsideClose = (e: MouseEvent) => {
    if (lbMoved) return
    const r = curImg()?.getBoundingClientRect()
    const inside = r && e.clientX >= r.left && e.clientX <= r.right && e.clientY >= r.top && e.clientY <= r.bottom
    if (e.target === dialog || (e.target as HTMLElement).closest('.lb-view') && !inside) dialog?.close()
  }
  const key = (e: KeyboardEvent) => {
    if (e.key === 'ArrowRight') go(1); else if (e.key === 'ArrowLeft') go(-1)
  }

  return (
    <div class="shots">
      <button type="button" class="shots-stage" onPointerDown={stageDown} onPointerMove={stageMove} onPointerUp={stageUp} onPointerCancel={stageUp} onClick={stageClick}>
        <div class="shots-track" classList={{ dragging: stageDrag() }} style={{ transform: trackX(stageDx()) }}>
          <For each={props.shots}>{(s, i) => (
            <div class="shots-slide" aria-hidden={i() !== idx()}><img src={s.path} alt={alt(s)} draggable={false} loading={i() === 0 ? 'eager' : 'lazy'} /></div>
          )}</For>
        </div>
      </button>
      <Show when={props.shots.length > 1}>
        <ul class="shots-keys">
          <For each={props.shots}>{(s, i) => (
            <li>
              <button type="button" class="shots-key" classList={{ on: i() === idx() }} aria-label={`${i() + 1} / ${props.shots.length}`} aria-pressed={i() === idx()} onClick={() => setCur(i())}>
                <img src={s.path} alt="" loading="lazy" draggable={false} />
              </button>
            </li>
          )}</For>
        </ul>
      </Show>
      <dialog ref={dialog} class="lightbox" onKeyDown={key} onClick={outsideClose} onClose={resetZoom}>
        <figure>
          <div ref={view} class="lb-view" onPointerDown={lbDown} onPointerMove={lbMove} onPointerUp={lbUp} onPointerCancel={lbUp} onDblClick={lbDouble}>
            <div class="lb-track" classList={{ dragging: lbDrag() }} style={{ transform: trackX(lbDx()) }}>
              <For each={props.shots}>{(s, i) => (
                <div class="lb-slide" aria-hidden={i() !== idx()}>
                  <img src={s.path} alt={alt(s)} draggable={false} classList={{ dragging: lbDrag() }}
                    style={{ transform: i() === idx() ? `translate(${tx()}px, ${ty()}px) scale(${scale()})` : undefined }} />
                </div>
              )}</For>
            </div>
          </div>
          <Show when={shot().alt}><figcaption>{shot().alt}</figcaption></Show>
          <button type="button" class="lightbox-close" onClick={() => dialog?.close()} aria-label={props.closeLabel ?? 'Close'}><IconClose width={18} height={18} /></button>
        </figure>
      </dialog>
    </div>
  )
}
