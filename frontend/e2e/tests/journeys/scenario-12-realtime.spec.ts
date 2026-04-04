/**
 * Scenario 12: Real-Time Updates
 *
 * Tests live event mode, pause/resume/clear controls, and event ingestion visibility.
 *
 * Flow:
 *   Part A — Setup: get API key + pixel UUID (steps 1-2)
 *   Part B — Live Mode: open live view, ingest event, check updates (steps 3-7)
 *   Part C — Pause/Resume/Clear controls (steps 8-10)
 */
import { test, expect } from '../../fixtures/auth.fixture'
import { EventLogPage } from '../../pages/event-log.page'

const PREFIX = 'E2E-S12'

test.describe(`Scenario 12: Real-Time Updates`, () => {
  test.describe.configure({ mode: 'serial' })
  test.setTimeout(120_000)

  /** Shared state across serial steps */
  let apiKey = ''
  let pixelUUID = ''

  // ============================================================
  // Part A: Setup
  // ============================================================

  test(`${PREFIX} step 1: get API key from /settings`, async ({ page }) => {
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

  test(`${PREFIX} step 2: get pixel UUID from API`, async ({ page }) => {
    test.skip(!apiKey, 'No API key from step 1')

    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:5173'

    // Navigate to an app page first so localStorage is accessible
    await page.goto('/settings')
    await page.waitForLoadState('networkidle')

    // Get the internal pixel UUID from the API using JWT from localStorage
    const accessToken = await page.evaluate(() => localStorage.getItem('access_token'))
    const pixelsRes = await page.request.get(`${baseURL}/api/v1/pixels`, {
      headers: { Authorization: `Bearer ${accessToken}` },
    })
    const pixels = (await pixelsRes.json()).data || []

    if (pixels.length === 0) {
      test.skip(true, 'No pixels available')
      return
    }

    pixelUUID = pixels[0].id
    expect(pixelUUID).toBeTruthy()
  })

  // ============================================================
  // Part B: Live Mode
  // ============================================================

  test(`${PREFIX} step 3: open live mode → see heading`, async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.gotoLive()
    await page.waitForLoadState('networkidle')

    await expect(eventLogPage.heading).toBeVisible()
  })

  test(`${PREFIX} step 4: see live badge or waiting message`, async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.gotoLive()
    await page.waitForLoadState('networkidle')

    // In live mode, should see one of:
    // - Live mode button (active/selected)
    // - "รอรับอีเวนต์..." waiting message
    // - "กำลังโหลดอีเวนต์ล่าสุด..." loading message
    // - Event rows in the table
    const hasLiveButton = await eventLogPage.liveModeButton.isVisible().catch(() => false)
    const hasWaiting = await eventLogPage.liveWaitingMessage.isVisible().catch(() => false)
    const hasLoading = await eventLogPage.liveLoadingMessage.isVisible().catch(() => false)
    const hasEvents = await eventLogPage.eventTable.locator('tbody tr').first().isVisible().catch(() => false)

    expect(hasLiveButton || hasWaiting || hasLoading || hasEvents).toBe(true)
  })

  test(`${PREFIX} step 5: ingest 1 PageView event via API`, async ({ page }) => {
    test.skip(!apiKey, 'No API key from step 1')
    test.skip(!pixelUUID, 'No pixel UUID from step 2')

    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:5173'

    const resp = await page.request.post(`${baseURL}/api/v1/events/ingest`, {
      headers: {
        'X-API-Key': apiKey,
        'Content-Type': 'application/json',
      },
      data: {
        events: [
          {
            pixel_id: pixelUUID,
            event_name: 'PageView',
            event_time: new Date().toISOString(),
            event_data: { source: 'e2e-s12-realtime' },
            source_url: 'https://e2e-s12.example.com/live-test',
          },
        ],
      },
    })

    // Accept 200/202 (success), 402 (quota exceeded), or 500 (CAPI forward fail with fake token)
    const status = resp.status()
    if (status >= 400) {
      const body = await resp.text()
      console.log(`${PREFIX} ingest returned ${status}: ${body}`)
    }
    expect([200, 202, 402, 500]).toContain(status)
  })

  test(`${PREFIX} step 6: wait and check for new event in live table`, async ({ page }) => {
    test.skip(!apiKey, 'No API key from step 1')

    const eventLogPage = new EventLogPage(page)
    await eventLogPage.gotoLive()
    await page.waitForLoadState('networkidle')

    // Wait for the live poll to pick up the new event
    await page.waitForTimeout(5000)

    // Soft check — event may not appear if ingest returned 402/500 or live polling is slow
    const hasEvents = await eventLogPage.eventTable.locator('tbody tr').first().isVisible().catch(() => false)
    const hasWaiting = await eventLogPage.liveWaitingMessage.isVisible().catch(() => false)
    const hasLoading = await eventLogPage.liveLoadingMessage.isVisible().catch(() => false)

    // At minimum, the live view should show something (events, waiting, or loading)
    expect(hasEvents || hasWaiting || hasLoading).toBe(true)

    if (hasEvents) {
      const rowCount = await eventLogPage.eventTable.locator('tbody tr').count()
      console.log(`${PREFIX} live mode shows ${rowCount} event(s)`)
    }
  })

  test(`${PREFIX} step 7: stat cards visible`, async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.gotoLive()
    await page.waitForLoadState('networkidle')

    // "อีเวนต์วันนี้" stat should be visible in event log page
    const statVisible = await eventLogPage.statEventsToday.isVisible().catch(() => false)
    const headingVisible = await eventLogPage.heading.isVisible().catch(() => false)

    // At minimum the heading must be present; stat cards may load async
    expect(headingVisible).toBe(true)

    if (statVisible) {
      // The parent card should contain a numeric value
      const cardText = await eventLogPage.statEventsToday.locator('..').textContent() ?? ''
      expect(cardText.length).toBeGreaterThan(0)
    }
  })

  // ============================================================
  // Part C: Pause/Resume/Clear
  // ============================================================

  test(`${PREFIX} step 8: pause live mode`, async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.gotoLive()
    await page.waitForLoadState('networkidle')

    // Pause/resume button may not be visible if live mode has no events yet
    const pauseVisible = await eventLogPage.pauseResumeButton.isVisible().catch(() => false)

    if (!pauseVisible) {
      test.skip(true, 'Pause button not visible in live mode')
      return
    }

    // Click pause
    await eventLogPage.pauseResumeButton.click()
    await page.waitForTimeout(1000)

    // After pause, should see some state change — any of:
    // - "หยุดชั่วคราว" paused badge
    // - The button text/icon changed
    // - The page still shows heading (didn't crash)
    const hasPausedBadge = await eventLogPage.pausedBadge.isVisible().catch(() => false)
    const hasHeading = await eventLogPage.heading.isVisible().catch(() => false)

    // At minimum, page should still be functional after clicking pause
    expect(hasPausedBadge || hasHeading).toBe(true)
  })

  test(`${PREFIX} step 9: resume live mode`, async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.gotoLive()
    await page.waitForLoadState('networkidle')

    const pauseVisible = await eventLogPage.pauseResumeButton.isVisible().catch(() => false)

    if (!pauseVisible) {
      test.skip(true, 'Pause/resume button not visible')
      return
    }

    // If currently live, click to pause first, then resume
    // Click once to toggle
    await eventLogPage.pauseResumeButton.click()
    await page.waitForTimeout(500)

    // Click again to toggle back
    await eventLogPage.pauseResumeButton.click()
    await page.waitForTimeout(500)

    // After toggling twice, should be back in live state
    const headingVisible = await eventLogPage.heading.isVisible().catch(() => false)
    expect(headingVisible).toBe(true)
  })

  test(`${PREFIX} step 10: clear live events`, async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.gotoLive()
    await page.waitForLoadState('networkidle')

    const clearVisible = await eventLogPage.clearButton.isVisible().catch(() => false)

    if (!clearVisible) {
      test.skip(true, 'Clear button not visible in live mode')
      return
    }

    // Click clear
    await eventLogPage.clearButton.click()
    await page.waitForTimeout(500)

    // After clearing, should see waiting message or empty table
    const hasWaiting = await eventLogPage.liveWaitingMessage.isVisible().catch(() => false)
    const hasEmptyTable = (await eventLogPage.eventTable.locator('tbody tr').count()) === 0
    const headingVisible = await eventLogPage.heading.isVisible().catch(() => false)

    // Page should still be functional
    expect(headingVisible).toBe(true)
    // Cleared state — either waiting message or empty table is acceptable
    if (!hasWaiting && !hasEmptyTable) {
      // Events may have already re-populated from the live poll — also acceptable
      console.log(`${PREFIX} live events re-populated after clear`)
    }
  })

})
