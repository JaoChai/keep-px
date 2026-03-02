import { test, expect } from '../fixtures/auth.fixture'
import { DashboardPage } from '../pages/dashboard.page'

test.describe('Dashboard @smoke', () => {
  test('5 stats cards visible', async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.goto()

    await expect(dashboardPage.heading).toBeVisible()

    // Verify all 5 stat card titles are visible
    await expect(page.getByText('Active Pixels')).toBeVisible()
    await expect(page.getByText('Events Today')).toBeVisible()
    await expect(page.getByText('CAPI Rate')).toBeVisible()
    await expect(page.getByText('Events This Week')).toBeVisible()
    await expect(page.getByText('Active Replays')).toBeVisible()
  })

  test('chart section renders', async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.goto()

    await expect(dashboardPage.chartSection).toBeVisible()
  })
})
