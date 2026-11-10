import { For, Show, type JSX } from 'solid-js'

export function Field(props: { label: string; error?: string; hint?: string; children: JSX.Element; for?: string }) {
  return (
    <div class="field">
      <label for={props.for}>{props.label}</label>
      {props.children}
      <Show when={props.hint}><span class="muted small">{props.hint}</span></Show>
      <Show when={props.error}><span class="field-error" role="alert">{props.error}</span></Show>
    </div>
  )
}

export function StatusPill(props: { status: string }) {
  return <span class={`pill ${props.status === 'published' ? 'pill-ok' : 'pill-draft'}`}>{props.status}</span>
}

export function Select(props: { id?: string; value: string | number; onChange: (v: string) => void; options: [string | number, string][]; empty?: string }) {
  return (
    <select id={props.id} value={String(props.value)} onChange={e => props.onChange(e.currentTarget.value)}>
      <Show when={props.empty !== undefined}><option value="">{props.empty}</option></Show>
      <For each={props.options}>{([v, label]) => <option value={String(v)}>{label}</option>}</For>
    </select>
  )
}
