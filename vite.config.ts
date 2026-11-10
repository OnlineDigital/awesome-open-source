import { defineConfig } from 'vite'
import solid from 'vite-plugin-solid'

export default defineConfig(({ command }) => {
  const developmentOrigin = process.env.VITE_DEV_SERVER_URL
  const address = process.env.APP_ADDR ?? '127.0.0.1:8080'
  const wildcard = /^(0\.0\.0\.0|\[::\]):/.test(address)
  const allowedOrigins: (string | RegExp)[] = [/^http:\/\/(127\.0\.0\.1|localhost):\d+$/]
  if (!wildcard) allowedOrigins.push(`http://${address}`)
  // Binding to every interface explicitly opts development into LAN access.
  if (wildcard) allowedOrigins.push(/^http:\/\/(?:\d{1,3}\.){3}\d{1,3}:\d+$/)
  if (developmentOrigin) allowedOrigins.push(developmentOrigin)
  return {
    base: command === 'build' ? '/build/' : '/',
    publicDir: false,
    plugins: [solid()],
    server: { host: '127.0.0.1', port: 5173, strictPort: true, cors: { origin: allowedOrigins } },
    build: {
      outDir: 'public/build',
      manifest: true,
      rollupOptions: { input: 'resources/js/app.tsx' },
    },
  }
})
