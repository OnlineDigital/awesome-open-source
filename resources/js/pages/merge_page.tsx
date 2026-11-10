import { router } from '@inertiajs/core'
import { usePage } from 'inertia-adapter-solid'
import { For } from 'solid-js'
import Snapshot from '../components/page_snapshot'

export default function MergePage() {
  const page = usePage<any>()
  return <><Snapshot name="MergePage" title="Append pages without duplicates" description="Merge append with MatchOn(id) appends items during partial reloads and replaces an item with the same id. Reset or a full visit replaces the list." /><section class="demo-section"><h2>Accumulated list</h2><ul class="item-list"><For each={page.props.items ?? []}>{item => <li>{item.id} · {item.name}</li>}</For></ul><div class="controls"><button onClick={() => router.visit('/merge?page=2', { only: ['items'], preserveState: true })}>Append page 2</button><button class="secondary" onClick={() => router.visit('/merge?page=3', { only: ['items'], reset: ['items'], preserveState: true })}>Reset with page 3</button><button class="secondary" onClick={() => router.visit('/merge?page=1')}>Full visit: page 1</button></div></section></>
}
