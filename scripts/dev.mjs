import { spawn } from 'node:child_process'
import { watch } from 'node:fs'
import { mkdir, rm } from 'node:fs/promises'
import path from 'node:path'
import { networkInterfaces } from 'node:os'
import { execute, root, goEnv, terminate } from './process.mjs'

await mkdir(path.join(root, 'build'), { recursive: true })
const vitePort = process.env.VITE_PORT ?? '5173'
const appAddress = process.env.APP_ADDR ?? '127.0.0.1:8080'
const appURL = new URL(`http://${appAddress}`)
const appHost = appURL.hostname.replace(/^\[|\]$/g, '')
const viteHost = process.env.VITE_HOST ?? appHost
let advertisedHost = viteHost
if (['0.0.0.0', '::', ''].includes(advertisedHost)) {
  advertisedHost = Object.values(networkInterfaces()).flat().find(address => address && !address.internal && address.family === 'IPv4')?.address ?? '127.0.0.1'
}
const advertisedOrigin = `http://${advertisedHost.includes(':') ? `[${advertisedHost}]` : advertisedHost}:${vitePort}`
const devServerURL = process.env.VITE_DEV_SERVER_URL ?? advertisedOrigin
process.env.VITE_HOST = viteHost
process.env.VITE_DEV_SERVER_URL = devServerURL
process.env.APP_ADDR = appAddress
const vite = spawn(process.execPath, [path.join(root, 'node_modules/vite/bin/vite.js'), '--host', viteHost, '--port', vitePort, '--strictPort'], { cwd: root, stdio: 'inherit' })
let server, serverBinary, timer, building = false, queued = false, stopping = false, generation = 0
const watchers = []
async function rebuild() {
  if (stopping) return
  if (building) { queued = true; return }
  building = true
  try {
    // Compile separately so an invalid edit does not stop the working server.
    const binary = path.join(root, 'build', `server-dev-${process.pid}-${++generation}${process.platform === 'win32' ? '.exe' : ''}`)
    await execute('go', ['build', '-o', binary, '.'], { env: goEnv })
    await terminate(server)
    if (serverBinary) await rm(serverBinary, { force: true })
    if (stopping) { await rm(binary, { force: true }); return }
    serverBinary = binary
    server = spawn(binary, ['artisan', 'serve', '--env', 'development'], {
      cwd: root, stdio: 'inherit',
      env: { ...process.env, APP_ROOT: root, APP_ENV: 'development', APP_ADDR: appAddress, VITE_DEV_SERVER_URL: devServerURL },
    })
    server.once('error', error => console.error(error))
  } catch (error) { console.error(error.message) }
  finally { building = false; if (queued && !stopping) { queued = false; await rebuild() } }
}
async function shutdown() {
  if (stopping) return
  stopping = true
  clearTimeout(timer)
  watchers.forEach(watcher => watcher.close())
  await Promise.all([terminate(server), terminate(vite)])
  if (serverBinary) await rm(serverBinary, { force: true })
  process.exit(0)
}
for (const signal of ['SIGINT', 'SIGTERM']) process.on(signal, shutdown)
vite.once('error', error => { console.error(error); shutdown() })
vite.once('exit', () => { if (!stopping) shutdown() })
for (const directory of ['app', 'cmd', 'routes', 'internal']) {
  watchers.push(watch(path.join(root, directory), { recursive: true }, (_event, filename) => {
    if (!filename?.endsWith('.go') && !filename?.endsWith('.html')) return
    clearTimeout(timer)
    timer = setTimeout(rebuild, 250)
  }))
}
watchers.push(watch(root, (_event, filename) => {
  if (!['main.go', 'go.mod', 'go.sum'].includes(filename)) return
  clearTimeout(timer)
  timer = setTimeout(rebuild, 250)
}))
await rebuild()
