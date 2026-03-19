---
name: e2e-write
description: Proactive rules for writing Playwright E2E tests that pass CI — Thai text selectors, custom dialogs, responsive duplicates, sandbox guards, and spec templates (159 tests, 12 page objects)
---

# E2E Write

## When to Activate

Activate this skill when:
- Writing new E2E test specs or page objects
- Adding test coverage for new features
- Creating journey/scenario tests
- User says "write E2E test" / "add test for X page" / "create spec"

## Test Suite Context

- **19 spec files** in `frontend/e2e/tests/` (+ `journeys/` subfolder)
- **12 page objects** in `frontend/e2e/pages/`
- **159 tests** total
- Auth: token-based via `E2E_ACCESS_TOKEN` + `E2E_REFRESH_TOKEN` env vars
- Config: `frontend/playwright.config.ts`
- Global setup: `frontend/e2e/support/global-setup.ts`

---

## Rule 1: Thai Text + `getByText` = `{ exact: true }` Always

Thai text is reused heavily across components. Without `{ exact: true }`, Playwright matches partial substrings and hits strict mode violations.

**Bad:**
```typescript
await page.getByText('สร้าง').click()         // Matches 16+ elements!
await page.getByText('ยกเลิก').click()        // Matches 14+ elements!
```

**Good:**
```typescript
await page.getByText('สร้างเพจ', { exact: true }).click()
await page.getByRole('button', { name: 'ยกเลิก' }).first().click()
// Or scope to a specific container:
await dialog.getByText('บันทึก', { exact: true }).click()
```

### Thai Text Collision Table

| Thai Text | English | Files | Risk Level |
|-----------|---------|-------|------------|
| สร้าง | create | 16 | Critical |
| ยกเลิก | cancel | 14 | Critical |
| จัดการ | manage | 12 | High |
| เพิ่ม | add | 11 | High |
| บันทึก | save | 10 | High |
| รายละเอียด | details | 8 | Medium |
| ตั้งค่า | settings | 7 | Medium |
| ค้นหา | search | 4 | Low |
| แก้ไข | edit | 4 | Low |
| ลบ | delete | 4 | Low |

**Strategy for high-collision words:**
1. Use the **full button/link text** (e.g., "สร้างเพจขาย" not just "สร้าง")
2. Scope to parent: `dialog.getByRole('button', { name: 'บันทึก' })`
3. Use `{ exact: true }` to prevent substring matches
4. Last resort: `.first()` with a comment explaining why

---

## Rule 2: Sandbox User Empty State = `test.skip()` Guard

The E2E test user (sandbox) may have no data (no pixels, no events, no sale pages). Tests that interact with existing data MUST guard against empty state.

**Pattern:**
```typescript
test('should edit existing pixel', async ({ page }) => {
  const pixelsPage = new PixelsPage(page)
  await pixelsPage.goto()

  const hasPixels = await pixelsPage.pixelRows.count() > 0
  test.skip(!hasPixels, 'No pixels in sandbox — skipping edit test')

  // Safe to interact now
  await pixelsPage.pixelRows.first().click()
})
```

**When to use:**
- Any test that clicks on a list item (pixel, sale page, event, replay)
- Any test that expects data in a table/list
- Any test that deletes an item (must exist first)

---

## Rule 3: Custom Dialog — No ARIA `role="dialog"`

Keep-PX dialogs are **custom implementations** (NOT Radix Dialog). They do NOT emit `role="dialog"` or `aria-modal`.

**Bad:**
```typescript
await page.getByRole('dialog')  // Will NOT find our custom dialogs!
```

**Good:**
```typescript
// Wait for dialog by its heading
await expect(page.getByRole('heading', { name: 'แก้ไข Pixel' })).toBeVisible()

// Scope to dialog container by class or test-id
const dialog = page.locator('[class*="fixed inset-0"]').last()
await dialog.getByRole('button', { name: 'บันทึก' }).click()

// Wait for dialog to close
await expect(page.getByRole('heading', { name: 'แก้ไข Pixel' })).not.toBeVisible()
```

**Dialog lifecycle in tests:**
1. Trigger open → wait for heading `toBeVisible()`
2. Fill fields
3. Click submit → wait for heading `not.toBeVisible()`
4. Assert page content updated

---

## Rule 4: Collapsible Sections — Closed by Default

Custom `Collapsible` starts closed. Click header text to expand before interacting with content inside.

---

## Rule 5: Conditional Visibility = Safe Check

For elements that may not exist, use `.isVisible().catch(() => false)` before interacting.

---

## Rule 6: Unique Test Data — Prefix + Timestamp

Test data names MUST be unique to avoid collision between parallel runs or reruns.

**Pattern:** `` `E2E Test Pixel ${Date.now()}` `` — prefix `E2E ` + `Date.now()` for uniqueness. Store in variable, reuse for assertions.

---

## Rule 7: Cleanup Before Create (Quota Management)

Free-tier sandbox users have quota limits (e.g., max 5 pixels). Tests that create resources MUST clean up first if needed.

Before creating resources, check count and delete if at limit: `if (count >= 5) { await deletePixel(last) }`. Applies to pixels, sale pages, and any quota-limited resource.

---

## Rule 8: Responsive Layout = Duplicate Elements

Sidebar and navigation render different markup for mobile vs desktop. Playwright sees BOTH in the DOM even if only one is visible.

**Bad:**
```typescript
await page.getByRole('link', { name: 'Pixels' }).click()  // 2 matches!
```

**Good:**
```typescript
// Use .first() — desktop nav renders first in DOM
await page.getByRole('link', { name: 'Pixels' }).first().click()

// Or scope to the visible sidebar
const sidebar = page.locator('nav[class*="hidden lg:flex"]')
await sidebar.getByRole('link', { name: 'Pixels' }).click()
```

---

## Rule 9: Zod Schema = Fill ALL Required Fields

Forms use Zod validation with `zodResolver`. If a required field is empty, `handleSubmit` never fires — the dialog stays open silently.

**Diagnostic:** When dialog stays open after clicking save:
1. Find the component's Zod schema (`zodResolver(...)`)
2. List ALL required fields
3. Ensure test fills every single one

**Common miss:** `fb_access_token` is required by schema even though UI says "(leave blank to keep current)"

---

## Rule 10: Production Smoke Tests Must Be State-Agnostic

`post-deploy-e2e` runs `@smoke` tests against **production** — the test user's data changes over time. Never assume a specific state.

**Bad:**
```typescript
// Hardcodes "no credits" — breaks when user has credits
await expect(page.getByText('ยังไม่มีเครดิตรีเพลย์')).toBeVisible()
```

**Good:**
```typescript
// State-agnostic: accept either state
const hasCredits = await page.getByText('เครดิตที่มีอยู่').isVisible().catch(() => false)
if (hasCredits) {
  await expect(page.getByText('เครดิตที่มีอยู่')).toBeVisible()
} else {
  await expect(page.getByText('ยังไม่มีเครดิตรีเพลย์')).toBeVisible()
}
```

**Principles:**
1. Assert **structure** (sections exist, buttons present) — not **specific content state**
2. Use conditional checks when UI depends on user data (credits, subscriptions, list items)
3. Add generous `timeout` for `toHaveURL` after save/redirect — production is slower:
   ```typescript
   await expect(page).toHaveURL(/\/sale-pages$/, { timeout: 15000 })
   ```
4. Distinguish: PR CI tests (predictable state) vs smoke tests (unpredictable state)

---

## Conventions

| Convention | Rule |
|-----------|------|
| Page object file | `frontend/e2e/pages/{feature}.page.ts` |
| Spec file | `frontend/e2e/tests/{feature}.spec.ts` |
| Journey spec | `frontend/e2e/tests/journeys/scenario-{N}-{name}.spec.ts` |
| Locators | Define ALL in constructor, not in methods |
| `goto()` | Navigate + `waitForLoadState('networkidle')` |
| Locator priority | `getByRole` > `getByLabel` > `getByText` > `locator` |
| Thai text locators | Always `{ exact: true }` or scoped |

---

## Checklist Before Commit

Before pushing E2E test code, verify:

- [ ] All `getByText` with Thai text use `{ exact: true }` or are scoped
- [ ] No bare `getByRole('dialog')` — use heading-based detection
- [ ] Tests that interact with list items have `test.skip()` guard for empty state
- [ ] Test data names include `Date.now()` for uniqueness
- [ ] All Zod-required form fields are filled
- [ ] Dialog lifecycle: wait open → interact → wait close → assert
- [ ] Responsive nav locators use `.first()` or scope
- [ ] Resource creation tests handle quota limits
- [ ] `waitForLoadState('networkidle')` in page object `goto()`
- [ ] `@smoke` tests are state-agnostic — no hardcoded assumptions about user data
- [ ] `toHaveURL` after save/redirect uses `{ timeout: 15000 }` for production

---

## Related

- `e2e-debug` for debugging CI failures after they happen
- `ci-pipeline` for CI job structure and E2E auth setup
- `dev-workflow` for file co-change patterns when adding new features
