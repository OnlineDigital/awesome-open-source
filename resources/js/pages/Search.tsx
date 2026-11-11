import { router } from 'inertia-adapter-solid'
import { For, Show } from 'solid-js'
import { Pagination } from '../components/pagination'
import { ProjectCard } from '../components/project_card'
import { ScoreFilter } from '../components/score_filter'
import { SearchBox } from '../components/search_box'
import { useI18n } from '../lib/i18n'
import type { SearchResult } from '../types'

export default function Search(props: { query: string; category: string; min: number; sort: string; page: number; unavailable: boolean; result: SearchResult | null }) {
  const { t, props: shared } = useI18n()
  const base = () => (shared().locale === shared().site.defaultLocale ? '' : `/${shared().locale}`) + '/search'
  const visit = (patch: Record<string, string | number | undefined>) => {
    const q: Record<string, string | number> = {}
    const next = { q: props.query, category: props.category, min_score: props.min || undefined, sort: props.sort, ...patch }
    for (const [k, v] of Object.entries(next)) if (v !== undefined && v !== '' && v !== 0) q[k] = v
    router.get(base(), q, { preserveScroll: true })
  }
  const searchHref = (patch: Record<string, string | number | undefined>) => {
    const next: Record<string, string | number | undefined> = { q: props.query, category: props.category, min_score: props.min || undefined, sort: props.sort, ...patch }
    const q = new URLSearchParams()
    for (const [k, v] of Object.entries(next)) if (v !== undefined && v !== '' && v !== 0) q.set(k, String(v))
    const str = q.toString()
    return str ? `${base()}?${str}` : base()
  }
  const totalPages = () => (props.result ? Math.max(1, Math.ceil(props.result.found / props.result.perPage)) : 1)
  return (
    <div class="wrap main page">
      <header class="page-head">
        <h1>{props.query ? t('search.results_for', { q: props.query }) : t('search.title')}</h1>
        <SearchBox initial={props.query} large autofocus={!props.query} />
      </header>
      <Show when={!props.unavailable} fallback={<p class="empty" role="alert">{t('search.unavailable')}</p>}>
        <div class="toolbar">
          <label class="inline">{t('search.all_categories')}
            <select value={props.category} onChange={e => visit({ category: e.currentTarget.value, page: undefined })}>
              <option value="">{t('search.all_categories')}</option>
              <For each={props.result?.facets?.category_slug ?? []}>{f => <option value={f.value}>{f.value} ({f.count})</option>}</For>
            </select>
          </label>
          <label class="inline">{t('category.sort')}
            <select value={props.sort} onChange={e => visit({ sort: e.currentTarget.value, page: undefined })}>
              <option value="">{t('nav.search')}</option>
              <option value="score">{t('category.sort_score')}</option>
              <option value="stars">{t('category.sort_stars')}</option>
              <option value="new">{t('category.sort_new')}</option>
            </select>
          </label>
          <ScoreFilter value={props.min} href={min => searchHref({ min_score: min || undefined, page: undefined })} />
          <span class="muted small" aria-live="polite">{t('search.found', { n: props.result?.found ?? 0 })}</span>
        </div>
        <Show when={(props.result?.projects.length ?? 0) > 0} fallback={<p class="empty">{t('search.no_results')}</p>}>
          <div class="tracklist"><For each={props.result!.projects}>{p => <ProjectCard project={p} />}</For></div>
        </Show>
        <Pagination page={props.page} pages={totalPages()} path={base()} params={{ q: props.query, category: props.category, min_score: props.min || undefined, sort: props.sort }} />
      </Show>
    </div>
  )
}
