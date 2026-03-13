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

    // Account status — scoped to the card containing events quota
    this.accountStatusCard = page.locator('[class*="card"]').filter({ hasText: 'อีเวนต์เดือนนี้' }).first()
    this.eventsQuota = this.accountStatusCard.getByText('อีเวนต์เดือนนี้')
    this.replaysQuota = this.accountStatusCard.getByText('รีเพลย์คงเหลือ')
    this.pixelsQuota = this.accountStatusCard.getByText('พิกเซล')
    this.retentionInfo = this.accountStatusCard.getByText(/เก็บข้อมูล \d+ วัน/)

    // Pixel Slots
    this.pixelSlotsHeading = page.getByRole('heading', { name: 'Pixel Slots' })
    this.quantityDisplay = page.getByText('pixel slots', { exact: true })
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
