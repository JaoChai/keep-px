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
    this.heading = page.getByRole('heading', { name: 'Events' })
    this.eventTable = page.locator('table')
    this.emptyState = page.getByText('No events recorded yet')
    this.previousButton = page.getByRole('button', { name: 'Previous' })
    this.nextButton = page.getByRole('button', { name: 'Next' })
  }

  async goto() {
    await this.page.goto('/events?mode=history')
  }
}
