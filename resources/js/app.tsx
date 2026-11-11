import { render } from 'solid-js/web'
import { createInertiaApp } from 'inertia-adapter-solid'
import '../css/app.css'
import '../css/public.css'
import AdminLayout from './layouts/admin_layout'
import PublicLayout from './layouts/public_layout'

type PageModule = { default: ((props: any) => any) & { layout?: unknown } }
const modules = import.meta.glob<PageModule>('./pages/**/*.tsx', { eager: true })

createInertiaApp({
  resolve: name => {
    const module = modules[`./pages/${name}.tsx`]
    if (!module) throw new Error(`Unknown Inertia component: ${name}`)
    const component = module.default
    component.layout = name.startsWith('Admin/') ? AdminLayout : PublicLayout
    return component
  },
  setup: ({ el, App, props }) => { render(() => <App {...props} />, el); window.__aos_ready = true },
})
