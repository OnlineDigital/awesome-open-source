import { LANGUAGES } from '../lib/languages'

/** Programming-language logo tile in the language's brand colour; unknown languages get a neutral bracket tile. */
export function LangMark(props: { name: string; size?: number }) {
  const m = () => LANGUAGES[props.name]
  const px = () => props.size ?? 22
  return (
    <span class="lang-tile" style={{ '--lc': m()?.hex ?? '#2a2699', '--li': m()?.ink ?? '#f4f2ff', width: `${px()}px`, height: `${px()}px` }} aria-hidden="true">
      {m() ? (
        <svg viewBox="0 0 24 24" width={px() * 0.74} height={px() * 0.74} fill="currentColor"><path d={m()!.d} /></svg>
      ) : (
        <svg viewBox="0 0 24 24" width={px() * 0.74} height={px() * 0.74} fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 7 4 12l5 5M15 7l5 5-5 5" /></svg>
      )}
    </span>
  )
}
