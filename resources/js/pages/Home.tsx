import { Link } from 'inertia-adapter-solid'
import { For } from 'solid-js'
import { ProjectCard } from '../components/project_card'
import { SearchBox } from '../components/search_box'
import { useI18n } from '../lib/i18n'
import type { Card, CategoryCard } from '../types'

export default function Home(props: { categories: CategoryCard[]; top: Card[]; latest: Card[] }) {
  const { t } = useI18n()
  return (
    <>
      <section class="hero">
        <h1>{t('home.title')}</h1>
        <p class="lead">{t('home.subtitle')}</p>
        <SearchBox large />
      </section>
      <section aria-labelledby="cats">
        <h2 id="cats">{t('home.browse')}</h2>
        <ul class="cat-grid">
          <For each={props.categories.filter(c => c.projectCount > 0)}>{c => (
            <li><Link href={c.path} class="cat-card"><strong>{c.name}</strong><span class="muted small">{t('common.projects', { n: c.projectCount })}</span></Link></li>
          )}</For>
        </ul>
      </section>
      <section aria-labelledby="top">
        <h2 id="top">{t('home.top_active')}</h2>
        <div class="grid"><For each={props.top}>{p => <ProjectCard project={p} />}</For></div>
      </section>
      <section aria-labelledby="latest">
        <h2 id="latest">{t('home.latest')}</h2>
        <div class="grid"><For each={props.latest}>{p => <ProjectCard project={p} />}</For></div>
      </section>
    </>
  )
}
