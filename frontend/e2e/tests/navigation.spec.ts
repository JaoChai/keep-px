import { test, expect } from '../fixtures/auth.fixture'
import { test as baseTest } from '@playwright/test'
import { SidebarPage } from '../pages/sidebar.page'

test.describe('Navigation @smoke', () => {
  test('all sidebar items visible', async ({ page }) => {
    await page.goto('/dashboard')
    const sidebar = new SidebarPage(page)

    await expect(sidebar.brand).toBeVisible()
    await expect(sidebar.dashboardLink).toBeVisible()
    await expect(sidebar.pixelsLink).toBeVisible()
    await expect(sidebar.salePagesLink).toBeVisible()
    await expect(sidebar.customDomainsLink).toBeVisible()
    await expect(sidebar.eventLogLink).toBeVisible()
    await expect(sidebar.realtimeLink).toBeVisible()
    await expect(sidebar.replayCenterLink).toBeVisible()
    await expect(sidebar.settingsLink).toBeVisible()
    await expect(sidebar.logoutButton).toBeVisible()
  })

  test('navigate to each page via sidebar', async ({ page }) => {
    await page.goto('/dashboard')
    const sidebar = new SidebarPage(page)

    const routes = [
      { name: 'Pixels', url: '/pixels' },
      { name: 'Sale Pages', url: '/sale-pages' },
      { name: 'Custom Domains', url: '/domains' },
      { name: 'Event Log', url: '/events/log' },
      { name: 'Realtime', url: '/events/realtime' },
      { name: 'Replay Center', url: '/replay' },
      { name: 'Settings', url: '/settings' },
      { name: 'Dashboard', url: '/dashboard' },
    ]

    for (const route of routes) {
      await sidebar.navigateTo(route.name)
      await expect(page).toHaveURL(new RegExp(route.url))
    }
  })
})

baseTest.describe('Navigation - Unauthenticated', () => {
  baseTest('protected route redirects to /login when not authenticated', async ({ page }) => {
    await page.goto('/dashboard')

    await expect(page).toHaveURL(/\/login/)
  })
})
