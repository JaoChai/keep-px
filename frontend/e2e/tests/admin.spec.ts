import { test, expect } from '../fixtures/auth.fixture'
import {
  AdminCustomersPage,
  AdminBillingPage,
  AdminSalePagesPage,
  AdminPixelsPage,
  AdminReplaysPage,
  AdminEventsPage,
  AdminAuditLogPage,
} from '../pages/admin.page'

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
      await expect(page.getByRole('heading', { name: 'ลูกค้า' }).first()).toBeVisible()
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

test.describe('Admin - Customer Filters', () => {
  test('filter customers by plan (admin only)', async ({ page }) => {
    await page.goto('/admin/customers')
    await page.waitForLoadState('networkidle')
    test.skip(!page.url().includes('/admin/'), 'E2E user is not admin')

    const customersPage = new AdminCustomersPage(page)
    await expect(customersPage.planFilter).toBeVisible()

    // Select a specific plan
    await customersPage.planFilter.selectOption('sandbox')
    await page.waitForTimeout(500)

    // Table should still be visible (may show filtered results or empty)
    await expect(customersPage.table).toBeVisible()
  })

  test('filter customers by status (admin only)', async ({ page }) => {
    await page.goto('/admin/customers')
    await page.waitForLoadState('networkidle')
    test.skip(!page.url().includes('/admin/'), 'E2E user is not admin')

    const customersPage = new AdminCustomersPage(page)
    await expect(customersPage.statusFilter).toBeVisible()

    // Select active status
    await customersPage.statusFilter.selectOption('active')
    await page.waitForTimeout(500)

    // Table should still be visible
    await expect(customersPage.table).toBeVisible()
  })
})

test.describe('Admin - Customer Detail', () => {
  test('view customer detail dialog (admin only)', async ({ page }) => {
    await page.goto('/admin/customers')
    await page.waitForLoadState('networkidle')
    test.skip(!page.url().includes('/admin/'), 'E2E user is not admin')

    const customersPage = new AdminCustomersPage(page)

    // Wait for table data to load
    await expect(customersPage.table).toBeVisible()

    // Check if there are customer rows to click
    const hasCustomers = await customersPage.firstCustomerRow.isVisible().catch(() => false)
    if (!hasCustomers) {
      test.skip(true, 'No customers in the table')
      return
    }

    // Check the row is not the "no customers" message
    const isEmptyRow = await customersPage.noCustomersMessage.isVisible().catch(() => false)
    if (isEmptyRow) {
      test.skip(true, 'No customers found')
      return
    }

    // Click the first customer row
    await customersPage.firstCustomerRow.click()
    await page.waitForTimeout(500)

    // Dialog should open with customer detail
    await expect(customersPage.customerDetailTitle).toBeVisible()
    await expect(customersPage.customerDetailDialog).toBeVisible()

    // Verify key sections are present in the dialog
    await expect(page.getByText('ข้อมูลบัญชี')).toBeVisible()
    await expect(page.getByText('สถิติการใช้งาน')).toBeVisible()
    await expect(page.getByText('จัดการ')).toBeVisible()
  })

  test.fixme('grant credits to customer (admin only)', async ({ page }) => {
    // Requires admin access + mutation capability
    // Would: open customer detail -> fill credit form -> click "เพิ่มเครดิต"
    // Verify: grantCreditsSection, creditAmountInput, grantCreditsButton
    await page.goto('/admin/customers')
    await page.waitForLoadState('networkidle')
  })

  test.fixme('change customer plan (admin only)', async ({ page }) => {
    // Requires admin access + mutation capability
    // Would: open customer detail -> select new plan -> click "บันทึก"
    // Verify: changePlanSelect, savePlanButton
    await page.goto('/admin/customers')
    await page.waitForLoadState('networkidle')
  })

  test.fixme('suspend and activate customer (admin only)', async ({ page }) => {
    // Requires admin access + safe test customer
    // Would: open customer detail -> click "ระงับบัญชี" -> confirm
    // Then: click "เปิดใช้งานบัญชี" to revert
    // Verify: suspendButton or activateButton, confirmSuspendButton
    await page.goto('/admin/customers')
    await page.waitForLoadState('networkidle')
  })
})

test.describe('Admin - Resource Pages', () => {
  test('admin sale pages page loads (admin only)', async ({ page }) => {
    await page.goto('/admin/sale-pages')
    await page.waitForLoadState('networkidle')
    test.skip(!page.url().includes('/admin/'), 'E2E user is not admin')

    const salePagesPage = new AdminSalePagesPage(page)
    await expect(salePagesPage.heading).toBeVisible()
    await expect(salePagesPage.table).toBeVisible()
    await expect(salePagesPage.searchInput).toBeVisible()
    await expect(salePagesPage.publishedFilter).toBeVisible()
  })

  test('admin pixels page loads (admin only)', async ({ page }) => {
    await page.goto('/admin/pixels')
    await page.waitForLoadState('networkidle')
    test.skip(!page.url().includes('/admin/'), 'E2E user is not admin')

    const pixelsPage = new AdminPixelsPage(page)
    await expect(pixelsPage.heading).toBeVisible()
    await expect(pixelsPage.table).toBeVisible()
    await expect(pixelsPage.searchInput).toBeVisible()
    await expect(pixelsPage.activeFilter).toBeVisible()
  })

  test('admin replays page loads (admin only)', async ({ page }) => {
    await page.goto('/admin/replays')
    await page.waitForLoadState('networkidle')
    test.skip(!page.url().includes('/admin/'), 'E2E user is not admin')

    const replaysPage = new AdminReplaysPage(page)
    await expect(replaysPage.heading).toBeVisible()
    await expect(replaysPage.table).toBeVisible()
    await expect(replaysPage.statusFilter).toBeVisible()
  })

  test('admin events page loads with stats and table (admin only)', async ({ page }) => {
    await page.goto('/admin/events')
    await page.waitForLoadState('networkidle')
    test.skip(!page.url().includes('/admin/'), 'E2E user is not admin')

    const eventsPage = new AdminEventsPage(page)
    await expect(eventsPage.heading).toBeVisible()
    await expect(eventsPage.table).toBeVisible()

    // Verify stat cards are present
    await expect(page.getByText('อีเวนต์วันนี้')).toBeVisible()
    await expect(page.getByText('ชั่วโมงนี้')).toBeVisible()

    // Verify filter inputs are present
    await expect(eventsPage.customerIdInput).toBeVisible()
    await expect(eventsPage.pixelIdInput).toBeVisible()
    await expect(eventsPage.eventNameInput).toBeVisible()
  })
})

test.describe('Admin - Audit Log', () => {
  test('admin audit log page loads (admin only)', async ({ page }) => {
    await page.goto('/admin/audit-log')
    await page.waitForLoadState('networkidle')
    test.skip(!page.url().includes('/admin/'), 'E2E user is not admin')

    const auditLogPage = new AdminAuditLogPage(page)
    await expect(auditLogPage.heading).toBeVisible()
    await expect(auditLogPage.table).toBeVisible()
    await expect(auditLogPage.actionFilter).toBeVisible()
  })

  test('audit log action filter works (admin only)', async ({ page }) => {
    await page.goto('/admin/audit-log')
    await page.waitForLoadState('networkidle')
    test.skip(!page.url().includes('/admin/'), 'E2E user is not admin')

    const auditLogPage = new AdminAuditLogPage(page)

    // Select a specific action filter
    await auditLogPage.actionFilter.selectOption('grant_credits')
    await page.waitForTimeout(500)

    // Table should still be visible (may show filtered results or empty)
    await expect(auditLogPage.table).toBeVisible()
  })
})
