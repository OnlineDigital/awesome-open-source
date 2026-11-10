// WebMCP (https://webmachinelearning.github.io/webmcp/): expose site search as tools that
// in-browser AI agents can call through navigator.modelContext. Feature-detected: a no-op
// in browsers without WebMCP. The tools call the same public JSON API used by agents.

interface ToolResult { content: { type: 'text'; text: string }[] }
interface ModelContextTool {
  name: string
  description: string
  inputSchema: Record<string, unknown>
  annotations?: Record<string, unknown>
  execute: (args: Record<string, unknown>) => Promise<ToolResult>
}
interface ModelContext {
  registerTool?: (tool: ModelContextTool) => unknown
  provideContext?: (ctx: { tools: ModelContextTool[] }) => unknown
}

const text = (value: unknown): ToolResult => ({ content: [{ type: 'text', text: JSON.stringify(value) }] })

async function api(path: string, params: Record<string, unknown>, locale: string): Promise<unknown> {
  const url = new URL(path, location.origin)
  for (const [k, v] of Object.entries(params)) if (v !== undefined && v !== null && v !== '') url.searchParams.set(k, String(v))
  url.searchParams.set('locale', locale)
  const res = await fetch(url, { headers: { Accept: 'application/json' } })
  if (!res.ok) throw new Error(`Request failed with HTTP ${res.status}`)
  return res.json()
}

let registered = false
export function registerWebMCP(getLocale: () => string) {
  const mc = (navigator as unknown as { modelContext?: ModelContext }).modelContext
  if (!mc || registered) return
  const tools: ModelContextTool[] = [
    {
      name: 'search_projects',
      description: 'Search open source projects by keyword, topic or language. Each result has an activity score from 1 (abandoned) to 5 (thriving). Returns project cards with page URLs.',
      inputSchema: {
        type: 'object',
        properties: {
          query: { type: 'string', description: 'Free-text query, e.g. "self-hosted analytics" or "rust http server"' },
          category: { type: 'string', description: 'Category slug from list_categories' },
          min_score: { type: 'number', minimum: 1, maximum: 5, description: 'Only projects with at least this activity score' },
          sort: { type: 'string', enum: ['relevance', 'score', 'stars', 'new'] },
          limit: { type: 'integer', minimum: 1, maximum: 50 },
        },
      },
      annotations: { readOnlyHint: true },
      execute: async a => text(await api('/api/search', { q: a.query, category: a.category, min_score: a.min_score, sort: a.sort === 'relevance' ? '' : a.sort, limit: a.limit ?? 10 }, getLocale())),
    },
    {
      name: 'list_categories',
      description: 'List all software categories with their slugs and project counts.',
      inputSchema: { type: 'object', properties: {} },
      annotations: { readOnlyHint: true },
      execute: async () => text(await api('/api/categories', {}, getLocale())),
    },
    {
      name: 'get_project',
      description: 'Get full details for one project by slug (as returned by search_projects): description, license, activity score breakdown, demo video and links.',
      inputSchema: { type: 'object', properties: { slug: { type: 'string' } }, required: ['slug'] },
      annotations: { readOnlyHint: true },
      execute: async a => text(await api(`/api/projects/${encodeURIComponent(String(a.slug))}`, {}, getLocale())),
    },
  ]
  try {
    if (typeof mc.registerTool === 'function') tools.forEach(t => mc.registerTool!(t))
    else if (typeof mc.provideContext === 'function') mc.provideContext({ tools })
    else return
    registered = true
  } catch (e) {
    console.warn('WebMCP registration failed', e)
  }
}
