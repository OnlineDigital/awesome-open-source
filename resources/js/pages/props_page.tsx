import { createSignal } from 'solid-js'
import { router } from '@inertiajs/core'
import { Link } from 'inertia-adapter-solid'
import Snapshot from '../components/page_snapshot'
import EvaluationCounters from '../components/evaluation_counters'

export default function Props() {
  const [local, setLocal] = createSignal('initial')
  return <><Snapshot name="Props" title="Request only the data you need" description="only selects props; except excludes props. Excluded lazy functions do not run, and the browser preserves the other data it already received." />
    <section class="demo-section"><h2>Partial reload</h2><div class="controls"><button id="partial-only-users" onClick={() => router.reload({ only: ['users'] })}>Only users</button><button id="partial-except-companies" onClick={() => router.reload({ except: ['companies'] })}>Except companies</button><button id="partial-nested" onClick={() => router.reload({ only: ['auth.notifications'] })}>Only auth.notifications</button><button id="partial-optional" onClick={() => router.reload({ only: ['optional'] })}>Request the optional prop</button><Link id="only-link" href="/props" only={['users']} preserveState>The same only through Link</Link></div><p>Optional is initially absent. Always is included in partial reloads as well. Ordinary callbacks use the template's local lazy resolver.</p></section>
    <section class="demo-section"><h2>Local state and scroll</h2><label for="local-state">Value preserved by router.reload</label><input id="local-state" value={local()} onInput={event => setLocal(event.currentTarget.value)} /><div class="controls"><button class="secondary" onClick={() => router.visit('/props', { preserveState: false })}>New visit: reset local state</button></div></section>
    <EvaluationCounters /><div class="scroll-fixture">Scroll preservation demo: scroll down, then request only users. The position remains unchanged.</div>
  </>
}
