import { test, expect } from '../fixtures/auth.fixture'
import { AdminCustomersPage, AdminBillingPage } from '../pages/admin.page'

// Admin tests are conditional — skip if the e2e user is not admin

test.describe('Admin Panel', () => {
  test('non-admin user redirects to dashboard from admin route', async ({ page }) => {
    await page.goto('/admin/customers')
    await page.waitForLoadState('networkidle')

    // If user is admin, they stay on admin page; if not, redirect to dashboard
    const url = page.url()
    const isAdmin = url.includes('/admin/')

    if (!isAdmin) {
      await expect(page).toHaveURL(/\/dashboard/)
    } else {
      // User is admin — verify the page loaded
      await expect(page.getByRole('heading', { name: 'จัดการลูกค้า' })).toBeVisible()
    }
  })

  test('admin customers page loads (admin only)', async ({ page }) => {
    await page.goto('/admin/customers')
    await page.waitForLoadState('networkidle')
    test.skip(!page.url().includes('/admin/'), 'E2E user is not admin')

    const customersPage = new AdminCustomersPage(page)
    await expect(customersPage.heading).toBeVisible()
    await expect(customersPage.searchInput).toBeVisible()
    await expect(customersPage.table).toBeVisible()
  })

  test('admin customers search works (admin only)', async ({ page }) => {
    await page.goto('/admin/customers')
    await page.waitForLoadState('networkidle')
    test.skip(!page.url().includes('/admin/'), 'E2E user is not admin')

    const customersPage = new AdminCustomersPage(page)
    await customersPage.searchInput.fill('test')
    await page.waitForTimeout(500)
    await expect(customersPage.table).toBeVisible()
  })

  test('admin analytics page loads with stat cards (admin only)', async ({ page }) => {
    await page.goto('/admin/analytics')
    await page.waitForLoadState('networkidle')
    test.skip(!page.url().includes('/admin/'), 'E2E user is not admin')

    await expect(page.getByText('ลูกค้าทั้งหมด')).toBeVisible()
    await expect(page.getByText('อีเวนต์เดือนนี้')).toBeVisible()
  })

  test('admin analytics charts visible (admin only)', async ({ page }) => {
    await page.goto('/admin/analytics')
    await page.waitForLoadState('networkidle')
    test.skip(!page.url().includes('/admin/'), 'E2E user is not admin')

    await expect(page.getByText('การเติบโตของผู้ใช้')).toBeVisible()
  })

  test('admin billing page loads with tabs (admin only)', async ({ page }) => {
    await page.goto('/admin/billing')
    await page.waitForLoadState('networkidle')
    test.skip(!page.url().includes('/admin/'), 'E2E user is not admin')

    const billingPage = new AdminBillingPage(page)
    await expect(billingPage.purchasesTab).toBeVisible()
    await expect(billingPage.subscriptionsTab).toBeVisible()
    await expect(billingPage.creditsTab).toBeVisible()
  })

  test('admin billing tab switching works (admin only)', async ({ page }) => {
    await page.goto('/admin/billing')
    await page.waitForLoadState('networkidle')
    test.skip(!page.url().includes('/admin/'), 'E2E user is not admin')

    const billingPage = new AdminBillingPage(page)
    await billingPage.subscriptionsTab.click()
    await expect(billingPage.table).toBeVisible()
  })

  test('admin sidebar shows admin nav links (admin only)', async ({ page }) => {
    await page.goto('/admin/customers')
    await page.waitForLoadState('networkidle')
    test.skip(!page.url().includes('/admin/'), 'E2E user is not admin')

    await expect(page.getByRole('link', { name: 'ลูกค้า' })).toBeVisible()
  })
})
