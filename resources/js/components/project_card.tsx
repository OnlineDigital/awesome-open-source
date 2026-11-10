import { Link } from 'inertia-adapter-solid'
import { useI18n } from '../lib/i18n'
import type { Card } from '../types'
import { ScoreBadge } from './score_badge'

export function ProjectCard(props: { project: Card }) {
  const { number, t } = useI18n()
  const p = () => props.project
  return (
    <article class="card">
      <div class="card-head">
        {p().avatar ? <img src={p().avatar} alt="" width="40" height="40" loading="lazy" class="avatar" /> : <span class="avatar avatar-empty" aria-hidden="true" />}
        <div class="card-title">
          <h3><Link href={p().path}>{p().name}</Link></h3>
          <span class="muted small">{p().fullName}</span>
        </div>
      </div>
      <p class="card-tagline">{p().tagline}</p>
      <div class="card-meta">
        <ScoreBadge score={p().score} tier={p().tier} />
        <span class="muted small" title={t('common.stars')}>★ {number(p().stars)}</span>
        {p().language && <span class="chip">{p().language}</span>}
        {p().hasVideo && <span class="chip chip-video" title={t('project.demo')}>▶ {t('project.demo')}</span>}
      </div>
    </article>
  )
}
