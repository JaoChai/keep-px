import type { Page, Locator } from '@playwright/test'

export class SettingsPage {
  readonly page: Page
  readonly heading: Locator
  readonly profileSection: Locator
  readonly apiKeySection: Locator
  readonly nameInput: Locator
  readonly emailInput: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'ตั้งค่า' })
    this.profileSection = page.getByText('โปรไฟล์')
    this.apiKeySection = page.getByText('API Key', { exact: true })
    // Labels lack htmlFor — use CSS sibling combinator with Playwright text matching
    this.nameInput = page.locator('label:has-text("ชื่อ") + input').first()
    this.emailInput = page.locator('label:has-text("อีเมล") + input')
  }

  async goto() {
    await this.page.goto('/settings')
  }
}
