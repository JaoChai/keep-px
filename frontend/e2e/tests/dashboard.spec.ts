import { test, expect } from '../fixtures/auth.fixture'
import { DashboardPage } from '../pages/dashboard.page'

test.describe('Dashboard @smoke', () => {
  test('5 stats cards visible', async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.goto()

    await expect(dashboardPage.heading).toBeVisible()

    // Verify all 5 stat card titles are visible
    await expect(page.getByText('พิกเซลที่ใช้งาน')).toBeVisible()
    await expect(page.getByText('อีเวนต์วันนี้')).toBeVisible()
    await expect(page.getByText('อัตรา CAPI')).toBeVisible()
    await expect(page.getByText('อีเวนต์สัปดาห์นี้')).toBeVisible()
    await expect(page.getByText('รีเพลย์ที่ทำงาน')).toBeVisible()
  })

  test('chart section renders', async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.goto()

    await expect(dashboardPage.chartSection).toBeVisible()
  })
})
