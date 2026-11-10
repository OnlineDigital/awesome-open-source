import { Link } from 'inertia-adapter-solid'
import { For, Show } from 'solid-js'
import { Pagination } from '../components/pagination'
import { ProjectCard } from '../components/project_card'
import { useI18n } from '../lib/i18n'
import type { Card, CategoryCard } from '../types'

export default function Category(props: { category: CategoryCard; categories: CategoryCard[]; projects: Card[]; total: number; page: number; pages: number; sort: string; min: number }) {
  const { t, props: shared } = useI18n()
  const sorts = [['score', 'category.sort_score'], ['stars', 'category.sort_stars'], ['new', 'category.sort_new']]
  const sortHref = (s: string) => `${props.category.path}${s === 'score' ? '' : `?sort=${s}`}`
  return (
    <>
      <nav class="crumbs" aria-label="Breadcrumb"><Link href={shared().site.homePath}>{shared().site.name}</Link> <span aria-hidden="true">/</span> <span>{props.category.name}</span></nav>
      <header class="page-head">
        <h1>{t('category.title', { name: props.category.name })}</h1>
        <Show when={props.category.description}><p class="lead">{props.category.description}</p></Show>
        <p class="muted">{t('common.projects', { n: props.total })}</p>
      </header>
      <div class="toolbar">
        <span class="muted small">{t('category.sort')}:</span>
        <For each={sorts}>{([s, key]) => <Link href={sortHref(s)} class={`pill ${props.sort === s ? 'pill-active' : ''}`} aria-current={props.sort === s ? 'true' : undefined}>{t(key)}</Link>}</For>
      </div>
      <Show when={props.projects.length > 0} fallback={<p class="empty">{t('category.empty')}</p>}>
        <div class="grid"><For each={props.projects}>{p => <ProjectCard project={p} />}</For></div>
      </Show>
      <Pagination page={props.page} pages={props.pages} path={props.category.path} params={{ sort: props.sort === 'score' ? undefined : props.sort }} />
      <aside class="related-cats" aria-label={t('nav.categories')}>
        <h2>{t('nav.categories')}</h2>
        <ul class="pill-list"><For each={props.categories.filter(c => c.projectCount > 0 && c.slug !== props.category.slug)}>{c => <li><Link href={c.path} class="pill">{c.name}</Link></li>}</For></ul>
      </aside>
    </>
  )
}
