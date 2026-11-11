import type { JSX } from 'solid-js'

const base = { width: 16, height: 16, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': 1.75, 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'aria-hidden': 'true', focusable: 'false' } as const

export const IconSearch = (p: JSX.SvgSVGAttributes<SVGSVGElement>) => <svg {...base} {...p}><circle cx="11" cy="11" r="6.5" /><path d="m16 16 4.5 4.5" /></svg>
export const IconPlay = (p: JSX.SvgSVGAttributes<SVGSVGElement>) => <svg {...base} {...p}><path d="M8 5.5v13l11-6.5z" fill="currentColor" stroke="none" /></svg>
export const IconExternal = (p: JSX.SvgSVGAttributes<SVGSVGElement>) => <svg {...base} {...p}><path d="M14 4h6v6M20 4l-9 9M18 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h5" /></svg>
export const IconStar = (p: JSX.SvgSVGAttributes<SVGSVGElement>) => <svg {...base} {...p}><path d="m12 3.5 2.6 5.4 5.9.8-4.3 4.1 1 5.9L12 16.9 6.8 19.7l1-5.9-4.3-4.1 5.9-.8z" /></svg>
export const IconArrow = (p: JSX.SvgSVGAttributes<SVGSVGElement>) => <svg {...base} {...p}><path d="M5 12h14M13 6l6 6-6 6" /></svg>
export const IconClose = (p: JSX.SvgSVGAttributes<SVGSVGElement>) => <svg {...base} {...p}><path d="M6 6l12 12M18 6 6 18" /></svg>
export const IconDoc = (p: JSX.SvgSVGAttributes<SVGSVGElement>) => <svg {...base} {...p}><path d="M7 3.5h7l4 4V20a.5.5 0 0 1-.5.5h-10.5a.5.5 0 0 1-.5-.5V4a.5.5 0 0 1 .5-.5zM14 3.5V8h4M9.5 12.5h5M9.5 16h5" /></svg>
export const IconCheck = (p: JSX.SvgSVGAttributes<SVGSVGElement>) => <svg {...base} stroke-width={2.6} {...p}><path d="m5 12.5 4.5 4.5L19 7.5" /></svg>
export const IconCross = (p: JSX.SvgSVGAttributes<SVGSVGElement>) => <svg {...base} stroke-width={2.4} {...p}><path d="M7 7l10 10M17 7 7 17" /></svg>
export const IconHalf = (p: JSX.SvgSVGAttributes<SVGSVGElement>) => <svg {...base} stroke-width={2} {...p}><circle cx="12" cy="12" r="7.5" /><path d="M12 4.5a7.5 7.5 0 0 1 0 15z" fill="currentColor" stroke="none" /></svg>
/** Brand mark: a seedling in a ring. On hover the ring draws, the leaves unfurl and sway, and four step keys light beneath. */
export const BrandMark = () => (
  <svg class="brandmark" width="30" height="30" viewBox="0 0 32 32" aria-hidden="true">
    <circle class="bm-ring" cx="16" cy="16" r="14.5" fill="none" stroke="currentColor" stroke-width="1.6" pathLength="100" />
    <path class="bm-stem" d="M16 25v-9" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
    <g class="bm-leaves"><path class="bm-l bm-l1" d="M16 16c0-4.2-2.8-6.6-7.5-6.6 0 4.4 2.8 6.6 7.5 6.6z" fill="currentColor" /><path class="bm-l bm-l2" d="M16 13c0-3.6 2.4-5.6 6.5-5.6 0 3.8-2.4 5.6-6.5 5.6z" fill="currentColor" /></g>
  </svg>
)

/** Generic wiki mark (open book with a spark) standing in for DeepWiki; swap for the official logo asset if licensed. */
export const IconWiki = (p: JSX.SvgSVGAttributes<SVGSVGElement>) => <svg {...base} {...p}><path d="M3.5 6.5C6 5 9 5 12 6.8c3-1.8 6-1.8 8.5-.3v11.5c-2.5-1.5-5.5-1.5-8.5.3-3-1.8-6-1.8-8.5-.3z" /><path d="M12 6.8V18.3" /><path d="M17.5 2.5v3M16 4h3" /></svg>
