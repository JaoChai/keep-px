import type { Page, Locator } from '@playwright/test'

export class DashboardPage {
  readonly page: Page
  readonly heading: Locator
  readonly statCards: Locator
  readonly chartSection: Locator

  // Chart time range buttons
  readonly chartRange7d: Locator
  readonly chartRange14d: Locator
  readonly chartRange30d: Locator
  readonly chartRange90d: Locator

  // Recent Activity feed
  readonly recentActivityCard: Locator
  readonly recentActivityHeading: Locator
  readonly recentActivityViewAllLink: Locator

  // Pixel Status list
  readonly pixelStatusCard: Locator
  readonly pixelStatusHeading: Locator
  readonly pixelStatusManageLink: Locator

  // Top Event Types
  readonly topEventTypesCard: Locator
  readonly topEventTypesHeading: Locator

  // Recent Replays
  readonly recentReplaysCard: Locator
  readonly recentReplaysHeading: Locator

  // Monthly Event Usage
  readonly monthlyEventUsageSection: Locator

  // Onboarding Wizard
  readonly onboardingWizard: Locator
  readonly onboardingDismissButton: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: 'แดชบอร์ด' })
    this.statCards = page.locator('[class*="card"]').filter({ has: page.locator('p.text-2xl') })
    this.chartSection = page.getByText('ปริมาณอีเวนต์')

    // Chart time range buttons — inside the chart card's button group
    const chartCard = page.locator('[class*="card"]').filter({ hasText: 'ปริมาณอีเวนต์' })
    this.chartRange7d = chartCard.getByRole('button', { name: '7d' })
    this.chartRange14d = chartCard.getByRole('button', { name: '14d' })
    this.chartRange30d = chartCard.getByRole('button', { name: '30d' })
    this.chartRange90d = chartCard.getByRole('button', { name: '90d' })

    // Recent Activity feed
    this.recentActivityCard = page.locator('[class*="card"]').filter({ hasText: 'กิจกรรมล่าสุด' })
    this.recentActivityHeading = this.recentActivityCard.getByText('กิจกรรมล่าสุด')
    this.recentActivityViewAllLink = this.recentActivityCard.getByRole('link', { name: 'ดูทั้งหมด' })

    // Pixel Status list
    this.pixelStatusCard = page.locator('[class*="card"]').filter({ hasText: 'สถานะพิกเซล' })
    this.pixelStatusHeading = this.pixelStatusCard.getByText('สถานะพิกเซล')
    this.pixelStatusManageLink = this.pixelStatusCard.getByRole('link', { name: 'จัดการ' })

    // Top Event Types
    this.topEventTypesCard = page.locator('[class*="card"]').filter({ hasText: 'ประเภทอีเวนต์ยอดนิยม' })
    this.topEventTypesHeading = this.topEventTypesCard.getByText('ประเภทอีเวนต์ยอดนิยม')

    // Recent Replays
    this.recentReplaysCard = page.locator('[class*="card"]').filter({ hasText: 'รีเพลย์ล่าสุด' })
    this.recentReplaysHeading = this.recentReplaysCard.getByText('รีเพลย์ล่าสุด')

    // Monthly Event Usage
    this.monthlyEventUsageSection = page.getByText('ปริมาณอีเวนต์รายเดือน')

    // Onboarding Wizard
    this.onboardingWizard = page.locator('[class*="card"]').filter({ hasText: 'ยินดีต้อนรับสู่ Keep-PX!' })
    this.onboardingDismissButton = this.onboardingWizard.getByRole('button', { name: 'ซ่อน' })
  }

  async goto() {
    await this.page.goto('/dashboard')
  }
}
