import { router, useForm } from 'inertia-adapter-solid'
import { createSignal, For, Show } from 'solid-js'
import { Field } from '../../components/admin_ui'

interface McpLink { id: number; name: string; token: string; sseUrl: string; httpUrl: string; active: boolean; createdAt: string; rotatedAt: string | null; lastUsedAt: string | null; revokedAt: string | null }

const fmt = (iso: string | null) => (iso ? new Date(iso).toLocaleString() : 'never')

function CopyField(props: { label: string; value: string; display: string }) {
  const [copied, setCopied] = createSignal(false)
  return (
    <Field label={props.label}>
      <div class="copy">
        <input readonly value={props.display} onFocus={e => e.currentTarget.select()} aria-label={props.label} />
        <button type="button" class="secondary" onClick={async () => { await navigator.clipboard?.writeText(props.value); setCopied(true); setTimeout(() => setCopied(false), 1500) }}>{copied() ? 'Copied' : 'Copy'}</button>
      </div>
    </Field>
  )
}

function LinkCard(props: { link: McpLink }) {
  const [name, setName] = createSignal(props.link.name)
  const [reveal, setReveal] = createSignal(false)
  const l = () => props.link
  const mask = (u: string) => (reveal() ? u : u.replace(l().token, `${l().token.slice(0, 4)}••••••••••••${l().token.slice(-4)}`))
  return (
    <section class={`panel ${l().active ? '' : 'danger'}`}>
      <form class="row" onSubmit={e => { e.preventDefault(); router.put(`/admin/settings/mcp-tokens/${l().id}`, { name: name() }, { preserveScroll: true }) }}>
        <div class="copy">
          <input aria-label="Link name" value={name()} onInput={e => setName(e.currentTarget.value)} />
          <button type="submit" class="secondary" disabled={name().trim() === l().name}>Rename</button>
        </div>
        <span class={`pill ${l().active ? 'pill-ok' : 'pill-draft'}`}>{l().active ? 'active' : 'revoked'}</span>
      </form>
      <p class="muted small">Created {fmt(l().createdAt)} · Last used {fmt(l().lastUsedAt)}<Show when={l().rotatedAt}> · Rotated {fmt(l().rotatedAt)}</Show><Show when={l().revokedAt}> · Revoked {fmt(l().revokedAt)}</Show></p>
      <div class="form-grid">
        <CopyField label="SSE URL (most clients)" value={l().sseUrl} display={mask(l().sseUrl)} />
        <CopyField label="Streamable HTTP URL" value={l().httpUrl} display={mask(l().httpUrl)} />
      </div>
      <Show when={reveal()}>
        <details open>
          <summary>Add to Claude Code</summary>
          <pre class="snippet">{`claude mcp add --transport sse aos-admin ${l().sseUrl}`}</pre>
        </details>
      </Show>
      <div class="actions">
        <button type="button" class="secondary" onClick={() => setReveal(!reveal())}>{reveal() ? 'Hide token' : 'Show full link'}</button>
        <button type="button" class="secondary" onClick={() => { if (confirm('Rotate this token? The current URL stops working immediately.')) router.post(`/admin/settings/mcp-tokens/${l().id}/rotate`, {}, { preserveScroll: true }) }}>Rotate</button>
        <Show when={l().active}><button type="button" class="secondary" onClick={() => { if (confirm('Revoke this link? Connected agents lose access immediately.')) router.post(`/admin/settings/mcp-tokens/${l().id}/revoke`, {}, { preserveScroll: true }) }}>Revoke</button></Show>
        <button type="button" class="danger-btn" onClick={() => { if (confirm('Delete this link permanently?')) router.delete(`/admin/settings/mcp-tokens/${l().id}`, { preserveScroll: true }) }}>Delete</button>
      </div>
    </section>
  )
}

export default function Settings(props: { links: McpLink[] }) {
  const form = useForm({ name: '' })
  return (
    <>
      <h1>Settings</h1>
      <h2>MCP access</h2>
      <p class="muted">Each link lets an AI agent manage the whole catalog (projects, categories, topics, comparison tables, texts, screenshots) through the Model Context Protocol. The secret is part of the URL, so treat it like a password: no other authentication is needed.</p>
      <form class="panel" onSubmit={e => { e.preventDefault(); form.post('/admin/settings/mcp-tokens', { preserveScroll: true, onSuccess: () => form.reset() }) }}>
        <div class="copy">
          <input aria-label="New link name" value={form.data.name} onInput={e => form.setData('name', e.currentTarget.value)} placeholder="Name, e.g. Claude Code on my laptop" />
          <button type="submit" disabled={form.processing}>Generate link</button>
        </div>
      </form>
      <Show when={props.links.length > 0} fallback={<p class="empty">No links yet.</p>}>
        <For each={props.links}>{l => <LinkCard link={l} />}</For>
      </Show>
    </>
  )
}
