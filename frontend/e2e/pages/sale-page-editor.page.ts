import type { Page, Locator } from '@playwright/test'

export class SalePageEditorPage {
  readonly page: Page
  readonly pageNameInput: Locator
  readonly saveDraftButton: Locator
  readonly publishButton: Locator
  readonly addTextBlockButton: Locator
  readonly backLink: Locator

  // Published success dialog
  readonly successDialogTitle: Locator
  readonly goBackButton: Locator

  constructor(page: Page) {
    this.page = page
    this.pageNameInput = page.getByLabel('ชื่อหน้าเพจ')
    this.saveDraftButton = page.getByRole('button', { name: 'บันทึกแบบร่าง' })
    this.publishButton = page.getByRole('button', { name: 'เผยแพร่' })
    this.addTextBlockButton = page.getByRole('button', { name: 'ข้อความ' })
    this.backLink = page.getByRole('link', { name: 'Sale Pages' })

    this.successDialogTitle = page.getByRole('heading', { name: /เผยแพร่สำเร็จ/ })
    this.goBackButton = page.getByRole('button', { name: 'กลับไปหน้ารายการ' })
  }

  async goto() {
    await this.page.goto('/sale-pages/new')
  }

  /** Fill page name and add a text block (minimum required to save) */
  async fillMinimum(name: string) {
    // The settings collapsible is open by default for new pages
    await this.pageNameInput.fill(name)
    // Add at least one block
    await this.addTextBlockButton.click()
  }

  async saveDraft() {
    await this.saveDraftButton.click()
  }

  async publish() {
    await this.publishButton.click()
  }
}
