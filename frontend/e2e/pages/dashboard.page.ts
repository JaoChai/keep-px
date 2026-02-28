import type { Page, Locator } from '@playwright/test'

export class DashboardPage {
  readonly page: Page
  readonly heading: Locator
  readonly statCards: Locator
  readonly chartSection: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'Dashboard' })
    this.statCards = page.locator('[class*="card"]').filter({ has: page.locator('p.text-2xl') })
    this.chartSection = page.getByText('Events (Last 30 Days)')
  }

  async goto() {
    await this.page.goto('/dashboard')
  }
}
