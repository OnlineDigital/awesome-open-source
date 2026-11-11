# Design: Awesome Open Source (public site)

Scope: Home, Category, Search, Project and Error pages. Admin keeps its earlier styling (`resources/css/app.css`); public tokens in `resources/css/public.css` apply only when the public layout is mounted (`:root:has(.site)`).

## World
A step sequencer. Each project is a track of 16 step keys; lit keys = score / 5 (rounded). Keys are coloured by quarter: 1 pink, 2 salmon, 3 yellow, 4 mint. The score numeral takes the colour of its last lit key, so low scores read pink and healthy ones mint. Night scene: dark ultramarine panel only (no light mode).

## Color
- Ground `#090828`, panel `#0e0c3a`, rail `#0d0b36`, deep edge `#04031a`, surface `#1a1766`, border `#312d96`. Kept dark on purpose so the lit keys carry the colour.
- Text `#f4f2ff`, muted `#b3b0f5`. Glows are kept subtle.
- Keys: `#ff4fa3` `#ff8a5c` `#ffe14d` `#5cf2c8`. Yellow is also the primary action and link colour; focus ring is mint.

## Type
- Unbounded (variable, OFL): headings, track names, brand.
- Anybody (variable, OFL): uppercase labels, buttons, banks, tier names; tracking .1 to .16em.
- Sometype Mono (variable, OFL): scores, repos, counts, facts; tabular numerals.
- Hanken Grotesk (variable, OFL): body and taglines.
- Self-hosted woff2 in `resources/fonts/` with licences.

## Components
- Step row (`step_row.tsx`): 16 keys, sizes sm/md/lg. Decorative; the score is always written beside it.
- Track (`project_card.tsx`): name + repo, step row, score readout, tagline and marks beneath. Hairline-separated, no cards.
- Bank: category key with mono count. Seg: sort keys. Physical keys use an inset bottom shadow and a 2px press.
- Home panel: sticky left panel (top-score readout, title, search, tier legend). Chase light is a one-key highlight stepping across 16 slots.
- Plate (project page): tempo readout, large 16-key row that lights in sequence once, one step row per measured component, facts.

- Track columns: name | language (brand-colour logo tile from Simple Icons, CC0, `lib/languages.ts`) | stars (filled yellow star) | score with a compact 7px step row. Description underneath.
- Score filter (`score_filter.tsx`): minimum score keys (Any, 2+, 3+, 3.5+, 4+, 4.5+) on Category (`?min=`) and Search (`min_score=`); keeps projects scoring that or higher.
- Brand hover: the seedling's ring draws and leaves unfurl; the wordmark lights letter by letter through the four key colours.

## Layout
Max width 1280px. Home: 420px sticky panel plus tracks; under 1000px it stacks. Track: name column, keys, score; under 760px keys drop to their own row.

## Do and don't
- Keep key colours bound to score quarters and nothing else.
- Don't box tracks as cards, add eyebrows, gradients, glass or a light theme.
