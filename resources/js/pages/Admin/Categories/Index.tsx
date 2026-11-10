import { router, useForm } from 'inertia-adapter-solid'
import { createSignal, For, Show } from 'solid-js'
import { Field } from '../../../components/admin_ui'
import type { Shared } from '../../../types'

interface Cat { id: number; slug: string; icon: string; sortOrder: number; name: string; projectCount: number }
type Trs = Record<string, { name: string; description: string }>

function emptyTrs(locales: string[], src?: Record<string, { name: string; description: string }>): Trs {
  const t: Trs = {}
  for (const l of locales) t[l] = { name: src?.[l]?.name ?? '', description: src?.[l]?.description ?? '' }
  return t
}

function CategoryForm(props: { cat?: Cat; trs?: Record<string, { name: string; description: string }>; locales: string[]; defaultLocale: string; translate: boolean }) {
  const form = useForm({ slug: props.cat?.slug ?? '', icon: props.cat?.icon ?? '', sortOrder: props.cat?.sortOrder ?? 0, translations: emptyTrs(props.locales, props.trs) })
  const [open, setOpen] = createSignal(!props.cat)
  const url = () => (props.cat ? `/admin/categories/${props.cat.id}` : '/admin/categories')
  const submit = (e: SubmitEvent) => { e.preventDefault(); const opts = { onSuccess: () => { if (!props.cat) form.reset() } }; props.cat ? form.put(url(), opts) : form.post(url(), opts) }
  return (
    <form class="panel" onSubmit={submit}>
      <div class="row">
        <h3>{props.cat ? <button type="button" class="linklike" aria-expanded={open()} onClick={() => setOpen(!open())}>{props.cat.name} <span class="muted small">({props.cat.projectCount})</span></button> : 'New category'}</h3>
      </div>
      <Show when={open()}>
        <div class="form-grid">
          <Field label="Slug" for={`slug-${props.cat?.id ?? 'new'}`} hint="URL segment; generated from the name when empty" error={form.errors.slug}><input id={`slug-${props.cat?.id ?? 'new'}`} value={form.data.slug} onInput={e => form.setData('slug', e.currentTarget.value)} /></Field>
          <Field label="Icon" for={`icon-${props.cat?.id ?? 'new'}`} hint="Emoji or short text"><input id={`icon-${props.cat?.id ?? 'new'}`} value={form.data.icon} onInput={e => form.setData('icon', e.currentTarget.value)} /></Field>
          <Field label="Sort order" for={`sort-${props.cat?.id ?? 'new'}`}><input id={`sort-${props.cat?.id ?? 'new'}`} type="number" value={form.data.sortOrder} onInput={e => form.setData('sortOrder', Number(e.currentTarget.value))} /></Field>
        </div>
        <For each={props.locales}>{l => (
          <fieldset>
            <legend>{l.toUpperCase()}{l === props.defaultLocale ? ' (required)' : ''}</legend>
            <Field label="Name" for={`n-${props.cat?.id ?? 'new'}-${l}`} error={l === props.defaultLocale ? (form.errors as Record<string, string>).name : undefined}><input id={`n-${props.cat?.id ?? 'new'}-${l}`} value={form.data.translations[l].name} onInput={e => form.setData('translations', l, 'name', e.currentTarget.value)} /></Field>
            <Field label="Description" for={`d-${props.cat?.id ?? 'new'}-${l}`}><textarea id={`d-${props.cat?.id ?? 'new'}-${l}`} rows="2" value={form.data.translations[l].description} onInput={e => form.setData('translations', l, 'description', e.currentTarget.value)} /></Field>
          </fieldset>
        )}</For>
        <div class="actions">
          <button type="submit" disabled={form.processing}>{props.cat ? 'Save' : 'Create'}</button>
          <Show when={props.cat}>
            <button type="button" class="secondary" disabled={!props.translate} onClick={() => router.post(`/admin/categories/${props.cat!.id}/translate`)}>Translate missing with AI</button>
            <button type="button" class="danger-btn" onClick={() => { if (confirm(`Delete category "${props.cat!.name}"? Its projects become uncategorized.`)) router.delete(url()) }}>Delete</button>
          </Show>
        </div>
      </Show>
    </form>
  )
}

export default function CategoriesIndex(props: { categories: Cat[]; translations: Record<number, Record<string, { name: string; description: string }>>; adminLocales: string[]; defaultLocale: string; capabilities: NonNullable<Shared['capabilities']> }) {
  return (
    <>
      <h1>Categories</h1>
      <CategoryForm locales={props.adminLocales} defaultLocale={props.defaultLocale} translate={props.capabilities.translate} />
      <For each={props.categories}>{c => <CategoryForm cat={c} trs={props.translations[c.id]} locales={props.adminLocales} defaultLocale={props.defaultLocale} translate={props.capabilities.translate} />}</For>
    </>
  )
}
