import type { Page, Locator } from '@playwright/test'

export class SalePagesPage {
  readonly page: Page
  readonly heading: Locator
  readonly createButton: Locator
  readonly grid: Locator
  readonly emptyState: Locator

  // Delete dialog
  readonly deleteDialogTitle: Locator
  readonly deleteConfirmButton: Locator
  readonly deleteCancelButton: Locator

  // Quota limit
  readonly quotaLimitMessage: Locator
  readonly upgradeButton: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'เซลเพจ' })
    this.createButton = page.getByRole('button', { name: 'สร้างเซลเพจ', exact: true })
    this.grid = page.locator('[data-testid="sale-page-grid"]')
    this.emptyState = page.getByText('ยังไม่มีเซลเพจ')

    this.deleteDialogTitle = page.getByRole('heading', { name: 'ลบเซลเพจ' })
    // The destructive button in the delete dialog (distinct from delete button in cards)
    this.deleteConfirmButton = page.locator('button.bg-destructive', { hasText: 'ลบ' })
    this.deleteCancelButton = page.getByRole('button', { name: 'ยกเลิก' })

    // Quota limit (shows in editor when sale page limit reached)
    this.quotaLimitMessage = page.getByText('ถึงขีดจำกัดเซลเพจแล้ว')
    this.upgradeButton = page.getByRole('button', { name: 'อัปเกรดแพ็คเกจ' })
  }

  async goto() {
    await this.page.goto('/sale-pages', { waitUntil: 'networkidle' })
    await this.heading.waitFor({ state: 'visible', timeout: 15000 })
  }

  /** Click delete button on a card matching the given text */
  async clickDeleteOnCard(name: string) {
    const card = this.page.locator('[data-testid="sale-page-card"]', { hasText: name })
    await card.getByRole('button', { name: 'ลบ' }).click()
  }

  /** Click edit button on a card matching the given text */
  async clickEditOnCard(name: string) {
    const card = this.page.locator('[data-testid="sale-page-card"]', { hasText: name })
    await card.getByRole('button', { name: 'แก้ไข' }).click()
  }

  /** Get card element matching the given text */
  getCard(name: string) {
    return this.page.locator('[data-testid="sale-page-card"]', { hasText: name })
  }

  // Backward-compatible aliases
  /** @deprecated Use clickDeleteOnCard */
  async clickDeleteOnRow(name: string) {
    return this.clickDeleteOnCard(name)
  }

  /** @deprecated Use clickEditOnCard */
  async clickEditOnRow(name: string) {
    return this.clickEditOnCard(name)
  }

  /** @deprecated Use getCard */
  getRow(name: string) {
    return this.getCard(name)
  }
}
