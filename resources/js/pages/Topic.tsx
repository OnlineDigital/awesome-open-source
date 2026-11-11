import { For, Show } from 'solid-js'
import { Link } from '../components/link'
import { IconCheck, IconCross, IconHalf } from '../components/icons'
import { ProjectCard } from '../components/project_card'
import { ScoreBadge } from '../components/score_badge'
import { useI18n } from '../lib/i18n'
import type { Card, Cell, FeatureRow, TopicCard } from '../types'

function CellView(props: { cell?: Cell }) {
  const { t } = useI18n()
  const c = () => props.cell
  return (
    <>
      <Show when={c()} fallback={<span class="cmp-none"><span aria-hidden="true">–</span><span class="sr-only">{t('topic.unrated')}</span></span>}>
        {cell => (
          <>
            <Show when={cell().state !== 'text'}>
              <span class={`cmp-state cmp-${cell().state}`}>
                {cell().state === 'yes' ? <IconCheck width={20} height={20} /> : cell().state === 'no' ? <IconCross width={18} height={18} /> : <IconHalf width={20} height={20} />}
                <span class="sr-only">{t(`topic.${cell().state}`)}</span>
              </span>
            </Show>
            <Show when={cell().note}><span class={`cmp-note ${cell().state === 'text' ? 'cmp-note-solo' : ''}`}>{cell().note}</span></Show>
          </>
        )}
      </Show>
    </>
  )
}

export default function TopicPage(props: { topic: TopicCard; body: string; features: FeatureRow[]; cells: Record<string, Record<string, Cell>>; projects: Card[]; others: TopicCard[] }) {
  const { t, props: shared } = useI18n()
  return (
    <div class="wrap main page topic-page">
      <nav class="crumbs" aria-label="Breadcrumb"><Link href={shared().site.homePath}>{shared().site.name}</Link> <span aria-hidden="true">/</span> <span>{props.topic.name}</span></nav>
      <header class="page-head">
        <h1>{props.topic.name}</h1>
        <Show when={props.topic.summary}><p class="lead">{props.topic.summary}</p></Show>
        <p class="muted">{t('common.projects', { n: props.projects.length })}</p>
      </header>

      <Show when={props.features.length > 0 && props.projects.length > 0}>
        <section aria-labelledby="cmp-h">
          <h2 id="cmp-h" class="deck-h">{t('topic.table')}</h2>
          <div class="cmp-scroll" role="region" aria-labelledby="cmp-h" tabindex="0">
            <table class="cmp">
              <thead>
                <tr>
                  <th scope="col" class="cmp-corner">{t('topic.feature')}</th>
                  <For each={props.projects}>{p => (
                    <th scope="col" class="cmp-proj">
                      <Link href={p.path} class="cmp-proj-link">
                        {p.avatar ? <img src={p.avatar} alt="" width="32" height="32" loading="lazy" class="avatar" /> : null}
                        <span>{p.name}</span>
                      </Link>
                      <ScoreBadge score={p.score} tier={p.tier} />
                    </th>
                  )}</For>
                </tr>
              </thead>
              <tbody>
                <For each={props.features}>{f => (
                  <tr>
                    <th scope="row" class="cmp-feature"><span>{f.name}</span><Show when={f.description}><span class="cmp-desc">{f.description}</span></Show></th>
                    <For each={props.projects}>{p => <td><CellView cell={props.cells[p.slug]?.[String(f.id)]} /></td>}</For>
                  </tr>
                )}</For>
              </tbody>
            </table>
          </div>
        </section>
      </Show>

      <Show when={props.body}>
        <section class="prose topic-body">
          <For each={props.body.split(/\n{2,}/)}>{para => <p>{para}</p>}</For>
        </section>
      </Show>

      <section aria-labelledby="tp-h">
        <h2 id="tp-h" class="deck-h">{t('common.projects', { n: props.projects.length })}</h2>
        <div class="tracklist"><For each={props.projects}>{p => <ProjectCard project={p} />}</For></div>
      </section>

      <Show when={props.others.length > 0}>
        <aside class="related-cats" aria-label={t('topic.more')}>
          <h2 class="deck-h">{t('topic.more')}</h2>
          <ul class="pill-list"><For each={props.others}>{o => <li><Link href={o.path} class="pill">{o.name}</Link></li>}</For></ul>
        </aside>
      </Show>
    </div>
  )
}
