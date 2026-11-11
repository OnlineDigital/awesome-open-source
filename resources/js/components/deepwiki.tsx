import { createResource, Match, Switch } from 'solid-js'
import { useI18n } from '../lib/i18n'
import { IconExternal, IconWiki } from './icons'

type Reply = { status: 'ready' | 'missing' | 'unknown'; url: string }

async function check(slug: string): Promise<Reply> {
  try {
    const res = await fetch(`/api/projects/${encodeURIComponent(slug)}/deepwiki`, { headers: { Accept: 'application/json' } })
    return res.ok ? await res.json() : { status: 'unknown', url: '' }
  } catch {
    return { status: 'unknown', url: '' }
  }
}

// Asks our API (which asks DeepWiki) whether the repo is indexed. Clicking only opens deepwiki.com/owner/repo,
// which is also where indexing is requested. Hidden when DeepWiki can't be reached.
export function DeepWiki(props: { slug: string; fullName: string }) {
  const { t } = useI18n()
  const [reply] = createResource(() => props.slug, check)
  const href = () => reply()?.url || `https://deepwiki.com/${props.fullName}`
  return (
    <Switch>
      <Match when={reply.loading}>
        <div class="deepwiki deepwiki-loading" aria-hidden="true" />
      </Match>
      <Match when={reply()?.status === 'ready'}>
        <a class="deepwiki" href={href()} rel="noopener" target="_blank">
          <span class="deepwiki-logo"><IconWiki width={26} height={26} /></span>
          <span class="deepwiki-text"><strong>{t('deepwiki.title')}</strong><span>{t('deepwiki.cta')}</span></span>
          <IconExternal />
        </a>
      </Match>
      <Match when={reply()?.status === 'missing'}>
        <a class="deepwiki deepwiki-missing" href={href()} rel="noopener" target="_blank">
          <span class="deepwiki-logo"><IconWiki width={26} height={26} /></span>
          <span class="deepwiki-text"><strong>{t('deepwiki.not_ready')}</strong><span>{t('deepwiki.request')}</span></span>
          <IconExternal />
        </a>
      </Match>
    </Switch>
  )
}
