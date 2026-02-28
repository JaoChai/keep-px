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
    this.heading = page.getByRole('heading', { name: 'Settings' })
    this.profileSection = page.getByText('Profile')
    this.apiKeySection = page.getByText('API Key', { exact: true })
    // Labels lack htmlFor — use CSS sibling combinator with Playwright text matching
    this.nameInput = page.locator('label:has-text("Name") + input').first()
    this.emailInput = page.locator('label:has-text("Email") + input')
  }

  async goto() {
    await this.page.goto('/settings')
  }
}
