---
name: e2e-debug
description: Debug Playwright E2E test failures in CI — root cause analysis, Zod schema alignment, and dialog timing patterns
---

# E2E Debug

## When to Activate

Activate this skill when the user says:
- "E2E tests failing" / "CI tests fail" / "Tests pass locally but fail in CI"
- "Dialog won't close" / "Form not submitting in test"
- "Playwright timeout" / "Strict mode violation"
- "Fix flaky test" / "Debug CI failure"

## Step 1: Read the CI Error

Never guess. Read the exact error first:

```bash
gh run view <run-id> --log-failed | tail -80
```

From the error, identify:
1. Which locator failed? What was expected vs received?
2. Did the previous action complete? (dialog open, form submit, navigation)
3. Is it a timing issue? (CI is 2-5x slower than local)
4. Is it a validation issue? (form didn't submit due to Zod schema)

Then read the actual React component code — understand what triggers the state change the test expects.

## Step 2: Check Zod Schema Alignment

If a dialog stays open after clicking submit, the most likely cause is a Zod validation failure.

**Rule**: Fill ALL fields required by the Zod schema — not just what the UI suggests.

```typescript
// Schema says: fb_access_token: z.string().min(1, 'Required')
// UI says: "(leave blank to keep current)"
// Test MUST fill it anyway:

await pixelsPage.nameInput.fill(updatedName)
await pixelsPage.accessTokenInput.fill('EAAtest123token')  // Required by schema!
await pixelsPage.saveButton.click()
```

**Why it silently fails**:
- Zod validation runs client-side before `handleSubmit`
- If validation fails, the callback never executes
- Dialog stays open, test times out — no visible error

**Diagnostic**:
1. Find the Zod schema for the form (search for `zodResolver` in the component)
2. List all required fields in the schema
3. Compare with what the test fills
4. Missing field = root cause

## Step 3: Add Dialog Timing Waits

CI environments are 2-5x slower. Always add explicit waits around dialog interactions:

```typescript
// 1. Wait for dialog to OPEN
await expect(
  page.getByRole('heading', { name: 'Edit Pixel' })
).toBeVisible()

// 2. Interact with fields
await nameInput.fill('new value')
await saveButton.click()

// 3. Wait for dialog to CLOSE
await expect(
  page.getByRole('heading', { name: 'Edit Pixel' })
).not.toBeVisible()

// 4. NOW assert page content
await expect(page.getByText('new value')).toBeVisible()
```

**Rules**:
- Before interacting with dialog → wait for heading `toBeVisible()`
- After clicking submit/save → wait for heading `not.toBeVisible()`
- Between sequential dialogs → wait for previous to fully close first
- After table update → wait for expected content to appear

## Quick Reference: Common CI Failures

| Symptom | Root Cause | Fix |
|---------|-----------|-----|
| Dialog stays open after submit | Zod validation failure | Fill ALL required schema fields |
| Element not found after action | Animation/render delay | Add `toBeVisible()` / `not.toBeVisible()` wait |
| Strict mode violation | Multiple matching elements | Scope locator: `.locator('form').getByRole(...)` |
| Button click has no effect | Element obscured by overlay | Wait for previous dialog/overlay to close first |
| Test passes locally, fails CI | CI slower than local | Add explicit waits, never rely on implicit timing |
| Stale data after mutation | TanStack Query cache | Wait for updated content to appear in DOM |

## Decision Tree

```
E2E test fails in CI
├── Read CI logs (gh run view --log-failed)
├── Timeout on dialog?
│   ├── After clicking submit? → Check Zod schema (Step 2)
│   └── On open/close? → Add timing waits (Step 3)
├── Strict mode violation?
│   └── Scope locator with parent: .locator('form').getByRole(...)
├── Element not found?
│   ├── After dialog close? → Wait for not.toBeVisible() first
│   └── After navigation? → Wait for new page content
└── Passes locally but fails CI?
    └── Add explicit waits — CI is always slower
```

## Verification

After fixing:
```bash
cd frontend && npx playwright test              # Run all E2E tests locally
cd frontend && npx playwright test --grep "test name"  # Run specific test
```

If tests pass locally, push and verify CI:
```bash
git push && gh run watch
```

## Related

- `frontend-feature` for creating new pages with testable structure
- `deploy-check` for pre-deployment verification including E2E
