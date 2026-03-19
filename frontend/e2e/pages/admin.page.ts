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

  // Customer detail dialog
  readonly customerDetailDialog: Locator
  readonly customerDetailTitle: Locator

  // Grant credits
  readonly grantCreditsSection: Locator
  readonly creditAmountInput: Locator
  readonly grantCreditsButton: Locator

  // Change plan
  readonly changePlanSelect: Locator
  readonly savePlanButton: Locator

  // Suspend / Activate
  readonly suspendButton: Locator
  readonly activateButton: Locator
  readonly confirmSuspendButton: Locator

  // Customer rows
  readonly firstCustomerRow: Locator
  readonly noCustomersMessage: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'ลูกค้า' }).first()
    this.searchInput = page.getByPlaceholder('ค้นหาอีเมลหรือชื่อ')
    this.planFilter = page.locator('select').filter({ has: page.locator('option:has-text("ทุกแผน")') })
    this.statusFilter = page.locator('select').filter({ has: page.locator('option:has-text("ทุกสถานะ")') })
    this.table = page.locator('table')
    this.prevButton = page.getByRole('button', { name: 'ก่อนหน้า' })
    this.nextButton = page.getByRole('button', { name: 'ถัดไป' })

    // Customer detail dialog
    this.customerDetailDialog = page.getByRole('dialog')
    this.customerDetailTitle = page.getByRole('heading', { name: 'รายละเอียดลูกค้า' })

    // Grant credits (inside dialog)
    this.grantCreditsSection = page.getByText('ให้เครดิตรีเพลย์')
    this.creditAmountInput = page.locator('label:has-text("จำนวนรีเพลย์") + input')
    this.grantCreditsButton = page.getByRole('button', { name: 'เพิ่มเครดิต' })

    // Change plan (inside dialog)
    this.changePlanSelect = page.locator('label:has-text("เปลี่ยนแผน") + select')
    this.savePlanButton = page.getByRole('button', { name: 'บันทึก' })

    // Suspend / Activate (inside dialog)
    this.suspendButton = page.getByRole('button', { name: 'ระงับบัญชี' })
    this.activateButton = page.getByRole('button', { name: 'เปิดใช้งานบัญชี' })
    this.confirmSuspendButton = page.getByRole('button', { name: 'ยืนยันระงับ' })

    // Customer rows
    this.firstCustomerRow = page.locator('table tbody tr').first()
    this.noCustomersMessage = page.getByText('ไม่พบลูกค้า')
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

export class AdminSalePagesPage {
  readonly page: Page
  readonly heading: Locator
  readonly searchInput: Locator
  readonly publishedFilter: Locator
  readonly table: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'เซลเพจ' })
    this.searchInput = page.getByPlaceholder('ค้นหาชื่อ/slug...')
    this.publishedFilter = page.locator('select').filter({ has: page.locator('option:has-text("เผยแพร่")') })
    this.table = page.locator('table')
  }

  async goto() {
    await this.page.goto('/admin/sale-pages')
  }
}

export class AdminPixelsPage {
  readonly page: Page
  readonly heading: Locator
  readonly searchInput: Locator
  readonly activeFilter: Locator
  readonly table: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'พิกเซล' })
    this.searchInput = page.getByPlaceholder('ค้นหาชื่อ/Pixel ID...')
    this.activeFilter = page.locator('select').filter({ has: page.locator('option:has-text("ใช้งาน")') })
    this.table = page.locator('table')
  }

  async goto() {
    await this.page.goto('/admin/pixels')
  }
}

export class AdminReplaysPage {
  readonly page: Page
  readonly heading: Locator
  readonly statusFilter: Locator
  readonly table: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'รีเพลย์' })
    this.statusFilter = page.locator('select').filter({ has: page.locator('option:has-text("ทุกสถานะ")') })
    this.table = page.locator('table')
  }

  async goto() {
    await this.page.goto('/admin/replays')
  }
}

export class AdminEventsPage {
  readonly page: Page
  readonly heading: Locator
  readonly table: Locator
  readonly customerIdInput: Locator
  readonly pixelIdInput: Locator
  readonly eventNameInput: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'อีเวนต์' })
    this.table = page.locator('table')
    this.customerIdInput = page.getByPlaceholder('Customer ID')
    this.pixelIdInput = page.getByPlaceholder('Pixel ID')
    this.eventNameInput = page.getByPlaceholder('ชื่ออีเวนต์ (PageView, Purchase...)')
  }

  async goto() {
    await this.page.goto('/admin/events')
  }
}

export class AdminAuditLogPage {
  readonly page: Page
  readonly heading: Locator
  readonly actionFilter: Locator
  readonly table: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'บันทึกกิจกรรม' })
    this.actionFilter = page.locator('select').filter({ has: page.locator('option:has-text("ทุกการกระทำ")') })
    this.table = page.locator('table')
  }

  async goto() {
    await this.page.goto('/admin/audit-log')
  }
}
