import { Link, router, usePage } from 'inertia-adapter-solid'
import { createEffect, For, Show, type ParentProps } from 'solid-js'
import type { Shared } from '../types'

const nav = [['/admin', 'Dashboard'], ['/admin/projects', 'Projects'], ['/admin/categories', 'Categories']] as const

export default function AdminLayout(props: ParentProps) {
  const page = usePage<any>()
  const shared = () => page.props as Shared
  createEffect(() => { document.title = 'Admin'; document.documentElement.lang = 'en'; document.head.querySelectorAll('[data-seo]:not(title)').forEach(el => el.remove()); const m = document.createElement('meta'); m.setAttribute('data-seo', ''); m.name = 'robots'; m.content = 'noindex,nofollow'; document.head.appendChild(m) })
  const errors = () => Object.entries(shared().errors ?? {})
  return (
    <div class="admin">
      <Show when={shared().auth} fallback={<main class="admin-main admin-narrow">{props.children}</main>}>
        <aside class="admin-side">
          <Link href="/admin" class="brand"><span class="brand-mark" aria-hidden="true">◆</span> Admin</Link>
          <nav aria-label="Admin">
            <For each={nav}>{([href, label]) => <Link href={href} aria-current={(href === '/admin' ? page.url === '/admin' : page.url.startsWith(href)) ? 'page' : undefined}>{label}</Link>}</For>
            <a href="/" target="_blank" rel="noopener">View site ↗</a>
          </nav>
          <div class="admin-user">
            <span class="muted small">{shared().auth?.email}</span>
            <button type="button" class="secondary" onClick={() => router.post('/admin/logout')}>Sign out</button>
          </div>
        </aside>
        <main class="admin-main">
          <Show when={shared().flash?.success}><div class="notice notice-ok" role="status">{shared().flash?.success}</div></Show>
          <Show when={errors().length > 0}><div class="notice notice-err" role="alert"><For each={errors()}>{([, msg]) => <p>{msg}</p>}</For></div></Show>
          {props.children}
        </main>
      </Show>
    </div>
  )
}
