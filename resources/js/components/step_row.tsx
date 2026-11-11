import { For } from 'solid-js'
import { STEPS } from '../lib/steps'

/** Sixteen step keys; the first `lit` are on, coloured in quarters. Decorative: the score is always written out beside it. */
export function StepRow(props: { lit: number; animate?: boolean; size?: 'xs' | 'sm' | 'md' | 'lg' }) {
  return (
    <span class={`keys keys-${props.size ?? 'md'} ${props.animate ? 'keys-anim' : ''}`} aria-hidden="true">
      <For each={Array.from({ length: STEPS }, (_, i) => i)}>{i => <i class={`key ${i < props.lit ? `on q${Math.floor(i / 4) + 1}` : ''}`} style={{ '--n': i }} />}</For>
    </span>
  )
}
