import { test, expect } from '../fixtures/auth.fixture'
import { DashboardPage } from '../pages/dashboard.page'

test.describe('Dashboard @smoke', () => {
  test('4 stats cards visible', async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.goto()

    await expect(dashboardPage.heading).toBeVisible()

    // Verify all 4 stat card titles are visible
    await expect(page.getByText('Active Pixels')).toBeVisible()
    await expect(page.getByText('Events Today')).toBeVisible()
    await expect(page.getByText('Total Events')).toBeVisible()
    await expect(page.getByText('Replays')).toBeVisible()
  })

  test('chart section renders', async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.goto()

    await expect(dashboardPage.chartSection).toBeVisible()
  })
})
