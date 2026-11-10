import { expect, test, type Page } from '@playwright/test'

async function open(page: Page, path: string) {
  await page.goto(path)
  await page.waitForFunction(() => window.__lab?.ready)
}
async function evaluations(page: Page) {
  return (await (await page.request.get('/stats')).json()).evaluations
}

test.beforeEach(async ({ request }) => { await request.post('/reset') })

test('demonstration navigation reaches every page without runtime exceptions', async ({ page }) => {
  const errors: string[] = []
  page.on('pageerror', error => errors.push(error.message))
  await open(page, '/home')
  const boot = await page.evaluate(() => window.__lab.boot)
  for (const [path, component] of [
    ['/props', 'Props'], ['/deferred', 'DeferredPage'], ['/once', 'OncePage'],
    ['/merge', 'MergePage'], ['/deep', 'DeepPage'], ['/scroll', 'ScrollPage'],
    ['/visible', 'VisiblePage'], ['/poll', 'PollPage'], ['/form', 'FormPage'],
    ['/history', 'HistoryPage'], ['/users/37', 'Users/Show'], ['/home', 'Home'],
  ]) {
    await page.locator(`nav a[href="${path}"]`).first().click()
    await expect(page.locator('#component')).toHaveText(component)
    if (component === 'Users/Show') {
      await expect(page.locator('#user-id')).toHaveText('37')
      await page.locator('main a[href="/users/42"]').click()
      await expect(page.locator('#user-id')).toHaveText('42')
    }
  }
  expect(await page.evaluate(() => window.__lab.boot)).toBe(boot)
  expect(errors).toEqual([])
})

test('visible only and except controls demonstrate avoided companies work', async ({ page }) => {
  await open(page, '/props')
  let before = await evaluations(page)
  await page.locator('#partial-only-users').click()
  await expect.poll(async () => (await evaluations(page)).users).toBe(before.users + 1)
  expect((await evaluations(page)).companies).toBe(before.companies)
  before = await evaluations(page)
  await page.locator('#partial-except-companies').click()
  await expect.poll(async () => (await evaluations(page)).users).toBe(before.users + 1)
  expect((await evaluations(page)).companies).toBe(before.companies)
  await expect(page.locator('#evaluation-counters')).toContainText('companies')
})

test('manual pagination reaches its end and reset replaces accumulated rows', async ({ page }) => {
  const errors: string[] = []
  page.on('pageerror', error => errors.push(error.message))
  await open(page, '/scroll')
  await page.locator('#scroll-next').click()
  await expect(page.locator('#scroll-items [data-item]')).toHaveCount(2)
  await page.locator('#scroll-next').click()
  await expect(page.locator('#scroll-items [data-item]')).toHaveCount(3)
  expect(await page.evaluate(() => window.__lab.scroll.hasNext())).toBe(false)
  await expect(page.locator('#scroll-next')).toBeDisabled()
  const pagingRequests = async () => (await (await page.request.get('/stats')).json()).requests.filter((r: {path: string}) => r.path.startsWith('/scroll')).length
  const count = await pagingRequests()
  await page.evaluate(() => window.__lab.scroll.fetchNext())
  expect(await pagingRequests()).toBe(count)
  await page.locator('#scroll-reset').click()
  await expect(page.locator('#scroll-items [data-item]')).toHaveCount(1)
  expect(await page.evaluate(() => window.__lab.scroll.hasNext())).toBe(true)
  expect(errors).toEqual([])
})

test('JSON wrapper keeps response reactive and releases page hooks on unmount', async ({ page }) => {
  await open(page, '/form')
  const result = await page.evaluate(() => window.__lab.http.get('/http'))
  expect(result).toEqual({ ok: true, value: 42 })
  await expect.poll(async () => JSON.parse(await page.locator('#http-state').innerText()).response)
    .toEqual({ ok: true, value: 42 })
  await page.locator('#nav-other').click()
  await expect(page.locator('#component')).toHaveText('Other')
  expect(await page.evaluate(() => window.__lab.form)).toBeNull()
  expect(await page.evaluate(() => window.__lab.http)).toBeNull()
})

test('every demonstration fits a mobile viewport', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  const errors: string[] = []
  page.on('pageerror', error => errors.push(error.message))
  for (const path of ['/home', '/props', '/deferred', '/once', '/merge', '/deep', '/scroll', '/visible', '/poll', '/form', '/history', '/users/37']) {
    await open(page, path)
    await expect(page.locator('#component')).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true)
  }
  expect(errors).toEqual([])
})
