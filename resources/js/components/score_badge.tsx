import { useI18n } from '../lib/i18n'
import { quarterForScore } from '../lib/steps'
import { TIER_KEY } from './status_ladder'

/** Score as a lit numeral (colour = quarter of the last lit step) with its tier label. */
export function ScoreBadge(props: { score: number | null; tier: number | null; size?: 'sm' | 'lg' }) {
  const { t } = useI18n()
  const tier = () => props.tier ?? 0
  return (
    <>
      {props.score !== null && props.tier !== null && (
        <span class={`score score-${tier()} q${quarterForScore(props.score)} ${props.size === 'lg' ? 'score-lg' : ''}`} title={`${t('score.title')}: ${props.score.toFixed(1)} / 5`}>
          <span class="score-num" aria-hidden="true">{props.score.toFixed(1)}</span>
          <span class="score-label" aria-hidden="true">{t(`tier.${TIER_KEY[tier()]}`)}</span>
          <span class="sr-only">{`${t('score.title')}: ${props.score.toFixed(1)} ${t('score.out_of')}, ${t(`tier.${TIER_KEY[tier()]}`)}`}</span>
        </span>
      )}
    </>
  )
}
