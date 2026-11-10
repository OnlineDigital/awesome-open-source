import { router } from '@inertiajs/core'
import Snapshot from '../components/page_snapshot'
import { usePage } from 'inertia-adapter-solid'

export default function DeepPage() {
  const page = usePage()
  return <><Snapshot name="DeepPage" title="Deep merge for nested objects" description="DeepMerge combines the tree structure and its nested arrays during partial reloads. This example uses tree.items; inspect the resulting object after each request." /><section class="demo-section"><h2>Received tree</h2><pre>{JSON.stringify(page.props.tree, null, 2)}</pre><div class="controls"><button onClick={() => router.visit('/deep?page=2', { only: ['tree'], preserveState: true })}>Merge page 2</button><button class="secondary" onClick={() => router.visit('/deep?page=1')}>Reset with a full visit</button></div></section></>
}
