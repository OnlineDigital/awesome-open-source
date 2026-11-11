import { router } from 'inertia-adapter-solid'
import { createSignal } from 'solid-js'
import { useI18n } from '../lib/i18n'
import { IconSearch } from './icons'

export function SearchBox(props: { initial?: string; large?: boolean; autofocus?: boolean }) {
  const { t, props: shared } = useI18n()
  const [q, setQ] = createSignal(props.initial ?? '')
  const submit = (e: SubmitEvent) => {
    e.preventDefault()
    const base = shared().locale === shared().site.defaultLocale ? '' : `/${shared().locale}`
    router.get(`${base}/search`, q().trim() ? { q: q().trim() } : {})
  }
  return (
    <form class={`searchbox ${props.large ? 'searchbox-lg' : ''}`} role="search" onSubmit={submit}>
      <label class="sr-only" for={props.large ? 'q-hero' : 'q-nav'}>{t('nav.search')}</label>
      <IconSearch class="searchbox-icon" width={18} height={18} />
      <input id={props.large ? 'q-hero' : 'q-nav'} type="search" name="q" value={q()} onInput={e => setQ(e.currentTarget.value)} placeholder={t('search.placeholder')} autocomplete="off" autofocus={props.autofocus} />
      <button type="submit">{t('nav.search')}</button>
    </form>
  )
}
