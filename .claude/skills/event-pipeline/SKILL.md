---
name: event-pipeline
description: Checklist and pitfalls for the Keep-PX event tracking pipeline — sale page templates, Meta CAPI forwarding, and template co-change rules
---

# Event Pipeline

## When to Activate

Activate this skill when the user says:
- "Edit sale page" / "Add field to sale page" / "Change sale page template"
- "Events not showing" / "Events not appearing"
- "Send event to Meta" / "Fix CAPI" / "Test pixel connection"
- "Events not appearing in Meta Events Manager"
- "Change event tracking" / "Update pixel integration"

## Architecture

```
Sale page HTML (blocks.html / simple.html)
    → tracking.html (injected via Go template)
    → fetch POST /api/v1/events/ingest (API Key auth)
    → backend event_service.go (store in DB)
    → go s.forwardToCAPI() (async Meta CAPI forward)
```

Events flow from sale page templates directly to the backend ingest endpoint. There is no client-side SDK — all tracking is embedded in the sale page HTML templates.

## Part 1: Sale Page Co-Change Checklist

When modifying the Sale Page feature, a single change typically touches 8-15 files. Update all layers in order:

### Backend (change first)
1. `backend/internal/domain/sale_page.go` — Domain struct
2. `backend/db/migrations/` — SQL migration (if schema change)
3. `backend/internal/repository/postgres/sale_page_repo.go` — Query updates
4. `backend/internal/service/sale_page_service.go` — Business logic
5. `backend/internal/handler/sale_page_handler.go` — HTTP handler
6. `backend/internal/router/router.go` — New routes (if needed)
7. `backend/internal/templates/sale_pages/blocks.html` — Block template
8. `backend/internal/templates/sale_pages/simple.html` — Simple template
9. `backend/internal/templates/sale_pages/tracking.html` — Tracking script (shared)

### Frontend (change after backend)
10. `frontend/src/types/index.ts` — TypeScript types (must match domain struct)
11. `frontend/src/hooks/use-sale-pages.ts` — TanStack Query hooks
12. `frontend/src/pages/SalePageEditorPage.tsx` — Editor page
13. `frontend/src/components/sale-pages/BlockEditor.tsx` — Block editor component
14. `frontend/src/components/sale-pages/SalePagePreview.tsx` — Page preview

**Critical**: `blocks.html`, `simple.html`, and `tracking.html` ALWAYS need the same tracking changes. Forgetting one causes silent rendering or tracking failures.

## Part 2: Meta CAPI Required Fields

When sending events to Facebook Conversions API, ALL of these fields are required:

```go
EventData{
    EventName:      "PageView",           // Required
    EventTime:      time.Now().Unix(),     // Required, Unix timestamp
    ActionSource:   "website",            // Required
    EventSourceURL: "https://...",        // Required, valid URL
    UserData: UserData{                   // Required, at least one identifier
        ClientUserAgent: "...",           // From User-Agent header
        ExternalID:      "sha256...",     // SHA-256 hashed
    },
}
```

### Rules
- `event_source_url` — MUST be a valid URL. Meta rejects empty.
- `user_data` — MUST contain at least `client_user_agent`. Without it → HTTP 400.
- `client_ip_address` — Only from real client requests. For test events → omit it. Invalid IPs → HTTP 400.
- PII hashing — `email`, `phone`, `external_id` MUST be SHA-256 hashed. `client_ip_address` and `client_user_agent` are NOT hashed.
- Event dedup — Use `event_id` (UUID). Same ID + name within 48h = deduplicated.

### Test Connection vs Production

| Field | Test Event | Production Event |
|-------|-----------|-----------------|
| `event_source_url` | `https://keepx.io/test` | Actual page URL |
| `client_ip_address` | Don't include | From X-Forwarded-For |
| `client_user_agent` | `KeepPX/1.0 Connection Test` | From User-Agent header |
| `external_id` | `test-{pixelID}-{timestamp}` | Hashed real user ID |

### Debugging "Events Not in Meta Events Manager"
1. Backend logs show CAPI response 200 but not visible? → Wait 20+ minutes for Overview tab
2. Response 400 → Missing `user_data` or `event_source_url`
3. Events with `test_event_code` → only in Test Events tab, not Overview

## Diagnostic: Events Not Appearing

1. Check sale page HTML source → confirm tracking.html is injected
2. Check browser Network tab → POST `/api/v1/events/ingest`
3. No POST at all → `tracking.html` not rendering or API key missing
4. POST returns 4xx → Check API key validity and request body format
5. Check backend logs for received events and CAPI response

**Trap**: Browser pixel (fbq) works independently. Pixel Helper shows green, but server-side CAPI path may be broken.

## Verification

After changes to the event pipeline:
1. Check sale page renders tracking script (view page source at `/p/<slug>`)
2. Check browser Network for POST to `/api/v1/events/ingest`
3. Check backend logs for received events
4. Check CAPI response status in backend logs (should be 200)
5. Verify events appear in Meta Events Manager (allow 20 min)

## Related

- `sale-page-editor` for sale page CRUD and template patterns
- `go-service-scaffold` for creating new backend resources
- `api-endpoint` for adding new endpoints
- `deploy-check` for pre-deployment verification
