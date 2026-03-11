import type { Page, Locator } from '@playwright/test'

export class BillingPage {
  readonly page: Page
  readonly heading: Locator

  // Tabs
  readonly plansTab: Locator
  readonly replaysTab: Locator
  readonly addonsTab: Locator

  // Account status card
  readonly currentPlanBadge: Locator
  readonly eventsQuota: Locator
  readonly replaysQuota: Locator
  readonly salePagesQuota: Locator
  readonly pixelsQuota: Locator

  // Manage billing button
  readonly manageBillingButton: Locator

  // Purchase history
  readonly purchaseHistorySection: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'การเงิน' })

    this.plansTab = page.getByRole('button', { name: 'แผน', exact: true })
    this.replaysTab = page.getByRole('button', { name: 'รีเพลย์', exact: true })
    this.addonsTab = page.getByRole('button', { name: 'ส่วนเสริม', exact: true })

    this.currentPlanBadge = page.getByText('แผนปัจจุบัน').first()
    this.eventsQuota = page.getByText('อีเวนต์เดือนนี้')
    this.replaysQuota = page.getByText('รีเพลย์คงเหลือ')
    this.salePagesQuota = page.getByText('Sale Pages', { exact: true })
    this.pixelsQuota = page.locator('span', { hasText: /^พิกเซล$/ })

    this.manageBillingButton = page.getByRole('button', { name: 'จัดการการชำระเงิน' })
    this.purchaseHistorySection = page.getByText(/ประวัติการซื้อ/)
  }

  async goto() {
    await this.page.goto('/billing')
  }

  async switchToTab(tab: 'plans' | 'replays' | 'addons') {
    const tabMap = { plans: this.plansTab, replays: this.replaysTab, addons: this.addonsTab }
    await tabMap[tab].click()
  }
}
