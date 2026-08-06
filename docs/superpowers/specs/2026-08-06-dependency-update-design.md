# Dependency Update — Frontend, Backend, and Stripe SDK

**Date**: 2026-08-06
**Status**: Approved for planning
**Type**: Dependency maintenance — no new features, no speculative refactor

## Background & Motivation

The repository has been idle since 2026-05-29 (`bbf643e`). A scan on 2026-08-06 found:

- **Frontend**: 36 outdated npm packages, including 6 major bumps (`typescript` 5.9→7.0, `vite` 7→8, `react-router` 7→8, `eslint` 9→10, `lucide-react` 0.575→1.28, `@vitejs/plugin-react` 5→6).
- **Backend**: 10 outdated direct Go modules (all minor/patch), plus `stripe-go` stranded **4 major versions** behind (`v82.5.1`, latest stable `v86.2.0`). Stripe majors do not surface in `go list -m -u` because Go changes the module path per major.
- **Toolchain**: Go 1.25.6 → 1.26.5 stable available.
- **Config drift**: `frontend/Dockerfile` builds on `node:23-alpine` while CI runs Node 22. Node 23 is an odd-numbered release that has already reached end of life.
- **Stale memory**: recorded that no event-retention cleanup cron and no `/ready` endpoint exist. Both exist in code (`cmd/server/main.go:112`, `internal/router/router.go:108`).

Upstream findings verified through Context7 documentation lookups:

- **React Router 8** is a non-breaking upgrade from v7 when future flags are already adopted. It raises baselines to Node 22+, Vite 7+, React 19+, ESM-only — all of which this project already satisfies.
- **Vite 8** replaces the esbuild/Rollup pipeline with Rolldown. Breaking: `build.commonjsOptions` becomes a no-op, CJS interop semantics change (opt-out via `legacy.inconsistentCjsInterop`), the default browser target moves, and the `import.meta.hot.accept` resolution fallback is removed.
- **TypeScript 7** is the native Go compiler port (`typescript-go`). It does not support `baseUrl`, `outFile`, `target: ES5`, `moduleResolution: node10/classic`, or `esModuleInterop: false`.

## Goals

- Bring every low-risk dependency current on both frontend and backend.
- Adopt React Router 8 — the one major upgrade that is verifiably non-breaking here.
- Close the 4-major `stripe-go` gap so the billing system stops accumulating migration debt.
- Move the Go toolchain to 1.26.5.
- Fix the Node 23 / Node 22 mismatch and remove the `baseUrl` that would block a future TypeScript 7 adoption.
- Bring documentation and memory back in sync with the code.

## Non-Goals

- **No Vite 8.** Rolldown swaps the entire bundler; `vite.config.ts` carries five `manualChunks` groups that would all need re-validation. Deferred to a dedicated future spec.
- **No TypeScript 7.** It is an RC-stage native port with documented gaps. Deferred.
- **No other frontend major bumps.** ESLint 10, `lucide-react` 1.x, `@vitejs/plugin-react` 6, `@types/node` 26, `dotenv` 17, `globals` 17, `jsdom` 29, and `react-doctor` 0.9 all stay where they are. None is required by anything in this scope, and each would add an unattributable variable to the upgrade.
- **No best-practice refactor beyond what an upgrade forces.** Only APIs that upgrades deprecate or change get touched. Existing working code is left alone (CLAUDE.md §3, surgical changes).
- **No new features, no test-coverage expansion, no CI restructuring.**

## Design

Four independent pull requests, ordered by ascending risk. Each is revertible without affecting the others.

### PR1 — Config Drift Fixes

Pre-existing defects found during the scan, unrelated to any version bump. Shipped first so the rest builds on a correct baseline.

| Change | File | Rationale |
|---|---|---|
| `node:23-alpine` → `node:22-alpine` | `frontend/Dockerfile:1` | Match CI (Node 22); Node 23 is EOL |
| Remove `"baseUrl": "."` | `frontend/tsconfig.app.json` | `paths` already resolves as `./src/*`; unblocks future TypeScript 7 |
| Correct two stale entries | agent memory | Cleanup cron and `/ready` both exist |

**Verify**: `npm run build` succeeds; `docker build` of the frontend image succeeds; the `@/` import alias still resolves in both `tsc -b` and Vite.

### PR2 — Frontend Dependencies

- `react-router` 7.13.0 → 8.3.0
- 29 remaining minor/patch upgrades: `react`/`react-dom` 19.2.4→19.2.8, `@tanstack/react-query` 5.90→5.101, `tailwindcss` + `@tailwindcss/vite` 4.2→4.3, `@playwright/test` 1.59→1.62, `axios` 1.13→1.19, `react-hook-form` 7.71→7.84, `recharts` 3.7→3.10, `zod` 4.3→4.4, `zustand` 5.0.11→5.0.14, `vite` 7.3.1→7.3.6 (stays on 7), `vitest` 4.0→4.1, `typescript-eslint` 8.56→8.66, `@types/node` 24.10→24.13, and the Radix UI set.

Six packages have no minor/patch available at all — only a major — and are therefore left untouched: `typescript`, `lucide-react`, `dotenv`, `globals`, `jsdom`, `eslint-plugin-react-refresh`.

**Verify**: `npm run lint`, `npm run test`, `npm run build`, and all 19 Playwright specs in `frontend/e2e/tests/` pass.

### PR3 — Backend Dependencies + Go 1.26

- `jackc/pgx/v5` 5.8.0 → 5.10.0
- `go-chi/chi/v5` 5.2.5 → 5.3.1
- `aws-sdk-go-v2` 1.41.2 → 1.43.4, `aws-sdk-go-v2/service/s3` 1.96.1 → 1.106.5, `aws-sdk-go-v2/credentials` 1.19.10 → 1.19.34
- `go-playground/validator/v10` 10.30.1 → 10.30.3
- `golang.org/x/sync` 0.19.0 → 0.22.0, `golang.org/x/time` 0.14.0 → 0.15.0
- `google.golang.org/api` 0.269.0 → 0.292.0
- `caarlos0/env/v11` 11.4.0 → 11.4.1
- Go toolchain 1.25.6 → 1.26.5 in `backend/go.mod`, `backend/Dockerfile:1`, and `.github/workflows/ci.yml`

`stripe-go` is explicitly excluded from this PR.

**Highest-attention item**: pgx is the driver that talks to the Neon pooler, which has produced parameter-type bugs before (`#198`, `#187`). Integration tests must run against a real Postgres, not only the mocked unit suite.

**Verify**: `go vet ./...`, `go test -race -count=1 ./...`, `go test -race -tags=integration ./internal/repository/postgres/...`, `golangci-lint` (v2.12.1, as pinned in CI), and `go build ./cmd/server` all pass.

### PR4 — stripe-go v82 → v86

Isolated because it is the only change that can break payment collection.

**Surface area** — two non-test files:

| File | Stripe packages used |
|---|---|
| `internal/service/billing_service.go` | `stripe`, `checkout/session`, `billingportal/session`, `customer`, `subscription` |
| `internal/handler/billing_handler.go` | `stripe`, `webhook` |

**Method**: apply the official migration guide one major at a time — v82→v83→v84→v85→v86 — rather than jumping straight to v86. Each Stripe major corresponds to an API-version bump with its own struct and field changes; a single combined jump makes failures unattributable.

**Known fragile point**: `billing_service.go` carries the comment *"stripe-go v82, period dates are on SubscriptionItem, not Subscription"* — a scar from the previous major migration. Verify whether v83–v86 relocate these fields again.

**Verify**: 13 existing billing tests pass (8 in `billing_service_test.go`, 5 in `billing_handler_test.go`); `frontend/e2e/tests/billing.spec.ts` passes; webhook signature verification and the `checkout.session.completed` path are exercised against Stripe CLI test mode before merge.

### Documentation

Folded into the relevant PRs rather than shipped separately.

- `docs/user-guide-th.md` — reconcile against current behavior. The notification feature was removed in `#203`; Google OAuth moved from popup to redirect in `#202`/`#204`.
- Agent memory — correct the two stale entries listed in PR1.
- `CLAUDE.md` — untouched; it was deliberately rewritten in `#208`.

## Risk & Rollback

| Risk | Mitigation |
|---|---|
| pgx 5.10 breaks Neon pooler queries | Integration tests gate the merge; roll back pgx alone, keeping the rest of PR3 |
| Stripe migration breaks billing | PR4 merges last, only after PR1–3 are stable in production; revert is a single PR |
| React Router 8 regression | Full E2E suite covers routing; revert is a single dependency pin |
| A PR breaks CI ambiguously | Four separate PRs keep every failure attributable to one change set |

## Success Criteria

1. `npm outdated` shows no remaining minor/patch drift. Exactly 12 rows remain, all of them deliberately-deferred majors: `typescript`, `vite`, `eslint`, `@eslint/js`, `eslint-plugin-react-refresh`, `@vitejs/plugin-react`, `@types/node`, `dotenv`, `globals`, `jsdom`, `lucide-react`, `react-doctor`. `react-router` no longer appears.
2. `go list -m -u -f '{{if and (not .Indirect) .Update}}...'` returns empty for direct modules.
3. `stripe-go` is at v86.2.0 and the full billing test suite passes.
4. CI is green on Go 1.26 and Node 22.
5. `docs/user-guide-th.md` contains no reference to removed features.
