import type { Page, Locator } from '@playwright/test'

export class BillingPage {
  readonly page: Page
  readonly heading: Locator

  // Account status card
  readonly accountStatusCard: Locator
  readonly eventsQuota: Locator
  readonly replaysQuota: Locator
  readonly pixelsQuota: Locator
  readonly retentionInfo: Locator

  // Pixel Slots section
  readonly pixelSlotsHeading: Locator
  readonly quantityDisplay: Locator
  readonly slotPriceDisplay: Locator
  readonly subscribeButton: Locator

  // Replay section
  readonly replayHeading: Locator
  readonly replaySingleCard: Locator
  readonly replayMonthlyCard: Locator

  // Manage billing button
  readonly manageBillingButton: Locator

  // Purchase history
  readonly purchaseHistorySection: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'การเงิน' })

    // Account status
    this.accountStatusCard = page.locator('[class*="card"]').filter({ hasText: 'อีเวนต์เดือนนี้' })
    this.eventsQuota = page.getByText('อีเวนต์เดือนนี้')
    this.replaysQuota = page.getByText('รีเพลย์คงเหลือ')
    this.pixelsQuota = page.getByText('พิกเซล')
    this.retentionInfo = page.getByText(/เก็บข้อมูล \d+ วัน/)

    // Pixel Slots
    this.pixelSlotsHeading = page.getByRole('heading', { name: 'Pixel Slots' })
    this.quantityDisplay = page.getByText('pixel slots')
    this.slotPriceDisplay = page.getByText('฿199/pixel/เดือน')
    this.subscribeButton = page.getByRole('button', { name: 'สมัครสมาชิก' })

    // Replay
    this.replayHeading = page.getByRole('heading', { name: 'รีเพลย์' })
    this.replaySingleCard = page.locator('[class*="card"]').filter({ hasText: 'ครั้งเดียว' })
    this.replayMonthlyCard = page.locator('[class*="card"]').filter({ hasText: 'ไม่จำกัด' })

    // Other
    this.manageBillingButton = page.getByRole('button', { name: /จัดการ/ })
    this.purchaseHistorySection = page.getByText(/ประวัติการซื้อ/)
  }

  async goto() {
    await this.page.goto('/billing')
  }
}
