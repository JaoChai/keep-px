import type { Page, Locator } from '@playwright/test'

export class SalePagesPage {
  readonly page: Page
  readonly heading: Locator
  readonly createButton: Locator
  readonly table: Locator
  readonly emptyState: Locator

  // Delete dialog
  readonly deleteDialogTitle: Locator
  readonly deleteConfirmButton: Locator
  readonly deleteCancelButton: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'เซลเพจ' })
    this.createButton = page.getByRole('button', { name: 'สร้างเซลเพจ', exact: true })
    this.table = page.locator('table')
    this.emptyState = page.getByText('ยังไม่มีเซลเพจ')

    this.deleteDialogTitle = page.getByRole('heading', { name: 'ลบเซลเพจ' })
    // The destructive button in the delete dialog (distinct from trash icon in table rows)
    this.deleteConfirmButton = page.locator('button.bg-destructive', { hasText: 'ลบ' })
    this.deleteCancelButton = page.getByRole('button', { name: 'ยกเลิก' })
  }

  async goto() {
    await this.page.goto('/sale-pages')
  }

  /** Click trash icon on a row matching the given text */
  async clickDeleteOnRow(name: string) {
    const row = this.page.locator('tr', { hasText: name })
    await row.getByRole('button', { name: 'ลบ' }).click()
  }

  /** Click edit icon on a row matching the given text */
  async clickEditOnRow(name: string) {
    const row = this.page.locator('tr', { hasText: name })
    await row.getByRole('button', { name: 'แก้ไข' }).click()
  }

  /** Get row element matching the given text */
  getRow(name: string) {
    return this.page.locator('tr', { hasText: name })
  }
}
