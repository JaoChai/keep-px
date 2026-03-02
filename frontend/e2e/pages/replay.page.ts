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
  readonly paywallMessage: Locator
  readonly viewReplayPacksButton: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'ศูนย์รีเพลย์' })
    // Labels lack htmlFor — use CSS sibling combinator with Playwright text matching
    this.sourcePixelSelect = page.locator('label:has-text("พิกเซลต้นทาง") + select')
    this.targetPixelSelect = page.locator('label:has-text("พิกเซลปลายทาง") + select')
    this.dateFromInput = page.locator('label:has-text("วันที่เริ่มต้น") + input')
    this.dateToInput = page.locator('label:has-text("วันที่สิ้นสุด") + input')
    this.previewButton = page.getByRole('button', { name: 'ตัวอย่าง' })
    this.replayHistory = page.getByText('ประวัติรีเพลย์')
    this.paywallMessage = page.getByText('ไม่มีเครดิตรีเพลย์')
    this.viewReplayPacksButton = page.getByRole('button', { name: 'ดูแพ็กรีเพลย์' })
  }

  async goto() {
    await this.page.goto('/replay')
  }
}
