import { execute, root, goEnv } from './process.mjs'
await execute('go', ['run', '.', 'artisan', 'build', ...process.argv.slice(2)], { env: goEnv })
