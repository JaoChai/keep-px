---
name: event-pipeline
description: Checklist and pitfalls for the Keep-PX event tracking pipeline — sale page templates, SDK endpoint sync, and Meta CAPI required fields
---

# Event Pipeline

## When to Activate

Activate this skill when the user says:
- "Edit sale page" / "Add field to sale page" / "Change sale page template"
- "Fix SDK endpoint" / "Events not showing" / "SDK not sending events"
- "Send event to Meta" / "Fix CAPI" / "Test pixel connection"
- "Events not appearing in Meta Events Manager"
- "Change event tracking" / "Update pixel integration"

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

### Frontend (change after backend)
9. `frontend/src/types/index.ts` — TypeScript types (must match domain struct)
10. `frontend/src/hooks/use-sale-pages.ts` — TanStack Query hooks
11. `frontend/src/pages/SalePageEditorPage.tsx` — Editor page
12. `frontend/src/pages/BlockEditorPage.tsx` — Block editor
13. `frontend/src/components/sale-pages/BlockEditor.tsx` — Block editor component
14. `frontend/src/components/sale-pages/BlockPreview.tsx` — Block preview
15. `frontend/src/components/sale-pages/SalePagePreview.tsx` — Page preview

**Critical**: `blocks.html` and `simple.html` ALWAYS need the same changes. Forgetting one causes silent rendering failures.

## Part 2: SDK Endpoint Three-Point Sync

When changing the SDK endpoint, you MUST sync all three locations:

```
sdk/src/tracker.ts (DEFAULT_ENDPOINT)
        ↕ must match
backend/internal/templates/sale_pages/*.html (data-endpoint="...")
        ↕ must match
frontend/src/pages/PixelsPage.tsx ("Get Code" snippet)
```

If any one is out of sync, events silently fail — SDK sends to wrong URL, backend never receives POST, no CAPI forwarding.

### Diagnostic: Events Not Appearing
1. Check browser Network tab → POST `/api/v1/events/ingest`
2. POST goes to wrong domain → SDK endpoint misconfigured
3. No POST at all → `data-endpoint` missing from script tag
4. Check backend logs for incoming event requests

**Trap**: Browser pixel (fbq) works independently. Pixel Helper shows green, but server-side CAPI path may be broken.

## Part 3: Meta CAPI Required Fields

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

## Verification

After changes to the event pipeline:
1. Check SDK sends POST to correct backend URL (browser Network tab)
2. Check backend logs for received events
3. Check CAPI response status in backend logs (should be 200)
4. Verify events appear in Meta Events Manager (allow 20 min)

## Related

- `go-service-scaffold` for creating new backend resources
- `api-endpoint` for adding new endpoints
- `deploy-check` for pre-deployment verification
