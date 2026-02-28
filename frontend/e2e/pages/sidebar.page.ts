import type { Page, Locator } from '@playwright/test'

export class SidebarPage {
  readonly page: Page
  readonly brand: Locator
  readonly dashboardLink: Locator
  readonly pixelsLink: Locator
  readonly salePagesLink: Locator
  readonly customDomainsLink: Locator
  readonly eventLogLink: Locator
  readonly realtimeLink: Locator
  readonly replayCenterLink: Locator
  readonly settingsLink: Locator
  readonly logoutButton: Locator

  constructor(page: Page) {
    this.page = page
    this.brand = page.getByText('Pixlinks')
    this.dashboardLink = page.getByRole('link', { name: 'Dashboard' })
    this.pixelsLink = page.getByRole('link', { name: 'Pixels' })
    this.salePagesLink = page.getByRole('link', { name: 'Sale Pages' })
    this.customDomainsLink = page.getByRole('link', { name: 'Custom Domains' })
    this.eventLogLink = page.getByRole('link', { name: 'Event Log' })
    this.realtimeLink = page.getByRole('link', { name: 'Realtime' })
    this.replayCenterLink = page.getByRole('link', { name: 'Replay Center' })
    this.settingsLink = page.getByRole('link', { name: 'Settings' })
    this.logoutButton = page.getByRole('button', { name: 'Logout' })
  }

  async navigateTo(linkName: string) {
    await this.page.getByRole('link', { name: linkName }).click()
  }
}
