import type { Page, Locator } from '@playwright/test'

export class GuidePage {
  readonly page: Page
  readonly heading: Locator
  readonly searchInput: Locator
  readonly sections: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'คู่มือการใช้งาน' })
    this.searchInput = page.getByPlaceholder('ค้นหาหัวข้อ')
    // Each guide section is a clickable button with icon + title
    this.sections = page.locator('button').filter({ has: page.locator('svg') })
  }

  async goto() {
    await this.page.goto('/guide')
    await this.page.waitForLoadState('networkidle')
  }
}
