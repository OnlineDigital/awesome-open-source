import { spawn } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

export const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
try { process.loadEnvFile(path.join(root, '.env')) }
catch (error) { if (error.code !== 'ENOENT') throw error }
export function execute(command, args, options = {}) {
  return new Promise((resolve, reject) => {
    // npm is a .cmd launcher on Windows and needs the native command processor.
    if (process.platform === 'win32' && command === 'npm') {
      args = ['/d', '/c', 'npm', ...args]
      command = process.env.ComSpec ?? 'cmd.exe'
    }
    const child = spawn(command, args, { cwd: root, stdio: 'inherit', ...options })
    child.once('error', reject)
    child.once('exit', (code, signal) => {
      if (code === 0) resolve()
      else reject(new Error(`${command} ${args.join(' ')} exited with ${code ?? signal}`))
    })
  })
}
export const goEnv = { ...process.env, GOFLAGS: [process.env.GOFLAGS, '-buildvcs=false'].filter(Boolean).join(' ') }

export async function terminate(child) {
  if (!child || child.exitCode !== null || child.signalCode !== null) return
  await new Promise(resolve => {
    const timer = setTimeout(() => child.kill('SIGKILL'), 4000)
    child.once('exit', () => { clearTimeout(timer); resolve() })
    child.kill('SIGTERM')
  })
}
