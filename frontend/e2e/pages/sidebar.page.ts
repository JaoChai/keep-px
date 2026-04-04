import type { Page, Locator } from '@playwright/test'

export class SidebarPage {
  readonly page: Page
  readonly nav: Locator
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
    // Scope all locators to the sidebar nav to avoid conflicts with page content
    this.nav = page.locator('nav')
    this.brand = page.getByRole('heading', { name: 'Pixlinks' })
    this.dashboardLink = this.nav.getByRole('link', { name: 'แดชบอร์ด' }).first()
    this.pixelsLink = this.nav.getByRole('link', { name: 'พิกเซล' }).first()
    this.salePagesLink = this.nav.getByRole('link', { name: 'เซลเพจ' }).first()
    this.eventsLink = this.nav.getByRole('link', { name: 'อีเวนต์' }).first()
    this.replayCenterLink = this.nav.getByRole('link', { name: 'รีเพลย์' }).first()
    this.billingLink = this.nav.getByRole('link', { name: 'การเงิน' }).first()
    this.settingsLink = this.nav.getByRole('link', { name: 'ตั้งค่า' }).first()
    this.guideLink = this.nav.getByRole('link', { name: 'คู่มือ' }).first()
    this.logoutButton = page.getByRole('button', { name: 'ออกจากระบบ' })
  }

  async navigateTo(linkName: string) {
    await this.nav.getByRole('link', { name: linkName }).click()
    await this.page.waitForLoadState('networkidle')
  }
}
