import type { Page, Locator } from '@playwright/test'

export class EventLogPage {
  readonly page: Page
  readonly heading: Locator
  readonly eventTable: Locator
  readonly emptyState: Locator
  readonly previousButton: Locator
  readonly nextButton: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'อีเวนต์' })
    this.eventTable = page.locator('table')
    this.emptyState = page.getByText('ยังไม่มีอีเวนต์ที่บันทึก')
    this.previousButton = page.getByRole('button', { name: 'ก่อนหน้า' })
    this.nextButton = page.getByRole('button', { name: 'ถัดไป' })
  }

  async goto() {
    await this.page.goto('/events?mode=history')
  }
}
