import { spawn } from 'node:child_process'
import { access } from 'node:fs/promises'
import path from 'node:path'
import { root, terminate } from './process.mjs'

const test = process.argv.includes('--test')
const binary = path.join(root, 'build', `inertia-go-solid-template${process.platform === 'win32' ? '.exe' : ''}`)
try { await access(binary) }
catch { throw new Error('Build the executable first: go run . artisan build') }
const child = spawn(binary, [], {
  cwd: root, stdio: 'inherit',
  env: { ...process.env, APP_ROOT: root, APP_ENV: test ? 'test' : (process.env.APP_ENV ?? 'production'), APP_ADDR: test ? '127.0.0.1:8102' : (process.env.APP_ADDR ?? '127.0.0.1:8080') },
})
for (const signal of ['SIGINT', 'SIGTERM']) process.on(signal, async () => { await terminate(child); process.exit(0) })
child.once('error', error => { console.error(error); process.exitCode = 1 })
child.once('exit', code => { process.exitCode = code ?? 0 })
