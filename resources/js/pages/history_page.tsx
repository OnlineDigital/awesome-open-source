import { router } from '@inertiajs/core'
import { Link, usePage } from 'inertia-adapter-solid'
import { eventLog, events, setEventLog } from '../lib/demo_state'
import Snapshot from '../components/page_snapshot'

export default function HistoryPage() {
  const page = usePage<any>()
  const local = () => router.replace({ url: page.url, component: 'HistoryPage', props: { ...page.props, localLabel: 'Initial', localItems: [1] } })
  return <><Snapshot name="HistoryPage" title="History, events, and local props" description="The server enables encryptHistory on this page. The core encrypts history data; Back restores it. The local controls modify props in the browser without making requests." /><section class="demo-section"><h2>Encrypted history</h2><div class="controls"><Link href="/other">Navigate away, then return with Back</Link><Link href="/clear-history">ClearHistory on the server</Link></div><p>History encryption does not replace server-side data authorization.</p></section>
    <section class="demo-section"><h2>Local changes through the core</h2><div class="controls"><button onClick={local}>Initialize local props</button><button class="secondary" disabled={!page.props.localItems} onClick={() => router.replaceProp('localLabel', 'Changed')}>Replace label</button><button class="secondary" disabled={!page.props.localItems} onClick={() => router.appendToProp('localItems', 2)}>Append 2</button><button class="secondary" disabled={!page.props.localItems} onClick={() => router.prependToProp('localItems', 0)}>Prepend 0</button></div><pre>{JSON.stringify({ label: page.props.localLabel, items: page.props.localItems }, null, 2)}</pre><div class="controls"><button class="secondary" onClick={() => router.push({ url: '/local', component: 'Other', props: { label: 'local', items: [1] } })}>Client push to Other</button></div><p>The /local URL demonstrates a client-only visit and should not be used as a direct link.</p></section>
    <section class="demo-section"><h2>Router events</h2><p>The list is limited to the last 100 events to prevent unbounded memory growth.</p><output id="event-log" aria-live="polite">{eventLog().join(' → ') || 'No events.'}</output><button class="secondary" onClick={() => { events.length = 0; setEventLog([]) }}>Clear the list</button></section>
  </>
}
