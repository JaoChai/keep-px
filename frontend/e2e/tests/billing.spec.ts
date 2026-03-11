import { test, expect } from '../fixtures/auth.fixture'
import { BillingPage } from '../pages/billing.page'

test.describe('Billing @smoke', () => {
  test('page loads with heading and tabs', async ({ page }) => {
    const billing = new BillingPage(page)
    await billing.goto()

    await expect(billing.heading).toBeVisible()
    await expect(billing.plansTab).toBeVisible()
    await expect(billing.replaysTab).toBeVisible()
    await expect(billing.addonsTab).toBeVisible()
  })

  test('account status shows current plan and quotas', async ({ page }) => {
    const billing = new BillingPage(page)
    await billing.goto()

    await expect(billing.currentPlanBadge).toBeVisible()
    await expect(billing.eventsQuota).toBeVisible()
    await expect(billing.replaysQuota).toBeVisible()
    await expect(billing.salePagesQuota).toBeVisible()
    await expect(billing.pixelsQuota).toBeVisible()
  })

  test('plans tab shows 4 plan cards', async ({ page }) => {
    const billing = new BillingPage(page)
    await billing.goto()

    // Plans tab is active by default
    for (const planName of ['Sandbox', 'Launch', 'Shield', 'Vault']) {
      await expect(page.getByText(planName, { exact: true }).first()).toBeVisible()
    }
  })

  test('replays tab shows replay packs', async ({ page }) => {
    const billing = new BillingPage(page)
    await billing.goto()
    await billing.switchToTab('replays')

    // Should show 3 replay packs
    for (const packName of ['Single', 'Triple', 'Unlimited']) {
      await expect(page.getByText(packName, { exact: true }).first()).toBeVisible()
    }

    // Sandbox user should see no credits message
    await expect(page.getByText('ยังไม่มีเครดิตรีเพลย์')).toBeVisible()
  })

  test('addons tab shows 3 addon toggles', async ({ page }) => {
    const billing = new BillingPage(page)
    await billing.goto()
    await billing.switchToTab('addons')

    // Should show 3 addons with prices
    await expect(page.getByText('Pixels +10')).toBeVisible()
    await expect(page.getByText('Sale Pages +10')).toBeVisible()
    await expect(page.getByText('Events +1M')).toBeVisible()

    // Each has a price
    await expect(page.getByText('฿290')).toBeVisible()
    await expect(page.getByText('฿190')).toBeVisible()
    await expect(page.getByText('฿490')).toBeVisible()
  })

  test('manage billing button is visible', async ({ page }) => {
    const billing = new BillingPage(page)
    await billing.goto()

    await expect(billing.manageBillingButton).toBeVisible()
  })

  test('purchase history section is visible', async ({ page }) => {
    const billing = new BillingPage(page)
    await billing.goto()

    await expect(billing.purchaseHistorySection).toBeVisible()
  })
})
