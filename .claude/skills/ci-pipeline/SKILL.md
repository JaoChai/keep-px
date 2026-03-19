---
name: ci-pipeline
description: GitHub Actions CI pipeline structure and common CI failure debugging for Keep-PX
---

# CI Pipeline

## When to Activate

- "CI failed"
- "Why is the build red?"
- "Fix CI"
- "Pipeline error"

## Pipeline Structure

Read `.github/workflows/` for the current pipeline definition. Standard flow:

```
changes → backend → frontend → e2e → ci-gate → deploy-verify → post-deploy-e2e
```

- `ci-gate` is the required status check for PR merges
- `deploy-verify` runs after merge to main
- `post-deploy-e2e` runs `@smoke` tagged tests against production

## Debugging Decision Tree

### Backend Job Failed

```
go build failed?
  → Check syntax errors in changed .go files
  → Did you update interfaces.go without updating mocks?

go vet failed?
  → Usually: unused variables, unreachable code, printf format mismatches

go test failed?
  → Read the specific test failure message
  → Mock expectations not met? Check service/mocks_test.go matches interfaces
  → Tests needing DATABASE_URL will fail in CI (integration tests)
```

### Frontend Job Failed

```
TypeScript errors?
  → Run `cd frontend && npx tsc --noEmit` locally
  → LSP diagnostics can be stale — trust tsc output

Lint errors?
  → Run `cd frontend && npm run lint -- --fix`

Build failed but types pass?
  → Check Vite-specific issues (env vars, imports)
```

### E2E Job Failed

```
Auth setup failed?
  → Token expired or E2E env vars missing in CI secrets
  → Check .auth/user.json generation step

Test timeout?
  → Usually network/dialog timing — add explicit waits
  → Check if test relies on data that doesn't exist in sandbox

Strict mode violation?
  → Multiple elements matched — use .first() or scope with parent locator
  → See e2e-debug skill for detailed patterns
```

### deploy-verify / post-deploy-e2e Failed

```
Health check failed?
  → Check Railway logs: `railway logs`
  → Did migration fail? Check startup logs for golang-migrate errors

Smoke test failed?
  → Production URL changed? Check VITE_API_URL
  → CSP blocking? See nginx-csp skill
```

## Related

- `e2e-debug` — detailed E2E failure analysis
- `railway-deploy` — deployment issues
- `nginx-csp` — CSP/proxy issues
