import { test, expect } from '../fixtures/auth.fixture'
import { BillingPage } from '../pages/billing.page'

test.describe('Billing @smoke', () => {
  test('page loads with heading', async ({ page }) => {
    const billing = new BillingPage(page)
    await billing.goto()

    await expect(billing.heading).toBeVisible()
  })

  test('account status shows quotas', async ({ page }) => {
    const billing = new BillingPage(page)
    await billing.goto()

    await expect(billing.eventsQuota).toBeVisible()
    await expect(billing.pixelsQuota).toBeVisible()
    await expect(billing.replaysQuota).toBeVisible()
    await expect(billing.retentionInfo).toBeVisible()
  })

  test('pixel slots section is visible with pricing', async ({ page }) => {
    const billing = new BillingPage(page)
    await billing.goto()

    await expect(billing.pixelSlotsHeading).toBeVisible()
    await expect(billing.quantityDisplay).toBeVisible()
    await expect(billing.slotPriceDisplay).toBeVisible()
  })

  test('replay section shows 2 options', async ({ page }) => {
    const billing = new BillingPage(page)
    await billing.goto()

    await expect(billing.replayHeading).toBeVisible()
    await expect(billing.replaySingleCard).toBeVisible()
    await expect(billing.replayMonthlyCard).toBeVisible()

    // Check prices
    await expect(page.getByText('฿299')).toBeVisible()
    await expect(page.getByText('฿1,990')).toBeVisible()

    // Credits section: either active credits or "no credits" message (state-agnostic)
    const hasCredits = await page.getByText('เครดิตที่มีอยู่').isVisible().catch(() => false)
    if (hasCredits) {
      await expect(page.getByText('เครดิตที่มีอยู่')).toBeVisible()
    } else {
      await expect(page.getByText('ยังไม่มีเครดิตรีเพลย์')).toBeVisible()
    }
  })

  test('purchase history section is visible', async ({ page }) => {
    const billing = new BillingPage(page)
    await billing.goto()

    await expect(billing.purchaseHistorySection).toBeVisible()
  })
})

// --- Scenario 6: Stripe Portal Button ---

test.describe('Billing - Manage Billing', () => {
  test('manage billing button or upgrade button is visible', async ({ page }) => {
    const billing = new BillingPage(page)
    await billing.goto()

    // For paid users: "จัดการการชำระเงิน" button
    // For free users: "อัปเกรด" button
    // Either one should be present in the account status card
    const manageBillingVisible = await billing.manageBillingButton.isVisible().catch(() => false)
    const upgradeButtonVisible = await page.getByRole('button', { name: 'อัปเกรด', exact: true }).isVisible().catch(() => false)

    // At least one action button should be present
    expect(manageBillingVisible || upgradeButtonVisible).toBeTruthy()

    // If the manage billing button is visible, we just verify it exists
    // (don't click — it redirects to Stripe portal)
    if (manageBillingVisible) {
      await expect(billing.manageBillingButton).toBeVisible()
    }
  })
})
