import { writeFile, mkdir } from 'node:fs/promises'
import path from 'node:path'
import { execute, root, goEnv } from './process.mjs'

const checks = []
await mkdir(path.join(root, 'reports'), { recursive: true })
async function check(name, command, args, options) {
  try { await execute(command, args, options); checks.push({ name, status: 'PASS' }) }
  catch (error) { checks.push({ name, status: 'FAIL', error: error.message }); throw error }
}
try {
  await check('TypeScript', 'npm', ['run', 'typecheck'])
  await check('Embedded production build', 'go', ['run', '-buildvcs=false', '.', 'artisan', 'build'], { env: goEnv })
  await check('Go tests with race detector', 'go', ['test', '-race', '-count=1', './...'], { env: goEnv })
  await check('Portable embedded server', 'node', ['scripts/smoke.mjs'])
  await check('Chromium browser scenarios', 'npm', ['exec', '--', 'playwright', 'test', ...process.argv.slice(2)])
} catch (error) { console.error(error.message); process.exitCode = 1 }
finally {
  await writeFile(path.join(root, 'reports/template-checks.json'), JSON.stringify({ date: new Date().toISOString(), checks }, null, 2) + '\n')
  if (checks.some(check => check.name === 'Chromium browser scenarios')) await execute('node', ['scripts/report.mjs'])
}
