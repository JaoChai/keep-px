---
name: ci-pipeline
description: GitHub Actions CI pipeline structure and failure debugging — ใช้เมื่อ: CI แดง, CI fail, build fail, pipeline error, GitHub Actions พัง, CI ไม่ผ่าน, build ไม่ผ่าน, workflow error
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
changes → backend ‖ frontend ‖ worker → ci-gate → deploy
```

- `changes` เป็น paths filter — job ปลายทางรันเฉพาะเมื่อ path ของมันถูกแตะ
  **เพิ่ม path ใหม่เข้า filter ทุกครั้งที่สร้างโฟลเดอร์ระดับบนสุด** ไม่งั้นทั้ง test และ deploy เงียบ
- `worker` รัน `npm run typecheck` + `npm test` (vitest ที่ root) สำหรับ `worker/**` และ `wrangler.jsonc`
- `ci-gate` is the required status check for PR merges
- `deploy` runs after merge to main — CI deploys via `npm run deploy` (wrangler) แล้ว verify `/health`
  และ security header 7 ตัว; ดู skill `cloudflare-deploy`
  · `if` ของมันต้องมี `always()` นำหน้าเสมอ ไม่งั้นจะรับ skip ต่อมาจาก needs chain
- **ไม่มี e2e แล้ว** — ถูกลบทั้งชุดเมื่อ 2026-08-12

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

### Worker Job Failed

```
typecheck failed?
  → Env type ล้าสมัย — รัน `npx wrangler types` หลังแก้ vars/secret ใน wrangler.jsonc

vitest failed?
  → worker/headers.test.ts หรือ client-ip.test.ts — ดู skill `worker-headers`
```

### deploy Failed

```
job ขึ้นว่า skipped ไม่ใช่ failed?
  → path ที่แก้ไม่อยู่ใน `changes` filter หรือ `if` ขาด `always()`
  → เช็คด้วย `gh run view <id> --json jobs` ดู conclusion ของ job ไม่ใช่ของ run

Health check failed?
  → Check Worker logs: `npx wrangler tail`
  → Container ไม่ตื่น / cold start ช้า / secret หาย? ดู `cloudflare-deploy` skill
  → Did migration fail? Check startup logs for golang-migrate errors

Security header หาย?
  → CSP / header ตกหล่น? See `worker-headers` skill
```

## Related

- `cloudflare-deploy` — deployment issues (Worker + Container)
- `worker-headers` — CSP / security headers
