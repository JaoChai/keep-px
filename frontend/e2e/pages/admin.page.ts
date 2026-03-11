import type { Page, Locator } from '@playwright/test'

export class AdminCustomersPage {
  readonly page: Page
  readonly heading: Locator
  readonly searchInput: Locator
  readonly planFilter: Locator
  readonly statusFilter: Locator
  readonly table: Locator
  readonly prevButton: Locator
  readonly nextButton: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'จัดการลูกค้า' })
    this.searchInput = page.getByPlaceholder('ค้นหาอีเมลหรือชื่อ')
    this.planFilter = page.locator('select').filter({ has: page.locator('option:has-text("ทุกแผน")') })
    this.statusFilter = page.locator('select').filter({ has: page.locator('option:has-text("ทุกสถานะ")') })
    this.table = page.locator('table')
    this.prevButton = page.getByRole('button', { name: 'ก่อนหน้า' })
    this.nextButton = page.getByRole('button', { name: 'ถัดไป' })
  }

  async goto() {
    await this.page.goto('/admin/customers')
  }
}

export class AdminAnalyticsPage {
  readonly page: Page
  readonly heading: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'แอดมิน' })
  }

  async goto() {
    await this.page.goto('/admin/analytics')
  }
}

export class AdminBillingPage {
  readonly page: Page
  readonly heading: Locator
  readonly purchasesTab: Locator
  readonly subscriptionsTab: Locator
  readonly creditsTab: Locator
  readonly table: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'การเงิน (แอดมิน)' })
    this.purchasesTab = page.getByRole('button', { name: 'การซื้อ', exact: true })
    this.subscriptionsTab = page.getByRole('button', { name: 'สมาชิก', exact: true })
    this.creditsTab = page.getByRole('button', { name: 'เครดิตที่ให้', exact: true })
    this.table = page.locator('table')
  }

  async goto() {
    await this.page.goto('/admin/billing')
  }
}
