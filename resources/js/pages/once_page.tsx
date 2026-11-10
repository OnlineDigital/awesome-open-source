import { router } from '@inertiajs/core'
import { Link } from 'inertia-adapter-solid'
import Snapshot from '../components/page_snapshot'
import EvaluationCounters from '../components/evaluation_counters'

export default function OncePage() {
  return <><Snapshot name="OncePage" title="Once: reuse the lookup" description="The US / RO list is evaluated once while you navigate between pages that declare the same once prop. An explicit only or server-side Fresh requests a new value." /><section class="demo-section"><h2>Reuse and invalidation</h2><div class="controls"><Link id="once-next" href="/once-other">Navigate within the same section</Link><Link href="/once">First once page</Link><button onClick={() => router.reload({ only: ['lookup'] })}>Explicitly reload the lookup</button><Link href="/once?fresh=1">Request Fresh on the server</Link><Link href="/other">Leave this section</Link></div><p>After leaving the section, returning recalculates the lookup. The counter shows exactly when the function ran.</p></section><EvaluationCounters /></>
}
