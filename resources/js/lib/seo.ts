import type { Seo } from '../types'

// The server renders the first response's <head> (every element has data-seo).
// On client-side navigation we replace those elements with the new page's tags.
export function applySeo(seo: Seo | undefined, locale: string) {
  if (!seo) return
  document.title = seo.title
  document.documentElement.lang = locale
  document.head.querySelectorAll('[data-seo]:not(title)').forEach(el => el.remove())
  for (const tag of seo.tags) {
    const el = document.createElement(tag.kind)
    el.setAttribute('data-seo', '')
    for (const [k, v] of Object.entries(tag.attrs)) el.setAttribute(k, v)
    if (tag.kind === 'script' && tag.content) el.textContent = tag.content
    document.head.appendChild(el)
  }
}
