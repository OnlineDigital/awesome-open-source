import { useI18n } from '../lib/i18n'

const TIER_LABEL: Record<number, string> = { 5: 'thriving', 4: 'active', 3: 'maintained', 2: 'slow', 1: 'abandoned' }

export function ScoreBadge(props: { score: number | null; tier: number | null; size?: 'sm' | 'lg' }) {
  const { t } = useI18n()
  const tier = () => props.tier ?? 0
  return (
    <>
      {props.score !== null && props.tier !== null && (
        <span class={`score score-${tier()} ${props.size === 'lg' ? 'score-lg' : ''}`} title={`${t('score.title')}: ${props.score.toFixed(1)} / 5`}>
          <span class="score-num" aria-hidden="true">{props.score.toFixed(1)}</span>
          <span class="score-label">{t(`tier.${TIER_LABEL[tier()]}`)}</span>
          <span class="sr-only">{`${t('score.title')}: ${props.score.toFixed(1)} ${t('score.out_of')}`}</span>
        </span>
      )}
    </>
  )
}
