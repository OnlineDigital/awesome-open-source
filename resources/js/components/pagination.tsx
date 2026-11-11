import { Link } from './link'
import { useI18n } from '../lib/i18n'
import { IconArrow } from './icons'

// Pages are plain links (crawlable): ?page=N preserving other query parameters.
export function Pagination(props: { page: number; pages: number; path: string; params?: Record<string, string | number | undefined> }) {
  const { t } = useI18n()
  const href = (n: number) => {
    const q = new URLSearchParams()
    for (const [k, v] of Object.entries(props.params ?? {})) if (v !== undefined && v !== '') q.set(k, String(v))
    if (n > 1) q.set('page', String(n))
    const s = q.toString()
    return s ? `${props.path}?${s}` : props.path
  }
  return (
    <>
      {props.pages > 1 && (
        <nav class="pagination" aria-label="Pagination">
          {props.page > 1 ? <Link href={href(props.page - 1)} rel="prev" class="page-link"><IconArrow class="flip" width={16} height={16} /> {t('common.prev')}</Link> : <span />}
          <span class="muted">{t('common.page', { n: `${props.page} / ${props.pages}` })}</span>
          {props.page < props.pages ? <Link href={href(props.page + 1)} rel="next" class="page-link">{t('common.next')} <IconArrow width={16} height={16} /></Link> : <span />}
        </nav>
      )}
    </>
  )
}
