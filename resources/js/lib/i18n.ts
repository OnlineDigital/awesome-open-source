import { usePage } from 'inertia-adapter-solid'
import type { Shared } from '../types'

export function useI18n() {
  const page = usePage<any>()
  const props = () => page.props as Shared
  const t = (key: string, vars?: Record<string, string | number>) => {
    let s = props().i18n?.[key] ?? key
    if (vars) for (const [k, v] of Object.entries(vars)) s = s.replaceAll(`{${k}}`, String(v))
    return s
  }
  const locale = () => props().locale ?? 'en'
  const number = (n: number) => new Intl.NumberFormat(locale(), { notation: 'compact', maximumFractionDigits: 1 }).format(n)
  const ago = (iso: string | null | undefined) => {
    if (!iso) return t('common.never')
    const days = Math.floor((Date.now() - new Date(iso).getTime()) / 86400000)
    if (days < 1) return t('common.today')
    if (days < 60) return t('common.days_ago', { n: days })
    if (days < 730) return t('common.months_ago', { n: Math.floor(days / 30) })
    return t('common.years_ago', { n: Math.floor(days / 365) })
  }
  return { t, locale, number, ago, props }
}
