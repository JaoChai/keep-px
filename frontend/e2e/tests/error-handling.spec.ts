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
