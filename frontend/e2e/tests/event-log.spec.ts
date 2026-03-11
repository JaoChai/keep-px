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

    // If table exists, check for pagination elements
    const hasTable = await eventLogPage.eventTable.isVisible()
    if (hasTable) {
      // Page info text should be visible
      await expect(page.getByText(/หน้า \d+ จาก \d+/)).toBeVisible()
    }
  })
})
