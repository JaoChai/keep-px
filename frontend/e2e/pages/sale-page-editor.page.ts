import type { Page, Locator } from '@playwright/test'

export class SalePageEditorPage {
  readonly page: Page
  readonly pageNameInput: Locator
  readonly saveDraftButton: Locator
  readonly publishButton: Locator
  readonly addTextBlockButton: Locator
  readonly backLink: Locator

  // Published success dialog
  readonly successDialogTitle: Locator
  readonly goBackButton: Locator

  // Custom slug
  readonly customSlugToggle: Locator
  readonly slugInput: Locator

  // Hero section fields (classic editor)
  readonly heroTitleInput: Locator
  readonly heroSubtitleInput: Locator
  readonly heroImageUrlInput: Locator

  // Description (classic editor)
  readonly descriptionTextarea: Locator

  // Features (classic editor)
  readonly addFeatureButton: Locator

  // CTA settings (classic editor)
  readonly ctaButtonTextInput: Locator
  readonly ctaButtonLinkInput: Locator

  // Tracking settings
  readonly ctaEventNameSelect: Locator
  readonly trackingContentNameInput: Locator
  readonly trackingContentValueInput: Locator
  readonly trackingCurrencySelect: Locator

  // Style settings
  readonly styleSection: Locator

  // Contact info (classic editor)
  readonly contactLineIdInput: Locator
  readonly contactPhoneInput: Locator
  readonly contactWebsiteUrlInput: Locator

  // Preview toggle (mobile)
  readonly previewToggleButton: Locator
  readonly previewLabel: Locator

  // Published dialog URL + copy
  readonly publishedUrlCode: Locator
  readonly copyUrlButton: Locator

  // Block editor specific
  readonly settingsCollapsible: Locator
  readonly addImageBlockButton: Locator
  readonly addLineBlockButton: Locator
  readonly addWebsiteBlockButton: Locator
  readonly addLinkBlockButton: Locator

  // Block delete dialog
  readonly deleteBlockDialogTitle: Locator
  readonly deleteBlockConfirmButton: Locator
  readonly deleteBlockCancelButton: Locator

  // Unsaved changes dialog
  readonly unsavedChangesDialog: Locator
  readonly stayButton: Locator
  readonly leaveButton: Locator

  // Tracking collapsible (block editor)
  readonly trackingCollapsible: Locator

  // Style collapsible (block editor)
  readonly styleCollapsible: Locator

  constructor(page: Page) {
    this.page = page

    // Block editor page name input (id="page-name")
    this.pageNameInput = page.getByLabel('ชื่อหน้าเพจ')
    this.saveDraftButton = page.getByRole('button', { name: 'บันทึกแบบร่าง' })
    this.publishButton = page.getByRole('button', { name: 'เผยแพร่' })
    this.addTextBlockButton = page.getByRole('button', { name: 'ข้อความ' })
    this.backLink = page.getByRole('link', { name: 'Sale Pages' })

    this.successDialogTitle = page.getByRole('heading', { name: /เผยแพร่สำเร็จ/ })
    this.goBackButton = page.getByRole('button', { name: 'กลับไปหน้ารายการ' })

    // Custom slug
    this.customSlugToggle = page.getByText('ตั้ง URL เอง (ไม่บังคับ)')
    this.slugInput = page.locator('#page-slug, #slug')

    // Hero section fields (classic editor uses htmlFor ids)
    this.heroTitleInput = page.getByLabel('หัวข้อ')
    this.heroSubtitleInput = page.getByLabel('คำบรรยาย')
    this.heroImageUrlInput = page.locator('#hero_image_url')

    // Description textarea (classic editor)
    this.descriptionTextarea = page.getByLabel('รายละเอียด')

    // Features (classic editor)
    this.addFeatureButton = page.getByRole('button', { name: 'เพิ่มจุดเด่น' })

    // CTA settings (classic editor)
    this.ctaButtonTextInput = page.getByLabel('ข้อความปุ่ม')
    this.ctaButtonLinkInput = page.getByLabel('ลิงก์ปุ่ม')

    // Tracking settings (both editors)
    this.ctaEventNameSelect = page.locator('#cta_event_name, #cta-event')
    this.trackingContentNameInput = page.locator('#tracking_content_name, #content-name')
    this.trackingContentValueInput = page.locator('#tracking_content_value, #content-value')
    this.trackingCurrencySelect = page.locator('#tracking_currency, #currency')

    // Style settings section
    this.styleSection = page.getByText('รูปแบบหน้าเพจ')

    // Contact info (classic editor)
    this.contactLineIdInput = page.getByLabel('LINE ID')
    this.contactPhoneInput = page.getByLabel('เบอร์โทรศัพท์')
    this.contactWebsiteUrlInput = page.getByLabel('URL เว็บไซต์')

    // Preview toggle (mobile)
    this.previewToggleButton = page.getByRole('button', { name: /Preview|พรีวิว/ })
    this.previewLabel = page.getByText('Preview').or(page.getByText('ตัวอย่าง'))

    // Published dialog URL + copy button
    this.publishedUrlCode = page.locator('code').filter({ hasText: '/p/' })
    this.copyUrlButton = page.locator('[role="dialog"]').getByRole('button').filter({
      has: page.locator('[class*="lucide-copy"], [class*="lucide-check"]'),
    })

    // Block editor specific add block buttons
    this.settingsCollapsible = page.getByText('ตั้งค่าหน้าเพจ')
    this.addImageBlockButton = page.getByRole('button', { name: 'รูปภาพ' })
    this.addLineBlockButton = page.getByRole('button', { name: 'LINE' })
    this.addWebsiteBlockButton = page.getByRole('button', { name: 'เว็บไซต์' })
    this.addLinkBlockButton = page.getByRole('button', { name: 'ลิงก์' })

    // Block delete confirmation dialog (custom Dialog component — no role="dialog")
    this.deleteBlockDialogTitle = page.getByRole('heading', { name: 'ลบบล็อก' })
    // The dialog is a div.rounded-lg.border.bg-card inside a fixed overlay
    const deleteBlockDialog = page.locator('div.rounded-lg').filter({ has: page.getByRole('heading', { name: 'ลบบล็อก' }) })
    this.deleteBlockConfirmButton = deleteBlockDialog.getByRole('button', { name: 'ลบ' })
    this.deleteBlockCancelButton = deleteBlockDialog.getByRole('button', { name: 'ยกเลิก' })

    // Unsaved changes dialog
    this.unsavedChangesDialog = page.getByRole('heading', { name: 'มีการเปลี่ยนแปลงที่ยังไม่ได้บันทึก' })
    this.stayButton = page.getByRole('button', { name: 'อยู่ต่อ' })
    this.leaveButton = page.getByRole('button', { name: 'ออกโดยไม่บันทึก' })

    // Collapsible sections in block editor
    this.trackingCollapsible = page.getByRole('button', { name: 'ตั้งค่าการติดตาม' })
    this.styleCollapsible = page.getByRole('button', { name: 'รูปแบบหน้าเพจ' })
  }

  async goto() {
    await this.page.goto('/sale-pages/new')
  }

  async gotoClassic() {
    await this.page.goto('/sale-pages/new-classic')
  }

  /** Fill page name and add a text block (minimum required to save) */
  async fillMinimum(name: string) {
    // The settings collapsible is open by default for new pages
    await this.pageNameInput.fill(name)
    // Add at least one block
    await this.addTextBlockButton.click()
  }

  async saveDraft() {
    await this.saveDraftButton.click()
  }

  async publish() {
    await this.publishButton.click()
  }

  /** Open the settings collapsible (in block editor, settings is a collapsible section) */
  async openSettings() {
    // Check if the settings section is already open by looking for the name input
    const isVisible = await this.pageNameInput.isVisible({ timeout: 1000 }).catch(() => false)
    if (!isVisible) {
      await this.settingsCollapsible.click()
    }
  }

  /** Get feature input by index (classic editor) */
  getFeatureInput(index: number) {
    return this.page.getByPlaceholder(`จุดเด่นที่ ${index + 1}`)
  }

  /** Get feature remove button by index (classic editor) */
  getFeatureRemoveButton(index: number) {
    const featureRow = this.page.locator('.flex.items-center.gap-2').filter({
      has: this.page.getByPlaceholder(`จุดเด่นที่ ${index + 1}`),
    })
    return featureRow.getByRole('button')
  }

  /** Get the number of blocks in the block editor */
  async getBlockCount() {
    return await this.page.locator('.border.border-border.rounded-lg.p-4.bg-card').count()
  }

  /** Click the delete button on a specific block by index */
  async clickDeleteBlock(index: number) {
    const blocks = this.page.locator('.border.border-border.rounded-lg.p-4.bg-card')
    const block = blocks.nth(index)
    // The delete button contains a red trash icon
    await block.locator('button').filter({ has: this.page.locator('[class*="lucide-trash"]') }).click()
  }

  /** Open tracking settings collapsible in block editor */
  async openTracking() {
    const isVisible = await this.ctaEventNameSelect.isVisible({ timeout: 1000 }).catch(() => false)
    if (!isVisible) {
      await this.trackingCollapsible.click()
    }
  }

  /** Open style settings collapsible in block editor */
  async openStyle() {
    await this.styleCollapsible.click()
  }
}
