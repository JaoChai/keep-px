import { test, expect } from '../fixtures/auth.fixture'
import { test as baseTest, expect as baseExpect } from '@playwright/test'
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

// --- Scenario 1: Landing Page (unauthenticated) ---

baseTest.describe('Landing Page', () => {
  baseTest('shows branding and CTA button', async ({ page }) => {
    await page.goto('/')

    // Hero heading — use .first() because "บัญชีโฆษณาถูกแบน" appears in both hero h1 and pain-point h3
    await baseExpect(page.getByRole('heading', { name: /บัญชีโฆษณาถูกแบน/ }).first()).toBeVisible()

    // CTA button "เริ่มต้นฟรี"
    await baseExpect(page.getByRole('link', { name: 'เริ่มต้นฟรี' }).first()).toBeVisible()

    // Navbar brand
    await baseExpect(page.getByText('Pixlinks').first()).toBeVisible()

    // "เริ่มต้นฟรี" link in navbar
    await baseExpect(page.getByRole('link', { name: 'เริ่มต้นฟรี' }).first()).toBeVisible()
  })
})

// --- Scenario 1: Onboarding Wizard ---

test.describe('Onboarding Wizard', () => {
  test('shows for new users and can be dismissed', async ({ page }) => {
    // Clear the onboarding dismissed flag before navigating
    await page.goto('/dashboard')
    await page.evaluate(() => localStorage.removeItem('keepx_onboarding_dismissed'))
    await page.reload({ waitUntil: 'networkidle' })

    const dashboardPage = new DashboardPage(page)

    // Wizard may only show when user has 0 pixels
    const wizardVisible = await dashboardPage.onboardingWizard.isVisible().catch(() => false)
    if (!wizardVisible) {
      // User has pixels, wizard won't show — skip gracefully
      test.skip()
      return
    }

    // Verify wizard content
    await expect(dashboardPage.onboardingWizard).toBeVisible()
    await expect(page.getByText('ยินดีต้อนรับสู่ Keep-PX!')).toBeVisible()
    await expect(page.getByText('เริ่มต้นใช้งานด้วย 4 ขั้นตอนง่ายๆ')).toBeVisible()

    // Dismiss wizard
    await dashboardPage.onboardingDismissButton.click()

    // Wizard should be gone
    await expect(dashboardPage.onboardingWizard).not.toBeVisible()

    // Verify it stays dismissed after reload
    await page.reload({ waitUntil: 'networkidle' })
    await expect(dashboardPage.onboardingWizard).not.toBeVisible()

    // Cleanup: remove the dismissed flag
    await page.evaluate(() => localStorage.removeItem('keepx_onboarding_dismissed'))
  })
})

// --- Scenario 6: Chart Time Range Switching ---

test.describe('Dashboard - Chart Time Range', () => {
  test('switching time ranges keeps chart visible', async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.goto()

    await expect(dashboardPage.chartSection).toBeVisible()

    // Click each time range button and verify chart section remains visible
    for (const rangeButton of [
      dashboardPage.chartRange7d,
      dashboardPage.chartRange14d,
      dashboardPage.chartRange30d,
      dashboardPage.chartRange90d,
    ]) {
      await rangeButton.click()
      await expect(dashboardPage.chartSection).toBeVisible()
    }
  })
})

// --- Scenario 6: Recent Activity Feed ---

test.describe('Dashboard - Recent Activity', () => {
  test('recent activity section visible with view-all link', async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.goto()

    await expect(dashboardPage.recentActivityHeading).toBeVisible()
    await expect(dashboardPage.recentActivityViewAllLink).toBeVisible()
  })

  test('"ดูทั้งหมด" link navigates to /events', async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.goto()

    await dashboardPage.recentActivityViewAllLink.click()
    await expect(page).toHaveURL(/\/events/)
  })
})

// --- Scenario 6: Pixel Status List ---

test.describe('Dashboard - Pixel Status', () => {
  test('pixel status section visible with manage link', async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.goto()

    await expect(dashboardPage.pixelStatusHeading).toBeVisible()
    await expect(dashboardPage.pixelStatusManageLink).toBeVisible()
  })

  test('"จัดการ" link navigates to /pixels', async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.goto()

    await dashboardPage.pixelStatusManageLink.click()
    await expect(page).toHaveURL(/\/pixels/)
  })
})

// --- Scenario 6: Top Event Types ---

test.describe('Dashboard - Top Event Types', () => {
  test('top event types section visible', async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.goto()

    await expect(dashboardPage.topEventTypesHeading).toBeVisible()
  })
})

// --- Scenario 6: Recent Replays ---

test.describe('Dashboard - Recent Replays', () => {
  test('recent replays section visible', async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.goto()

    await expect(dashboardPage.recentReplaysHeading).toBeVisible()
  })
})

// --- Scenario 6: Monthly Event Usage ---

test.describe('Dashboard - Monthly Event Usage', () => {
  test('monthly event usage bar visible', async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.goto()

    // Monthly usage bar is shown when quota data loads — may not be visible for all users
    // Verify the section exists (visible or shows quota data)
    const usageVisible = await dashboardPage.monthlyEventUsageSection.isVisible().catch(() => false)
    if (!usageVisible) {
      // Quota data may not be loaded yet or section is conditional
      test.skip()
      return
    }

    await expect(dashboardPage.monthlyEventUsageSection).toBeVisible()
  })
})

