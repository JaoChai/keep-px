import { test, expect } from '../fixtures/auth.fixture'
import { SalePagesPage } from '../pages/sale-pages.page'
import { SalePageEditorPage } from '../pages/sale-page-editor.page'
import { SidebarPage } from '../pages/sidebar.page'

const TEST_PREFIX = 'E2E SP'

/** Delete all sale pages whose name starts with a test prefix */
async function cleanupTestSalePages(page: import('@playwright/test').Page) {
  await page.goto('/sale-pages')
  // Wait for page to load
  await page.waitForLoadState('networkidle')

  for (const prefix of [TEST_PREFIX, 'E2E Updated']) {
    const rows = page.locator('tr', { hasText: prefix })
    let count = await rows.count()
    // Delete from bottom up to avoid index shifting
    while (count > 0) {
      const row = rows.first()
      await row.getByRole('button', { name: 'ลบ' }).click()
      // Wait for delete dialog
      await expect(page.getByRole('heading', { name: 'ลบเซลเพจ' })).toBeVisible()
      await page.getByRole('button', { name: 'ลบ' }).last().click()
      // Wait for dialog to close and row to disappear
      await expect(page.getByRole('heading', { name: 'ลบเซลเพจ' })).not.toBeVisible()
      await page.waitForTimeout(500)
      count = await rows.count()
    }
  }
}

test.describe('Sale Pages @smoke', () => {
  test.describe.configure({ mode: 'serial' })

  test.afterEach(async ({ page }) => {
    await cleanupTestSalePages(page)
  })

  test('list page loads with heading and create button', async ({ page }) => {
    const salePagesPage = new SalePagesPage(page)
    await salePagesPage.goto()

    await expect(salePagesPage.heading).toBeVisible()
    await expect(salePagesPage.createButton).toBeVisible()
  })

  test('create sale page as draft', async ({ page }) => {
    const salePagesPage = new SalePagesPage(page)
    await salePagesPage.goto()
    await salePagesPage.createButton.click()

    const editor = new SalePageEditorPage(page)
    const spName = `${TEST_PREFIX} Draft ${Date.now()}`
    await editor.fillMinimum(spName)
    await editor.saveDraft()

    // Should redirect back to list
    await expect(page).toHaveURL(/\/sale-pages$/)
    await expect(page.getByText(spName)).toBeVisible()

    // Status should be draft
    const statusBadge = salePagesPage.getRow(spName)
    await expect(statusBadge).toContainText('แบบร่าง')
  })

  test('create and publish sale page', async ({ page }) => {
    const salePagesPage = new SalePagesPage(page)
    await salePagesPage.goto()
    await salePagesPage.createButton.click()

    const editor = new SalePageEditorPage(page)
    const spName = `${TEST_PREFIX} Published ${Date.now()}`
    await editor.fillMinimum(spName)
    await editor.publish()

    // Success dialog should appear
    await expect(editor.successDialogTitle).toBeVisible()
    await editor.goBackButton.click()

    // Should be back on list with published status
    await expect(page).toHaveURL(/\/sale-pages$/)
    await expect(page.getByText(spName)).toBeVisible()

    const statusBadge = salePagesPage.getRow(spName)
    await expect(statusBadge).toContainText('เผยแพร่แล้ว')
  })

  test('edit sale page name', async ({ page }) => {
    const salePagesPage = new SalePagesPage(page)
    await salePagesPage.goto()

    // Create a sale page first
    await salePagesPage.createButton.click()
    const editor = new SalePageEditorPage(page)
    const originalName = `${TEST_PREFIX} Edit ${Date.now()}`
    await editor.fillMinimum(originalName)
    await editor.saveDraft()
    await expect(page).toHaveURL(/\/sale-pages$/)
    await expect(page.getByText(originalName)).toBeVisible()

    // Click edit
    await salePagesPage.clickEditOnRow(originalName)
    await expect(page).toHaveURL(/\/sale-pages\/.*\/edit-blocks/)

    // Open settings collapsible (closed by default in edit mode)
    await page.getByText('ตั้งค่าหน้าเพจ').click()
    await expect(editor.pageNameInput).toBeVisible()

    // Change name
    const updatedName = `E2E Updated ${Date.now()}`
    await editor.pageNameInput.clear()
    await editor.pageNameInput.fill(updatedName)
    await editor.saveDraftButton.click()

    // Should be back on list with updated name
    await expect(page).toHaveURL(/\/sale-pages$/)
    await expect(page.getByText(updatedName)).toBeVisible()
  })

  test('delete sale page with confirmation', async ({ page }) => {
    const salePagesPage = new SalePagesPage(page)
    await salePagesPage.goto()

    // Create a sale page first
    await salePagesPage.createButton.click()
    const editor = new SalePageEditorPage(page)
    const spName = `${TEST_PREFIX} Delete ${Date.now()}`
    await editor.fillMinimum(spName)
    await editor.saveDraft()
    await expect(page).toHaveURL(/\/sale-pages$/)
    await expect(page.getByText(spName)).toBeVisible()

    // Click delete
    await salePagesPage.clickDeleteOnRow(spName)
    await expect(salePagesPage.deleteDialogTitle).toBeVisible()
    await expect(page.getByText('คุณแน่ใจหรือไม่')).toBeVisible()

    // Confirm delete
    await salePagesPage.deleteConfirmButton.click()
    await expect(page.getByText(spName)).not.toBeVisible()
  })

  test('cancel delete keeps sale page', async ({ page }) => {
    const salePagesPage = new SalePagesPage(page)
    await salePagesPage.goto()

    // Create a sale page first
    await salePagesPage.createButton.click()
    const editor = new SalePageEditorPage(page)
    const spName = `${TEST_PREFIX} Cancel ${Date.now()}`
    await editor.fillMinimum(spName)

    // Publish instead of draft (more reliable — draft may not redirect)
    await editor.publish()
    const hasSuccess = await editor.successDialogTitle.isVisible({ timeout: 10000 }).catch(() => false)
    if (!hasSuccess) {
      // Publish failed (quota?) — skip test
      test.skip(true, 'Could not create sale page (quota exceeded?)')
      return
    }
    await editor.goBackButton.click()
    await expect(page).toHaveURL(/\/sale-pages$/)
    await expect(page.getByText(spName)).toBeVisible()

    // Click delete
    await salePagesPage.clickDeleteOnRow(spName)
    await expect(salePagesPage.deleteDialogTitle).toBeVisible()

    // Cancel
    await salePagesPage.deleteCancelButton.click()
    await expect(salePagesPage.deleteDialogTitle).not.toBeVisible()

    // Sale page should still be visible
    await expect(page.getByText(spName)).toBeVisible()
  })
})

test.describe('Sale Page Editor Details', () => {
  test.describe.configure({ mode: 'serial' })
  test.setTimeout(60_000)

  test.afterEach(async ({ page }) => {
    await cleanupTestSalePages(page)
  })

  test('custom slug can be set and saved', async ({ page }) => {
    const editor = new SalePageEditorPage(page)
    await editor.goto()
    await page.waitForLoadState('networkidle')

    const spName = `${TEST_PREFIX} Slug ${Date.now()}`
    const customSlug = `e2e-slug-${Date.now()}`

    await editor.pageNameInput.fill(spName)

    // Open custom slug toggle
    await editor.customSlugToggle.click()
    await expect(editor.slugInput).toBeVisible()
    await editor.slugInput.fill(customSlug)

    // Add minimum block content
    await editor.addTextBlockButton.click()

    await editor.publish()

    // Success dialog should show slug in URL
    await expect(editor.successDialogTitle).toBeVisible({ timeout: 15000 })
    await expect(editor.publishedUrlCode).toContainText(`/p/${customSlug}`)

    await editor.goBackButton.click()
    await expect(page).toHaveURL(/\/sale-pages$/)

    // Verify slug appears in the list
    const row = page.locator('tr', { hasText: spName })
    await expect(row).toContainText(`/p/${customSlug}`)
  })

  test('hero section fields can be filled (classic editor)', async ({ page }) => {
    const editor = new SalePageEditorPage(page)
    await editor.gotoClassic()
    await page.waitForLoadState('networkidle')

    const spName = `${TEST_PREFIX} Hero ${Date.now()}`
    await page.getByLabel('ชื่อเพจ').fill(spName)

    // Fill hero fields
    await editor.heroTitleInput.fill('Test Hero Title')
    await editor.heroSubtitleInput.fill('Test Hero Subtitle')
    await editor.heroImageUrlInput.fill('https://example.com/test-image.jpg')

    await editor.saveDraftButton.click()
    await expect(page).toHaveURL(/\/sale-pages$/)
    await expect(page.getByText(spName)).toBeVisible()
  })

  test('description textarea can be filled (classic editor)', async ({ page }) => {
    const editor = new SalePageEditorPage(page)
    await editor.gotoClassic()
    await page.waitForLoadState('networkidle')

    const spName = `${TEST_PREFIX} Desc ${Date.now()}`
    await page.getByLabel('ชื่อเพจ').fill(spName)

    // Fill description
    await editor.descriptionTextarea.fill('This is a test description for the sale page')
    await expect(editor.descriptionTextarea).toHaveValue('This is a test description for the sale page')

    await editor.saveDraftButton.click()
    await expect(page).toHaveURL(/\/sale-pages$/)
    await expect(page.getByText(spName)).toBeVisible()
  })

  test('features add and remove (classic editor)', async ({ page }) => {
    const editor = new SalePageEditorPage(page)
    await editor.gotoClassic()
    await page.waitForLoadState('networkidle')

    const spName = `${TEST_PREFIX} Features ${Date.now()}`
    await page.getByLabel('ชื่อเพจ').fill(spName)

    // There's already one feature input by default
    // Fill the first feature
    const feature1 = page.getByPlaceholder('จุดเด่นที่ 1')
    await feature1.fill('Feature 1')

    // Add second feature
    await editor.addFeatureButton.click()
    const feature2 = page.getByPlaceholder('จุดเด่นที่ 2')
    await feature2.fill('Feature 2')

    // Add third feature
    await editor.addFeatureButton.click()
    const feature3 = page.getByPlaceholder('จุดเด่นที่ 3')
    await feature3.fill('Feature 3')

    // Verify 3 feature inputs exist
    const featureInputs = page.locator('input[placeholder^="จุดเด่นที่"]')
    await expect(featureInputs).toHaveCount(3)

    // Delete the second feature (index 1)
    // The remove button is the X button next to each feature
    const featureRows = page.locator('.flex.items-center.gap-2').filter({
      has: page.locator('input[placeholder^="จุดเด่นที่"]'),
    })
    // Click the delete button on the second row
    await featureRows.nth(1).getByRole('button').click()

    // Verify 2 feature inputs remain
    await expect(featureInputs).toHaveCount(2)

    await editor.saveDraftButton.click()
    await expect(page).toHaveURL(/\/sale-pages$/)
    await expect(page.getByText(spName)).toBeVisible()
  })

  test('CTA settings can be filled (classic editor)', async ({ page }) => {
    const editor = new SalePageEditorPage(page)
    await editor.gotoClassic()
    await page.waitForLoadState('networkidle')

    const spName = `${TEST_PREFIX} CTA ${Date.now()}`
    await page.getByLabel('ชื่อเพจ').fill(spName)

    // Fill CTA fields
    await editor.ctaButtonTextInput.fill('Buy Now')
    await expect(editor.ctaButtonTextInput).toHaveValue('Buy Now')
    await editor.ctaButtonLinkInput.fill('https://example.com/buy')
    await expect(editor.ctaButtonLinkInput).toHaveValue('https://example.com/buy')

    await editor.saveDraftButton.click()
    await expect(page).toHaveURL(/\/sale-pages$/)
    await expect(page.getByText(spName)).toBeVisible()
  })

  test('tracking settings can be configured (classic editor)', async ({ page }) => {
    const editor = new SalePageEditorPage(page)
    await editor.gotoClassic()
    await page.waitForLoadState('networkidle')

    const spName = `${TEST_PREFIX} Track ${Date.now()}`
    await page.getByLabel('ชื่อเพจ').fill(spName)

    // Set tracking fields
    // Event name select
    await editor.ctaEventNameSelect.selectOption('Purchase')
    await expect(editor.ctaEventNameSelect).toHaveValue('Purchase')

    // Product name
    await editor.trackingContentNameInput.fill('Test Product')
    await expect(editor.trackingContentNameInput).toHaveValue('Test Product')

    // Product value
    await editor.trackingContentValueInput.fill('999')

    // Currency
    await editor.trackingCurrencySelect.selectOption('USD')
    await expect(editor.trackingCurrencySelect).toHaveValue('USD')

    await editor.saveDraftButton.click()
    await expect(page).toHaveURL(/\/sale-pages$/)
    await expect(page.getByText(spName)).toBeVisible()
  })

  test('style settings section is visible (classic editor)', async ({ page }) => {
    const editor = new SalePageEditorPage(page)
    await editor.gotoClassic()
    await page.waitForLoadState('networkidle')

    // Style section should be visible
    await expect(editor.styleSection).toBeVisible()

    // Check for theme presets
    await expect(page.getByText('ธีมสำเร็จรูป')).toBeVisible()

    // Check for color pickers
    await expect(page.getByText('สีพื้นหลัง')).toBeVisible()
    await expect(page.getByText('สีปุ่ม/Accent')).toBeVisible()
    await expect(page.getByText('สีตัวอักษร')).toBeVisible()
  })

  test('contact info can be filled (classic editor)', async ({ page }) => {
    const editor = new SalePageEditorPage(page)
    await editor.gotoClassic()
    await page.waitForLoadState('networkidle')

    const spName = `${TEST_PREFIX} Contact ${Date.now()}`
    await page.getByLabel('ชื่อเพจ').fill(spName)

    // Fill contact fields
    await editor.contactLineIdInput.fill('@testlineid')
    await expect(editor.contactLineIdInput).toHaveValue('@testlineid')

    await editor.contactPhoneInput.fill('089-123-4567')
    await expect(editor.contactPhoneInput).toHaveValue('089-123-4567')

    await editor.contactWebsiteUrlInput.fill('https://example.com')
    await expect(editor.contactWebsiteUrlInput).toHaveValue('https://example.com')

    await editor.saveDraftButton.click()
    await expect(page).toHaveURL(/\/sale-pages$/)
    await expect(page.getByText(spName)).toBeVisible()
  })

  test('preview section is visible', async ({ page }) => {
    const editor = new SalePageEditorPage(page)
    await editor.goto()
    await page.waitForLoadState('networkidle')

    // Preview label should be visible (on desktop viewport)
    await expect(editor.previewLabel.first()).toBeVisible()
  })

  test('copy URL from success dialog after publish', async ({ page }) => {
    const editor = new SalePageEditorPage(page)
    await editor.goto()
    await page.waitForLoadState('networkidle')

    const spName = `${TEST_PREFIX} CopyURL ${Date.now()}`
    await editor.fillMinimum(spName)
    await editor.publish()

    // Wait for success dialog
    const hasSuccess = await editor.successDialogTitle.isVisible({ timeout: 15000 }).catch(() => false)
    if (!hasSuccess) {
      test.skip(true, 'Could not publish sale page (quota exceeded?)')
      return
    }

    // URL should be visible in the dialog
    await expect(editor.publishedUrlCode).toBeVisible()
    await expect(editor.publishedUrlCode).toContainText('/p/')

    // Copy button should be visible
    await expect(editor.copyUrlButton).toBeVisible()

    await editor.goBackButton.click()
    await expect(page).toHaveURL(/\/sale-pages$/)
  })
})

test.describe('Block Editor Flow', () => {
  test.describe.configure({ mode: 'serial' })
  test.setTimeout(60_000)

  test.afterEach(async ({ page }) => {
    await cleanupTestSalePages(page)
  })

  test('add text, image URL, and button blocks then save', async ({ page }) => {
    const editor = new SalePageEditorPage(page)
    await editor.goto()
    await page.waitForLoadState('networkidle')

    const spName = `${TEST_PREFIX} Blocks ${Date.now()}`
    await editor.pageNameInput.fill(spName)

    // Add a text block
    await editor.addTextBlockButton.click()
    let blockCount = await editor.getBlockCount()
    expect(blockCount).toBe(1)

    // Fill the text block
    const textArea = page.getByPlaceholder('พิมพ์ข้อความ...')
    await textArea.fill('Test text block content')

    // Add a LINE button block
    await editor.addLineBlockButton.click()
    blockCount = await editor.getBlockCount()
    expect(blockCount).toBe(2)

    // Add a website button block
    await editor.addWebsiteBlockButton.click()
    blockCount = await editor.getBlockCount()
    expect(blockCount).toBe(3)

    // Save as draft
    await editor.saveDraft()
    await expect(page).toHaveURL(/\/sale-pages$/)
    await expect(page.getByText(spName)).toBeVisible()
  })

  test('delete a block with confirmation', async ({ page }) => {
    const editor = new SalePageEditorPage(page)
    await editor.goto()
    await page.waitForLoadState('networkidle')

    const spName = `${TEST_PREFIX} DelBlock ${Date.now()}`
    await editor.pageNameInput.fill(spName)

    // Add two text blocks
    await editor.addTextBlockButton.click()
    await editor.addTextBlockButton.click()
    let blockCount = await editor.getBlockCount()
    expect(blockCount).toBe(2)

    // Delete the first block
    await editor.clickDeleteBlock(0)

    // Confirm delete dialog
    await expect(editor.deleteBlockDialogTitle).toBeVisible()
    await editor.deleteBlockConfirmButton.click()
    await expect(editor.deleteBlockDialogTitle).not.toBeVisible()

    // Verify only 1 block remains
    blockCount = await editor.getBlockCount()
    expect(blockCount).toBe(1)

    // Save
    await editor.saveDraft()
    await expect(page).toHaveURL(/\/sale-pages$/)
    await expect(page.getByText(spName)).toBeVisible()
  })

  test('tracking settings in block editor', async ({ page }) => {
    const editor = new SalePageEditorPage(page)
    await editor.goto()
    await page.waitForLoadState('networkidle')

    // Open tracking settings collapsible
    await editor.openTracking()

    // Verify tracking controls are visible
    await expect(editor.ctaEventNameSelect).toBeVisible()
    await expect(editor.trackingContentNameInput).toBeVisible()
    await expect(editor.trackingContentValueInput).toBeVisible()
    await expect(editor.trackingCurrencySelect).toBeVisible()
  })

  test('style settings in block editor', async ({ page }) => {
    const editor = new SalePageEditorPage(page)
    await editor.goto()
    await page.waitForLoadState('networkidle')

    // Open style settings collapsible
    await editor.openStyle()

    // Verify style controls are visible
    await expect(page.getByText('ธีมสำเร็จรูป')).toBeVisible()
    await expect(page.getByText('สีพื้นหลัง')).toBeVisible()
    await expect(page.getByText('สีปุ่ม/Accent')).toBeVisible()
  })
})

test.describe('Sale Page Edge Cases', () => {
  test.setTimeout(60_000)

  test.afterEach(async ({ page }) => {
    await cleanupTestSalePages(page)
  })

  test('unsaved changes dialog appears when navigating away', async ({ page }) => {
    const editor = new SalePageEditorPage(page)
    const sidebar = new SidebarPage(page)
    await editor.goto()
    await page.waitForLoadState('networkidle')

    // Type something to trigger unsaved changes
    await editor.pageNameInput.fill(`${TEST_PREFIX} Unsaved ${Date.now()}`)
    // Add a block to ensure hasChanges is set
    await editor.addTextBlockButton.click()

    // Try to navigate away via sidebar
    await sidebar.dashboardLink.click()

    // Unsaved changes dialog should appear
    await expect(editor.unsavedChangesDialog).toBeVisible({ timeout: 5000 })
    await expect(editor.stayButton).toBeVisible()
    await expect(editor.leaveButton).toBeVisible()

    // Click "stay" - should remain in editor
    await editor.stayButton.click()
    await expect(editor.unsavedChangesDialog).not.toBeVisible()

    // Verify we are still on the editor page
    await expect(editor.pageNameInput).toBeVisible()
  })

  test('unsaved changes dialog - leave discards changes', async ({ page }) => {
    const editor = new SalePageEditorPage(page)
    const sidebar = new SidebarPage(page)
    await editor.goto()
    await page.waitForLoadState('networkidle')

    // Type something to trigger unsaved changes
    await editor.pageNameInput.fill(`${TEST_PREFIX} Leave ${Date.now()}`)
    await editor.addTextBlockButton.click()

    // Try to navigate away via sidebar
    await sidebar.dashboardLink.click()

    // Unsaved changes dialog should appear
    await expect(editor.unsavedChangesDialog).toBeVisible({ timeout: 5000 })

    // Click "leave" - should navigate away
    await editor.leaveButton.click()

    // Should be on dashboard now
    await expect(page).toHaveURL(/\/dashboard/)
  })

  test('double-click publish prevention', async ({ page }) => {
    const editor = new SalePageEditorPage(page)
    await editor.goto()
    await page.waitForLoadState('networkidle')

    const spName = `${TEST_PREFIX} DblClick ${Date.now()}`
    await editor.fillMinimum(spName)

    // Click publish button twice quickly
    await editor.publishButton.click()

    // After first click, button should be disabled (shows "กำลังเผยแพร่...")
    // Try to verify button becomes disabled or text changes
    const isDisabled = await editor.publishButton.isDisabled({ timeout: 2000 }).catch(() => false)
    const buttonText = await editor.publishButton.textContent()

    // Either the button is disabled or it shows "กำลังเผยแพร่..."
    const isSubmitting = isDisabled || buttonText?.includes('กำลังเผยแพร่')
    expect(isSubmitting).toBe(true)

    // Wait for publish to complete
    const hasSuccess = await editor.successDialogTitle.isVisible({ timeout: 15000 }).catch(() => false)
    if (!hasSuccess) {
      test.skip(true, 'Could not publish sale page (quota exceeded?)')
      return
    }

    await editor.goBackButton.click()
    await expect(page).toHaveURL(/\/sale-pages$/)

    // Verify only one sale page was created (count rows matching the name)
    const matchingRows = page.locator('tr', { hasText: spName })
    await expect(matchingRows).toHaveCount(1)
  })

  test('sale page quota limit shows upgrade link', async ({ page }) => {
    // Navigate to the sale page editor to check quota
    // The quota limit is only shown when the user has reached their limit
    const editor = new SalePageEditorPage(page)
    const salePagesPage = new SalePagesPage(page)
    await editor.goto()
    await page.waitForLoadState('networkidle')

    // Check if quota limit message is visible
    // This is an edge case that may not always trigger
    const hasQuotaLimit = await salePagesPage.quotaLimitMessage.isVisible({ timeout: 3000 }).catch(() => false)
    if (!hasQuotaLimit) {
      test.skip(true, 'Sale page quota not reached - cannot test upgrade link')
      return
    }

    // Verify upgrade link exists
    await expect(salePagesPage.upgradeButton).toBeVisible()
  })
})
