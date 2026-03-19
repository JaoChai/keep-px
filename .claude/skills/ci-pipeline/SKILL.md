---
name: ci-pipeline
description: GitHub Actions CI pipeline structure, deploy verification, post-deploy E2E patterns, and common CI failure debugging for Keep-PX
---

# CI Pipeline

## When to Activate

Activate this skill when the user says:
- "CI failing" / "Pipeline broken" / "Build failed on GitHub"
- "Deploy verification" / "Smoke test failed"
- "E2E fails in CI but passes locally"
- "Add CI step" / "Modify GitHub Actions"
- "Post-deploy test" / "Health check failed"

## Architecture

The CI pipeline has 6 jobs in a dependency chain:

```
changes (path filter)
  ├── backend  (if backend/** changed)
  ├── frontend (if frontend/** changed)
  └── e2e      (if either changed, uses containerized PostgreSQL)
        ↓
    ci-gate (aggregates results, blocks deploy on failure)
        ↓
    deploy-verify (push to main only, waits for Railway deploy)
        ↓
    post-deploy-e2e (smoke tests against production)
```

**File:** `.github/workflows/ci.yml`

## Job Details

### 1. `changes` — Path-based filtering
Uses `dorny/paths-filter@v3` to detect which packages changed. Jobs only run if their paths are affected.

```yaml
outputs:
  backend: ${{ steps.filter.outputs.backend }}   # backend/**
  frontend: ${{ steps.filter.outputs.frontend }} # frontend/**
```

### 2. `backend` — Go quality gates
Runs if `backend/**` changed:
1. `golangci-lint` (golangci/golangci-lint-action@v7)
2. `go vet ./...`
3. `go test -race -count=1 ./...`
4. `go build -o /dev/null ./cmd/server`
5. `gosec` (continue-on-error) — security scanner
6. `govulncheck` (continue-on-error) — vulnerability scanner

### 3. `frontend` — Node quality gates
Runs if `frontend/**` changed:
1. `npm ci`
2. `npm run lint`
3. `npm run test`
4. `npm run build`
5. `npm audit --audit-level=high` (continue-on-error)

### 4. `e2e` — Integration tests
Runs if **either** package changed. Spins up PostgreSQL service container:
1. Apply all `backend/db/migrations/*.up.sql` to test DB
2. `npm ci` + `npx playwright install --with-deps chromium`
3. `npx playwright test`
4. On failure: upload `playwright-report/` + `test-results/` as artifacts

**Database config:**
```yaml
services:
  postgres:
    image: postgres:17-alpine
    env:
      POSTGRES_USER: testuser
      POSTGRES_PASSWORD: testpass
      POSTGRES_DB: keeppx_test
env:
  DATABASE_URL: postgres://testuser:testpass@localhost:5432/keeppx_test?sslmode=disable
  JWT_SECRET: e2e-test-secret-key-for-ci
```

### 5. `ci-gate` — Status aggregation
Runs `if: always()`. Checks all job results — only allows "success" or "skipped". This is the branch protection check.

### 6. `deploy-verify` — Post-deploy health check
Only runs on push to main + ci-gate passed:
1. `sleep 60` (wait for Railway to pick up the push)
2. Backend health: retry up to 10x with 30s intervals at `$BACKEND_PROD_URL/health`
3. Frontend health: retry up to 5x with 30s intervals at `$FRONTEND_PROD_URL`

### 7. `post-deploy-e2e` — Production smoke tests
Only runs after deploy-verify succeeds:
1. Re-verify backend health (5 retries, 10s intervals) — handles cold start timing
2. Run `npx playwright test --grep @smoke` against production
3. Auth setup uses `fetchWithRetry` with exponential backoff (3→6→12→24→48s)

**Environment:**
```yaml
env:
  E2E_BASE_URL: ${{ vars.FRONTEND_PROD_URL }}
  E2E_ACCESS_TOKEN: ${{ secrets.E2E_ACCESS_TOKEN }}
  E2E_REFRESH_TOKEN: ${{ secrets.E2E_REFRESH_TOKEN }}
```

## Debugging CI Failures

### Decision Tree

```
CI job failed
├── Which job?
│   ├── backend → Read golangci-lint / go test output
│   ├── frontend → Read npm run lint / npm run build output
│   ├── e2e → Check Playwright report artifact
│   ├── ci-gate → One of the above jobs failed
│   ├── deploy-verify → Backend/frontend not responding after deploy
│   └── post-deploy-e2e → Auth setup or smoke test failed
│
├── deploy-verify failed?
│   ├── HTTP 000 → Backend service crashed or not deployed yet
│   ├── HTTP 502 → Backend starting but not ready (Neon cold start)
│   └── HTTP 504 → Railway internal networking timeout
│
└── post-deploy-e2e failed?
    ├── Auth setup failed (502) → Backend cold start, retry handles it
    ├── Auth setup failed (401) → Credentials wrong or password auth disabled
    ├── Smoke test timeout → Page not loading, check frontend deploy
    └── Smoke test assertion → UI changed, update test expectations
```

### Common CI Failure Patterns

| Pattern | Root Cause | Fix |
|---------|-----------|-----|
| `go test` fails with `-race` flag | Data race in concurrent code | Fix the race condition (don't remove `-race`) |
| golangci-lint new errors | Linter updated or new rules | Fix or add `//nolint:` with reason |
| E2E migration fails | Non-idempotent SQL | Add `IF NOT EXISTS` / `IF EXISTS` |
| E2E auth 502 | Backend cold start during setup | `fetchWithRetry` handles this automatically |
| deploy-verify 10 retries fail | Railway deploy stuck or failed | Check Railway dashboard manually |
| Smoke test auth fails | `E2E_PROD_USER_EMAIL` secret missing | Set in GitHub repo Settings → Secrets |
| `npm audit` blocks build | Won't block — it's `continue-on-error` | But review the vulnerabilities |

### Reading CI Logs

```bash
# View failed run logs
gh run view <run-id> --log-failed | tail -100

# List recent runs
gh run list --limit 10

# Watch current run
gh run watch

# Re-run failed jobs only
gh run rerun <run-id> --failed
```

## Modifying the Pipeline

### Adding a New CI Step

Add to the appropriate job based on what it tests:

```yaml
# Backend step:
- name: My new check
  run: my-command
  working-directory: backend

# Frontend step:
- name: My new check
  run: my-command
  working-directory: frontend
```

**Rules:**
- Security scanners should use `continue-on-error: true` to avoid blocking deploys
- Quality gates (lint, test, build) must NOT use `continue-on-error`
- Always specify `working-directory` for monorepo steps

### Adding a New Path Filter

In the `changes` job:
```yaml
filters: |
  backend:
    - 'backend/**'
  frontend:
    - 'frontend/**'
  newpackage:
    - 'newpackage/**'
```

Then add a new job with `if: needs.changes.outputs.newpackage == 'true'`.

### Adding Environment Variables

| Type | Where to Set | Access Pattern |
|------|-------------|----------------|
| Non-secret config | GitHub repo → Settings → Variables | `${{ vars.MY_VAR }}` |
| Secrets | GitHub repo → Settings → Secrets | `${{ secrets.MY_SECRET }}` |
| E2E test env | Inline in job `env:` block | `process.env.MY_VAR` |

## Key Variables Reference

| Variable | Type | Used By |
|----------|------|---------|
| `BACKEND_PROD_URL` | vars | deploy-verify, post-deploy-e2e |
| `FRONTEND_PROD_URL` | vars | deploy-verify, post-deploy-e2e |
| `E2E_ACCESS_TOKEN` | secrets | post-deploy-e2e auth (JWT access token) |
| `E2E_REFRESH_TOKEN` | secrets | post-deploy-e2e auth (JWT refresh token) |

## E2E Auth Setup

Auth is **token-based** (NOT password-based). Google OAuth only — no direct login.

`global-setup.ts` reads tokens from env vars and writes them to localStorage in storage state:

```typescript
// Required env vars:
// E2E_ACCESS_TOKEN  — JWT access token for test user
// E2E_REFRESH_TOKEN — JWT refresh token for test user
```

**Flow:**
1. Read `E2E_ACCESS_TOKEN` + `E2E_REFRESH_TOKEN` from env
2. Write `storageState` JSON with tokens in `localStorage`
3. If tokens missing → write empty state + warn (auth tests will fail)
4. Playwright tests load this state for authenticated access

**CI secrets needed:** `E2E_ACCESS_TOKEN`, `E2E_REFRESH_TOKEN` (not email/password)

See `e2e-write` skill for writing tests that work with this auth flow.

## Concurrency

```yaml
concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: true
```

Multiple pushes to the same branch cancel previous runs. This saves CI minutes but means rapid pushes may not get full verification.

## Related

- `e2e-write` for proactive rules when writing new E2E tests
- `e2e-debug` for debugging specific Playwright test failures
- `deploy-check` for local pre-push verification
- `railway-deploy` for Railway-specific deployment issues
