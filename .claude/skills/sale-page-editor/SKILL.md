---
name: sale-page-editor
description: Sale page CRUD, block editor, template rendering, pixel assignment patterns, and co-change checklist for Keep-PX
---

# Sale Page Editor

## When to Activate

Activate this skill when the user says:
- "Create/edit/delete sale page"
- "Add block type" / "New template"
- "Sale page preview" / "Sale page not rendering"
- "Assign pixel to sale page" / "Multi-pixel sale page"
- "Slug collision" / "Sale page URL"
- "Block editor" / "Content version"

## Part 1: Architecture Overview

```
Frontend Editor (SalePageEditorPage.tsx)
    → API: POST/PUT /api/v1/sale-pages
    → Backend: sale_page_service.go (validates, generates slug)
    → DB: sale_pages + sale_page_pixels (M:M join)
    → Public: GET /p/:slug → sale_page_handler.go renders HTML template
    → Template: blocks.html or simple.html + tracking.html injection
```

### Content Versions
- **V1 (SimpleContent)**: Single-section page with title, description, image, CTA button
- **V2 (BlocksContent)**: Multi-block page with ordered blocks (hero, text, image, cta, etc.)

Content version is stored in `content_version` column. Handler peeks at the struct to determine which template to render. Falls back to `simple.html` with a warning log if version detection fails.

### Template Map
| Template | Purpose | File |
|----------|---------|------|
| `simple.html` | V1 simple pages | `backend/internal/templates/sale_pages/simple.html` |
| `blocks.html` | V2 block-based pages | `backend/internal/templates/sale_pages/blocks.html` |
| `tracking.html` | Shared pixel tracking script | `backend/internal/templates/sale_pages/tracking.html` |

## Part 2: Co-Change Checklist

When modifying the Sale Page feature, update all layers in order:

### Backend (7 files)
1. `backend/internal/domain/sale_page.go` — Domain struct, content types, validation
2. `backend/internal/repository/postgres/sale_page_repo.go` — SQL queries
3. `backend/internal/service/sale_page_service.go` — Business logic, slug generation
4. `backend/internal/handler/sale_page_handler.go` — HTTP handler, template rendering
5. `backend/internal/templates/sale_pages/blocks.html` — Block template
6. `backend/internal/templates/sale_pages/simple.html` — Simple template
7. `backend/internal/templates/sale_pages/tracking.html` — Tracking script

### Frontend (5 files)
8. `frontend/src/types/index.ts` — TypeScript types (must match domain struct)
9. `frontend/src/hooks/use-sale-pages.ts` — TanStack Query hooks
10. `frontend/src/components/sale-pages/BlockEditor.tsx` — Block editor component
11. `frontend/src/components/sale-pages/SalePagePreview.tsx` — Page preview
12. `frontend/src/pages/SalePageEditorPage.tsx` — Editor page

**Rule**: `blocks.html` + `simple.html` + `tracking.html` ALWAYS change together when modifying tracking behavior.

## Part 3: Key Patterns

### Slug Generation
```go
// p-XXXXXXXX format with retry loop (5 attempts)
func generateSlug() string {
    return fmt.Sprintf("p-%s", randomHex(4)) // e.g., p-a1b2c3d4
}
```
Service retries slug generation up to 5 times if uniqueness check fails (DB unique constraint on `slug`).

### Pixel Ownership Validation
```go
// validatePixelOwnership checks ALL pixel_ids belong to the customer
func (s *SalePageService) validatePixelOwnership(ctx, customerID, pixelIDs) error
```
Every pixel ID in the request must be owned by the requesting customer. This runs before any create/update operation that assigns pixels.

### Transaction-Based Pixel Assignment
```go
// setPixels uses delete-all-then-reinsert within a transaction
func (r *SalePageRepo) SetPixels(ctx, tx, salePageID, pixelIDs) error {
    // DELETE FROM sale_page_pixels WHERE sale_page_id = $1
    // INSERT INTO sale_page_pixels (sale_page_id, pixel_id, position) VALUES ...
}
```
Position ordering is maintained — the order of `pixel_ids` in the request determines the `position` column value.

### Cache Invalidation on Slug Rename
When a sale page slug changes (update operation), both the old and new slugs must be invalidated:
```go
// Invalidate old slug cache entry
// Invalidate new slug cache entry
// This prevents stale cached pages from being served
```

### Content Version Detection
Handler peeks at the content JSON to determine which template to use:
```go
// If content has "blocks" key → use blocks.html (V2)
// Otherwise → use simple.html (V1)
// Fallback: simple.html with warning log
```

## Part 4: Common Pitfalls

1. **Template tracking sync**: `blocks.html` and `simple.html` must inject `tracking.html` identically. Forgetting one template means events are silently lost for pages using that template.

2. **Pixel ownership validation**: MUST validate every `pixel_id` before saving. Skipping this allows users to attach other customers' pixels to their pages.

3. **Slug validation**: Reserved words list + regex validation + DB uniqueness check. Slugs must be URL-safe and not conflict with route prefixes (`api`, `assets`, `auth`, etc.).

4. **Cache invalidation on rename**: When slug changes, invalidate BOTH old slug and new slug. Missing old slug invalidation = stale page served. Missing new slug = 404 until cache expires.

5. **Content version mismatch**: If handler can't detect content version, it falls back to `simple.html` and logs a warning. This is intentional — never crash on unknown content.

6. **sale_page_pixels join table ordering**: The `position` column determines pixel order. Reordering pixels requires delete-all + re-insert (not individual updates).

7. **Draft auto-save**: Frontend uses localStorage + debounce for unsaved changes. `useBlocker` hook warns users before navigating away from unsaved edits.

## Part 5: Adding a New Block Type

1. **Domain**: Add new block type constant in `domain/sale_page.go`
2. **Validation**: Add validation rules for the new block type
3. **Template**: Add `{{if eq .Type "new_type"}}` section in `blocks.html`
4. **Frontend BlockEditor**: Add new block component in `BlockEditor.tsx`
5. **Frontend Preview**: Add rendering logic in `SalePagePreview.tsx`

## Part 6: Adding a New Template

1. **HTML file**: Create `backend/internal/templates/sale_pages/new_template.html`
2. **Handler**: Register template in the handler's template map
3. **Validation**: Add `new_template` to allowed `template_name` values in service layer
4. **Frontend**: Add template option to the editor's template selector
5. **Tracking**: Include `tracking.html` injection in the new template

## Verification

After sale page changes:
```bash
cd backend && go vet ./... && go test -race ./...
cd frontend && npm run lint && npm run build
cd frontend && npx playwright test sale-pages.spec.ts
```

Check public page renders correctly:
```bash
curl -s http://localhost:8080/p/<slug> | head -50
```

## Related

- `event-pipeline` for tracking script and CAPI integration
- `db-migration` for schema changes to sale_pages/sale_page_pixels
- `e2e-debug` for debugging sale-pages.spec.ts failures
- `deploy-check` for pre-deployment verification
