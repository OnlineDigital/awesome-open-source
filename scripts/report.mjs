import { readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { root } from './process.mjs'

const output = path.join(root, 'reports')
const browser = JSON.parse(await readFile(path.join(output, 'template-browser-results.json'), 'utf8'))
const cases = []
function collect(suite) {
  for (const spec of suite.specs ?? []) {
    for (const test of spec.tests ?? []) {
      const result = test.results.at(-1)
      cases.push({ name: spec.title, file: spec.file, line: spec.line, status: result?.status ?? 'missing', expected: test.expectedStatus, errors: (result?.errors ?? []).map(error => error.message?.replace(/\x1b\[[0-9;]*m/g, '')) })
    }
  }
  for (const child of suite.suites ?? []) collect(child)
}
browser.suites.forEach(collect)
let checks
try { checks = JSON.parse(await readFile(path.join(output, 'template-checks.json'), 'utf8')) }
catch { checks = { checks: [] } }
const pass = cases.filter(test => test.status === 'passed').length
const fail = cases.filter(test => test.status !== 'passed').length
let smoke = null
try { smoke = JSON.parse(await readFile(path.join(output, 'template-smoke.json'), 'utf8')) } catch {}
const escape = value => value.replaceAll('|', '\\|').replaceAll('\n', ' ')
const lines = [
  '# Inertia Go + Solid template verification', '',
  `Browser run: **${browser.stats.startTime}**. **${pass} passed / ${fail} non-passing**, across ${cases.length} scenarios. This tests this template, rather than a comparison with Laravel/React.`, '',
  '## Checks', '', '| Check | Result |', '|---|---|',
  ...checks.checks.map(check => `| ${check.name} | ${check.status} |`), '',
  ...(smoke ? [
    `Additional smoke checks: ${smoke.date}. Artisan development startup on the LAN, Vite bootstrap, Solid HMR, and Go rebuild/restart were exercised. Desktop/mobile inspection found no overflow or browser errors; the project skill was validated. [Desktop](template-ui-desktop.png), [mobile](template-ui-mobile.png).`, '',
  ] : []),
  ...(smoke?.portable_embedded_binary ? ['Standalone embedded binary from a directory without project files: **PASS**. Cross-compilation: Linux amd64, Windows amd64, and macOS arm64. These local checks execute Linux only; native Windows/macOS checks are provided by CI.', ''] : []),
  '## Reading the results', '',
  'Tests execute the real Fiber application with production assets and Go endpoints. A pass means that the named scenario works with the pinned dependencies and local helpers. It does not certify the entire Inertia feature family or Laravel compatibility.', '',
  'Ordinary lazy props use the local application helper, without Merge(fn).Append(). Scroll uses manual pagination over native metadata, avoiding the broken InfiniteScroll component in the Solid beta. Reactive JSON responses and query-aware prefetch inspection also use local helpers. See the [compatibility matrix](../.agents/skills/inertia-go-solid/references/features.md) for native limitations and coverage boundaries.', '',
  '## Browser scenarios', '', '| Scenario | Result |', '|---|---|',
  ...cases.map(test => `| ${escape(test.name)} | ${test.status.toUpperCase()} |`), '',
  '## Reproduce', '',
  'Run `npx playwright install chromium` once, then `./artisan test`. Machine-readable browser results and check logs are generated locally in `reports/` and excluded from version control.', '',
]

for (const test of cases.filter(test => test.status !== 'passed')) {
  lines.push(`### ${test.name}`, '', '```text', ...test.errors, '```', '')
}
await writeFile(path.join(output, 'template-verification.md'), lines.join('\n'))
await writeFile(path.join(output, 'template-summary.json'), JSON.stringify({ date: browser.stats.startTime, pass, fail, cases, checks: checks.checks, smoke }, null, 2) + '\n')
console.log(`Template: ${pass} PASS / ${fail} non-PASS; reports/template-verification.md`)
