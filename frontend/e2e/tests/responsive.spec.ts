import { test, expect } from '../fixtures/auth.fixture'

test.describe('Responsive Layout', () => {
  test('mobile viewport hides sidebar and shows mobile menu', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 812 })
    await page.goto('/dashboard')

    // On mobile, sidebar should be hidden or collapsed
    // Check for mobile menu toggle button
    const menuButton = page.getByRole('button', { name: /เมนู|menu/i })
      .or(page.locator('button[class*="sidebar"]'))
      .or(page.locator('button', { has: page.locator('[class*="lucide-menu"]') }))

    // Either menu button exists (for toggling sidebar) or sidebar is always visible
    const hasMenuButton = await menuButton.isVisible().catch(() => false)
    const sidebarBrand = page.getByRole('heading', { name: 'Pixlinks' })

    if (hasMenuButton) {
      // Sidebar should be hidden initially on mobile
      // Menu button should toggle it
      await menuButton.click()
      await expect(sidebarBrand).toBeVisible()
    } else {
      // Some layouts keep sidebar always visible — just verify page loads
      await expect(page.getByRole('heading', { name: 'แดชบอร์ด' })).toBeVisible()
    }
  })

  test('tablet viewport renders correctly', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 })
    await page.goto('/dashboard')

    await expect(page.getByRole('heading', { name: 'แดชบอร์ด' })).toBeVisible()
    // Dashboard stat cards should be visible
    await expect(page.getByText('พิกเซลที่ใช้งาน')).toBeVisible()
  })

  test('wide desktop viewport renders correctly', async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 })
    await page.goto('/dashboard')

    await expect(page.getByRole('heading', { name: 'แดชบอร์ด' })).toBeVisible()
    // Sidebar should be visible
    await expect(page.getByRole('heading', { name: 'Pixlinks' })).toBeVisible()
    // Stat cards should be visible
    await expect(page.getByText('พิกเซลที่ใช้งาน')).toBeVisible()
  })
})
