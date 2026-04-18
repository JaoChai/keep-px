# Backend Mock Dedup + Admin File Split — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate ~1,200 LOC of duplicated mock types and split 3 admin god-files (2,353 LOC total) into sub-domain files, without changing any behavior, public API, or interface.

**Architecture:** Two-phase pure mechanical refactor. Phase 1 creates a shared `repository/mocks` package and deletes mock duplication from test files. Phase 2 splits `admin_repo.go`/`admin_service.go`/`admin_handler.go` into sub-domain files (customers, stats, billing, salepages, pixels, replays, events, audit). Each phase ships as its own PR.

**Tech Stack:** Go 1.25, `github.com/stretchr/testify/mock`, `github.com/jackc/pgx/v5`, `chi/v5`. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-04-18-backend-test-and-admin-refactor-design.md`

**Starting branch:** `docs/refactor-backend-mocks-and-admin-spec` (where this plan lives). Phase 1 and Phase 2 branch off `main`, not off this branch.

---

## TDD Note

This refactor doesn't follow the classic "write failing test → implement" loop because we're not adding behavior — we're moving code. The discipline here is:

1. **Baseline**: before touching code, record the test pass count and `go vet` clean state.
2. **Move**: make each mechanical change.
3. **Verify**: `go vet ./... && go build ./... && go test -race ./...` must stay green. Same test count as baseline.
4. **Commit** frequently — every logically complete chunk.

If a step breaks the build, revert that step, diagnose, and retry. Never accumulate broken state across commits.

---

## Phase 1 — Shared Mocks Package

### Setup

- [ ] **Step 0.1: Create branch and capture baseline**

```bash
cd /home/jaochai/code/keep-px
git checkout main
git pull
git checkout -b refactor/shared-mocks-package
cd backend
go vet ./... && go test -race ./... 2>&1 | tee /tmp/phase1-baseline.txt
grep -c "^--- PASS\|^--- FAIL" /tmp/phase1-baseline.txt
```

Expected: `go vet` clean, all tests PASS. Record the final test count from the last line.

### Task 1: Create the Shared Mocks Package

**Files to create:**
- `backend/internal/repository/mocks/customer.go`
- `backend/internal/repository/mocks/pixel.go`
- `backend/internal/repository/mocks/event.go`
- `backend/internal/repository/mocks/replay_session.go`
- `backend/internal/repository/mocks/sale_page.go`
- `backend/internal/repository/mocks/refresh_token.go`
- `backend/internal/repository/mocks/purchase.go`
- `backend/internal/repository/mocks/replay_credit.go`
- `backend/internal/repository/mocks/subscription.go`
- `backend/internal/repository/mocks/event_usage.go`
- `backend/internal/repository/mocks/webhook_event.go`
- `backend/internal/repository/mocks/admin.go`

**Source of truth for mock bodies:** `backend/internal/service/mocks_test.go`. Every mock type, with all its methods, already exists there — copy them verbatim; only change the package header.

- [ ] **Step 1.1: Create `mocks/customer.go` (template — copy pattern for all 12 files)**

File: `backend/internal/repository/mocks/customer.go`

```go
package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)

// MockCustomerRepo is a testify mock for repository.CustomerRepository.
type MockCustomerRepo struct{ mock.Mock }

func (m *MockCustomerRepo) Create(ctx context.Context, c *domain.Customer) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}
func (m *MockCustomerRepo) GetByID(ctx context.Context, id string) (*domain.Customer, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Customer), args.Error(1)
}
func (m *MockCustomerRepo) GetByEmail(ctx context.Context, email string) (*domain.Customer, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Customer), args.Error(1)
}
func (m *MockCustomerRepo) GetByGoogleID(ctx context.Context, googleID string) (*domain.Customer, error) {
	args := m.Called(ctx, googleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Customer), args.Error(1)
}
func (m *MockCustomerRepo) GetByAPIKey(ctx context.Context, key string) (*domain.Customer, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Customer), args.Error(1)
}
func (m *MockCustomerRepo) GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*domain.Customer, error) {
	args := m.Called(ctx, stripeCustomerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Customer), args.Error(1)
}
func (m *MockCustomerRepo) Update(ctx context.Context, c *domain.Customer) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}
func (m *MockCustomerRepo) UpdateStripeCustomerID(ctx context.Context, customerID string, stripeCustomerID string) error {
	args := m.Called(ctx, customerID, stripeCustomerID)
	return args.Error(0)
}
func (m *MockCustomerRepo) UpdatePlan(ctx context.Context, customerID string, plan string) error {
	args := m.Called(ctx, customerID, plan)
	return args.Error(0)
}
func (m *MockCustomerRepo) UpdateRetentionDays(ctx context.Context, customerID string, days int) error {
	args := m.Called(ctx, customerID, days)
	return args.Error(0)
}
func (m *MockCustomerRepo) RegenerateAPIKey(ctx context.Context, customerID, newKey string) (*domain.Customer, error) {
	args := m.Called(ctx, customerID, newKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Customer), args.Error(1)
}

// Keep the blank `time` import only if later methods need it; otherwise drop the import.
var _ = time.Time{}
```

Note: `time` may be unused in this specific file — if `go vet` complains, remove the `"time"` import and the `var _ = time.Time{}` line. Keep them only where a method signature mentions `time.Time` (e.g. `refresh_token.go`, `event.go`, `purchase.go`).

- [ ] **Step 1.2: Verify the file compiles**

```bash
cd backend && go build ./internal/repository/mocks/
```

Expected: no output (success). If it fails, check imports.

- [ ] **Step 1.3: Create the remaining 11 mock files by copying from `service/mocks_test.go`**

For each target file below, find the corresponding `type MockXxxRepo struct{ mock.Mock }` declaration and all its method receivers in `backend/internal/service/mocks_test.go`, then copy the block (type + all methods) into the new file with the package header:

```go
package mocks

import (
	"context"
	// add "time" only if any method signature uses time.Time

	"github.com/stretchr/testify/mock"

	"github.com/jaochai/pixlinks/backend/internal/domain"
)
```

Target files and their source blocks in `mocks_test.go`:

| New file | Mock type(s) to copy |
|----------|----------------------|
| `mocks/purchase.go` | `MockPurchaseRepo` + 5 methods — needs `time` import |
| `mocks/replay_credit.go` | `MockReplayCreditRepo` + 6 methods |
| `mocks/subscription.go` | `MockSubscriptionRepo` + 7 methods |
| `mocks/event_usage.go` | `MockEventUsageRepo` + 4 methods |
| `mocks/sale_page.go` | `MockSalePageRepo` + 8 methods |
| `mocks/webhook_event.go` | `MockWebhookEventRepo` + 2 methods |
| `mocks/refresh_token.go` | `MockRefreshTokenRepo` + 4 methods — needs `time` import |
| `mocks/pixel.go` | `MockPixelRepo` + 7 methods |
| `mocks/event.go` | `MockEventRepo` + 14 methods — needs `time` import |
| `mocks/replay_session.go` | `MockReplaySessionRepo` + ~10 methods |
| `mocks/admin.go` | `MockAdminRepo` + ~25 methods — needs `time` import |

**Mechanical rule:** only change the package declaration (to `package mocks`) and imports; do NOT edit method bodies or signatures.

- [ ] **Step 1.4: Verify the package compiles with all 12 files**

```bash
cd backend && go build ./internal/repository/mocks/
```

Expected: success. If a type assertion fails or import is wrong, inspect the corresponding block in `mocks_test.go` and copy again.

- [ ] **Step 1.5: Commit the new package**

```bash
cd /home/jaochai/code/keep-px
git add backend/internal/repository/mocks/
git commit -m "$(cat <<'EOF'
refactor: extract test mocks to shared internal/repository/mocks package

Adds a non-test Go package containing public MockXxxRepo types for
every repository interface. Subsequent commits replace the duplicated
definitions in service/mocks_test.go and handler/testhelpers_test.go.

No tests changed in this commit.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 2: Switch `service` tests to shared mocks

**Files to modify:**
- `backend/internal/service/auth_service_test.go`
- `backend/internal/service/billing_service_test.go`
- `backend/internal/service/cleanup_service_test.go`
- `backend/internal/service/event_service_test.go`
- `backend/internal/service/pixel_service_test.go`
- `backend/internal/service/quota_service_test.go`
- `backend/internal/service/replay_service_test.go`
- `backend/internal/service/sale_page_service_test.go`
- `backend/internal/service/admin_service_test.go`

**Files to delete:**
- `backend/internal/service/mocks_test.go`

- [ ] **Step 2.1: Add mocks import to each service test file**

For each of the 9 test files, add this import to the existing import block:

```go
"github.com/jaochai/pixlinks/backend/internal/repository/mocks"
```

- [ ] **Step 2.2: Rewrite mock type references**

In every service test file, replace bare mock references with the package-qualified form. Use `sed` for a mechanical pass, then review each file:

```bash
cd /home/jaochai/code/keep-px/backend/internal/service
for f in auth_service_test.go billing_service_test.go cleanup_service_test.go \
         event_service_test.go pixel_service_test.go quota_service_test.go \
         replay_service_test.go sale_page_service_test.go admin_service_test.go; do
  sed -i \
    -e 's/\bMockCustomerRepo\b/mocks.MockCustomerRepo/g' \
    -e 's/\bMockRefreshTokenRepo\b/mocks.MockRefreshTokenRepo/g' \
    -e 's/\bMockPixelRepo\b/mocks.MockPixelRepo/g' \
    -e 's/\bMockEventRepo\b/mocks.MockEventRepo/g' \
    -e 's/\bMockReplaySessionRepo\b/mocks.MockReplaySessionRepo/g' \
    -e 's/\bMockSalePageRepo\b/mocks.MockSalePageRepo/g' \
    -e 's/\bMockPurchaseRepo\b/mocks.MockPurchaseRepo/g' \
    -e 's/\bMockReplayCreditRepo\b/mocks.MockReplayCreditRepo/g' \
    -e 's/\bMockSubscriptionRepo\b/mocks.MockSubscriptionRepo/g' \
    -e 's/\bMockEventUsageRepo\b/mocks.MockEventUsageRepo/g' \
    -e 's/\bMockWebhookEventRepo\b/mocks.MockWebhookEventRepo/g' \
    -e 's/\bMockAdminRepo\b/mocks.MockAdminRepo/g' \
    "$f"
done
```

- [ ] **Step 2.3: Delete the old local mocks file**

```bash
rm backend/internal/service/mocks_test.go
```

- [ ] **Step 2.4: Run `goimports` or fix imports manually**

```bash
cd backend
gofmt -w ./internal/service/
```

If any test file ends up with an unused import, remove it manually (`go vet` will flag it).

- [ ] **Step 2.5: Verify service tests compile and pass**

```bash
cd backend && go vet ./internal/service/... && go test -race ./internal/service/...
```

Expected: all tests PASS, test count equal to baseline for `./internal/service/...`.

- [ ] **Step 2.6: Commit**

```bash
git add backend/internal/service/
git commit -m "$(cat <<'EOF'
refactor: service tests use shared mocks package

Replaces 12 locally-defined Mock*Repo types (in mocks_test.go) with
imports from internal/repository/mocks. Deletes the now-empty
mocks_test.go file.

No test logic or assertions changed.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 3: Switch `handler` tests to shared mocks, trim `testhelpers_test.go`

**Files to modify:**
- `backend/internal/handler/testhelpers_test.go` — keep only `testJWT`, `doRequest`, `testConfig`, `testLogger`, and the `test*` constants; delete all 12 `Mock*Repo` types and their methods.
- `backend/internal/handler/admin_handler_test.go`
- `backend/internal/handler/auth_handler_test.go`
- `backend/internal/handler/billing_handler_test.go`
- `backend/internal/handler/event_handler_test.go`
- `backend/internal/handler/event_ingest_handler_test.go`
- `backend/internal/handler/pixel_handler_test.go`
- `backend/internal/handler/replay_handler_test.go`
- `backend/internal/handler/sale_page_handler_test.go`
- `backend/internal/handler/analytics_handler_test.go` (if it references mocks)

- [ ] **Step 3.1: Trim `testhelpers_test.go`**

Open `backend/internal/handler/testhelpers_test.go`. Keep **only** the top portion up to (and including) `testLogger`. Delete every line from the first `type MockXxxRepo struct{ mock.Mock }` to end-of-file.

After the trim, the file should contain (verify with `grep -E '^(type|func|const)' testhelpers_test.go`):

- `const (` block with test constants
- `func testJWT(...)`
- `func doRequest(...)`
- `func testConfig() *config.Config`
- `func testLogger() *slog.Logger`

Remove these imports if they are no longer referenced after the trim (check with `goimports`):
- `github.com/stretchr/testify/mock` (only used by mock types)
- `github.com/jaochai/pixlinks/backend/internal/domain` (only used by mock methods)

- [ ] **Step 3.2: Add mocks import to each handler test file**

For each handler test file that references mocks, add the import:

```go
"github.com/jaochai/pixlinks/backend/internal/repository/mocks"
```

- [ ] **Step 3.3: Rewrite mock type references in handler tests**

```bash
cd /home/jaochai/code/keep-px/backend/internal/handler
for f in admin_handler_test.go auth_handler_test.go billing_handler_test.go \
         event_handler_test.go event_ingest_handler_test.go pixel_handler_test.go \
         replay_handler_test.go sale_page_handler_test.go analytics_handler_test.go; do
  [ -f "$f" ] && sed -i \
    -e 's/\bMockCustomerRepo\b/mocks.MockCustomerRepo/g' \
    -e 's/\bMockRefreshTokenRepo\b/mocks.MockRefreshTokenRepo/g' \
    -e 's/\bMockPixelRepo\b/mocks.MockPixelRepo/g' \
    -e 's/\bMockEventRepo\b/mocks.MockEventRepo/g' \
    -e 's/\bMockReplaySessionRepo\b/mocks.MockReplaySessionRepo/g' \
    -e 's/\bMockSalePageRepo\b/mocks.MockSalePageRepo/g' \
    -e 's/\bMockPurchaseRepo\b/mocks.MockPurchaseRepo/g' \
    -e 's/\bMockReplayCreditRepo\b/mocks.MockReplayCreditRepo/g' \
    -e 's/\bMockSubscriptionRepo\b/mocks.MockSubscriptionRepo/g' \
    -e 's/\bMockEventUsageRepo\b/mocks.MockEventUsageRepo/g' \
    -e 's/\bMockWebhookEventRepo\b/mocks.MockWebhookEventRepo/g' \
    -e 's/\bMockAdminRepo\b/mocks.MockAdminRepo/g' \
    "$f"
done
gofmt -w .
```

- [ ] **Step 3.4: Verify handler tests compile and pass**

```bash
cd backend && go vet ./internal/handler/... && go test -race ./internal/handler/...
```

Expected: all tests PASS.

- [ ] **Step 3.5: Commit**

```bash
git add backend/internal/handler/
git commit -m "$(cat <<'EOF'
refactor: handler tests use shared mocks package

Trims testhelpers_test.go down to JWT/request helpers only.
Replaces 12 duplicated Mock*Repo type definitions with imports
from internal/repository/mocks.

No test logic or assertions changed.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 4: Final Phase 1 Verification and PR

- [ ] **Step 4.1: Full backend verification**

```bash
cd backend
go vet ./...
go build ./...
go test -race ./... 2>&1 | tee /tmp/phase1-final.txt
grep -c "^--- PASS\|^--- FAIL" /tmp/phase1-final.txt
```

Expected:
- `go vet` clean
- `go build` success
- `go test -race` all PASS
- Final PASS/FAIL count equals baseline from Step 0.1 (no tests added, none removed)

- [ ] **Step 4.2: Confirm `go.mod` / `go.sum` unchanged**

```bash
git status backend/go.mod backend/go.sum
```

Expected: both clean (no modifications).

- [ ] **Step 4.3: Confirm no mock types leaked outside the new package**

```bash
cd /home/jaochai/code/keep-px
grep -rn "^type Mock[A-Z][a-zA-Z]*Repo" backend/internal/ --include="*.go"
```

Expected: all matches are under `backend/internal/repository/mocks/`.

- [ ] **Step 4.4: Push branch and open PR**

```bash
git push -u origin refactor/shared-mocks-package
gh pr create --title "refactor: shared mocks package (Phase 1 of admin refactor)" --body "$(cat <<'EOF'
## Summary
- Consolidates 12 duplicated Mock*Repo types from service/mocks_test.go and handler/testhelpers_test.go into a new non-test package internal/repository/mocks
- Removes ~600 LOC of duplication; CLAUDE.md gotcha "must update mocks in both files when changing interfaces" no longer applies

## Test plan
- [x] go vet ./... passes
- [x] go test -race ./... passes with same test count as baseline
- [x] No changes to go.mod / go.sum
- [x] No changes to production code (only test and new mocks package)

Spec: docs/superpowers/specs/2026-04-18-backend-test-and-admin-refactor-design.md
EOF
)"
```

**Do not start Phase 2 until this PR merges into `main`.**

---

## Phase 2 — Admin File Split

### Setup (after Phase 1 merged)

- [ ] **Step 5.0: Branch from updated main**

```bash
cd /home/jaochai/code/keep-px
git checkout main
git pull
git checkout -b refactor/admin-file-split
cd backend
go vet ./... && go test -race ./... 2>&1 | tee /tmp/phase2-baseline.txt
grep -c "^--- PASS\|^--- FAIL" /tmp/phase2-baseline.txt
# Capture current byte-identical checksums for invariant files
md5sum internal/repository/interfaces.go internal/router/router.go > /tmp/phase2-invariants.md5
cat /tmp/phase2-invariants.md5
```

Record the test PASS count and the two MD5 hashes. These must not change by end of phase.

### Task 5: Split `admin_repo.go` by Sub-Domain

**Source:** `backend/internal/repository/postgres/admin_repo.go` (1,275 LOC)

**New files** (all in `backend/internal/repository/postgres/`):

| File | Keeps |
|------|-------|
| `admin_repo.go` | `AdminRepo` struct + `NewAdminRepo` + imports (lines 1–27 of original) — **~30 LOC after trim** |
| `admin_repo_customers.go` | `ListCustomers` (28–84), `scanCustomerFromRows` helper (85–102), `GetCustomerDetail` (103–236), `SuspendCustomer` (237–249), `ActivateCustomer` (250–262) |
| `admin_repo_stats.go` | `GetPlatformStats` (263–370), `GetRevenueChart` (371–397), `GetGrowthChart` (398–429) |
| `admin_repo_billing.go` | `ListAllPurchases` (430–479), `ListAllSubscriptions` (480–531), `ListCreditGrants` (532–570) |
| `admin_repo_salepages.go` | `ListAllSalePages` (571–638), `GetSalePageAdminDetail` (639–704), `SetSalePagePublished` (705–717), `DeleteSalePageByAdmin` (718–730) |
| `admin_repo_pixels.go` | `ListAllPixels` (731–798), `GetPixelAdminDetail` (799–863), `SetPixelActive` (864–878) |
| `admin_repo_replays.go` | `ListAllReplaySessions` (879–943), `GetReplaySessionAdminDetail` (944–1020) |
| `admin_repo_events.go` | `ListAllEvents` (1021–1087), `GetEventStats` (1088–1194) |
| `admin_repo_audit.go` | `CreateAuditLog` (1195–1203), `ListAuditLogs` (1204–end) |

Line numbers refer to the **original** file before any edits. After the first split, the remaining file shrinks and line numbers no longer apply — always use the original file as the source of truth during this task.

- [ ] **Step 5.1: Snapshot the original file before splitting**

```bash
cd /home/jaochai/code/keep-px
cp backend/internal/repository/postgres/admin_repo.go /tmp/admin_repo_original.go
```

The `/tmp/admin_repo_original.go` copy is the authoritative source for every subsequent extraction.

- [ ] **Step 5.2: Create `admin_repo_customers.go`**

File: `backend/internal/repository/postgres/admin_repo_customers.go`

Structure:

```go
package postgres

import (
	// Copy only the imports used by the moved functions. Typical set:
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/jaochai/pixlinks/backend/internal/domain"
	"github.com/jaochai/pixlinks/backend/internal/repository"
)

// Copy verbatim from /tmp/admin_repo_original.go:
// - func (r *AdminRepo) ListCustomers(...)        (lines 28–84)
// - func scanCustomerFromRows(...)                (lines 85–102)
// - func (r *AdminRepo) GetCustomerDetail(...)    (lines 103–236)
// - func (r *AdminRepo) SuspendCustomer(...)      (lines 237–249)
// - func (r *AdminRepo) ActivateCustomer(...)     (lines 250–262)
```

Copy the **exact function bodies** from `/tmp/admin_repo_original.go`. Let `goimports` adjust imports at the end; if in doubt, copy all of the original file's imports into the new file and remove unused ones after build.

- [ ] **Step 5.3: Remove the moved functions from `admin_repo.go`**

Open `backend/internal/repository/postgres/admin_repo.go` and delete lines 28–262 (the functions moved in 5.2). The file should now contain only the imports, struct, `NewAdminRepo`, and the remaining (not-yet-moved) functions.

- [ ] **Step 5.4: Build incrementally after each split to catch issues early**

```bash
cd backend && go build ./internal/repository/postgres/
```

If it fails because the original file references a helper that's now in the new file, that's expected — keep moving functions. If it fails because an import is now unused, run `goimports -w internal/repository/postgres/`.

- [ ] **Step 5.5: Repeat Steps 5.2–5.4 for the remaining 7 sub-domain files**

Apply the same pattern for each remaining file in the table at the top of Task 5:

1. Create `admin_repo_<domain>.go` with `package postgres` + needed imports.
2. Copy the listed functions verbatim from `/tmp/admin_repo_original.go`.
3. Delete the same lines from `admin_repo.go`.
4. `go build ./internal/repository/postgres/` — must pass after each iteration.

Do them in this order (shortest last so `admin_repo.go` shrinks incrementally):
`stats → billing → salepages → pixels → replays → events → audit`.

- [ ] **Step 5.6: Final verification of repo split**

```bash
cd backend
go vet ./internal/repository/postgres/
wc -l internal/repository/postgres/admin_repo*.go
go test -race ./...
```

Expected:
- `go vet` clean.
- No file `admin_repo*.go` exceeds 300 LOC; `admin_repo.go` itself should be ~30–50 LOC.
- All tests PASS (pass count matches baseline).

- [ ] **Step 5.7: Commit repo split**

```bash
git add backend/internal/repository/postgres/
git commit -m "$(cat <<'EOF'
refactor: split admin_repo.go into sub-domain files

Splits 1,275-line god-file into per-sub-domain files following the
F1-F5 boundaries already documented in AdminRepository interface:
customers, stats, billing, salepages, pixels, replays, events, audit.

Core struct, constructor, and interface unchanged. Pure file
relocation — no behavior change.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 6: Split `admin_service.go` by Sub-Domain

**Source:** `backend/internal/service/admin_service.go` (490 LOC)

**New files** (all in `backend/internal/service/`):

| File | Keeps |
|------|-------|
| `admin_service.go` | `const platformStatsCacheTTL`, `type platformStatsCache` + its methods (lines 17–45), `type AdminService` (47–55), `NewAdminService` (57–73), `logAudit` (293–311), `normalizePagination` (312–322) |
| `admin_service_customers.go` | `ListCustomers` (75–96), `GetCustomerDetail` (97–107), `ChangePlan` (108–127), `SuspendCustomer` (128–141), `ActivateCustomer` (142–152), `type GrantCreditsInput` (153–160), `GrantCredits` (161–233) |
| `admin_service_stats.go` | `GetPlatformOverview` (234–245), `GetRevenueChart` (246–252), `GetGrowthChart` (253–259) |
| `admin_service_billing.go` | `ListAllPurchases` (260–270), `ListAllSubscriptions` (271–281), `ListCreditGrants` (282–292) |
| `admin_service_salepages.go` | `ListAllSalePages` (324–328), `GetSalePageDetail` (329–339), `DisableSalePage` (340–351), `EnableSalePage` (352–363), `DeleteSalePage` (364–383) |
| `admin_service_pixels.go` | `ListAllPixels` (384–388), `GetPixelDetail` (389–399), `DisablePixel` (400–411), `EnablePixel` (412–425) |
| `admin_service_replays.go` | `ListAllReplaySessions` (426–430), `GetReplayDetail` (431–441), `CancelReplay` (442–456) |
| `admin_service_events.go` | `ListAllEvents` (457–461), `GetEventStats` (462–483) |
| `admin_service_audit.go` | `ListAuditLogs` (484–end) |

- [ ] **Step 6.1: Snapshot original**

```bash
cp backend/internal/service/admin_service.go /tmp/admin_service_original.go
```

- [ ] **Step 6.2: Apply the same split pattern as Task 5**

For each file in the table above:
1. Create new file with `package service` header and required imports.
2. Copy listed functions/types **verbatim** from `/tmp/admin_service_original.go`.
3. Delete the same content from `admin_service.go`.
4. `go build ./internal/service/` after each file.

Notes:
- `GrantCreditsInput` is a type, not a function — move it together with `GrantCredits` into `admin_service_customers.go`.
- `logAudit` and `normalizePagination` are helpers used by many sub-domain methods — keep them in `admin_service.go` (the core file).
- `platformStatsCache` is used only by `GetPlatformOverview`, but move it as well into `admin_service.go` because the cache instance is a field on `AdminService` and its definition belongs with the struct.

- [ ] **Step 6.3: Verify service split**

```bash
cd backend
go vet ./internal/service/
wc -l internal/service/admin_service*.go
go test -race ./internal/service/
```

Expected: clean vet, no file > 300 LOC except possibly `admin_service_customers.go` (~120 LOC due to `GrantCredits` size), all tests PASS.

- [ ] **Step 6.4: Commit service split**

```bash
git add backend/internal/service/
git commit -m "$(cat <<'EOF'
refactor: split admin_service.go into sub-domain files

Same per-sub-domain split pattern as admin_repo.go.
AdminService struct, constructor, platformStatsCache, and helper
methods (logAudit, normalizePagination) stay in core admin_service.go.

No behavior change.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 7: Split `admin_handler.go` by Sub-Domain

**Source:** `backend/internal/handler/admin_handler.go` (588 LOC)

**New files** (all in `backend/internal/handler/`):

| File | Keeps |
|------|-------|
| `admin_handler.go` | `type AdminHandler` (18–23), `NewAdminHandler` (24–31), `queryInt` (289–301), `queryPerPage` (302–309), `queryBool` (310–318), `paginatedResponse` (319–334) |
| `admin_handler_customers.go` | `ListCustomers` (32–68), `GetCustomerDetail` (69–87), `ChangePlan` (88–118), `SuspendCustomer` (119–137), `ActivateCustomer` (138–153), `GrantCredits` (154–180) |
| `admin_handler_stats.go` | `GetPlatformOverview` (181–190), `GetRevenueChart` (191–202), `GetGrowthChart` (203–214) |
| `admin_handler_billing.go` | `ListPurchases` (215–239), `ListSubscriptions` (240–264), `ListCreditGrants` (265–288) |
| `admin_handler_salepages.go` | `ListSalePages` (335–350), `GetSalePageDetail` (351–366), `DisableSalePage` (367–382), `EnableSalePage` (383–398), `DeleteSalePage` (399–416) |
| `admin_handler_pixels.go` | `ListPixels` (417–432), `GetPixelDetail` (433–448), `DisablePixel` (449–464), `EnablePixel` (465–482) |
| `admin_handler_replays.go` | `ListReplays` (483–497), `GetReplayDetail` (498–513), `CancelReplay` (514–531) |
| `admin_handler_events.go` | `ListEvents` (532–547), `GetEventStats` (548–561) |
| `admin_handler_audit.go` | `ListAuditLog` (562–end) |

- [ ] **Step 7.1: Snapshot original**

```bash
cp backend/internal/handler/admin_handler.go /tmp/admin_handler_original.go
```

- [ ] **Step 7.2: Apply the same split pattern as Tasks 5 and 6**

For each file:
1. Create new file with `package handler` header + needed imports.
2. Copy listed functions verbatim.
3. Delete from `admin_handler.go`.
4. `go build ./internal/handler/` after each.

Notes:
- The helper functions `queryInt`, `queryPerPage`, `queryBool` are package-level (not method receivers) — they live in `admin_handler.go` and are used by other sub-domain files without needing re-export.
- `paginatedResponse` is a method on `AdminHandler` — stays in `admin_handler.go`.

- [ ] **Step 7.3: Verify handler split**

```bash
cd backend
go vet ./internal/handler/
wc -l internal/handler/admin_handler*.go
go test -race ./internal/handler/
```

Expected: clean vet, every file ≤300 LOC, all tests PASS.

- [ ] **Step 7.4: Commit handler split**

```bash
git add backend/internal/handler/
git commit -m "$(cat <<'EOF'
refactor: split admin_handler.go into sub-domain files

Final layer of the admin god-file split. Query-param helpers and
paginatedResponse stay in core admin_handler.go so all sub-domain
handler files share them package-locally.

No behavior change.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 8: Final Phase 2 Verification and PR

- [ ] **Step 8.1: Full backend verification**

```bash
cd backend
go vet ./...
go build ./...
go test -race ./... 2>&1 | tee /tmp/phase2-final.txt
grep -c "^--- PASS\|^--- FAIL" /tmp/phase2-final.txt
```

Expected:
- `go vet` clean.
- `go build` success.
- All tests PASS with the **same test count** as the Phase 2 baseline from Step 5.0.

- [ ] **Step 8.2: Confirm invariant files are byte-identical**

```bash
cd backend
md5sum internal/repository/interfaces.go internal/router/router.go
diff /tmp/phase2-invariants.md5 <(md5sum internal/repository/interfaces.go internal/router/router.go)
```

Expected: `diff` produces no output (MD5s unchanged).

- [ ] **Step 8.3: Confirm no admin file exceeds 500 LOC**

```bash
wc -l internal/repository/postgres/admin_repo*.go internal/service/admin_service*.go internal/handler/admin_handler*.go | awk '$1 > 500 && $2 != "total"'
```

Expected: no output (every file ≤500 LOC).

- [ ] **Step 8.4: Confirm `go.mod` / `go.sum` unchanged**

```bash
git status backend/go.mod backend/go.sum
```

Expected: both clean.

- [ ] **Step 8.5: Push branch and open PR**

```bash
git push -u origin refactor/admin-file-split
gh pr create --title "refactor: split admin_repo/service/handler by sub-domain (Phase 2)" --body "$(cat <<'EOF'
## Summary
- Splits three admin god-files into per-sub-domain files (customers, stats, billing, salepages, pixels, replays, events, audit)
- `admin_repo.go` 1,275 → ~30 LOC core + 8 sub-files; `admin_service.go` 490 → core + 8 sub-files; `admin_handler.go` 588 → core + 8 sub-files
- AdminRepository interface and router wiring byte-identical to pre-refactor

## Test plan
- [x] go vet ./... passes
- [x] go test -race ./... passes with same test count as baseline
- [x] interfaces.go and router/router.go MD5 unchanged
- [x] No file exceeds 500 LOC
- [x] No go.mod / go.sum changes

Spec: docs/superpowers/specs/2026-04-18-backend-test-and-admin-refactor-design.md
Phase 1: (previous PR link)
EOF
)"
```

---

## Self-Review Checklist (for plan author)

- [x] Every spec section has at least one task: shared mocks package (Task 1), mock removals (Tasks 2–3), admin repo split (Task 5), admin service split (Task 6), admin handler split (Task 7), and each Task N has its own verification step.
- [x] No TBD / TODO / "similar to" / "add appropriate error handling" placeholders.
- [x] Method signatures referenced in later steps (e.g. `GetPlatformOverview`, `queryInt`) are defined verbatim in the cited source files.
- [x] Line-range references are for the **original** file, captured in `/tmp/admin_*_original.go` snapshots before the destructive edits begin.
- [x] Every task ends with a commit, every phase ends with a PR, and every PR has an explicit test-plan checklist derived from spec success criteria.
