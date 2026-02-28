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

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'Pixels' })
    this.addPixelButton = page.getByRole('button', { name: 'Add Pixel' }).first()
    this.pixelTable = page.locator('table')
    this.emptyState = page.getByText('No pixels yet')

    this.dialogTitle = page.getByRole('heading', { name: /Add Pixel|Edit Pixel/ })
    this.nameInput = page.getByLabel('Name')
    this.pixelIdInput = page.getByLabel('Facebook Pixel ID')
    this.accessTokenInput = page.getByLabel('Access Token')
    this.saveButton = page.locator('form').getByRole('button', { name: /Add Pixel|Save Changes/ })
    this.cancelButton = page.getByRole('button', { name: 'Cancel' })

    this.deleteConfirmButton = page.getByRole('button', { name: 'Delete' })
  }

  async goto() {
    await this.page.goto('/pixels')
  }

  async createPixel(name: string, pixelId: string, accessToken: string) {
    await this.addPixelButton.click()
    await this.nameInput.fill(name)
    await this.pixelIdInput.fill(pixelId)
    await this.accessTokenInput.fill(accessToken)
    await this.saveButton.click()
  }
}
