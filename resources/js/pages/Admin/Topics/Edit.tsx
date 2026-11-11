import { router, useForm } from 'inertia-adapter-solid'
import { createSignal, For, Show } from 'solid-js'
import { createStore, produce } from 'solid-js/store'
import { Field } from '../../../components/admin_ui'
import { Link } from '../../../components/link'
import type { Shared } from '../../../types'

interface Topic { id: number; slug: string; published: boolean; sortOrder: number; name: string }
interface Feature { id: number; name: string; translations: Record<string, { name: string; description: string }> }
interface Row { id: number; slug: string; fullName: string; avatar: string; status: string; score: number | null; curated: boolean; github: boolean }
interface Value { state: string; notes: Record<string, string> }
type Trs = Record<string, { name: string; summary: string; body: string }>

const STATES: [string, string][] = [['', '— not rated —'], ['yes', 'Yes'], ['no', 'No'], ['partial', 'Partial'], ['text', 'Text only']]

function Cell(props: { topicId: number; row: Row; feature: Feature; value?: Value; locales: string[]; defaultLocale: string }) {
  const [open, setOpen] = createSignal(false)
  const [notes, setNotes] = createStore<Record<string, string>>({ ...(props.value?.notes ?? {}) })
  const save = (state: string, n: Record<string, string>) => router.put(`/admin/topics/${props.topicId}/values`, { projectId: props.row.id, featureId: props.feature.id, state, notes: { ...n } }, { preserveScroll: true, preserveState: true })
  const state = () => props.value?.state ?? ''
  const hasNote = () => Object.values(notes).some(v => v?.trim())
  return (
    <td>
      <div class="mx-cell">
      <select aria-label={`${props.row.fullName}: ${props.feature.name}`} value={state()} onChange={e => save(e.currentTarget.value, notes)}>
        <For each={STATES}>{([v, label]) => <option value={v}>{label}</option>}</For>
      </select>
      <button type="button" class="secondary mx-note-btn" aria-expanded={open()} onClick={() => setOpen(!open())}>{hasNote() ? 'Note ✎' : 'Note +'}</button>
      <Show when={open()}>
        <div class="mx-notes">
          <For each={props.locales}>{l => (
            <label class="mx-note"><span class="small muted">{l.toUpperCase()}{l === props.defaultLocale ? '' : ''}</span>
              <input value={notes[l] ?? ''} maxLength={500} onInput={e => setNotes(l, e.currentTarget.value)} placeholder={l === props.defaultLocale ? 'e.g. paid first-party plugin' : 'translation (optional)'} />
            </label>
          )}</For>
          <button type="button" onClick={() => { save(state() || (hasNote() ? 'text' : ''), notes); setOpen(false) }}>Save note</button>
        </div>
      </Show>
      </div>
    </td>
  )
}

export default function TopicEdit(props: { topic: Topic; translations: Record<string, { name: string; summary: string; body: string }>; features: Feature[]; projects: Row[]; values: Record<string, Record<string, Value>>; publicPath: string; adminLocales: string[]; defaultLocale: string; capabilities: NonNullable<Shared['capabilities']> }) {
  const t = () => props.topic
  const base = () => `/admin/topics/${t().id}`
  const initial: Trs = {}
  for (const l of props.adminLocales) initial[l] = { name: props.translations[l]?.name ?? '', summary: props.translations[l]?.summary ?? '', body: props.translations[l]?.body ?? '' }
  const form = useForm({ slug: t().slug, published: t().published, sortOrder: t().sortOrder, translations: initial })
  const [locale, setLocale] = createSignal(props.defaultLocale)

  const [features, setFeatures] = createStore<{ id: number; translations: Record<string, { name: string; description: string }> }[]>(
    props.features.map(f => ({ id: f.id, translations: Object.fromEntries(props.adminLocales.map(l => [l, { name: f.translations[l]?.name ?? '', description: f.translations[l]?.description ?? '' }])) })),
  )
  const addFeature = () => setFeatures(produce(list => { list.push({ id: 0, translations: Object.fromEntries(props.adminLocales.map(l => [l, { name: '', description: '' }])) }) }))
  const move = (i: number, d: number) => setFeatures(produce(list => { const j = i + d; if (j < 0 || j >= list.length) return; [list[i], list[j]] = [list[j], list[i]] }))
  const saveFeatures = () => router.put(`${base()}/features`, { features: features.map(f => ({ id: f.id, translations: f.translations })) }, { preserveScroll: true })
  const [addRef, setAddRef] = createSignal('')

  return (
    <>
      <nav class="crumbs"><Link href="/admin/topics">Topics</Link> / <span>{t().name}</span></nav>
      <header class="page-head row">
        <h1>{t().name}</h1>
        <div class="actions">
          <Show when={t().published}><a class="button secondary" href={props.publicPath} target="_blank" rel="noopener">View ↗</a></Show>
          <button type="button" class="secondary" disabled={!props.capabilities.translate} onClick={() => router.post(`${base()}/translate`)} title={props.capabilities.translate ? '' : 'Set ANTHROPIC_API_KEY'}>Translate missing with AI</button>
        </div>
      </header>

      <form class="panel" onSubmit={e => { e.preventDefault(); form.put(base(), { preserveScroll: true }) }}>
        <h2>Page</h2>
        <div class="form-grid">
          <Field label="Slug" for="slug" hint="Projects with this GitHub topic (or added below) are listed." error={(form.errors as Record<string, string>).slug}><input id="slug" value={form.data.slug} onInput={e => form.setData('slug', e.currentTarget.value)} /></Field>
          <Field label="Sort order" for="sort"><input id="sort" type="number" value={form.data.sortOrder} onInput={e => form.setData('sortOrder', Number(e.currentTarget.value))} /></Field>
        </div>
        <label class="check"><input type="checkbox" checked={form.data.published} onChange={e => form.setData('published', e.currentTarget.checked)} /> Published (appears on the site, sitemap and llms.txt)</label>
        <div class="tabs" role="tablist" aria-label="Language">
          <For each={props.adminLocales}>{l => <button type="button" role="tab" aria-selected={locale() === l} class={locale() === l ? '' : 'secondary'} onClick={() => setLocale(l)}>{l.toUpperCase()}{props.translations[l]?.name ? ' ✓' : ''}</button>}</For>
        </div>
        <For each={props.adminLocales}>{l => (
          <Show when={locale() === l}>
            <Field label={`Name${l === props.defaultLocale ? ' (required)' : ''}`} for={`name-${l}`} error={l === props.defaultLocale ? (form.errors as Record<string, string>).name : undefined}><input id={`name-${l}`} value={form.data.translations[l].name} onInput={e => form.setData('translations', l, 'name', e.currentTarget.value)} /></Field>
            <Field label="Summary" for={`sum-${l}`} hint="Shown under the title and used as the meta description (max 400)." error={(form.errors as Record<string, string>).summary}><textarea id={`sum-${l}`} rows="2" value={form.data.translations[l].summary} onInput={e => form.setData('translations', l, 'summary', e.currentTarget.value)} /></Field>
            <Field label="SEO text" for={`body-${l}`} hint="Paragraphs separated by a blank line." error={(form.errors as Record<string, string>).body}><textarea id={`body-${l}`} rows="9" value={form.data.translations[l].body} onInput={e => form.setData('translations', l, 'body', e.currentTarget.value)} /></Field>
          </Show>
        )}</For>
        <div class="actions">
          <button type="submit" disabled={form.processing}>Save page</button>
          <button type="button" class="danger-btn" onClick={() => { if (confirm(`Delete topic "${t().name}" with its features and values?`)) router.delete(base()) }}>Delete topic</button>
        </div>
      </form>

      <section class="panel">
        <h2>Features</h2>
        <p class="muted small">The rows of the comparison table. Names and descriptions are per language; new rows get an id when saved. Removing a row deletes its values.</p>
        <For each={features}>{(f, i) => (
          <fieldset class="feature-row">
            <legend>#{i() + 1}</legend>
            <div class="form-grid">
              <For each={props.adminLocales}>{l => (
                <div>
                  <Field label={`Name · ${l.toUpperCase()}${l === props.defaultLocale ? ' *' : ''}`}><input value={f.translations[l].name} onInput={e => setFeatures(i(), 'translations', l, 'name', e.currentTarget.value)} placeholder={l === props.defaultLocale ? 'Custom fields' : ''} /></Field>
                  <Show when={l === props.defaultLocale}><Field label="Description (optional)"><input value={f.translations[l].description} onInput={e => setFeatures(i(), 'translations', l, 'description', e.currentTarget.value)} /></Field></Show>
                </div>
              )}</For>
            </div>
            <div class="actions">
              <button type="button" class="secondary" onClick={() => move(i(), -1)} disabled={i() === 0} aria-label="Move up">↑</button>
              <button type="button" class="secondary" onClick={() => move(i(), 1)} disabled={i() === features.length - 1} aria-label="Move down">↓</button>
              <button type="button" class="danger-btn" onClick={() => setFeatures(produce(list => { list.splice(i(), 1) }))}>Remove</button>
            </div>
          </fieldset>
        )}</For>
        <div class="actions">
          <button type="button" class="secondary" onClick={addFeature}>+ Add feature</button>
          <button type="button" onClick={saveFeatures}>Save features</button>
        </div>
      </section>

      <section class="panel">
        <h2>Values per project</h2>
        <p class="muted small">Choose Yes / No / Partial, or add a note per language (for example "paid first-party plugin"). "Text only" shows just the note. Changes save immediately.</p>
        <Show when={props.features.length > 0 && props.projects.length > 0} fallback={<p class="empty">Save at least one feature and add projects to rate them.</p>}>
          <div class="mx-scroll">
            <table class="table mx">
              <thead><tr><th>Project</th><For each={props.features}>{f => <th>{f.name}</th>}</For></tr></thead>
              <tbody>
                <For each={props.projects}>{r => (
                  <tr>
                    <th scope="row"><Link href={`/admin/projects/${r.id}`}>{r.fullName}</Link> <Show when={r.status !== 'published'}><span class="pill pill-draft">draft</span></Show></th>
                    <For each={props.features}>{f => <Cell topicId={t().id} row={r} feature={f} value={props.values[String(r.id)]?.[String(f.id)]} locales={props.adminLocales} defaultLocale={props.defaultLocale} />}</For>
                  </tr>
                )}</For>
              </tbody>
            </table>
          </div>
        </Show>
      </section>

      <section class="panel">
        <h2>Projects in this topic</h2>
        <p class="muted small">Projects carrying the GitHub topic <code>{t().slug}</code> are included automatically. Curated additions are stored separately, so a GitHub re-sync never removes them.</p>
        <ul class="shot-list">
          <For each={props.projects}>{r => (
            <li class="shot">
              <span><Link href={`/admin/projects/${r.id}`}>{r.fullName}</Link></span>
              <span>
                <Show when={r.github}><span class="pill pill-ok">GitHub topic</span> </Show>
                <Show when={r.curated}><span class="pill">added by hand</span></Show>
              </span>
              <Show when={r.curated}><button type="button" class="secondary" onClick={() => router.delete(`${base()}/projects/${r.id}`, { preserveScroll: true })}>Remove</button></Show>
            </li>
          )}</For>
        </ul>
        <form class="shot" onSubmit={e => { e.preventDefault(); router.post(`${base()}/projects`, { project: addRef() }, { preserveScroll: true, onSuccess: () => setAddRef('') }) }}>
          <input aria-label="Project to add" value={addRef()} onInput={e => setAddRef(e.currentTarget.value)} placeholder="owner/repo, slug or id" />
          <button type="submit" disabled={!addRef().trim()}>Add project</button>
        </form>
      </section>
    </>
  )
}
