// End-to-end smoke test: drives a real browser through the main user journeys and
// fails (exit code 1) if anything is wrong.
//
//   cd e2e && npm install && npx playwright install chromium
//   node smoke.cjs                                          # against localhost:5173
//   BASE_URL=https://eneba-marketplace-zeta.vercel.app node smoke.cjs   # against the live site
//
// It registers a new random account each run, so it can be repeated on the same database.
const { chromium } = require('playwright')
const assert = require('node:assert/strict')

const BASE = (process.env.BASE_URL || 'http://localhost:5173').replace(/\/$/, '')
const TIMEOUT = BASE.includes('localhost') ? 10_000 : 90_000 // the free Render server may need to wake up

let step = 0
async function check(name, fn) {
  step++
  try {
    await fn()
    console.log(`✓ ${step}. ${name}`)
  } catch (err) {
    console.error(`✗ ${step}. ${name}\n  ${err.message.split('\n')[0]}`)
    process.exitCode = 1
    throw err // stop: later steps depend on earlier ones
  }
}

;(async () => {
  const browser = await chromium.launch()
  const context = await browser.newContext({ viewport: { width: 1280, height: 900 } })
  const page = await context.newPage()
  page.setDefaultTimeout(TIMEOUT)
  const pageErrors = []
  page.on('pageerror', (e) => pageErrors.push(e.message))
  const search = page.locator('.search-input')

  try {
    await check('homepage blocks load from the API', async () => {
      await page.goto(BASE + '/')
      await page.waitForSelector('.carousel-slide.active')
      await page.waitForSelector('.row-track .card:not(.card-skeleton)')
      const rows = await page.locator('.row-header h2').allTextContents()
      assert.ok(rows.length >= 2, `expected 2+ product rows, got ${rows}`)
    })

    await check('typo-tolerant search: "wticher" finds The Witcher 3', async () => {
      await search.fill('wticher')
      const first = page.locator('.search-result-title').first()
      await first.waitFor()
      assert.match(await first.textContent(), /Witcher 3/)
    })

    await check('"Did you mean": "gpa" suggests Grand Theft Auto V', async () => {
      await search.fill('gpa')
      await search.press('Enter')
      await page.waitForURL(/\/store\?q=gpa/)
      const empty = page.locator('.empty-state')
      await empty.waitFor()
      assert.match(await empty.textContent(), /Did you mean.*Grand Theft Auto V/s)
    })

    await check('store filters live in the URL and survive a reload', async () => {
      await page.goto(BASE + '/store?platform=Nintendo&sort=price_asc')
      await page.waitForSelector('.card:not(.card-skeleton)')
      await page.reload()
      await page.waitForSelector('.card:not(.card-skeleton)')
      assert.equal(await page.getByRole('checkbox', { name: 'Nintendo' }).isChecked(), true)
      assert.equal(await page.locator('.sort-select select').inputValue(), 'price_asc')
      const regions = await page.locator('.card-title').allTextContents()
      assert.ok(regions.every((t) => /Nintendo|Switch/.test(t)), 'non-Nintendo product in the results')
    })

    await check('product page lists offers; add to cart updates the badge', async () => {
      await page.goto(BASE + '/product/hades-pc-steam-global')
      await page.waitForSelector('.offer')
      await page.locator('.offer').first().locator('button').click()
      assert.equal(await page.locator('.cart-badge').textContent(), '1')
    })

    await check('checkout requires login; register returns to checkout', async () => {
      await page.goto(BASE + '/cart')
      await page.getByRole('link', { name: 'Go to checkout' }).click()
      await page.waitForURL(/\/login\?next=%2Fcheckout/)
      await page.getByRole('link', { name: 'Create an account' }).click()
      await page.getByLabel('Email').fill(`smoke-${Date.now()}@example.com`)
      await page.getByLabel('Password', { exact: true }).fill('smoke test password')
      await page.getByRole('button', { name: 'Create account' }).click()
      await page.waitForURL(/\/checkout$/)
      await page.waitForSelector('.summary-lines li')
    })

    await check('card ending in 0002 is declined', async () => {
      await page.getByRole('button', { name: /4000 0000 0000 0002/ }).click()
      await page.getByLabel('Expiry (MM/YY)').fill('1239')
      await page.getByLabel('CVC').fill('123')
      await page.getByRole('button', { name: /^Pay/ }).click()
      const error = page.locator('.form-error')
      await error.waitFor()
      assert.match(await error.textContent(), /declined/i)
    })

    await check('card 4242 pays; keys are delivered (hidden until revealed)', async () => {
      await page.getByRole('button', { name: /4242 4242 4242 4242/ }).click()
      await page.getByRole('button', { name: /^Pay/ }).click()
      await page.waitForURL(/\/orders\/\d+\?new=1/)
      await page.waitForSelector('.success-banner')
      assert.equal(await page.locator('.key-box').count(), 1)
      assert.match(await page.locator('.key-code').textContent(), /^•/)
      await page.getByRole('button', { name: 'Reveal' }).click()
      assert.match(await page.locator('.key-code').textContent(), /^[A-Z0-9]{5}-[A-Z0-9]{5}-[A-Z0-9]{5}$/)
      assert.equal(await page.locator('.cart-badge').count(), 0, 'cart should be empty after paying')
    })

    await check('order history shows the order; login survives a reload', async () => {
      await page.goto(BASE + '/orders')
      await page.waitForSelector('.order-row')
      await page.reload()
      await page.waitForSelector('.order-row')
      assert.match(await page.locator('.order-row').first().textContent(), /Hades/)
    })

    await check('demo account signs in and has a finished order with 2 keys', async () => {
      await page.getByRole('button', { name: 'Log out' }).click()
      await page.goto(BASE + '/login')
      await page.getByRole('button', { name: 'Fill in the demo account' }).click()
      await page.getByRole('button', { name: 'Sign in' }).click()
      await page.waitForURL(BASE + '/')
      await page.goto(BASE + '/orders')
      await page.locator('.order-row').last().click()
      await page.waitForSelector('.key-box')
      assert.equal(await page.locator('.key-box').count(), 2)
    })

    await check('no horizontal scrolling on a 390 px phone', async () => {
      const phone = await browser.newPage({ viewport: { width: 390, height: 844 } })
      phone.setDefaultTimeout(TIMEOUT)
      for (const path of ['/', '/store', '/product/elden-ring-pc-steam-global', '/cart', '/login', '/register']) {
        await phone.goto(BASE + path)
        await phone.waitForLoadState('networkidle')
        const overflow = await phone.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)
        assert.equal(overflow, false, `${path} scrolls sideways`)
      }
    })

    await check('no JavaScript errors on any page', async () => {
      assert.deepEqual(pageErrors, [])
    })

    console.log(`\nAll ${step} checks passed against ${BASE}`)
  } catch {
    console.error(`\nFailed against ${BASE}`)
  } finally {
    await browser.close()
  }
})()
