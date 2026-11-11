import { useForm } from 'inertia-adapter-solid'
import { For, Show } from 'solid-js'
import { Link } from '../../../components/link'
import { Field } from '../../../components/admin_ui'

interface Topic { id: number; slug: string; name: string; published: boolean; projectCount: number; translated: boolean }

export default function TopicsIndex(props: { topics: Topic[] }) {
  const form = useForm({ slug: '', name: '' })
  return (
    <>
      <h1>Topics</h1>
      <p class="muted">A topic page compares the projects that carry one GitHub topic (or that you add by hand) in a feature table. Use the topic's GitHub slug, e.g. <code>laravel-admin-panel</code>.</p>
      <form class="panel" onSubmit={e => { e.preventDefault(); form.post('/admin/topics', { onSuccess: () => form.reset() }) }}>
        <h2>New topic</h2>
        <div class="form-grid">
          <Field label="Name (default language)" for="tname" error={(form.errors as Record<string, string>).name}><input id="tname" value={form.data.name} onInput={e => form.setData('name', e.currentTarget.value)} placeholder="Laravel admin panels" /></Field>
          <Field label="Slug" for="tslug" hint="Matches the GitHub topic. Generated from the name when empty." error={(form.errors as Record<string, string>).slug}><input id="tslug" value={form.data.slug} onInput={e => form.setData('slug', e.currentTarget.value)} placeholder="laravel-admin-panel" /></Field>
        </div>
        <button type="submit" disabled={form.processing}>Create topic</button>
      </form>
      <Show when={props.topics.length > 0} fallback={<p class="empty">No topics yet.</p>}>
        <table class="table">
          <thead><tr><th>Name</th><th>Slug</th><th>Projects</th><th>Status</th></tr></thead>
          <tbody>
            <For each={props.topics}>{t => (
              <tr>
                <td><Link href={`/admin/topics/${t.id}`}><strong>{t.name}</strong></Link></td>
                <td><code>{t.slug}</code></td>
                <td>{t.projectCount}</td>
                <td><span class={`pill ${t.published ? 'pill-ok' : 'pill-draft'}`}>{t.published ? 'published' : 'draft'}</span></td>
              </tr>
            )}</For>
          </tbody>
        </table>
      </Show>
    </>
  )
}
