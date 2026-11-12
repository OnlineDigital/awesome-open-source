# Catalog copy

Use this skill whenever you write or update text shown in the Awesome Open Source catalog: project taglines and descriptions, topic names/summaries/bodies, category descriptions, feature names/descriptions, and comparison notes. It is the writing standard for both English source copy and its `ro`, `es`, `fr`, `de` translations.

## Evidence

- Base every sentence on the project's README, its GitHub description, or its own docs. Never invent capabilities, benchmarks, licensing, or deployment details.
- If the README is empty, unreadable, or only says "Contribute to…", write the minimum that is certainly true and do not guess.
- Do not copy README marketing lines verbatim, do not keep badges, emoji, HTML entities, "Contribute to…" text, or trailing `- owner/repo` suffixes.

## Project tagline

- One sentence, max ~120 characters, no trailing period needed if it reads cleanly.
- Formula: **what it is + the one thing it does best**. Example: "Self-hosted web analytics with funnels, retention and no cookies."
- No hype adjectives ("revolutionary", "best-in-class", "blazingly fast"), no "open-source" on its own, no repeating the repository name twice.

## Project description

2–3 sentences, 45–75 words, dense and specific. Cover, in order, skipping anything the README does not support:

1. **What it is** — category and core function, in plain language.
2. **Why you would use it** — the two or three capabilities that make it worth choosing, phrased as user outcomes.
3. **How it is made** — stack, deployment model (self-hosted, Docker, single binary), data model or integration surface, license only when stated.

Rules:

- One idea per sentence; no lists, no headings, no links.
- Prefer concrete nouns and numbers over generalities ("PostgreSQL and MySQL", not "many databases").
- No calls to action, no "get started", no "star us", no comparisons that the README does not make.
- Never start with "This is a…" or repeat the project name as the first word if the tagline already does.
- Present tense, third person, neutral tone.

## Topics and categories

- Name: 2–4 words, Title Case, the term a developer would search for.
- Summary: one sentence, max ~160 characters, what the collection covers.
- Body: 2 short paragraphs. First explains the problem space and what belongs in the topic; second helps the reader choose (criteria, trade-offs, who each option suits). No project ranking unless the criterion is stated.

## Features (comparison rows)

- Name: short capability label (2–5 words) that can truthfully be answered yes/no/partial for every member.
- Description: one neutral sentence defining what "yes" means.
- Avoid generic rows ("Open source", "Self-hosted") unless it is a defining property of the topic.

## Translations

- English is the source of truth; translate meaning, not words.
- Keep product names, library names, package names, protocol names, acronyms, versions, URLs, CLI flags and code identifiers untranslated.
- Keep the same sentence count and paragraph breaks as the English text.
- Use natural grammar in each locale; do not add or remove information.
- Romanian: no unnecessary English words when a normal Romanian term exists, but keep technical terms as developers use them.
- German: compound nouns are fine; keep the register neutral (`Sie`-free, infinitive style).
- Spanish and French: neutral international register, no regionalisms.

## Saving through MCP

- Project text: `set_project_text` with `project`, `locale`, `tagline`, `description`. Send all five locales; empty text deletes a locale, so never send blank values.
- Topic text: `save_topic` replaces every locale — always send `en`, `ro`, `es`, `fr`, `de`.
- Category text: `save_category` behaves the same way.
- Features: `save_features` replaces the row list; provide `translations` for all five locales.
- Never change category, topics, publication status or screenshots while writing copy.

## Checklist before saving

- [ ] Factual and supported by the README.
- [ ] Tagline ≤ 120 characters; description 45–75 words, 2–3 sentences.
- [ ] No marketing filler, no copied README lines, no emoji or HTML entities.
- [ ] Product names and technical identifiers preserved in all locales.
- [ ] All five locales present and non-empty.
