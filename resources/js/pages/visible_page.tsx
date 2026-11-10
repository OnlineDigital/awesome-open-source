import { usePage, WhenVisible } from 'inertia-adapter-solid'
import Snapshot from '../components/page_snapshot'

export default function VisiblePage() {
  const page = usePage()
  return <><Snapshot name="VisiblePage" title="Load when visible" description="The optional prop is not evaluated on entry. Scroll to the area below: WhenVisible requests its data when the area enters the viewport." /><div class="visibility-distance">Scroll to the area that triggers the request. The distance is intentional for this demonstration.</div><section class="demo-section"><h2>Observed area</h2><WhenVisible data="optional" fallback={<p id="visible-fallback" role="status">Waiting</p>}><p id="visible-value">{String(page.props.optional)}</p></WhenVisible></section></>
}
