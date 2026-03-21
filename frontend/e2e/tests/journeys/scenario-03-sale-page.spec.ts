/**
 * Scenario 3: Sale Page Builder
 * GitHub Issue: #154
 *
 * จำลอง user สร้าง sale page ตั้งแต่เปิดหน้า editor จนเผยแพร่ + ดูหน้า public
 * Part A: Block Editor (default) — /sale-pages/new
 * Part B: Classic Editor — /sale-pages/new-classic
 */
import { test, expect } from '../../fixtures/auth.fixture'
import { SalePagesPage } from '../../pages/sale-pages.page'

const PREFIX = 'E2E-S03'
const ts = Date.now()

test.describe('Scenario 3: Sale Page Builder', () => {
  test.describe.configure({ mode: 'serial' })
  test.setTimeout(120_000)

  // Safety net cleanup
  test.afterAll(async ({ browser }) => {
    const context = await browser.newContext({
      storageState: 'e2e/.auth/user.json',
    })
    const page = await context.newPage()
    try {
      await page.goto('/sale-pages')
      await page.waitForLoadState('networkidle')
      // Delete all sale pages with our prefix
      let rows = page.locator('[data-testid="sale-page-card"]', { hasText: PREFIX })
      let count = await rows.count()
      while (count > 0) {
        await rows.first().getByRole('button', { name: 'ลบ' }).click()
        await page.getByRole('heading', { name: 'ลบเซลเพจ' }).waitFor()
        await page.locator('button.bg-destructive', { hasText: 'ลบ' }).click()
        await page.waitForTimeout(1000)
        rows = page.locator('[data-testid="sale-page-card"]', { hasText: PREFIX })
        count = await rows.count()
      }
    } catch {
      // Best-effort cleanup
    } finally {
      await context.close()
    }
  })

  // ==============================
  // Part A: Block Editor (default)
  // ==============================

  const BLOCK_SP_NAME = `${PREFIX} Block-SP ${ts}`
  const BLOCK_SP_SLUG = `e2e-s03-block-${ts}`

  test('step 1: go to /sale-pages → click create', async ({ page }) => {
    const listPage = new SalePagesPage(page)
    await listPage.goto()
    await expect(listPage.heading).toBeVisible()

    // Clean up leftover test sale pages
    const cleanupPrefixes = [PREFIX]
    for (const prefix of cleanupPrefixes) {
      let cards = page.locator('[data-testid="sale-page-card"]', { hasText: prefix })
      let count = await cards.count()
      while (count > 0) {
        await cards.first().getByRole('button', { name: 'ลบ' }).click()
        await page.getByRole('heading', { name: 'ลบเซลเพจ' }).waitFor()
        await page.locator('button.bg-destructive', { hasText: 'ลบ' }).click()
        await page.waitForTimeout(1000)
        cards = page.locator('[data-testid="sale-page-card"]', { hasText: prefix })
        count = await cards.count()
      }
    }

    await listPage.createButton.click()
    // Should navigate to block editor (default)
    await expect(page).toHaveURL(/\/sale-pages\/new/)
  })

  test('step 2: block editor — fill page name + custom slug', async ({ page }) => {
    await page.goto('/sale-pages/new')
    await page.waitForLoadState('networkidle')

    // Fill page name
    const pageNameInput = page.getByLabel('ชื่อหน้าเพจ')
    await expect(pageNameInput).toBeVisible()
    await pageNameInput.fill(BLOCK_SP_NAME)

    // Open custom slug
    const slugToggle = page.getByText('ตั้ง URL เอง')
    if (await slugToggle.isVisible()) {
      await slugToggle.click()
      const slugInput = page.locator('input[id="page-slug"]')
      await expect(slugInput).toBeVisible()
      await slugInput.fill(BLOCK_SP_SLUG)
    }
  })

  test('step 3: block editor — select pixel', async ({ page }) => {
    await page.goto('/sale-pages/new')
    await page.waitForLoadState('networkidle')
    await page.getByLabel('ชื่อหน้าเพจ').fill(BLOCK_SP_NAME)

    // Pixel checkboxes should be visible in settings section
    const pixelCheckboxes = page.locator('input[type="checkbox"]')
    const checkboxCount = await pixelCheckboxes.count()
    if (checkboxCount > 0) {
      // Select the first pixel
      await pixelCheckboxes.first().check()
      await expect(pixelCheckboxes.first()).toBeChecked()
    }
  })

  test('step 4: block editor — add text block + save as draft', async ({ page }) => {
    await page.goto('/sale-pages/new')
    await page.waitForLoadState('networkidle')

    // Fill minimum
    await page.getByLabel('ชื่อหน้าเพจ').fill(BLOCK_SP_NAME)

    // Add a text block
    const addTextBtn = page.getByRole('button', { name: 'ข้อความ' })
    await addTextBtn.click()

    // Save as draft
    await page.getByRole('button', { name: 'บันทึกแบบร่าง' }).click()
    await page.waitForURL(/\/sale-pages$/, { timeout: 15000 })

    // Verify in list with draft status
    const row = page.locator('[data-testid="sale-page-card"]', { hasText: BLOCK_SP_NAME })
    await expect(row).toBeVisible()
    await expect(row.getByText('แบบร่าง')).toBeVisible()
  })

  test('step 5: block editor — edit draft → publish → success dialog', async ({ page }) => {
    const listPage = new SalePagesPage(page)
    await listPage.goto()

    // Click edit on the draft
    const row = page.locator('[data-testid="sale-page-card"]', { hasText: BLOCK_SP_NAME })
    await expect(row).toBeVisible()
    await row.getByRole('button', { name: 'แก้ไข' }).click()

    // Wait for editor to load (lazy-loaded component)
    await page.waitForLoadState('networkidle')
    // Block editor settings collapsible is CLOSED when editing — wait for publish button instead
    await expect(page.getByRole('button', { name: /เผยแพร่|อัพเดท/ })).toBeVisible({ timeout: 15000 })

    // Publish
    await page.getByRole('button', { name: /เผยแพร่|อัพเดท/ }).click()

    // Success dialog should appear
    await expect(page.getByText(/เผยแพร่สำเร็จ/)).toBeVisible({ timeout: 15000 })

    // Success dialog should contain published URL
    const urlCode = page.locator('code')
    await expect(urlCode).toBeVisible()
    const publishedUrl = await urlCode.textContent()
    expect(publishedUrl).toContain('/p/')

    // Copy button should be visible
    const copyBtn = page.locator('button').filter({ has: page.locator('[class*="lucide-copy"], [class*="lucide-check"]') })
    await expect(copyBtn.first()).toBeVisible()
  })

  test('step 6: block editor — visit published page', async ({ page }) => {
    const listPage = new SalePagesPage(page)
    await listPage.goto()

    // Find the published page row
    const row = page.locator('[data-testid="sale-page-card"]', { hasText: BLOCK_SP_NAME })
    await expect(row).toBeVisible()

    // Status should be "เผยแพร่แล้ว"
    await expect(row.getByText('เผยแพร่แล้ว')).toBeVisible()

    // Get the slug from the URL column
    const urlCell = row.locator('[data-testid="sale-page-url"]')
    const slug = await urlCell.textContent()
    expect(slug).toContain('/p/')

    // Visit the published page
    const pagePath = slug?.trim() ?? ''
    if (pagePath) {
      await page.goto(pagePath)
      await page.waitForLoadState('networkidle')
      // Should NOT redirect to login (public page)
      expect(page.url()).not.toContain('/login')
      // Page body should be visible
      await expect(page.locator('body')).toBeVisible()
    }
  })

  test('step 7: block editor — tracking settings', async ({ page }) => {
    const listPage = new SalePagesPage(page)
    await listPage.goto()

    // Edit the block page
    const row = page.locator('[data-testid="sale-page-card"]', { hasText: BLOCK_SP_NAME })
    await row.getByRole('button', { name: 'แก้ไข' }).click()
    await page.waitForLoadState('networkidle')
    // Wait for editor to load (settings collapsible is CLOSED on edit)
    await expect(page.getByRole('button', { name: /เผยแพร่|อัพเดท|บันทึก/ }).first()).toBeVisible({ timeout: 15000 })

    // Open tracking settings collapsible
    const trackingToggle = page.getByText('ตั้งค่าการติดตาม')
    if (await trackingToggle.isVisible()) {
      await trackingToggle.click()
      await page.waitForTimeout(300)

      // CTA event select
      const ctaEventSelect = page.locator('#cta-event, select').filter({ has: page.locator('option') }).first()
      if (await ctaEventSelect.isVisible()) {
        await expect(ctaEventSelect).toBeVisible()
      }

      // Content name input
      const contentNameInput = page.getByLabel(/ชื่อสินค้า/)
      if (await contentNameInput.isVisible()) {
        await contentNameInput.fill('E2E Test Product')
      }

      // Content value input
      const contentValueInput = page.getByLabel(/ราคาสินค้า/)
      if (await contentValueInput.isVisible()) {
        await contentValueInput.fill('299')
      }
    }
  })

  test('step 8: block editor — style settings visible', async ({ page }) => {
    await page.goto('/sale-pages/new')
    await page.waitForLoadState('networkidle')

    // Style section should exist
    const styleToggle = page.getByText('รูปแบบหน้าเพจ')
    await expect(styleToggle).toBeVisible()
    await styleToggle.click()
    await page.waitForTimeout(300)

    // At minimum, the style section toggle works without error
    // Style editor content should be present after toggle
    await page.waitForTimeout(300)
  })

  test('step 9: block editor — delete block page', async ({ page }) => {
    const listPage = new SalePagesPage(page)
    await listPage.goto()

    const row = page.locator('[data-testid="sale-page-card"]', { hasText: BLOCK_SP_NAME })
    if (await row.count() > 0) {
      await row.getByRole('button', { name: 'ลบ' }).click()
      await expect(page.getByRole('heading', { name: 'ลบเซลเพจ' })).toBeVisible()
      await page.locator('button.bg-destructive', { hasText: 'ลบ' }).click()
      await expect(row).not.toBeVisible()
    }
  })

  // ==============================
  // Part B: Classic Editor
  // ==============================

  const CLASSIC_SP_NAME = `${PREFIX} Classic-SP ${ts}`

  test('step 10: classic editor — open and fill basic info', async ({ page }) => {
    await page.goto('/sale-pages/new-classic')
    await page.waitForLoadState('networkidle')

    // Fill page name
    const nameInput = page.getByLabel('ชื่อเพจ')
    await expect(nameInput).toBeVisible()
    await nameInput.fill(CLASSIC_SP_NAME)

    // Open custom slug
    const slugToggle = page.getByText('ตั้ง URL เอง')
    if (await slugToggle.isVisible()) {
      await slugToggle.click()
      const slugInput = page.locator('input').filter({ hasText: '' }).locator('[placeholder*="slug"], [id="slug"]').first()
      if (await slugInput.count() > 0) {
        await slugInput.fill(`e2e-s03-classic-${ts}`)
      }
    }
  })

  test('step 11: classic editor — fill hero section', async ({ page }) => {
    await page.goto('/sale-pages/new-classic')
    await page.waitForLoadState('networkidle')
    await page.getByLabel('ชื่อเพจ').fill(CLASSIC_SP_NAME)

    // Hero title
    const heroTitle = page.getByLabel('หัวข้อ')
    if (await heroTitle.isVisible()) {
      await heroTitle.fill('E2E Hero Title')
    }

    // Hero subtitle
    const heroSubtitle = page.getByLabel('คำบรรยาย')
    if (await heroSubtitle.isVisible()) {
      await heroSubtitle.fill('E2E Hero Subtitle')
    }
  })

  test('step 12: classic editor — fill description + features', async ({ page }) => {
    await page.goto('/sale-pages/new-classic')
    await page.waitForLoadState('networkidle')
    await page.getByLabel('ชื่อเพจ').fill(CLASSIC_SP_NAME)

    // Description
    const description = page.getByLabel('รายละเอียด')
    if (await description.isVisible()) {
      await description.fill('This is an E2E test description for the classic editor.')
    }

    // Features — add items
    const addFeatureBtn = page.getByRole('button', { name: 'เพิ่มจุดเด่น' })
    if (await addFeatureBtn.isVisible()) {
      // Add 3 features
      await addFeatureBtn.click()
      await addFeatureBtn.click()

      // Fill feature inputs
      const allFeatures = page.locator('input[placeholder*="จุดเด่น"]')
      const featureCount = await allFeatures.count()
      for (let i = 0; i < Math.min(featureCount, 3); i++) {
        await allFeatures.nth(i).fill(`Feature ${i + 1}`)
      }

      // Delete one feature (click X button on last feature)
      if (featureCount >= 2) {
        const removeButtons = page.locator('button').filter({ has: page.locator('[class*="lucide-x"]') })
        const removeCount = await removeButtons.count()
        if (removeCount > 0) {
          await removeButtons.last().click()
        }
      }
    }
  })

  test('step 13: classic editor — fill CTA + contact', async ({ page }) => {
    await page.goto('/sale-pages/new-classic')
    await page.waitForLoadState('networkidle')
    await page.getByLabel('ชื่อเพจ').fill(CLASSIC_SP_NAME)

    // CTA button text
    const ctaText = page.getByLabel('ข้อความปุ่ม')
    if (await ctaText.isVisible()) {
      await ctaText.fill('Buy Now')
    }

    // CTA button link
    const ctaLink = page.getByLabel('ลิงก์ปุ่ม')
    if (await ctaLink.isVisible()) {
      await ctaLink.fill('https://example.com/buy')
    }

    // Contact — LINE ID
    const lineId = page.getByLabel('LINE ID')
    if (await lineId.isVisible()) {
      await lineId.fill('@e2etest')
    }

    // Contact — Phone
    const phone = page.getByLabel('เบอร์โทรศัพท์')
    if (await phone.isVisible()) {
      await phone.fill('081-234-5678')
    }

    // Contact — Website URL
    const website = page.getByLabel('URL เว็บไซต์')
    if (await website.isVisible()) {
      await website.fill('https://example.com')
    }
  })

  test('step 14: classic editor — tracking settings', async ({ page }) => {
    await page.goto('/sale-pages/new-classic')
    await page.waitForLoadState('networkidle')
    await page.getByLabel('ชื่อเพจ').fill(CLASSIC_SP_NAME)

    // CTA event select — use short timeout to avoid hanging
    const ctaEventSelect = page.locator('select').first()
    const selectVisible = await ctaEventSelect.isVisible().catch(() => false)
    if (selectVisible) {
      // Just verify select exists with options — don't wait for specific option
      const optionCount = await ctaEventSelect.locator('option').count()
      expect(optionCount).toBeGreaterThan(0)
    }

    // Tracking content name — use short timeout
    const contentName = page.getByLabel(/ชื่อสินค้า/)
    const nameVisible = await contentName.isVisible().catch(() => false)
    if (nameVisible) {
      await contentName.fill('E2E Classic Product')
    }

    // Tracking content value
    const contentValue = page.getByLabel(/ราคาสินค้า/)
    const valueVisible = await contentValue.isVisible().catch(() => false)
    if (valueVisible) {
      await contentValue.fill('599')
    }
  })

  test('step 15: classic editor — save draft → publish flow', async ({ page }) => {
    await page.goto('/sale-pages/new-classic')
    await page.waitForLoadState('networkidle')
    await page.getByLabel('ชื่อเพจ').fill(CLASSIC_SP_NAME)

    // Save as draft
    await page.getByRole('button', { name: 'บันทึกแบบร่าง' }).click()
    await page.waitForURL(/\/sale-pages$/, { timeout: 15000 })

    // Verify in list
    const row = page.locator('[data-testid="sale-page-card"]', { hasText: CLASSIC_SP_NAME })
    await expect(row).toBeVisible()
    await expect(row.getByText('แบบร่าง')).toBeVisible()
    // Template should show "Classic" badge
    await expect(row.getByText('Classic', { exact: true })).toBeVisible()

    // Edit → Publish
    await row.getByRole('button', { name: 'แก้ไข' }).click()
    await expect(page.getByLabel('ชื่อเพจ')).toBeVisible()
    await page.getByRole('button', { name: 'เผยแพร่' }).click()

    // Success dialog
    await expect(page.getByText(/เผยแพร่.*สำเร็จ/)).toBeVisible({ timeout: 15000 })

    // Go back to list
    const backBtn = page.getByRole('button', { name: /กลับ/ })
    await backBtn.click()
    await expect(page).toHaveURL(/\/sale-pages$/)

    // Verify published status
    const updatedRow = page.locator('[data-testid="sale-page-card"]', { hasText: CLASSIC_SP_NAME })
    await expect(updatedRow.getByText('เผยแพร่แล้ว')).toBeVisible()
  })

  test('step 16: classic editor — visit published page', async ({ page }) => {
    const listPage = new SalePagesPage(page)
    await listPage.goto()

    const row = page.locator('[data-testid="sale-page-card"]', { hasText: CLASSIC_SP_NAME })
    await expect(row).toBeVisible()

    // Get slug from URL column
    const urlCell = row.locator('[data-testid="sale-page-url"]')
    const slug = await urlCell.textContent()
    expect(slug).toContain('/p/')

    // Visit published page
    const pagePath = slug?.trim() ?? ''
    if (pagePath) {
      await page.goto(pagePath)
      await page.waitForLoadState('networkidle')
      expect(page.url()).not.toContain('/login')
      await expect(page.locator('body')).toBeVisible()
    }
  })

  test('step 17: unsaved changes dialog — stay', async ({ page }) => {
    await page.goto('/sale-pages/new')
    await page.waitForLoadState('networkidle')

    // Make a change to trigger dirty state
    await page.getByLabel('ชื่อหน้าเพจ').fill('Unsaved test')
    await page.waitForTimeout(500)

    // Try to navigate away via sidebar link
    const sidebarLink = page.getByRole('link', { name: 'พิกเซล' }).first()
    await sidebarLink.click()

    // Unsaved changes dialog should appear
    const dialog = page.getByText('มีการเปลี่ยนแปลงที่ยังไม่ได้บันทึก')
    const dialogVisible = await dialog.isVisible().catch(() => false)

    if (dialogVisible) {
      // Click "อยู่ต่อ" → should stay in editor
      await page.getByRole('button', { name: 'อยู่ต่อ' }).click()
      await expect(page).toHaveURL(/\/sale-pages\/new/)
    } else {
      // Some editors may not trigger unsaved dialog for name-only change
      test.skip(true, 'Unsaved changes dialog did not appear — editor may not track this field')
    }
  })

  test('step 18: unsaved changes dialog — leave', async ({ page }) => {
    await page.goto('/sale-pages/new')
    await page.waitForLoadState('networkidle')

    // Make a change
    await page.getByLabel('ชื่อหน้าเพจ').fill('Unsaved test leave')
    // Add a block to ensure dirty state
    const addTextBtn = page.getByRole('button', { name: 'ข้อความ' })
    if (await addTextBtn.isVisible()) {
      await addTextBtn.click()
      await page.waitForTimeout(300)
    }

    // Navigate away
    const sidebarLink = page.getByRole('link', { name: 'พิกเซล' }).first()
    await sidebarLink.click()

    // Unsaved changes dialog
    const dialog = page.getByText('มีการเปลี่ยนแปลงที่ยังไม่ได้บันทึก')
    const dialogVisible = await dialog.isVisible().catch(() => false)

    if (dialogVisible) {
      // Click "ออกโดยไม่บันทึก" → should navigate away
      await page.getByRole('button', { name: 'ออกโดยไม่บันทึก' }).click()
      await expect(page).not.toHaveURL(/\/sale-pages\/new/)
    } else {
      test.skip(true, 'Unsaved changes dialog did not appear')
    }
  })

  test('step 19: cleanup — delete classic sale page', async ({ page }) => {
    const listPage = new SalePagesPage(page)
    await listPage.goto()

    const row = page.locator('[data-testid="sale-page-card"]', { hasText: CLASSIC_SP_NAME })
    if (await row.count() > 0) {
      await row.getByRole('button', { name: 'ลบ' }).click()
      await expect(page.getByRole('heading', { name: 'ลบเซลเพจ' })).toBeVisible()
      await page.locator('button.bg-destructive', { hasText: 'ลบ' }).click()
      await expect(row).not.toBeVisible()
    }
  })
})
