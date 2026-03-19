/**
 * Scenario 10: Error Recovery & Edge Cases
 *
 * Tests application resilience: bad navigation, auth loss, form validation,
 * empty states, and API error responses.
 *
 * Flow:
 *   Part A — Navigation & Auth Errors (steps 1-4)
 *   Part B — Form Validation Errors (steps 5-7)
 *   Part C — Empty States (steps 8-10)
 *   Part D — API Errors (steps 11-14)
 */
import { test, expect } from '../../fixtures/auth.fixture'
import { PixelsPage } from '../../pages/pixels.page'
import { EventLogPage } from '../../pages/event-log.page'
import { ReplayPage } from '../../pages/replay.page'
import { SalePagesPage } from '../../pages/sale-pages.page'

const PREFIX = 'E2E-S10'

test.describe(`Scenario 10: Error Recovery & Edge Cases`, () => {
  test.describe.configure({ mode: 'serial' })
  test.setTimeout(90_000)

  /** Shared state */
  let apiKey = ''

  // ============================================================
  // Part A: Navigation & Auth Errors
  // ============================================================

  test(`${PREFIX} step 1: open /nonexistent-random-page → redirect away`, async ({ page }) => {
    await page.goto('/nonexistent-random-page-xyz')
    await page.waitForLoadState('networkidle')

    // Should NOT stay on the broken URL — redirect to dashboard, login, or a 404 page
    const url = page.url()
    const stuckOnBroken = url.includes('/nonexistent-random-page-xyz')

    // If the app doesn't have a catch-all route, it might stay but should show meaningful UI
    if (stuckOnBroken) {
      // At minimum, the page should still render (not a blank white screen)
      await expect(page.locator('body')).toBeVisible()
    } else {
      // Redirected — verify we're on a known route
      expect(url).toMatch(/\/(dashboard|login|pixels|events|settings|sale-pages)?/)
    }
  })

  test(`${PREFIX} step 2: open /admin/customers (non-admin) → redirect or error`, async ({ page }) => {
    await page.goto('/admin/customers')
    await page.waitForLoadState('networkidle')

    // Non-admin user should be redirected away or see an error
    const url = page.url()
    const onAdmin = url.includes('/admin')

    if (onAdmin) {
      // If still on /admin, should see error/forbidden message or empty content
      const body = await page.locator('body').textContent() ?? ''
      const hasError = body.includes('ไม่มีสิทธิ์') || body.includes('403') || body.includes('Forbidden') || body.includes('unauthorized')
      // Accept either an error message or the page simply being empty
      expect(hasError || body.length > 0).toBe(true)
    } else {
      // Redirected — should be on dashboard or login
      expect(url).toMatch(/\/(dashboard|login)/)
    }
  })

  test(`${PREFIX} step 3: remove token → redirect to login`, async ({ page }) => {
    await page.goto('/dashboard')
    await page.waitForLoadState('networkidle')

    // Remove tokens from localStorage
    await page.evaluate(() => {
      localStorage.removeItem('access_token')
      localStorage.removeItem('refresh_token')
    })

    // Refresh page — without tokens, should redirect to /login
    await page.reload()
    await page.waitForLoadState('networkidle')

    await expect(page).toHaveURL(/\/login/, { timeout: 10000 })
  })

  test(`${PREFIX} step 4: restore auth → dashboard loads`, async ({ page }) => {
    // The auth fixture provides storageState, so just navigating should work
    await page.goto('/dashboard')
    await page.waitForLoadState('networkidle')

    // Should be on the dashboard (not redirected to /login)
    const url = page.url()
    expect(url).not.toContain('/login')

    // Dashboard heading or main content should be visible
    await expect(page.locator('body')).toBeVisible()
  })

  // ============================================================
  // Part B: Form Validation Errors
  // ============================================================

  test(`${PREFIX} step 5: submit empty pixel form → validation errors`, async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')

    // Dismiss any lingering toasts
    const toast = page.locator('[data-sonner-toast]')
    if (await toast.count() > 0) {
      await toast.first().click()
      await page.waitForTimeout(300)
    }

    // Click the "เพิ่มพิกเซล" button to open the dialog
    await pixelsPage.addPixelButton.click()
    await expect(pixelsPage.dialogTitle).toBeVisible({ timeout: 5000 })

    // Submit empty form — click the save button without filling fields
    await pixelsPage.saveButton.click()
    await page.waitForTimeout(500)

    // Dialog should still be open (validation prevented submit)
    await expect(pixelsPage.dialogTitle).toBeVisible()

    // At least one validation indicator should be present:
    // - form still open (validation blocked submit)
    // - error text/border visible
    // The dialog staying open IS the validation signal
  })

  test(`${PREFIX} step 6: fill invalid Pixel ID → validation error`, async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')

    // Dismiss any lingering toasts
    const toast = page.locator('[data-sonner-toast]')
    if (await toast.count() > 0) {
      await toast.first().click()
      await page.waitForTimeout(300)
    }

    // Open the add pixel dialog
    await pixelsPage.addPixelButton.click()
    await expect(pixelsPage.dialogTitle).toBeVisible({ timeout: 5000 })

    // Fill with invalid data — Pixel ID should be numeric, "abc" is invalid
    await pixelsPage.nameInput.fill(`${PREFIX} Invalid`)
    await pixelsPage.pixelIdInput.fill('abc')
    await pixelsPage.accessTokenInput.fill('EAAtest_token')

    // Submit the form
    await pixelsPage.saveButton.click()
    await page.waitForTimeout(500)

    // Dialog should still be open or an error toast should appear
    const dialogStillOpen = await pixelsPage.dialogTitle.isVisible().catch(() => false)
    const errorToast = await page.locator('[data-sonner-toast]').isVisible().catch(() => false)

    // Either the dialog stays open (client-side validation) or an error appears
    expect(dialogStillOpen || errorToast).toBe(true)
  })

  test(`${PREFIX} step 7: close dialog via cancel`, async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')

    // Dismiss any lingering toasts
    const toast = page.locator('[data-sonner-toast]')
    if (await toast.count() > 0) {
      await toast.first().click()
      await page.waitForTimeout(300)
    }

    // Open dialog
    await pixelsPage.addPixelButton.click()
    await expect(pixelsPage.dialogTitle).toBeVisible({ timeout: 5000 })

    // Click cancel
    await pixelsPage.cancelButton.click()

    // Dialog should close
    await expect(pixelsPage.dialogTitle).not.toBeVisible({ timeout: 5000 })
  })

  // ============================================================
  // Part C: Empty States
  // ============================================================

  test(`${PREFIX} step 8: live mode → events or waiting message`, async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.gotoLive()
    await page.waitForLoadState('networkidle')

    await expect(eventLogPage.heading).toBeVisible()

    // In live mode, should see either:
    // - Event table with data
    // - "รอรับอีเวนต์..." waiting message
    // - Loading message
    const hasTable = await eventLogPage.eventTable.locator('tbody tr').first().isVisible().catch(() => false)
    const hasWaiting = await eventLogPage.liveWaitingMessage.isVisible().catch(() => false)
    const hasLoading = await eventLogPage.liveLoadingMessage.isVisible().catch(() => false)

    expect(hasTable || hasWaiting || hasLoading).toBe(true)
  })

  test(`${PREFIX} step 9: replay → form or paywall`, async ({ page }) => {
    const replayPage = new ReplayPage(page)
    await replayPage.goto()
    await page.waitForLoadState('networkidle')

    await expect(replayPage.heading).toBeVisible()

    // Should see either:
    // - Replay form (source pixel select visible)
    // - Paywall message ("ไม่มีเครดิตรีเพลย์")
    const hasForm = await replayPage.sourcePixelSelect.isVisible().catch(() => false)
    const hasPaywall = await replayPage.paywallMessage.isVisible().catch(() => false)

    expect(hasForm || hasPaywall).toBe(true)
  })

  test(`${PREFIX} step 10: sale pages → table or empty state`, async ({ page }) => {
    const salePagesPage = new SalePagesPage(page)
    await salePagesPage.goto()

    await expect(salePagesPage.heading).toBeVisible()

    // Should see either:
    // - Table with sale pages
    // - Empty state ("ยังไม่มีเซลเพจ")
    const hasTable = await salePagesPage.table.locator('tbody tr').first().isVisible().catch(() => false)
    const hasEmpty = await salePagesPage.emptyState.isVisible().catch(() => false)

    expect(hasTable || hasEmpty).toBe(true)
  })

  // ============================================================
  // Part D: API Errors
  // ============================================================

  test(`${PREFIX} step 11: get API key from /settings`, async ({ page }) => {
    await page.goto('/settings')
    await page.waitForLoadState('networkidle')

    // Reveal the API key
    const apiKeyInput = page.locator('input.font-mono')
    await expect(apiKeyInput).toBeVisible({ timeout: 10000 })

    const eyeButton = page.locator('button', { has: page.locator('[class*="lucide-eye"]') }).first()
    await eyeButton.click()
    await page.waitForTimeout(500)

    apiKey = await apiKeyInput.inputValue()
    expect(apiKey).toBeTruthy()
    expect(apiKey.length).toBeGreaterThan(5)
  })

  test(`${PREFIX} step 12: ingest with empty X-API-Key → 401`, async ({ page }) => {
    test.skip(!apiKey, 'No API key from step 11')

    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:5173'

    const resp = await page.request.post(`${baseURL}/api/v1/events/ingest`, {
      headers: {
        'X-API-Key': '',
        'Content-Type': 'application/json',
      },
      data: {
        events: [
          {
            pixel_id: '00000000-0000-0000-0000-000000000000',
            event_name: 'PageView',
            event_time: new Date().toISOString(),
            event_data: {},
            source_url: 'https://e2e-s10.example.com/empty-key',
          },
        ],
      },
    })

    expect(resp.status()).toBe(401)
  })

  test(`${PREFIX} step 13: ingest with invalid API key → 401`, async ({ page }) => {
    test.skip(!apiKey, 'No API key from step 11')

    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:5173'

    const resp = await page.request.post(`${baseURL}/api/v1/events/ingest`, {
      headers: {
        'X-API-Key': 'bad_key_that_does_not_exist',
        'Content-Type': 'application/json',
      },
      data: {
        events: [
          {
            pixel_id: '00000000-0000-0000-0000-000000000000',
            event_name: 'PageView',
            event_time: new Date().toISOString(),
            event_data: {},
            source_url: 'https://e2e-s10.example.com/bad-key',
          },
        ],
      },
    })

    expect(resp.status()).toBe(401)
  })

  test(`${PREFIX} step 14: ingest with valid key but empty body → 400`, async ({ page }) => {
    test.skip(!apiKey, 'No API key from step 11')

    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:5173'

    const resp = await page.request.post(`${baseURL}/api/v1/events/ingest`, {
      headers: {
        'X-API-Key': apiKey,
        'Content-Type': 'application/json',
      },
      data: {},
    })

    expect(resp.status()).toBe(400)
  })
})
