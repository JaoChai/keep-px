import type { Page, Locator } from '@playwright/test'

export class ReplayPage {
  readonly page: Page
  readonly heading: Locator
  readonly sourcePixelSelect: Locator
  readonly targetPixelSelect: Locator
  readonly dateFromInput: Locator
  readonly dateToInput: Locator
  readonly previewButton: Locator
  readonly replayHistory: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'Replay Center' })
    // Labels lack htmlFor — use CSS sibling combinator with Playwright text matching
    this.sourcePixelSelect = page.locator('label:has-text("Source Pixel") + select')
    this.targetPixelSelect = page.locator('label:has-text("Target Pixel") + select')
    this.dateFromInput = page.locator('label:has-text("Date From") + input')
    this.dateToInput = page.locator('label:has-text("Date To") + input')
    this.previewButton = page.getByRole('button', { name: 'Preview' })
    this.replayHistory = page.getByText('Replay History')
  }

  async goto() {
    await this.page.goto('/replay')
  }
}
