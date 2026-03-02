import type { Page, Locator } from '@playwright/test'

export class LoginPage {
  readonly page: Page
  readonly heading: Locator
  readonly subtitle: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'Pixlinks' })
    this.subtitle = page.getByText('ปกป้องข้อมูล Facebook Pixel ของคุณ')
  }

  async goto() {
    await this.page.goto('/login')
  }
}
