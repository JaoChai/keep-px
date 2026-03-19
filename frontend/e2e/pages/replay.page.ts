import type { Page, Locator } from '@playwright/test'

export class ReplayPage {
  readonly page: Page
  readonly heading: Locator
  readonly sourcePixelSelect: Locator
  readonly targetPixelSelect: Locator
  readonly dateFromInput: Locator
  readonly dateToInput: Locator
  readonly previewButton: Locator
  readonly replayHistory: Locator
  readonly paywallMessage: Locator
  readonly viewReplayPacksButton: Locator

  // Event type checkboxes
  readonly eventTypeContainer: Locator
  readonly selectAllButton: Locator

  // Time mode & batch delay
  readonly timeModeSelect: Locator
  readonly batchDelayInput: Locator

  // Preview summary
  readonly previewSummary: Locator
  readonly previewEventCount: Locator
  readonly previewSampleTable: Locator
  readonly confirmReplayButton: Locator
  readonly previewBackButton: Locator

  // Progress (active replay)
  readonly progressBar: Locator
  readonly totalEventsDisplay: Locator
  readonly replayedEventsDisplay: Locator
  readonly failedEventsDisplay: Locator
  readonly progressPercentage: Locator

  // Cancel / Retry
  readonly cancelButton: Locator
  readonly cancelConfirmDialog: Locator
  readonly cancelConfirmButton: Locator
  readonly retryButton: Locator

  // Replay history
  readonly replayHistorySection: Locator
  readonly replayHistoryTable: Locator
  readonly emptyHistoryMessage: Locator

  // Replay credits badge
  readonly creditsBadge: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'ศูนย์รีเพลย์' })

    // Labels lack htmlFor — use CSS sibling combinator with Playwright text matching
    this.sourcePixelSelect = page.locator('label:has-text("พิกเซลต้นทาง") + select')
    this.targetPixelSelect = page.locator('label:has-text("พิกเซลปลายทาง") + select')
    this.dateFromInput = page.locator('label:has-text("วันที่เริ่มต้น") + input')
    this.dateToInput = page.locator('label:has-text("วันที่สิ้นสุด") + input')
    this.previewButton = page.getByRole('button', { name: 'ตัวอย่าง' })
    this.replayHistory = page.getByText('ประวัติรีเพลย์')
    this.paywallMessage = page.getByText('ไม่มีเครดิตรีเพลย์')
    this.viewReplayPacksButton = page.getByRole('button', { name: 'ดูแพ็กรีเพลย์' })

    // Event type checkboxes
    this.eventTypeContainer = page.locator('label:has-text("ประเภทอีเวนต์") + *').first()
    this.selectAllButton = page.getByText('เลือกทั้งหมด')

    // Time mode & batch delay
    this.timeModeSelect = page.locator('label:has-text("โหมดเวลา") + select')
    this.batchDelayInput = page.locator('label:has-text("ดีเลย์ต่อชุด") + input')

    // Preview summary (shown after clicking preview)
    this.previewSummary = page.getByText('สรุปตัวอย่าง')
    this.previewEventCount = page.getByText(/จะรีเพลย์ \d+ อีเวนต์/)
    this.previewSampleTable = page.locator('table').filter({ has: page.getByText('ตัวอย่างอีเวนต์').or(page.locator('th:has-text("อีเวนต์")')) })
    this.confirmReplayButton = page.getByRole('button', { name: 'ยืนยันรีเพลย์' })
    this.previewBackButton = page.getByRole('button', { name: 'ย้อนกลับ' })

    // Progress (active replay)
    this.progressBar = page.locator('.bg-emerald-500').first()
    this.totalEventsDisplay = page.locator('text=ทั้งหมด').locator('..')
    this.replayedEventsDisplay = page.locator('text=รีเพลย์แล้ว').locator('..')
    this.failedEventsDisplay = page.locator('text=ล้มเหลว').locator('..')
    this.progressPercentage = page.getByText(/\d+% สำเร็จ/)

    // Cancel / Retry
    this.cancelButton = page.getByRole('button', { name: 'ยกเลิก' })
    this.cancelConfirmDialog = page.getByText('ยกเลิกรีเพลย์?')
    this.cancelConfirmButton = page.getByRole('button', { name: 'ใช่ ยกเลิกรีเพลย์' })
    this.retryButton = page.getByRole('button', { name: 'ลองใหม่ที่ล้มเหลว' })

    // Replay history
    this.replayHistorySection = page.getByRole('heading', { name: 'ประวัติรีเพลย์' }).locator('..').locator('..')
    this.replayHistoryTable = page.locator('.cursor-pointer.hover\\:bg-accent')
    this.emptyHistoryMessage = page.getByText('ยังไม่มีรีเพลย์')

    // Credits badge
    this.creditsBadge = page.getByText(/เหลือ \d+ รีเพลย์/).or(page.getByText('รีเพลย์ไม่จำกัด'))
  }

  async goto() {
    await this.page.goto('/replay')
  }

  getEventTypeCheckbox(type: string): Locator {
    return this.page.locator(`label:has-text("${type}") input[type="checkbox"]`)
  }
}
