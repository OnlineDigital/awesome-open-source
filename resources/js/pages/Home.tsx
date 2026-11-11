import { Link } from '../components/link'
import { For, Show } from 'solid-js'
import { ProjectCard } from '../components/project_card'
import { SearchBox } from '../components/search_box'
import { StatusLegend } from '../components/status_ladder'
import { useI18n } from '../lib/i18n'
import { quarterForScore } from '../lib/steps'
import type { Card, CategoryCard, TopicCard } from '../types'

export default function Home(props: { categories: CategoryCard[]; top: Card[]; latest: Card[]; topics: TopicCard[] }) {
  const { t, number } = useI18n()
  const cats = () => props.categories.filter(c => c.projectCount > 0)
  const lead = () => props.top.find(p => p.score !== null)
  return (
    <div class="deck">
      <aside class="panel-side">
        <div class="tempo" aria-hidden={lead() ? undefined : 'true'}>
          {lead() && (
            <Link href={lead()!.path} class={`tempo-link q${quarterForScore(lead()!.score)}`}>
              <span class="tempo-num">{lead()!.score!.toFixed(1)}</span>
              <span class="tempo-cap"><ScoreBadgeLabel card={lead()!} /> · {lead()!.name}</span>
            </Link>
          )}
        </div>
        <h1>{t('home.title')}</h1>
        <p class="lead">{t('home.subtitle')}</p>
        <SearchBox large />
        <StatusLegend />
      </aside>
      <div class="deck-main">
        <section aria-labelledby="cats">
          <h2 id="cats" class="deck-h">{t('home.browse')}</h2>
          <ul class="banks">
            <For each={cats()}>{c => (
              <li><Link href={c.path} class="bank"><span>{c.name}</span><b>{number(c.projectCount)}</b></Link></li>
            )}</For>
          </ul>
        </section>
        <Show when={props.topics.length > 0}>
          <section aria-labelledby="compare">
            <h2 id="compare" class="deck-h">{t('home.compare')}</h2>
            <ul class="topic-list">
              <For each={props.topics}>{tp => (
                <li><Link href={tp.path} class="topic-row"><span class="topic-name">{tp.name}</span><span class="topic-sum">{tp.summary}</span><span class="topic-count">{number(tp.projectCount)}</span></Link></li>
              )}</For>
            </ul>
          </section>
        </Show>
        <section aria-labelledby="top">
          <div class="deck-head"><h2 id="top" class="deck-h">{t('home.top_active')}</h2><span class="chase" aria-hidden="true"><i /></span></div>
          <div class="tracklist"><For each={props.top}>{p => <ProjectCard project={p} />}</For></div>
        </section>
        <section aria-labelledby="latest">
          <h2 id="latest" class="deck-h">{t('home.latest')}</h2>
          <div class="tracklist"><For each={props.latest}>{p => <ProjectCard project={p} />}</For></div>
        </section>
      </div>
    </div>
  )
}

function ScoreBadgeLabel(props: { card: Card }) {
  const { t } = useI18n()
  const keys: Record<number, string> = { 5: 'thriving', 4: 'active', 3: 'maintained', 2: 'slow', 1: 'abandoned' }
  return <>{props.card.tier ? t(`tier.${keys[props.card.tier]}`) : ''}</>
}
