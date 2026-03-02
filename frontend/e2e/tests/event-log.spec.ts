import { test, expect } from '../fixtures/auth.fixture'
import { EventLogPage } from '../pages/event-log.page'

test.describe('Event Log', () => {
  test('page loads with table or empty state', async ({ page }) => {
    const eventLogPage = new EventLogPage(page)
    await eventLogPage.goto()

    await expect(eventLogPage.heading).toBeVisible()

    // Either the table or empty state should be visible
    const hasTable = await eventLogPage.eventTable.isVisible()
    const hasEmptyState = await eventLogPage.emptyState.isVisible()
    expect(hasTable || hasEmptyState).toBeTruthy()
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
