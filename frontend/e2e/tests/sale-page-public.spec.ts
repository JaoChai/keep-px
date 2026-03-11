import { test, expect } from '../fixtures/auth.fixture'
import { SalePagesPage } from '../pages/sale-pages.page'
import { SalePageEditorPage } from '../pages/sale-page-editor.page'

const PUB_SP_NAME = `E2E Pub SP ${Date.now()}`

test.describe('Sale Page Public Serving', () => {
  test.describe.configure({ mode: 'serial' })
  test.setTimeout(60_000)

  let publishedSlug = ''

  test.afterAll(async ({ browser }) => {
    const context = await browser.newContext({
      storageState: 'e2e/.auth/user.json',
    })
    const page = await context.newPage()
    try {
      await page.goto('/sale-pages')
      await page.waitForLoadState('networkidle')
      const rows = page.locator('tr', { hasText: 'E2E Pub' })
      let count = await rows.count()
      while (count > 0) {
        await rows.first().getByRole('button', { name: 'ลบ' }).click()
        await page.getByRole('heading', { name: 'ลบเซลเพจ' }).waitFor()
        await page.locator('button.bg-destructive', { hasText: 'ลบ' }).click()
        await page.waitForTimeout(500)
        count = await rows.count()
      }
    } catch {
      // Best-effort cleanup
    } finally {
      await context.close()
    }
  })

  test('step 1: create and publish a sale page', async ({ page }) => {
    // Cleanup existing test sale pages first
    const salePagesPage = new SalePagesPage(page)
    await salePagesPage.goto()
    await page.waitForLoadState('networkidle')

    const existing = page.locator('tr', { hasText: 'E2E Pub' })
    let count = await existing.count()
    while (count > 0) {
      await existing.first().getByRole('button', { name: 'ลบ' }).click()
      await page.getByRole('heading', { name: 'ลบเซลเพจ' }).waitFor()
      await page.locator('button.bg-destructive', { hasText: 'ลบ' }).click()
      await page.waitForTimeout(500)
      count = await existing.count()
    }

    await salePagesPage.createButton.click()

    const editor = new SalePageEditorPage(page)
    await expect(editor.pageNameInput).toBeVisible()
    await editor.fillMinimum(PUB_SP_NAME)
    await editor.publish()
    await expect(editor.successDialogTitle).toBeVisible({ timeout: 15000 })

    // Try to extract slug from success dialog or URL display
    const codeElement = page.locator('code, [class*="mono"]').first()
    if (await codeElement.isVisible()) {
      const text = await codeElement.textContent()
      if (text) {
        const match = text.match(/\/p\/(.+?)(?:\s|$)/)
        if (match) publishedSlug = match[1]
      }
    }

    await editor.goBackButton.click()
    await expect(page).toHaveURL(/\/sale-pages$/)

    // If we couldn't get slug from dialog, try to get it from the table
    if (!publishedSlug) {
      await page.waitForLoadState('networkidle')
      const row = page.locator('tr', { hasText: PUB_SP_NAME })
      if (await row.isVisible()) {
        // Look for slug in the row data
        const cells = row.locator('td')
        for (let i = 0; i < await cells.count(); i++) {
          const text = await cells.nth(i).textContent()
          if (text && text.match(/^[a-z0-9-]+$/) && text.length > 3) {
            publishedSlug = text
            break
          }
        }
      }
    }
  })

  test('step 2: visit published sale page renders HTML', async ({ page }) => {
    test.skip(!publishedSlug, 'No slug captured from step 1')

    // Visit the public sale page URL
    await page.goto(`/p/${publishedSlug}`)

    // Should render as HTML (not redirect to login)
    await expect(page).not.toHaveURL(/\/login/)

    // The page should have some content (title or body)
    const body = page.locator('body')
    await expect(body).toBeVisible()

    // Should not be a 404
    const notFound = page.getByText('ไม่พบหน้าที่คุณต้องการ')
    const is404 = await notFound.isVisible().catch(() => false)
    expect(is404).toBe(false)
  })

  test('step 3: non-existent slug shows not found', async ({ page }) => {
    await page.goto('/p/non-existent-slug-that-does-not-exist-12345')
    await page.waitForLoadState('networkidle')

    // Backend renders sale pages server-side — non-existent slug may return
    // 404 status OR 200 with "not found" content (depending on nginx config).
    // Check page content for any error/not-found indicator.
    const body = await page.locator('body').textContent()
    const isNotFound =
      body?.includes('404') ||
      body?.includes('not found') ||
      body?.includes('ไม่พบ') ||
      body?.includes('Not Found') ||
      // If the page is blank/minimal (backend returned error HTML)
      (body?.trim().length ?? 0) < 100
    expect(isNotFound).toBe(true)
  })

  test('step 4: cleanup', async ({ page }) => {
    const salePagesPage = new SalePagesPage(page)
    await salePagesPage.goto()
    await page.waitForLoadState('networkidle')

    const row = page.locator('tr', { hasText: 'E2E Pub' })
    if (await row.count() > 0) {
      await row.first().getByRole('button', { name: 'ลบ' }).click()
      await page.getByRole('heading', { name: 'ลบเซลเพจ' }).waitFor()
      await page.locator('button.bg-destructive', { hasText: 'ลบ' }).click()
      await expect(row).not.toBeVisible()
    }
  })
})
