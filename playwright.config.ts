import { defineConfig } from '@playwright/test'
export default defineConfig({
  testDir: './tests/browser',
  fullyParallel: false, workers: 1, retries: 0,
  timeout: 10000, expect: {timeout:2000},
  reporter: [['list'], ['json', {outputFile:'reports/template-browser-results.json'}]],
  outputDir: 'test-results',
  use: {browserName:'chromium', headless:true, viewport:{width:1000,height:720}, trace:'retain-on-failure', screenshot:'only-on-failure'},
  projects: [{ name: 'inertia-go-solid', use: { baseURL: 'http://127.0.0.1:8102' } }],
  webServer: {
    command: 'node scripts/start.mjs --test',
    url: 'http://127.0.0.1:8102/health',
    reuseExistingServer: false,
    timeout: 20000,
  },
})
