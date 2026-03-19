import type { Page, Locator } from '@playwright/test'

export class PixelsPage {
  readonly page: Page
  readonly heading: Locator
  readonly addPixelButton: Locator
  readonly pixelTable: Locator
  readonly emptyState: Locator

  // Dialog elements
  readonly dialogTitle: Locator
  readonly nameInput: Locator
  readonly pixelIdInput: Locator
  readonly accessTokenInput: Locator
  readonly saveButton: Locator
  readonly cancelButton: Locator

  // Delete dialog
  readonly deleteConfirmButton: Locator

  // Backup pixel select in edit dialog
  readonly backupPixelSelect: Locator

  // Toast
  readonly toast: Locator

  // Quota elements
  readonly quotaLimitMessage: Locator
  readonly upgradeLink: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'พิกเซล' })
    this.addPixelButton = page.getByRole('button', { name: 'เพิ่มพิกเซล' }).first()
    this.pixelTable = page.locator('table')
    this.emptyState = page.getByText('ยังไม่มีพิกเซล')

    this.dialogTitle = page.getByRole('heading', { name: /เพิ่มพิกเซล|แก้ไขพิกเซล/ })
    this.nameInput = page.getByLabel('ชื่อ')
    this.pixelIdInput = page.getByLabel('Facebook Pixel ID')
    this.accessTokenInput = page.getByLabel(/Access Token/)
    this.saveButton = page.locator('form').getByRole('button', { name: /เพิ่มพิกเซล|บันทึกการเปลี่ยนแปลง/ })
    this.cancelButton = page.getByRole('button', { name: 'ยกเลิก' })

    this.deleteConfirmButton = page.locator('button.bg-destructive', { hasText: 'ลบ' })

    this.backupPixelSelect = page.locator('form select')

    this.toast = page.locator('[data-sonner-toast]')

    this.quotaLimitMessage = page.getByText('ถึงขีดจำกัด Pixel Slots แล้ว')
    this.upgradeLink = page.getByRole('link', { name: 'อัปเกรด' })
  }

  async goto() {
    await this.page.goto('/pixels')
  }

  async createPixel(name: string, pixelId: string, accessToken: string) {
    await this.addPixelButton.click()
    await this.dialogTitle.waitFor({ state: 'visible' })
    await this.nameInput.fill(name)
    await this.pixelIdInput.fill(pixelId)
    await this.accessTokenInput.fill(accessToken)
    await this.saveButton.click()
    // Wait for dialog to close (pixel created successfully)
    await this.dialogTitle.waitFor({ state: 'hidden', timeout: 10000 })
  }

  /** Click the test connection (Zap) button on a pixel row */
  async testConnection(name: string) {
    const row = this.page.locator('tr', { hasText: name })
    await row.getByRole('button', { name: 'ทดสอบการเชื่อมต่อ' }).click()
  }

  /** Click the status badge on a pixel row to toggle active/inactive */
  async toggleActive(name: string) {
    const row = this.page.locator('tr', { hasText: name })
    // The badge is wrapped in a <button> element
    await row.locator('button').filter({ has: this.page.locator('.inline-flex') }).first().click()
  }

  /** Get the status text from the badge in a pixel row */
  async getStatus(name: string): Promise<string> {
    const row = this.page.locator('tr', { hasText: name })
    // The status badge is in the third column; it contains either "ใช้งาน" or "หยุดชั่วคราว"
    const badge = row.locator('td').nth(2).locator('.inline-flex')
    return await badge.textContent() ?? ''
  }

  /** Get the backup pixel name text from the backup column of a pixel row */
  async getBackup(name: string): Promise<string> {
    const row = this.page.locator('tr', { hasText: name })
    // Backup column is the 4th column (index 3)
    const backupCell = row.locator('td').nth(3)
    return (await backupCell.textContent() ?? '').trim()
  }

  /** Open the edit dialog for a pixel by name */
  async openEdit(name: string) {
    const row = this.page.locator('tr', { hasText: name })
    await row.getByRole('button').filter({ has: this.page.locator('[class*="lucide-pencil"]') }).click()
    await this.page.getByRole('heading', { name: 'แก้ไขพิกเซล' }).waitFor({ state: 'visible' })
  }

  /** Delete a pixel by name using the delete button and confirmation dialog */
  async deletePixel(name: string) {
    const row = this.page.locator('tr', { hasText: name })
    await row.getByRole('button').filter({ has: this.page.locator('[class*="lucide-trash"]') }).click()
    await this.deleteConfirmButton.click()
    await row.waitFor({ state: 'hidden', timeout: 10000 })
  }
}
