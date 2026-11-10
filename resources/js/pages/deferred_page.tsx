import { Deferred, usePage } from 'inertia-adapter-solid'
import Snapshot from '../components/page_snapshot'
import EvaluationCounters from '../components/evaluation_counters'

export default function DeferredPage() {
  const page = usePage()
  return <><Snapshot name="DeferredPage" title="Load data after the initial page" description="The initial HTML response declares two deferred groups. The frontend requests them separately and replaces each fallback when its data is available." /><section class="demo-section"><h2>Default group</h2><Deferred data="slow" fallback={<p id="fallback" role="status">Loading</p>}><p id="slow">{String(page.props.slow)}</p></Deferred><h2>Extra group</h2><Deferred data="second" fallback={<p id="second-fallback" role="status">Loading second</p>}><p id="second">{String(page.props.second)}</p></Deferred></section><EvaluationCounters /></>
}
