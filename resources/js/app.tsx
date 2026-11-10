import { render } from 'solid-js/web'
import { createInertiaApp } from 'inertia-adapter-solid'
import './lib/demo_state'
import '../css/app.css'
import DemoLayout from './layouts/demo_layout'
import Home from './pages/home_page'
import Other from './pages/other_page'
import Props from './pages/props_page'
import DeferredPage from './pages/deferred_page'
import OncePage from './pages/once_page'
import MergePage from './pages/merge_page'
import DeepPage from './pages/deep_page'
import ScrollPage from './pages/scroll_page'
import VisiblePage from './pages/visible_page'
import PollPage from './pages/poll_page'
import FormPage from './pages/form_page'
import HistoryPage from './pages/history_page'
import UserPage from './pages/user_page'

const pages: Record<string, any> = { Home, Other, Props, DeferredPage, OncePage, MergePage, DeepPage, ScrollPage, VisiblePage, PollPage, FormPage, HistoryPage, 'Users/Show': UserPage }
Object.values(pages).forEach(page => { page.layout = DemoLayout })
const cache = new URLSearchParams(location.search).get('cache')
// The server emits the v3 JSON script bootstrap. No legacy data-page bridge.
createInertiaApp({
  defaults: cache ? { prefetch: { cacheFor: Number(cache) } } : {},
  resolve: name => {
    if (!pages[name]) throw new Error(`Unknown Inertia component: ${name}`)
    return pages[name]
  },
  setup: ({ el, App, props }) => { render(() => <App {...props} />, el); window.__lab.ready = true },
})
