import type { Page, Locator } from '@playwright/test'

export class EventLogPage {
  readonly page: Page
  readonly heading: Locator
  readonly eventTable: Locator
  readonly emptyState: Locator
  readonly previousButton: Locator
  readonly nextButton: Locator

  // Stat cards
  readonly statEventsToday: Locator
  readonly statTotalEvents: Locator
  readonly statCapiRate: Locator
  readonly statEventsPerMinute: Locator

  // Mode toggle
  readonly liveModeButton: Locator
  readonly historyModeButton: Locator

  // Live controls
  readonly pauseResumeButton: Locator
  readonly clearButton: Locator
  readonly refreshButton: Locator

  // Live status badge
  readonly liveBadge: Locator
  readonly pausedBadge: Locator

  // Filters
  readonly pixelFilter: Locator
  readonly eventTypeFilter: Locator
  readonly dateRangeButton: Locator
  readonly clearDateButton: Locator

  // Export
  readonly exportCsvButton: Locator

  // Event detail sheet
  readonly eventDetailSheet: Locator
  readonly eventDetailSheetTitle: Locator
  readonly eventDetailDescription: Locator

  // Live mode empty states
  readonly liveWaitingMessage: Locator
  readonly liveLoadingMessage: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'อีเวนต์' })
    this.eventTable = page.locator('table')
    this.emptyState = page.getByText('ยังไม่มีอีเวนต์ที่บันทึก')
    this.previousButton = page.getByRole('button', { name: 'ก่อนหน้า' })
    this.nextButton = page.getByRole('button', { name: 'ถัดไป' })

    // Stat cards
    this.statEventsToday = page.getByText('อีเวนต์วันนี้')
    this.statTotalEvents = page.getByText('อีเวนต์ทั้งหมด')
    this.statCapiRate = page.getByText('อัตรา CAPI')
    this.statEventsPerMinute = page.getByText('อีเวนต์/นาที')

    // Mode toggle buttons
    this.liveModeButton = page.getByRole('button', { name: 'สด' })
    this.historyModeButton = page.getByRole('button', { name: 'ประวัติ' })

    // Live controls
    this.pauseResumeButton = page.getByRole('button', { name: /หยุด|ดำเนินต่อ/ })
    this.clearButton = page.getByRole('button', { name: 'ล้าง' })
    this.refreshButton = page.getByRole('button', { name: 'รีเฟรช' })

    // Live status badges
    this.liveBadge = page.getByText('สด', { exact: true }).locator('xpath=ancestor::*[contains(@class,"badge") or contains(@class,"Badge")]').first()
    this.pausedBadge = page.getByText('หยุดชั่วคราว')

    // Pixel filter (Select trigger with "พิกเซลทั้งหมด" as default)
    this.pixelFilter = page.locator('button[role="combobox"]').filter({ hasText: /พิกเซล/ }).first()

    // Event type filter (Select trigger with "อีเวนต์ทั้งหมด" as default, only visible in history mode)
    this.eventTypeFilter = page.locator('button[role="combobox"]').filter({ hasText: /อีเวนต์/ }).first()

    // Date range popover button
    this.dateRangeButton = page.getByText('ช่วงวันที่')

    // Clear date button (inside popover)
    this.clearDateButton = page.getByRole('button', { name: 'ล้างช่วงวันที่' })

    // Export CSV
    this.exportCsvButton = page.getByRole('button', { name: 'Export CSV' })

    // Event detail sheet
    this.eventDetailSheet = page.locator('[role="dialog"]').filter({ hasText: 'รายละเอียดอีเวนต์' })
    this.eventDetailSheetTitle = page.getByText('รายละเอียดอีเวนต์')
    this.eventDetailDescription = page.getByText('รายละเอียดอีเวนต์')

    // Live mode empty states
    this.liveWaitingMessage = page.getByText('รอรับอีเวนต์...')
    this.liveLoadingMessage = page.getByText('กำลังโหลดอีเวนต์ล่าสุด...')
  }

  async goto() {
    await this.page.goto('/events?mode=history')
  }

  async gotoLive() {
    await this.page.goto('/events?mode=live')
    await this.heading.waitFor({ state: 'visible', timeout: 15000 })
  }

  async gotoHistory() {
    await this.page.goto('/events?mode=history')
    await this.heading.waitFor({ state: 'visible', timeout: 15000 })
  }

  /** Click the first event row in the table */
  async clickFirstEventRow() {
    const firstRow = this.eventTable.locator('tbody tr').first()
    await firstRow.click()
  }

  /** Get the date range "from" input inside the popover */
  getDateFromInput() {
    return this.page.locator('input[type="datetime-local"]').first()
  }

  /** Get the date range "to" input inside the popover */
  getDateToInput() {
    return this.page.locator('input[type="datetime-local"]').last()
  }
}
