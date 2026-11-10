// Run the compiled server from an otherwise empty directory, on any native OS.
import { spawn } from 'node:child_process'
import { randomBytes } from 'node:crypto'
import { copyFile, mkdtemp, rm } from 'node:fs/promises'
import { createServer } from 'node:net'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { root, terminate } from './process.mjs'

const directory = await mkdtemp(path.join(tmpdir(), 'inertia-go-solid-smoke-'))
const name = `inertia-go-solid-template${process.platform === 'win32' ? '.exe' : ''}`
const executable = path.join(directory, name)
let child
let output = ''
try {
  await copyFile(path.join(root, 'build', name), executable)
  const reservation = createServer()
  await new Promise((resolve, reject) => { reservation.once('error', reject); reservation.listen(0, '127.0.0.1', resolve) })
  const port = reservation.address().port
  await new Promise(resolve => reservation.close(resolve))
  const base = `http://127.0.0.1:${port}`
  child = spawn(executable, [], {
    cwd: directory, stdio: ['ignore', 'pipe', 'pipe'],
    env: { ...process.env, APP_ROOT: directory, APP_ENV: 'production', APP_ADDR: `127.0.0.1:${port}`, APP_KEY: randomBytes(32).toString('hex'), VITE_DEV_SERVER_URL: 'http://127.0.0.1:1' },
  })
  child.on('error', error => { output += error.message })
  for (const stream of [child.stdout, child.stderr]) stream.on('data', data => { output += data })
  let ready = false
  for (let attempt = 0; attempt < 100; attempt++) {
    try { ready = (await fetch(`${base}/health`, { signal: AbortSignal.timeout(500) })).ok } catch {}
    if (ready || child.exitCode !== null) break
    await new Promise(resolve => setTimeout(resolve, 100))
  }
  if (!ready) throw new Error(`Standalone server did not start: ${output}`)
  async function request(resource, status = 200) {
    const response = await fetch(new URL(resource, base))
    if (response.status !== status) throw new Error(`${resource}: HTTP ${response.status}, expected ${status}`)
    return response
  }
  const html = await (await request('/props')).text()
  const assets = [...html.matchAll(/(?:src|href)="(\/build\/[^"]+)"/g)].map(match => match[1])
  if (!assets.some(value => value.endsWith('.js')) || !assets.some(value => value.endsWith('.css'))) throw new Error('Missing embedded frontend links')
  let fontCount = 0
  for (const asset of assets) {
    const response = await request(asset)
    if (asset.endsWith('.css')) {
      if (!response.headers.get('content-type')?.includes('text/css')) throw new Error('Wrong CSS MIME')
      const css = await response.text()
      for (const match of css.matchAll(/url\(([^)]+\.woff2)\)/g)) {
        const font = new URL(match[1].replaceAll('"', ''), base + asset)
        const result = await request(font)
        if (!result.headers.get('content-type')?.includes('font/woff2')) throw new Error('Wrong font MIME')
        fontCount++
      }
    }
  }
  if (!fontCount) throw new Error('No embedded fonts exercised')
  await request('/build/.vite/manifest.json', 404)
  await request('/build/%252evite/manifest.json', 404)
  await request('/stats', 404)
  const response = await fetch(`${base}/submit`, { method: 'POST', redirect: 'manual', headers: { 'X-Inertia': 'true', 'Content-Type': 'application/x-www-form-urlencoded' }, body: 'name=' })
  if (response.status !== 303 || !response.headers.getSetCookie().some(cookie => cookie.includes('Secure') && cookie.includes('HttpOnly'))) throw new Error('Production validation/session cookie failed')
  console.log(`Portable production smoke PASS (${process.platform}/${process.arch}): empty directory, embedded JS/CSS/fonts, private manifest, secure sessions, no Vite fallback.`)
} finally {
  await terminate(child)
  await rm(directory, { recursive: true, force: true, maxRetries: 5, retryDelay: 200 })
}
