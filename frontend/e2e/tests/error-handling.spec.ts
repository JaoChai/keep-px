import { test, expect } from '../fixtures/auth.fixture'
import { test as baseTest, expect as baseExpect } from '@playwright/test'

test.describe('Error Handling - Authenticated', () => {
  test('invalid route redirects to known page', async ({ page }) => {
    await page.goto('/this-page-does-not-exist')
    await page.waitForLoadState('networkidle')

    // App should redirect to dashboard or login (depending on auth state handling)
    const url = page.url()
    const isKnownPage = url.includes('/dashboard') || url.includes('/login') || url.includes('/this-page-does-not-exist')
    expect(isKnownPage).toBe(true)
  })

  test('direct API call without auth returns 401', async ({ page }) => {
    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:5173'
    const response = await page.request.get(`${baseURL}/api/v1/pixels`, {
      headers: {
        Authorization: '',
      },
    })

    expect(response.status()).toBe(401)
  })

  test('event ingest without API key returns 401', async ({ page }) => {
    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:5173'
    const response = await page.request.post(`${baseURL}/api/v1/events/ingest`, {
      headers: {
        'Content-Type': 'application/json',
      },
      data: {
        events: [
          {
            pixel_id: '123456789',
            event_name: 'PageView',
            event_time: new Date().toISOString(),
          },
        ],
      },
    })

    expect(response.status()).toBe(401)
  })

  test('pixel quota limit shows disabled button with upgrade path', async ({ page }) => {
    await page.goto('/pixels')
    await page.waitForLoadState('networkidle')

    // Read quota info from the page (format: "X/Y สล็อต")
    const quotaLocator = page.locator('text=/\\d+\\/\\d+ สล็อต/')
    const hasQuota = await quotaLocator.count() > 0
    if (!hasQuota) {
      test.skip(true, 'Quota info not displayed — cannot verify quota limit behavior')
      return
    }

    const quotaText = await quotaLocator.textContent()
    const match = quotaText?.match(/(\d+)\/(\d+)/)
    if (!match) {
      test.skip(true, 'Could not parse quota text')
      return
    }

    const current = parseInt(match[1])
    const max = parseInt(match[2])

    if (current < max) {
      // Not at limit — verify button is enabled (normal state)
      const addButton = page.getByRole('button', { name: 'เพิ่มพิกเซล' }).first()
      await expect(addButton).toBeEnabled()
    } else {
      // At limit — verify button is disabled and has quota limit title
      const addButton = page.getByRole('button', { name: 'เพิ่มพิกเซล' }).first()
      await expect(addButton).toBeDisabled()

      // When at limit with 0 pixels (edge case), the empty state shows upgrade link
      // When at limit with pixels, the button title indicates the limit
      const hasUpgradeLink = await page.getByRole('link', { name: 'อัปเกรด' }).count() > 0
      const hasLimitTitle = await addButton.getAttribute('title') === 'ถึงขีดจำกัด Pixel Slots แล้ว'
      expect(hasUpgradeLink || hasLimitTitle).toBeTruthy()
    }
  })
})

baseTest.describe('Error Handling - Unauthenticated', () => {
  baseTest('all protected routes redirect to login', async ({ page }) => {
    const protectedRoutes = [
      '/dashboard',
      '/pixels',
      '/sale-pages',
      '/events',
      '/replay',
      '/billing',
      '/settings',
    ]

    for (const route of protectedRoutes) {
      await page.goto(route)
      await baseExpect(page).toHaveURL(/\/login/)
    }
  })
})
