import { Link } from './link'
import { For } from 'solid-js'
import { useI18n } from '../lib/i18n'

const OPTIONS = [0, 2, 3, 3.5, 4, 4.5]

/** Minimum maintenance score: picking 4 keeps projects scoring 4 or higher. Plain links, so filtered lists are crawlable and shareable. */
export function ScoreFilter(props: { value: number; href: (min: number) => string }) {
  const { t } = useI18n()
  return (
    <div class="score-filter" role="group" aria-label={t('search.min_score')}>
      <span class="toolbar-label">{t('search.min_score')}</span>
      <For each={OPTIONS}>{v => (
        <Link href={props.href(v)} class={`seg seg-score ${props.value === v ? 'seg-active' : ''}`} aria-current={props.value === v ? 'true' : undefined} preserveScroll>
          {v === 0 ? t('search.any_score') : <><span class="seg-num">{v}</span><span aria-hidden="true">+</span></>}
        </Link>
      )}</For>
    </div>
  )
}
