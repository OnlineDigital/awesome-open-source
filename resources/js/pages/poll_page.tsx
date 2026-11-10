import { createSignal, onMount, onCleanup } from 'solid-js'
import { usePage, usePoll } from 'inertia-adapter-solid'
import Snapshot from '../components/page_snapshot'

export default function PollPage() {
  const page = usePage()
  const [running, setRunning] = createSignal(false)
  const poll = usePoll(120, { only: ['tick'] }, { autoStart: false })
  const start = () => { poll.start(); setRunning(true) }
  const stop = () => { poll.stop(); setRunning(false) }
  onMount(() => { window.__lab.poll = { start, stop } })
  onCleanup(() => { window.__lab.poll = null })
  return <><Snapshot name="PollPage" title="Periodic updates with cleanup" description="Polling requests only tick. It starts stopped; the 120 ms interval is deliberately short for this demonstration and is not a recommended production setting." /><section class="demo-section"><h2>Server tick: {String(page.props.tick)}</h2><div class="controls"><button id="poll-start" disabled={running()} onClick={start}>Start polling</button><button id="poll-stop" class="secondary" disabled={!running()} onClick={stop}>Stop polling</button></div><output aria-live="polite">{running() ? 'Polling is active. Navigating to another page stops it.' : 'Polling is stopped.'}</output></section></>
}
