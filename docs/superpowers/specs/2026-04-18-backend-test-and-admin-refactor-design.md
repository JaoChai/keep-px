# Backend Test Mocks + Admin File Split Refactor

**Date**: 2026-04-18
**Status**: Approved for planning
**Type**: Pure mechanical refactor — zero behavior change

## Background & Motivation

Code audit of `keep-px` backend surfaced two concentrated pain points:

1. **Duplicated mocks** (`service/mocks_test.go` 620 LOC + `handler/testhelpers_test.go` 737 LOC) — ~70% overlap. CLAUDE.md documents this as a recurring gotcha: "Change `interfaces.go` → update mocks in BOTH files." Every repository interface change requires synchronized edits in two places. Root cause: mock definitions live inside `_test.go` files of two separate packages and cannot be shared.

2. **`admin_*` god-files** — `repository/postgres/admin_repo.go` (1,275 LOC, 25 methods), `service/admin_service.go` (490 LOC, 32 methods), `handler/admin_handler.go` (588 LOC). Each file spans 7 sub-domains (customers, platform stats, billing views, sale pages, pixels, replays, events, audit). Locating methods requires scrolling through unrelated code; diff review is painful.

Other audit findings (large service files, frontend page bloat, GuidePage.tsx) are explicitly **out of scope** for this spec and deferred to future specs.

## Goals

- Eliminate mock duplication — single source of truth imported by both service and handler tests.
- Reduce admin file sizes to ≤300 LOC each without changing any public API, interface, or behavior.
- Preserve existing test coverage and success criteria exactly.

## Non-Goals

- **No mock generator** (mockery, gomock). Keep `testify/mock` manual; future spec may revisit if pain reappears.
- **No interface splitting** for `AdminRepository`. Single interface stays; only file locations change.
- **No sub-service decomposition** for `AdminService`. Single struct stays.
- **No DI wiring or router changes.** `router.go` must not be touched.
- **No behavior change, no API change, no error-shape change.**
- **No refactor** of `replay_service.go`, `billing_service.go`, frontend pages, `GuidePage.tsx`, or anything else.

## Design

### Phase 1 — Shared Mocks Package

**New package**: `backend/internal/repository/mocks/` — non-test Go package with public exports, importable by any `*_test.go` in the module.

**File layout** (one file per repository interface):

```
backend/internal/repository/mocks/
├── admin.go              # MockAdminRepo
├── customer.go           # MockCustomerRepo
├── event.go              # MockEventRepo
├── event_usage.go        # MockEventUsageRepo
├── pixel.go              # MockPixelRepo
├── purchase.go           # MockPurchaseRepo
├── refresh_token.go      # MockRefreshTokenRepo
├── replay_credit.go      # MockReplayCreditRepo
├── replay_session.go     # MockReplaySessionRepo
├── sale_page.go          # MockSalePageRepo
├── subscription.go       # MockSubscriptionRepo
└── webhook_event.go      # MockWebhookEventRepo
```

**Conventions**:
- Package declaration: `package mocks`
- Each file defines one public mock type: `type MockXxxRepo struct{ mock.Mock }` and its methods.
- Methods mirror the interface in `backend/internal/repository/interfaces.go` exactly.
- No additional helpers, constructors, or fixtures — just the mock types.
- Dependencies: `github.com/stretchr/testify/mock` and `github.com/jaochai/pixlinks/backend/internal/domain`. No new `go.mod` entries.

**Removals**:
- `backend/internal/service/mocks_test.go` — delete entirely (all content moves to shared package).
- `backend/internal/handler/testhelpers_test.go` — retain only: `testJWT`, `doRequest`, `testConfig`, `testLogger`, the `test*` constants. Delete all `Mock*` type definitions and their methods.

**Import updates** — every `*_test.go` file that references mock types:
- `backend/internal/service/*_test.go` — change `MockXxxRepo{}` → `mocks.MockXxxRepo{}`, add import `"github.com/jaochai/pixlinks/backend/internal/repository/mocks"`.
- `backend/internal/handler/*_test.go` — same update.

### Phase 2 — Admin File Split

Three source files get split; each keeps the same package, struct, and interface. Only the physical location of method receivers changes.

**Sub-domain boundaries** follow the `F1..F5` comments already present in `AdminRepository`:

| Sub-domain | Methods moved (repo/service/handler) |
|------------|--------------------------------------|
| **customers** | `ListCustomers`, `GetCustomerDetail`, `SuspendCustomer`, `ActivateCustomer`, `ChangePlan` (service), `GrantCredits` (service) |
| **stats** | `GetPlatformStats`/`GetPlatformOverview`, `GetRevenueChart`, `GetGrowthChart` |
| **billing** | `ListAllPurchases`, `ListAllSubscriptions`, `ListCreditGrants` |
| **salepages** | `ListAllSalePages`, `GetSalePageAdminDetail`/`GetSalePageDetail`, `SetSalePagePublished`/`EnableSalePage`/`DisableSalePage`, `DeleteSalePageByAdmin`/`DeleteSalePage` |
| **pixels** | `ListAllPixels`, `GetPixelAdminDetail`/`GetPixelDetail`, `SetPixelActive`/`EnablePixel`/`DisablePixel` |
| **replays** | `ListAllReplaySessions`, `GetReplaySessionAdminDetail`/`GetReplayDetail`, `CancelReplay` (service) |
| **events** | `ListAllEvents`, `GetEventStats` |
| **audit** | `CreateAuditLog`, `ListAuditLogs` |

**Repository split** (`backend/internal/repository/postgres/`):

```
admin_repo.go                # AdminRepo struct, NewAdminRepo, shared scan helpers (scanCustomerFromRows, etc.)
admin_repo_customers.go
admin_repo_stats.go
admin_repo_billing.go
admin_repo_salepages.go
admin_repo_pixels.go
admin_repo_replays.go
admin_repo_events.go
admin_repo_audit.go
```

**Service split** (`backend/internal/service/`):

```
admin_service.go             # AdminService struct, NewAdminService, platformStatsCache type + methods, logAudit, normalizePagination
admin_service_customers.go
admin_service_stats.go       # also hosts GetPlatformOverview
admin_service_billing.go
admin_service_salepages.go
admin_service_pixels.go
admin_service_replays.go
admin_service_events.go
admin_service_audit.go
```

**Handler split** (`backend/internal/handler/`):

```
admin_handler.go             # AdminHandler struct, NewAdminHandler, query helpers (queryInt, queryPerPage, queryBool), paginatedResponse
admin_handler_customers.go
admin_handler_stats.go
admin_handler_billing.go
admin_handler_salepages.go
admin_handler_pixels.go
admin_handler_replays.go
admin_handler_events.go
admin_handler_audit.go
```

**Constraints**:
- `AdminRepository` interface in `interfaces.go` — **unchanged**.
- `AdminRepo`, `AdminService`, `AdminHandler` struct definitions — **unchanged**.
- Constructor signatures — **unchanged**.
- `internal/router/router.go` — **not touched**.
- Each resulting file ≤300 LOC after split.

### Execution Order

Two independent PRs, merged sequentially. Phase 1 goes first because it's foundational (test infra) and smaller.

1. **PR 1 — Phase 1 (shared mocks)**
2. **PR 2 — Phase 2 (admin split)** — based on `main` after PR 1 merges

Each PR must be independently reviewable and revertible.

## Success Criteria

Both phases, on every commit:

- [ ] `cd backend && go vet ./...` passes.
- [ ] `cd backend && go test -race ./...` passes with same test count as baseline.
- [ ] `cd backend && go build ./...` passes.
- [ ] `git grep -n "^package mocks"` returns only files under `internal/repository/mocks/` (Phase 1).
- [ ] `wc -l backend/internal/{repository/postgres/admin_repo*,service/admin_service*,handler/admin_handler*}.go` — no file > 500 LOC (Phase 2).
- [ ] `AdminRepository` interface in `interfaces.go` byte-identical to pre-refactor (Phase 2).
- [ ] `internal/router/router.go` byte-identical to pre-refactor.
- [ ] No new entries in `backend/go.mod`.

## Risks & Mitigations

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Missed mock reference during import update causes test compile failure | Medium | Run `go build ./...` after every file deletion; compile errors point directly at missed call sites |
| Method signature drift during copy-paste in Phase 2 | Low | Mechanical `git mv`-style moves using Edit tool; verify with `git diff --stat` shows only line relocations |
| Test helper (`testJWT`, etc.) accidentally deleted when pruning `testhelpers_test.go` | Low | Explicitly list retained symbols in spec; review diff against list before commit |
| Someone lands admin changes on main during Phase 2 work | Medium | Keep Phase 2 PR small and fast; rebase if conflicts appear |

## Open Questions

None at spec-writing time. All design decisions made: hand-written shared mocks (not generator), file-only split (not interface split), two-phase sequential PRs.
