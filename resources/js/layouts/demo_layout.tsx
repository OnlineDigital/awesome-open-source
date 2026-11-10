import { createSignal, For, Show, type ParentProps } from 'solid-js'
import { Link, usePage } from 'inertia-adapter-solid'
import { notice, setNotice } from '../lib/demo_state'

const navigation = [
  ['/home', 'Navigation & prefetch', 'nav-home'], ['/other', 'Secondary page', 'nav-other'],
  ['/users/37', 'Controller & route parameter'],
  ['/props', 'Partial reload & lazy props'], ['/deferred', 'Deferred groups'], ['/once', 'Once props'],
  ['/merge', 'Merge & reset'], ['/deep', 'Deep merge'], ['/scroll', 'Manual pagination'],
  ['/visible', 'WhenVisible'], ['/poll', 'Polling'], ['/form', 'Forms & JSON'], ['/history', 'History & client props'],
]
export default function DemoLayout(props: ParentProps) {
  const page = usePage()
  const [counter, setCounter] = createSignal(0)
  return <div class="app-shell">
    <a href="#demo-content" class="skip-link">Skip to content</a>
    <aside class="sidebar">
      <Link href="/home" class="brand">inertia-go <span>+ Solid</span></Link>
      <p class="sidebar-description">A reusable template with examples you can try directly.</p>
      <nav aria-label="Demonstrations"><For each={navigation}>{([href, label, id]) => <Link href={href} id={id} aria-current={page.url.split('?')[0] === href ? 'page' : undefined}>{label}</Link>}</For></nav>
      <div class="layout-state"><label for="layout-counter">Persistent layout state</label><button id="layout-counter" class="secondary" onClick={() => setCounter(counter() + 1)}>{counter()}</button><p>Navigation preserves this counter.</p></div>
    </aside>
    <main id="demo-content" class="content" tabindex="-1">
      <Show when={notice()}><div role="alert" class="notice"><p>{notice()}</p><button class="secondary" onClick={() => setNotice('')}>Dismiss message</button></div></Show>
      {props.children}
      <footer>Solid adapter beta.9 · Inertia core v3 · inertia-go v0.10.0. Limitations and local adapters are documented in the project skill.</footer>
    </main>
  </div>
}
