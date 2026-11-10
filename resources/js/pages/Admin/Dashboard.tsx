import { Link, router } from 'inertia-adapter-solid'
import { For } from 'solid-js'
import { StatusPill } from '../../components/admin_ui'
import type { Project, Shared } from '../../types'

export default function Dashboard(props: { stats: Record<string, number>; recent: Project[]; capabilities: NonNullable<Shared['capabilities']> }) {
  const tiles: [string, string][] = [['projects', 'Projects'], ['published', 'Published'], ['categories', 'Categories'], ['neverSynced', 'Never synced'], ['syncErrors', 'Sync errors']]
  return (
    <>
      <h1>Dashboard</h1>
      <ul class="tiles">
        <For each={tiles}>{([k, label]) => <li class="tile"><strong>{props.stats[k]}</strong><span class="muted">{label}</span></li>}</For>
      </ul>
      <section class="panel">
        <h2>Maintenance</h2>
        <div class="actions">
          <button type="button" onClick={() => router.post('/admin/sync')}>Refresh stale GitHub data &amp; scores</button>
          <button type="button" class="secondary" disabled={!props.capabilities.search} onClick={() => router.post('/admin/reindex')}>Rebuild search index</button>
        </div>
        <ul class="small muted caps">
          <li>GitHub token: {props.capabilities.githubToken ? 'set (GraphQL, all branches)' : 'not set (REST only, 60 req/h)'}</li>
          <li>Typesense: {props.capabilities.search ? 'configured' : 'not configured'}</li>
          <li>AI translation: {props.capabilities.translate ? 'enabled' : 'disabled (ANTHROPIC_API_KEY)'}</li>
          <li>Embeddings (pgvector): {props.capabilities.embeddings ? 'enabled' : 'disabled (EMBEDDINGS_URL)'}</li>
        </ul>
      </section>
      <section>
        <h2>Recently added</h2>
        <table class="table">
          <thead><tr><th>Project</th><th>Score</th><th>Status</th></tr></thead>
          <tbody><For each={props.recent}>{p => <tr><td><Link href={`/admin/projects/${p.id}`}>{p.fullName}</Link></td><td>{p.score?.toFixed(1) ?? '–'}</td><td><StatusPill status={p.status} /></td></tr>}</For></tbody>
        </table>
      </section>
    </>
  )
}
