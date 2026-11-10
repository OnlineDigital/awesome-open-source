import { createSignal } from 'solid-js'
import { router, config } from '@inertiajs/core'

declare global { interface Window { __lab: any } }

export const events: string[] = []
export const [eventLog, setEventLog] = createSignal<string[]>([])
export const [notice, setNotice] = createSignal('')
window.__lab = { router, config, events, boot: Math.random(), page: null, form: null, poll: null }
for (const name of ['before', 'start', 'success', 'error', 'finish', 'navigate', 'prefetching', 'prefetched'] as const) {
  router.on(name, () => {
    events.push(name)
    if (events.length > 100) events.splice(0, events.length - 100)
    setEventLog([...events])
  })
}
router.on('networkError', () => { setNotice('The connection failed. Check your network and try the action again.'); return false })
router.on('httpException', event => { setNotice(`The server responded with HTTP ${event.detail.response.status}. Try again or open a page from the menu.`); return false })
