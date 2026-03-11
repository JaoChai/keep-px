import type { Page, Locator } from '@playwright/test'

export class SidebarPage {
  readonly page: Page
  readonly brand: Locator
  readonly dashboardLink: Locator
  readonly pixelsLink: Locator
  readonly salePagesLink: Locator
  readonly eventsLink: Locator
  readonly replayCenterLink: Locator
  readonly billingLink: Locator
  readonly settingsLink: Locator
  readonly guideLink: Locator
  readonly logoutButton: Locator

  constructor(page: Page) {
    this.page = page
    this.brand = page.getByRole('heading', { name: 'Pixlinks' })
    this.dashboardLink = page.getByRole('link', { name: 'แดชบอร์ด' })
    this.pixelsLink = page.getByRole('link', { name: 'พิกเซล' })
    this.salePagesLink = page.getByRole('link', { name: 'เซลเพจ' })
    this.eventsLink = page.getByRole('link', { name: 'อีเวนต์' })
    this.replayCenterLink = page.getByRole('link', { name: 'รีเพลย์' })
    this.billingLink = page.getByRole('link', { name: 'การเงิน' })
    this.settingsLink = page.getByRole('link', { name: 'ตั้งค่า' })
    this.guideLink = page.getByRole('link', { name: 'คู่มือ' })
    this.logoutButton = page.getByRole('button', { name: 'ออกจากระบบ' })
  }

  async navigateTo(linkName: string) {
    await this.page.getByRole('link', { name: linkName }).click()
  }
}
