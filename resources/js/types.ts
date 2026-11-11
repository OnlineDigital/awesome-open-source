export interface Card {
  slug: string; name: string; fullName: string; tagline: string
  category: string; categorySlug: string
  score: number | null; tier: number | null; stars: number
  language: string; avatar: string; hasVideo: boolean; archived: boolean
  lastCommitAt: string | null; path: string; url: string
}
export interface CategoryCard { slug: string; icon: string; name: string; description: string; projectCount: number; path: string; url: string }
export interface TopicCard { slug: string; name: string; summary: string; projectCount: number; path: string; url: string }
export interface FeatureRow { id: number; name: string; description: string }
export interface Cell { state: 'yes' | 'no' | 'partial' | 'text'; note?: string }
export interface LocaleInfo { code: string; name: string; path: string }
export interface SeoTag { kind: 'meta' | 'link' | 'script'; attrs: Record<string, string>; content?: string }
export interface Seo { title: string; description: string; canonical: string; locale: string; tags: SeoTag[] }
export interface Shared {
  locale: string; locales: LocaleInfo[]; i18n: Record<string, string>
  site: { name: string; baseUrl: string; defaultLocale: string; homePath: string }
  seo?: Seo; flash?: { success?: string }; errors?: Record<string, string>
  auth?: { email: string }
  adminLocales?: string[]; defaultLocale?: string
  capabilities?: { translate: boolean; search: boolean; githubToken: boolean; embeddings: boolean }
}
export interface Component { name: string; value: number; weight: number; used: boolean; evidence?: string }
export interface Activity {
  score: number; tier: number; label: string; cap?: string; components: Component[]
  commits30: number; commits90: number; commits365: number; activeBranches90: number; totalBranches: number; releases12m: number
  allBranches: boolean; lastCommitAt: string | null; lastReleaseAt: string | null; lastReleaseTag: string
}
export interface Screenshot { id: number; path: string; alt: string; position: number }
export interface Project {
  id: number; slug: string; owner: string; repo: string; fullName: string
  categoryId: number | null; categorySlug: string; categoryName: string
  homepage: string; license: string; language: string; topics: string[]; extraTopics?: string[]; avatarUrl: string
  stars: number; forks: number; openIssues: number; archived: boolean
  pushedAt: string | null; lastCommitAt: string | null; lastReleaseAt: string | null; lastReleaseTag: string
  score: number | null; tier: number | null; youtubeId: string; status: 'draft' | 'published'; featured: boolean
  syncedAt: string | null; syncError: string; createdAt: string; updatedAt: string
  tagline: string; description: string; machineTranslated: boolean; screenshots?: Screenshot[]
}
export interface SearchResult { found: number; page: number; perPage: number; projects: Card[]; facets: Record<string, { value: string; count: number }[]> }

declare global {
  interface Window { __aos_ready?: boolean }
}
