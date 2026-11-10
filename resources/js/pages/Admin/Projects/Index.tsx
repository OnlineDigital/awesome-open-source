import { Link, router, useForm } from 'inertia-adapter-solid'
import { createSignal, For, Show } from 'solid-js'
import { Field, Select, StatusPill } from '../../../components/admin_ui'
import type { Project } from '../../../types'

interface Cat { id: number; name: string }

export default function ProjectsIndex(props: { projects: Project[]; total: number; page: number; pages: number; categories: Cat[]; filters: { q: string; status: string; category: number } }) {
  const add = useForm({ urls: '', categoryId: '' as string | number })
  const [q, setQ] = createSignal(props.filters.q)
  const filter = (patch: Record<string, string | number | undefined>) => {
    const next: Record<string, string | number | undefined> = { q: q(), status: props.filters.status, category: props.filters.category || undefined, ...patch }
    const clean: Record<string, string | number> = {}
    for (const [k, v] of Object.entries(next)) if (v !== undefined && v !== '' && v !== 0) clean[k] = v
    router.get('/admin/projects', clean)
  }
  return (
    <>
      <h1>Projects <span class="muted small">({props.total})</span></h1>
      <form class="panel" onSubmit={e => { e.preventDefault(); add.transform(d => ({ ...d, categoryId: Number(d.categoryId) || 0 })).post('/admin/projects', { onSuccess: () => add.reset('urls') }) }}>
        <h2>Add projects</h2>
        <Field label="GitHub repositories" for="urls" hint="One per line: https://github.com/owner/repo or owner/repo. Metadata and the activity score are fetched automatically." error={add.errors.urls}>
          <textarea id="urls" rows="4" value={add.data.urls} onInput={e => add.setData('urls', e.currentTarget.value)} placeholder="https://github.com/vercel/next.js" />
        </Field>
        <Field label="Category" for="cat">
          <Select id="cat" value={add.data.categoryId} onChange={v => add.setData('categoryId', v)} empty="— none —" options={props.categories.map(c => [c.id, c.name])} />
        </Field>
        <button type="submit" disabled={add.processing || !add.data.urls.trim()}>Add as draft</button>
      </form>

      <form class="toolbar" onSubmit={e => { e.preventDefault(); filter({ q: q() }) }} role="search">
        <input type="search" placeholder="Filter by name…" aria-label="Filter by name" value={q()} onInput={e => setQ(e.currentTarget.value)} />
        <Select value={props.filters.status} onChange={v => filter({ status: v })} empty="All statuses" options={[['draft', 'Draft'], ['published', 'Published']]} />
        <Select value={props.filters.category || ''} onChange={v => filter({ category: v })} empty="All categories" options={props.categories.map(c => [c.id, c.name])} />
        <button type="submit" class="secondary">Filter</button>
      </form>

      <table class="table">
        <thead><tr><th>Project</th><th>Category</th><th>Score</th><th>Video</th><th>Status</th><th>Synced</th></tr></thead>
        <tbody>
          <For each={props.projects} fallback={<tr><td colspan="6" class="muted">No projects.</td></tr>}>{p => (
            <tr>
              <td><Link href={`/admin/projects/${p.id}`}>{p.fullName}</Link><Show when={p.syncError}><div class="field-error small" title={p.syncError}>sync error</div></Show></td>
              <td>{p.categoryName || '–'}</td>
              <td>{p.score?.toFixed(1) ?? '–'}</td>
              <td>{p.youtubeId ? '▶' : ''}</td>
              <td><StatusPill status={p.status} /></td>
              <td class="muted small">{p.syncedAt ? new Date(p.syncedAt).toLocaleDateString() : 'never'}</td>
            </tr>
          )}</For>
        </tbody>
      </table>
      <Show when={props.pages > 1}>
        <nav class="pagination">
          {props.page > 1 ? <Link href={`/admin/projects?page=${props.page - 1}`}>← Previous</Link> : <span />}
          <span class="muted">{props.page} / {props.pages}</span>
          {props.page < props.pages ? <Link href={`/admin/projects?page=${props.page + 1}`}>Next →</Link> : <span />}
        </nav>
      </Show>
    </>
  )
}
