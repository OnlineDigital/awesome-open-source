import { Link } from './link'
import { useI18n } from '../lib/i18n'
import { stepsFor } from '../lib/steps'
import type { Card } from '../types'
import { IconPlay, IconStar } from './icons'
import { LangMark } from './lang_mark'
import { ScoreBadge } from './score_badge'
import { StepRow } from './step_row'

/** A track: name and repo, then the things you choose by (language, stars, score with a compact step row), then the one-line description. */
export function ProjectCard(props: { project: Card }) {
  const { number, t } = useI18n()
  const p = () => props.project
  return (
    <article class="track">
      <div class="track-id">
        {p().avatar ? <img src={p().avatar} alt="" width="40" height="40" loading="lazy" class="avatar" /> : <span class="avatar avatar-empty" aria-hidden="true" />}
        <div class="track-title">
          <h3><Link href={p().path}>{p().name}</Link></h3>
          <span class="track-repo">{p().fullName}</span>
        </div>
      </div>
      <div class="track-meta">
        <span class="track-lang">{p().language ? <><LangMark name={p().language} size={24} /><span>{p().language}</span></> : null}</span>
        <span class="track-stars" title={t('common.stars')}><IconStar width={18} height={18} /><span class="sr-only">{t('common.stars')}: </span>{number(p().stars)}</span>
      </div>
      <div class="track-score">
        <ScoreBadge score={p().score} tier={p().tier} />
        <StepRow lit={stepsFor(p().score)} size="xs" />
      </div>
      <div class="track-info">
        <p class="track-tagline">{p().tagline}</p>
        {p().hasVideo && <span class="mark-video"><IconPlay width={11} height={11} /> {t('project.demo')}</span>}
      </div>
    </article>
  )
}
