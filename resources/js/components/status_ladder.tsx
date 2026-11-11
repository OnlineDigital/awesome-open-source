import { For } from 'solid-js'
import { useI18n } from '../lib/i18n'
import { stepsFor } from '../lib/steps'
import { StepRow } from './step_row'

export const TIER_KEY: Record<number, string> = { 5: 'thriving', 4: 'active', 3: 'maintained', 2: 'slow', 1: 'abandoned' }
const TIERS: [number, string, number][] = [[5, '4.5–5', 4.75], [4, '3.5–4.4', 3.9], [3, '2.5–3.4', 2.9], [2, '1.7–2.4', 2.0], [1, '1–1.6', 1.2]]

/** The five tiers, each drawn as a step row at a representative score. */
export function StatusLegend() {
  const { t } = useI18n()
  return (
    <ol class="legend" aria-label={t('score.title')}>
      <For each={TIERS}>{([tier, range, sample]) => (
        <li class="legend-row">
          <StepRow lit={stepsFor(sample)} size="sm" />
          <span class="legend-label">{t(`tier.${TIER_KEY[tier]}`)}</span>
          <span class="legend-range">{range}</span>
        </li>
      )}</For>
    </ol>
  )
}
