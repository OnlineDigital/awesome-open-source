import { Link, router, useForm } from 'inertia-adapter-solid'
import { createSignal, For, Show } from 'solid-js'
import { Field, Select } from '../../../components/admin_ui'
import { ScoreBreakdown } from '../../../components/score_breakdown'
import type { Activity, Project, Shared } from '../../../types'

interface Tr { locale: string; tagline: string; description: string; source: string }
interface Cat { id: number; name: string }

export default function ProjectEdit(props: { project: Project; translations: Record<string, Tr>; categories: Cat[]; activity: Activity | null; publicPath: string; adminLocales: string[]; defaultLocale: string; capabilities: NonNullable<Shared['capabilities']> }) {
  const p = () => props.project
  const settings = useForm({ categoryId: p().categoryId ?? '', youtube: p().youtubeId, status: p().status as string, featured: p().featured, homepage: p().homepage })
  const initialTexts: Record<string, { tagline: string; description: string }> = {}
  for (const l of props.adminLocales) initialTexts[l] = { tagline: props.translations[l]?.tagline ?? '', description: props.translations[l]?.description ?? '' }
  const texts = useForm({ translations: initialTexts })
  const [locale, setLocale] = createSignal(props.defaultLocale)
  const shot = useForm({ file: null as File | null, alt: '' })
  const base = () => `/admin/projects/${p().id}`
  const machine = (l: string) => props.translations[l]?.source === 'machine'

  return (
    <>
      <nav class="crumbs"><Link href="/admin/projects">Projects</Link> / <span>{p().fullName}</span></nav>
      <header class="page-head row">
        <h1>{p().fullName}</h1>
        <div class="actions">
          <Show when={p().status === 'published'}><a class="button secondary" href={props.publicPath} target="_blank" rel="noopener">View ↗</a></Show>
          <button type="button" onClick={() => router.post(`${base()}/sync`)}>Sync with GitHub</button>
        </div>
      </header>
      <Show when={p().syncError}><div class="notice notice-err">Last sync failed: {p().syncError}</div></Show>

      <form class="panel" onSubmit={e => { e.preventDefault(); settings.transform(d => ({ ...d, categoryId: Number(d.categoryId) || 0 })).put(base()) }}>
        <h2>Publishing</h2>
        <div class="form-grid">
          <Field label="Status" for="status"><Select id="status" value={settings.data.status} onChange={v => settings.setData('status', v)} options={[['draft', 'Draft (hidden)'], ['published', 'Published']]} /></Field>
          <Field label="Category" for="category"><Select id="category" value={settings.data.categoryId} onChange={v => settings.setData('categoryId', v)} empty="— none —" options={props.categories.map(c => [c.id, c.name])} /></Field>
          <Field label="YouTube demo" for="youtube" hint="Video URL or 11-character id" error={settings.errors.youtube}><input id="youtube" value={settings.data.youtube} onInput={e => settings.setData('youtube', e.currentTarget.value)} placeholder="https://youtu.be/…" /></Field>
          <Field label="Website" for="homepage" error={settings.errors.homepage}><input id="homepage" type="url" value={settings.data.homepage} onInput={e => settings.setData('homepage', e.currentTarget.value)} /></Field>
        </div>
        <label class="check"><input type="checkbox" checked={settings.data.featured} onChange={e => settings.setData('featured', e.currentTarget.checked)} /> Featured</label>
        <div class="actions"><button type="submit" disabled={settings.processing}>Save</button></div>
      </form>

      <form class="panel" onSubmit={e => { e.preventDefault(); texts.put(`${base()}/translations`) }}>
        <h2>Texts &amp; translations</h2>
        <div class="tabs" role="tablist">
          <For each={props.adminLocales}>{l => <button type="button" role="tab" aria-selected={locale() === l} class={`pill ${locale() === l ? 'pill-active' : ''}`} onClick={() => setLocale(l)}>{l.toUpperCase()}{l === props.defaultLocale ? ' (source)' : machine(l) ? ' (AI)' : ''}</button>}</For>
        </div>
        <For each={props.adminLocales}>{l => (
          <Show when={locale() === l}>
            <Field label="Tagline" for={`tag-${l}`} hint="One sentence, shown in lists and search results (max 300)">
              <input id={`tag-${l}`} maxLength={300} value={texts.data.translations[l].tagline} onInput={e => texts.setData('translations', l, 'tagline', e.currentTarget.value)} />
            </Field>
            <Field label="Description" for={`desc-${l}`} hint="Plain text; blank lines make paragraphs. Used on the page, in schema.org data and llms .md files.">
              <textarea id={`desc-${l}`} rows="8" value={texts.data.translations[l].description} onInput={e => texts.setData('translations', l, 'description', e.currentTarget.value)} />
            </Field>
          </Show>
        )}</For>
        <div class="actions">
          <button type="submit" disabled={texts.processing}>Save texts</button>
          <button type="button" class="secondary" disabled={!props.capabilities.translate} title={props.capabilities.translate ? '' : 'Set ANTHROPIC_API_KEY'} onClick={() => router.post(`${base()}/translate`)}>Translate missing with AI</button>
        </div>
        <p class="muted small">Edited translations are marked as human-written and are never overwritten by AI translation.</p>
      </form>

      <section class="panel">
        <h2>Screenshots</h2>
        <ul class="shot-list">
          <For each={p().screenshots ?? []} fallback={<li class="muted">No screenshots yet.</li>}>{s => <ShotRow shot={s} base={base()} />}</For>
        </ul>
        <form class="inline-form" onSubmit={e => { e.preventDefault(); shot.post(`${base()}/screenshots`, { forceFormData: true, onSuccess: () => shot.reset() }) }}>
          <Field label="Add screenshot (PNG, JPEG, WebP, GIF · max 5 MB)" for="file"><input id="file" type="file" accept="image/png,image/jpeg,image/webp,image/gif" onChange={e => shot.setData('file', e.currentTarget.files?.[0] ?? null)} /></Field>
          <Field label="Alt text" for="alt"><input id="alt" value={shot.data.alt} onInput={e => shot.setData('alt', e.currentTarget.value)} placeholder="What the screenshot shows" /></Field>
          <button type="submit" disabled={!shot.data.file || shot.processing}>Upload</button>
        </form>
      </section>

      <Show when={props.activity}>{a => <ScoreBreakdown activity={a()} />}</Show>

      <section class="panel danger">
        <h2>Delete</h2>
        <button type="button" class="danger-btn" onClick={() => { if (confirm(`Delete ${p().fullName}? This removes its texts and screenshots.`)) router.delete(base()) }}>Delete project</button>
      </section>
    </>
  )
}

function ShotRow(props: { shot: { id: number; path: string; alt: string; position: number }; base: string }) {
  const f = useForm({ alt: props.shot.alt, position: props.shot.position })
  const url = `${props.base}/screenshots/${props.shot.id}`
  return (
    <li class="shot">
      <img src={props.shot.path} alt={props.shot.alt} width="120" />
      <form onSubmit={e => { e.preventDefault(); f.put(url) }}>
        <input aria-label="Alt text" value={f.data.alt} onInput={e => f.setData('alt', e.currentTarget.value)} />
        <input aria-label="Position" type="number" class="narrow" value={f.data.position} onInput={e => f.setData('position', Number(e.currentTarget.value))} />
        <button type="submit" class="secondary">Save</button>
        <button type="button" class="danger-btn" onClick={() => router.delete(url)}>Remove</button>
      </form>
    </li>
  )
}
