import { Link, usePage } from 'inertia-adapter-solid'
import { createEffect, For, onMount, type ParentProps } from 'solid-js'
import { SearchBox } from '../components/search_box'
import { useI18n } from '../lib/i18n'
import { applySeo } from '../lib/seo'
import { registerWebMCP } from '../lib/webmcp'
import type { Shared } from '../types'

export default function PublicLayout(props: ParentProps) {
  const page = usePage<any>()
  const { t, locale } = useI18n()
  const shared = () => page.props as Shared
  createEffect(() => applySeo(shared().seo, shared().locale))
  onMount(() => registerWebMCP(() => shared().locale))
  const llms = () => (locale() === shared().site.defaultLocale ? '' : `/${locale()}`) + '/llms.txt'
  return (
    <div class="site">
      <a class="skip-link" href="#main">{t('nav.skip')}</a>
      <header class="header">
        <div class="wrap header-row">
          <Link href={shared().site.homePath} class="brand"><span class="brand-mark" aria-hidden="true">◆</span> {shared().site.name}</Link>
          <div class="header-search"><SearchBox /></div>
          <nav class="lang" aria-label={t('nav.language')}>
            <For each={shared().locales}>{l => <Link href={l.path} class={l.code === locale() ? 'lang-active' : ''} aria-current={l.code === locale() ? 'true' : undefined} hreflang={l.code} lang={l.code} title={l.name}>{l.code.toUpperCase()}</Link>}</For>
          </nav>
        </div>
      </header>
      <main id="main" class="wrap main" tabindex="-1">{props.children}</main>
      <footer class="footer">
        <div class="wrap footer-row">
          <p class="muted">{t('footer.about')}</p>
          <p class="footer-links">
            <a href={llms()}>{t('footer.ai')}</a>
            <a href="/sitemap.xml">{t('footer.sitemap')}</a>
          </p>
        </div>
      </footer>
    </div>
  )
}
