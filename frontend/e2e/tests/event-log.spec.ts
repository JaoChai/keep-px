import { test, expect } from '../fixtures/auth.fixture'
import { EventLogPage } from '../pages/event-log.page'

test.describe('Event Log', () => {
  test('page loads with table or empty state', async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.goto()

    await expect(eventLogPage.heading).toBeVisible()

    // Wait for API data to load before checking content
    await page.waitForLoadState('networkidle')

    // Either the table or empty state should be visible
    await expect(eventLogPage.eventTable.or(eventLogPage.emptyState)).toBeVisible({ timeout: 10000 })
  })

  test('pagination controls visible when events exist', async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.goto()

    await expect(eventLogPage.heading).toBeVisible()

    // If table exists and has enough events for pagination, check pagination elements
    await page.waitForLoadState('networkidle')
    const hasTable = await eventLogPage.eventTable.isVisible()
    if (hasTable) {
      // Pagination only shows when totalPages > 1 — check if text exists
      const paginationText = page.getByText(/หน้า \d+ จาก \d+/)
      const hasPagination = await paginationText.isVisible({ timeout: 3000 }).catch(() => false)
      if (!hasPagination) {
        test.skip(true, 'Not enough events for pagination to appear')
        return
      }
      await expect(paginationText).toBeVisible()
    }
  })
})

test.describe('Event Pipeline Stats', () => {
  test('stat cards are visible on events page', async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.gotoLive()
    await page.waitForLoadState('networkidle')

    // All 4 stat cards should be visible
    await expect(eventLogPage.statEventsToday).toBeVisible({ timeout: 10000 })
    await expect(eventLogPage.statTotalEvents).toBeVisible()
    await expect(eventLogPage.statCapiRate).toBeVisible()
    await expect(eventLogPage.statEventsPerMinute).toBeVisible()
  })
})

test.describe('Event Pipeline Live Mode', () => {
  test('live mode loads by default', async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.gotoLive()

    await expect(eventLogPage.heading).toBeVisible()

    // Live mode button should be active/selected
    await expect(eventLogPage.liveModeButton).toBeVisible()

    // Live controls should be visible
    await expect(eventLogPage.pauseResumeButton).toBeVisible()
    await expect(eventLogPage.clearButton).toBeVisible()
    await expect(eventLogPage.refreshButton).toBeVisible()
  })

  test('pause and resume in live mode', async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.gotoLive()
    await page.waitForLoadState('networkidle')

    // Initially should show "หยุด" (pause) — skip if live controls didn't render
    const pauseButton = page.getByRole('button', { name: /หยุด/ }).first()
    const resumeButton = page.getByRole('button', { name: /ดำเนินต่อ/ }).first()

    const pauseVisible = await pauseButton.isVisible({ timeout: 5000 }).catch(() => false)
    if (!pauseVisible) {
      test.skip(true, 'Live mode controls not rendered — backend may be unavailable')
      return
    }

    // Click pause and wait for state change
    await pauseButton.click()
    await page.waitForTimeout(1000)

    // Button text should change to "ดำเนินต่อ" (resume) — CI needs more time
    await expect(resumeButton).toBeVisible({ timeout: 10000 })

    // Click resume and wait for state change
    await resumeButton.click()
    await page.waitForTimeout(1000)

    // Button should go back to "หยุด" (pause)
    await expect(pauseButton).toBeVisible({ timeout: 10000 })
  })

  test('clear button in live mode', async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.gotoLive()
    await page.waitForLoadState('networkidle')

    // Clear button should be visible
    await expect(eventLogPage.clearButton).toBeVisible()

    // Click clear
    await eventLogPage.clearButton.click()

    // After clearing, the waiting message should appear (if no new events come in)
    // or the table should be empty
    const waitingOrTable = eventLogPage.liveWaitingMessage.or(eventLogPage.eventTable)
    await expect(waitingOrTable).toBeVisible({ timeout: 5000 })
  })
})

test.describe('Event Pipeline Mode Switching', () => {
  test('switch between live and history mode', async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.gotoLive()
    await page.waitForLoadState('networkidle')

    // Should start in live mode
    await expect(eventLogPage.liveModeButton).toBeVisible()

    // Switch to history mode
    await eventLogPage.historyModeButton.click()
    await page.waitForLoadState('networkidle')

    // URL should change to include mode=history
    await expect(page).toHaveURL(/mode=history/)

    // History mode content should load (table or empty state)
    await expect(eventLogPage.eventTable.or(eventLogPage.emptyState)).toBeVisible({ timeout: 10000 })

    // Live controls should NOT be visible in history mode
    await expect(page.getByRole('button', { name: 'หยุด' })).not.toBeVisible()
    await expect(page.getByRole('button', { name: 'ดำเนินต่อ' })).not.toBeVisible()

    // Switch back to live mode
    await eventLogPage.liveModeButton.click()
    await page.waitForLoadState('networkidle')

    // URL should change to include mode=live
    await expect(page).toHaveURL(/mode=live/)

    // Live controls should be visible again
    await expect(eventLogPage.pauseResumeButton).toBeVisible()
  })
})

test.describe('Event Pipeline Filters', () => {
  test('pixel filter exists and has options', async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.gotoHistory()
    await page.waitForLoadState('networkidle')

    // Pixel filter select trigger should be visible
    await expect(eventLogPage.pixelFilter).toBeVisible()

    // Click the pixel filter to open options
    await eventLogPage.pixelFilter.click()

    // "พิกเซลทั้งหมด" option should be visible
    await expect(page.getByRole('option', { name: 'พิกเซลทั้งหมด' })).toBeVisible()

    // Close by pressing Escape
    await page.keyboard.press('Escape')
  })

  test('event type filter exists in history mode', async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.gotoHistory()
    await page.waitForLoadState('networkidle')

    // Event type filter should be visible in history mode
    await expect(eventLogPage.eventTypeFilter).toBeVisible()

    // Click to open
    await eventLogPage.eventTypeFilter.click()

    // "อีเวนต์ทั้งหมด" option should be visible
    await expect(page.getByRole('option', { name: 'อีเวนต์ทั้งหมด' })).toBeVisible()

    // Close by pressing Escape
    await page.keyboard.press('Escape')
  })

  test('date range filter opens and can be cleared', async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.gotoHistory()
    await page.waitForLoadState('networkidle')

    // Date range button should be visible
    await expect(eventLogPage.dateRangeButton).toBeVisible()

    // Click to open popover
    await eventLogPage.dateRangeButton.click()

    // Date inputs should be visible inside popover
    const fromInput = eventLogPage.getDateFromInput()
    const toInput = eventLogPage.getDateToInput()
    await expect(fromInput).toBeVisible()
    await expect(toInput).toBeVisible()

    // Set a date value
    await fromInput.fill('2026-01-01T00:00')

    // Clear date button should now be visible
    await expect(eventLogPage.clearDateButton).toBeVisible()

    // Click clear
    await eventLogPage.clearDateButton.click()

    // Clear button should disappear (no dates set)
    await expect(eventLogPage.clearDateButton).not.toBeVisible()
  })
})

test.describe('Event Pipeline Detail', () => {
  test('event detail sheet opens when clicking a row', async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.gotoHistory()
    await page.waitForLoadState('networkidle')

    // Check if there are events in the table
    const hasEvents = await eventLogPage.eventTable.locator('tbody tr').first().isVisible({ timeout: 5000 }).catch(() => false)
    if (!hasEvents) {
      test.skip(true, 'No events available to test detail sheet')
      return
    }

    // Click the first event row
    await eventLogPage.clickFirstEventRow()

    // Event detail sheet should open
    await expect(eventLogPage.eventDetailSheet).toBeVisible({ timeout: 5000 })
    await expect(eventLogPage.eventDetailDescription).toBeVisible()

    // Close the sheet by pressing Escape
    await page.keyboard.press('Escape')
    await expect(eventLogPage.eventDetailSheet).not.toBeVisible()
  })
})

test.describe('Event Pipeline Export', () => {
  test('export CSV button exists', async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.gotoHistory()
    await page.waitForLoadState('networkidle')

    // Export CSV button should be visible
    await expect(eventLogPage.exportCsvButton).toBeVisible()
  })
})
