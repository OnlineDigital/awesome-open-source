import { defineConfig } from 'vite'
import solid from 'vite-plugin-solid'

export default defineConfig(({ command }) => {
  const backend = process.env.VITE_BACKEND_URL
  return {
    base: command === 'build' ? '/build/' : '/',
    publicDir: false,
    plugins: [solid()],
    server: {
      // Reached through a reverse proxy under arbitrary hostnames.
      allowedHosts: true,
      // Everything except Vite's own modules goes to the Go server.
      proxy: backend ? { '^(?!/(@vite|@id|@fs|@solid-refresh|resources|node_modules)(/|$)).*': { target: backend } } : undefined,
    },
    build: {
      outDir: 'public/build',
      manifest: true,
      rollupOptions: { input: 'resources/js/app.tsx' },
    },
  }
})
