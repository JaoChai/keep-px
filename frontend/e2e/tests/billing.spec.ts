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

    // No credits message for free user
    await expect(page.getByText('ยังไม่มีเครดิตรีเพลย์')).toBeVisible()
  })

  test('purchase history section is visible', async ({ page }) => {
    const billing = new BillingPage(page)
    await billing.goto()

    await expect(billing.purchaseHistorySection).toBeVisible()
  })
})
