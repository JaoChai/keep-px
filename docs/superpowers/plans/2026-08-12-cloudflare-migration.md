# Cloudflare Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** ย้าย frontend และ backend ของ keep-px จาก Railway ไป Cloudflare (Workers Static Assets + Containers) โดยไม่เขียน Go ใหม่ และล้างไฟล์ Railway/Nginx ที่ตายแล้วทิ้ง

**Architecture:** Worker ตัวเดียวเป็นด่านหน้าทั้งหมด — เสิร์ฟ React SPA จาก Static Assets, ส่ง `/api/*` `/p/*` `/health` `/ready` ต่อเข้า Cloudflare Container ที่รัน Go binary เดิมจาก `backend/Dockerfile` และแนบ security headers ที่เคยอยู่ใน `nginx.conf` Neon Postgres และ R2 คงเดิมไม่แตะ

**Tech Stack:** Cloudflare Workers · Cloudflare Containers (`@cloudflare/containers`) · Wrangler · Go 1.26.5 (เดิม) · React 19 + Vite 7 (เดิม) · Vitest · Playwright

**Spec:** `docs/superpowers/specs/2026-08-12-cloudflare-migration-design.md`

## ⚠️ ข้อจำกัดลำดับ — อ่านก่อนแตะ git push

**ห้าม push Task 1 ขึ้น `main` จนกว่า Worker จะรับ traffic จริงแล้ว**

CI ตั้งไว้ให้ deploy อัตโนมัติเมื่อ push เข้า `main` และ DNS จริงเป็นแบบนี้ (ตรวจผ่าน Cloudflare API 2026-08-12):

| hostname | ชี้ไป | ผ่าน Cloudflare |
|---|---|---|
| `pixlinks.xyz` | `7o0pp82y.up.railway.app` | ❌ grey cloud |
| `api.pixlinks.xyz` | `plh1h140.up.railway.app` | ❌ grey cloud |
| `customer.pixlinks.xyz` | → `api.pixlinks.xyz` | ✅ proxied |

`api.pixlinks.xyz` ไม่ผ่าน Cloudflare → request ที่ยิงตรงเข้ามา**ไม่มี `CF-Connecting-IP`** ถ้า Task 1 ขึ้นโปรดักชันก่อน Worker จะขึ้น backend จะ fallback ไป RemoteAddr แล้วบันทึก IP ของ proxy แทน IP ลูกค้าจริง **เงียบ ๆ ไม่มี error** — คือความพังแบบเดียวกับที่ Task 1 ตั้งใจป้องกัน

`docs/superpowers/specs/2026-08-06-dependency-update-followups.md` ยืนยันไว้แล้วว่า backend เข้าถึงได้ตรงที่ `api.pixlinks.xyz` — Nginx ไม่ใช่ทางเข้าเดียว

**ลำดับที่ปลอดภัย:** Task 2-7 ให้ครบ → Worker รับ traffic → ค่อย push ทั้งชุดพร้อมกัน

## สิ่งที่มีอยู่จริงในโปรดักชัน แต่ไม่มีในโค้ด repo นี้

**1. zone `pixlinks.xyz` อยู่บน Cloudflare แล้ว (active)** → Task 8 step 1-2 (ย้าย nameserver) **ไม่ต้องทำ** เหลือแค่เปลี่ยน CNAME ให้ชี้ Worker

**2. Worker ชื่อ `pixlinks-worker` ถูก deploy ไว้แล้ว** — ฟีเจอร์ custom domain ของลูกค้า source ไม่อยู่ใน repo นี้
- bindings: `BACKEND_ORIGIN = https://api.pixlinks.xyz` · KV `DOMAIN_MAP` (`07285dcbbd434c4ba63a396e574f2dbe`) · `PLATFORM_DOMAINS`
- ตอนนี้ **ไม่มี route ผูกอยู่** (`routes: []`) = ยังไม่รับ traffic
- **KV `DOMAIN_MAP` ว่างเปล่า 0 key และ zone ไม่มี custom hostname เลย** (ตรวจ 2026-08-12)
  → ฟีเจอร์นี้ยังไม่มีลูกค้าใช้จริงสักราย ความเสี่ยงตอน cutover ต่ำ ไม่ใช่ blocker
- migration `000003_custom_domains` ถูก drop ไปแล้วที่ `000009` แต่ Worker กับ KV ยังอยู่
- **ต้องอัปเดต `BACKEND_ORIGIN` ตอน cutover** ไม่งั้นพอเปิดใช้จะชี้ไป Railway ที่ตายแล้ว

**3. สถาปัตยกรรมจริงเป็น 2 origin** — frontend `pixlinks.xyz` · backend `api.pixlinks.xyz`

**ตัดสินแล้ว: เก็บ `api.pixlinks.xyz` ไว้ ชี้เข้า Worker ตัวเดียวกัน** — e2e ผูกไว้ 5 จุด (`production-smoke.spec.ts`, `global-setup.ts`) และ snippet ที่ลูกค้าอาจฝังไปแล้วชี้โดเมนนั้น การยุบทิ้งเสี่ยงทำ pixel พังโดยไม่มีสัญญาณเตือน เก็บไว้ต้นทุนเป็นศูนย์

→ ผลต่อ Task 4: `VITE_API_URL` ตั้งเป็นค่าว่างได้ตามแผนเดิม (same-origin) แต่ Worker ต้องรับทั้งสอง hostname

## Global Constraints

- **DB ไม่ย้าย** — Neon Postgres (`us-east-1`) คงเดิม ห้ามแตะ migration หรือ schema
- **ห้ามเขียน Go ใหม่** — แก้ Go ได้เฉพาะ 2 จุดที่ระบุใน Task 1 และ Task 7 เท่านั้น
- `max_instances = 1` เสมอ — migration รันตอน boot (`main.go:78`), rate limiter และ sale page cache เก็บใน memory
- `instance_type = "basic"` · `defaultPort = 8080` · `sleepAfter = "1h"` · `pingEndpoint = "health"`
- security headers ที่ออกจาก Worker ต้องตรงกับ `frontend/nginx.conf` เดิมทุกตัว **ยกเว้น** `connect-src` ที่ตัด `${BACKEND_URL}` ออกเพราะกลายเป็น same-origin
- dashboard และ sale page ใช้ headers **คนละชุด** — sale page มีแค่ 4 ตัว ไม่มี `X-Frame-Options` (ตั้งใจ เพราะ sale page ต้องฝังในหน้าอื่นได้)
- commit message ต้องผ่าน commitlint — subject ขึ้นต้นด้วย **ตัวพิมพ์เล็ก**
- ห้ามลบไฟล์ใน `docs/superpowers/plans/` และ `docs/superpowers/specs/`

---

## File Structure

**สร้างใหม่:**
| ไฟล์ | หน้าที่ |
|---|---|
| `wrangler.jsonc` | config เดียวของทั้งระบบ — assets, container, DO binding, cron |
| `worker-configuration.d.ts` | generate ด้วย `wrangler types` · gitignore ไว้ |
| `worker/index.ts` | Container class + fetch handler + scheduled handler |
| `worker/headers.ts` | security headers ทั้ง 2 ชุด (pure function ทดสอบได้โดยไม่ต้องมี runtime) |
| `worker/headers.test.ts` | unit test ของ headers |
| `worker/client-ip.ts` | normalize client IP ก่อนส่งเข้า container |
| `worker/client-ip.test.ts` | unit test ของ client IP |
| — | — |

**แก้ไข:** `backend/internal/router/router.go:30` · `backend/internal/router/clientip_test.go` · `backend/internal/handler/health.go:72` · `.github/workflows/ci.yml` · `frontend/playwright.config.ts`

**ลบ:** `frontend/Dockerfile` · `frontend/nginx.conf` · `frontend/10-set-dns-resolver.envsh` · `frontend/railway.json` · `backend/railway.json` · `.claude/skills/railway-deploy/` · `.claude/skills/nginx-csp/`

**ไม่แตะ:** `backend/Dockerfile` (Containers ใช้) · `frontend/error-pages/*.html` (Worker เสิร์ฟแทน) · โค้ด Go ที่เหลือทั้งหมด · `frontend/src/**` ทั้งหมด

---

## Task 1: เปลี่ยน client IP เป็น CF-Connecting-IP

ทำก่อนทุกอย่างเพราะเป็นความเสี่ยงสูงสุด และทดสอบได้ทันทีโดยไม่ต้องมีบัญชี Cloudflare

**ทำไมสำคัญ:** `event_handler.go:47` อ่าน client IP → ส่งเข้า `event_service.go:206` → `userData["client_ip_address"]` → Meta CAPI ถ้าค่าผิดจะไม่มี error แต่ attribution พังเงียบ

**Files:**
- Modify: `backend/internal/router/router.go:30`
- Modify: `backend/internal/router/clientip_test.go`

**Interfaces:**
- Consumes: `chimiddleware.ClientIPFromHeader` จาก `github.com/go-chi/chi/v5/middleware` (มีอยู่แล้ว)
- Produces: router ที่ resolve client IP จาก `CF-Connecting-IP` — Task 3 พึ่งพาข้อนี้ตอน normalize header ใน Worker

- [ ] **Step 1: แก้ test ให้ fail ก่อน**

ใน `backend/internal/router/clientip_test.go` — `TestRouter_ForgedHeadersCannotBuyFreshBuckets` เปลี่ยน header ที่ใช้เป็นตัวจริง และย้าย `X-Real-IP` ไปอยู่ฝั่ง header ปลอม:

```go
	for i := 1; i <= 6; i++ {
		code := do(t, h,
			fmt.Sprintf("10.0.0.%d:1234", i),
			map[string]string{
				"CF-Connecting-IP": "203.0.113.1",
				"X-Real-IP":        fmt.Sprintf("198.51.100.%d", i),
				"True-Client-IP":   fmt.Sprintf("198.51.100.%d", i),
				"X-Forwarded-For":  fmt.Sprintf("192.0.2.%d", i),
			},
		)
		if code == http.StatusTooManyRequests {
			return // bucket collision observed — the guard holds
		}
	}

	t.Fatal("router does not resolve the client IP from CF-Connecting-IP — forged " +
		"X-Real-IP / True-Client-IP / X-Forwarded-For bought a fresh rate-limit " +
		"bucket on every request. chimiddleware.ClientIPFromHeader(\"CF-Connecting-IP\") " +
		"is not registered in router.New. See " +
		"docs/superpowers/specs/2026-08-12-cloudflare-migration-design.md")
```

และใน `TestRouter_DistinctRealIPsKeepSeparateBuckets`:

```go
			map[string]string{
				"CF-Connecting-IP": fmt.Sprintf("203.0.113.%d", i),
			},
```

พร้อมแก้ข้อความใน `t.Fatalf` ของ test นั้นจาก `X-Real-IP` เป็น `CF-Connecting-IP`

- [ ] **Step 2: รัน test ให้เห็นว่า fail**

```bash
cd backend && go test ./internal/router/ -run 'TestRouter_(ForgedHeaders|DistinctRealIPs)' -v
```

Expected: `TestRouter_ForgedHeadersCannotBuyFreshBuckets` FAIL พร้อมข้อความ "router does not resolve the client IP from CF-Connecting-IP"

- [ ] **Step 3: แก้ router ให้ test ผ่าน**

`backend/internal/router/router.go` บรรทัด 30 เปลี่ยนจาก:

```go
	r.Use(chimiddleware.ClientIPFromHeader("X-Real-IP"))
```

เป็น:

```go
	// Cloudflare sets CF-Connecting-IP to the real client address at the edge and
	// overwrites any client-supplied value, so it is the only header we trust.
	// The Worker strips X-Real-IP / X-Forwarded-For / True-Client-IP before
	// forwarding — see worker/client-ip.ts.
	r.Use(chimiddleware.ClientIPFromHeader("CF-Connecting-IP"))
```

- [ ] **Step 4: รัน test ให้ผ่าน**

```bash
cd backend && go test ./internal/router/ -run 'TestRouter_(ForgedHeaders|DistinctRealIPs)' -v
```

Expected: PASS ทั้งสองตัว

- [ ] **Step 5: แก้ comment ที่อ้าง Railway/Nginx ในไฟล์ test**

ใน `clientip_test.go` comment เหนือ `TestRouter_ForgedHeadersCannotBuyFreshBuckets` มีประโยค `the one header Railway's edge overwrites` และเหนือ `TestRouter_DistinctRealIPsKeepSeparateBuckets` มี `as all traffic is behind Nginx in production` — แก้ทั้งสองให้อ้าง Cloudflare แทน โดยคงเนื้อหาเชิงเทคนิคส่วนที่เหลือไว้ครบ (ข้อความเรื่อง burst = rps*2 และคู่ test ที่ต้องอยู่ด้วยกัน ห้ามตัด)

- [ ] **Step 6: รัน test ทั้ง package**

```bash
cd backend && go test ./internal/... 2>&1 | tail -30
```

Expected: ทุก package ผ่าน (ยกเว้น integration test ที่ต้องใช้ build tag `integration` ซึ่งจะถูกข้าม)

- [ ] **Step 7: Commit**

```bash
git add backend/internal/router/router.go backend/internal/router/clientip_test.go
git commit -m "fix(security): resolve client IP from CF-Connecting-IP instead of X-Real-IP

X-Real-IP ถูกเซ็ตโดย Nginx ซึ่งกำลังจะถูกลบทิ้งพร้อม Railway
Cloudflare เซ็ต CF-Connecting-IP ให้ที่ edge และ client ปลอมไม่ได้

client IP ไหลเข้า Meta CAPI ที่ event_service.go:206 ถ้าค่าผิด
attribution จะพังโดยไม่มี error ให้เห็น"
```

---

## Task 2: Spike — พิสูจน์ว่า Go backend รันบน Cloudflare Containers ได้จริง

**เป้าหมาย:** ตอบคำถามเดียว — Go + pgx pool + migration-on-boot ขึ้นบน Containers ได้ไหม ถ้าไม่ได้ต้องรู้ตอนนี้ ไม่ใช่ตอนทำไปครึ่งทาง

**Files:**
- Modify: `package.json` (root — **มีอยู่แล้ว ห้ามเขียนทับ** ถือ husky/commitlint/lint-staged)
- Create: `wrangler.jsonc`
- Create: `worker/index.ts` (ฉบับ minimal — จะขยายใน Task 3)

**Interfaces:**
- Produces: `export class Backend extends Container` และ binding ชื่อ `BACKEND` — Task 3, 4, 5 ใช้ทั้งคู่

- [ ] **Step 1: แก้ `package.json` ที่ root — ห้ามสร้างทับ**

⚠️ **ไฟล์นี้มีอยู่แล้วและ git ติดตามอยู่** มันถือ husky + commitlint + lint-staged ซึ่งเป็น hook ที่บังคับรูปแบบ commit ของทั้ง repo เขียนทับ = pre-commit hook พังทั้งโปรเจกต์

**เพิ่ม** 4 key นี้เข้าไป โดยคง `private`, `devDependencies` เดิมทั้ง 4 ตัว, `lint-staged` และ `scripts.prepare` ไว้ครบ:

```json
{
  "private": true,
  "devDependencies": {
    "@commitlint/cli": "^20.4.2",
    "@commitlint/config-conventional": "^20.4.2",
    "husky": "^9.1.7",
    "lint-staged": "^16.2.7",
    "wrangler": "^4.42.0",
    "vitest": "^4.0.18",
    "typescript": "~5.9.3"
  },
  "dependencies": {
    "@cloudflare/containers": "^0.0.42"
  },
  "lint-staged": {
    "frontend/src/**/*.{ts,tsx}": [
      "npx --prefix frontend eslint --fix --config frontend/eslint.config.js"
    ]
  },
  "scripts": {
    "prepare": "husky",
    "build": "npm run build --prefix frontend",
    "deploy": "npm run build && wrangler deploy",
    "dev": "wrangler dev",
    "test": "vitest run"
  }
}
```

⚠️ **ห้ามใส่ `"type": "module"`** — `commitlint.config.js` ที่ root ใช้ `module.exports` (CommonJS) การใส่ `type: module` จะทำให้ commit ทุกครั้งล้มด้วย `module is not defined` wrangler bundle โค้ด Worker เองอยู่แล้ว จึงไม่ต้องพึ่ง field นี้

**หมายเหตุเวอร์ชัน:** เลข 3 ตัวใหม่ (`wrangler`, `vitest`, `@cloudflare/containers`) เป็นค่าตั้งต้น — รัน `npm install -D wrangler vitest typescript && npm install @cloudflare/containers` แล้วปล่อยให้ npm เลือกเวอร์ชันล่าสุด จากนั้นค่อยบันทึกค่าที่ได้จริง

- [ ] **Step 1b: ยืนยันว่า commit hook ยังทำงาน**

```bash
git commit --allow-empty -m "test: verify commitlint still works" && git reset --soft HEAD~1
```

Expected: commit ผ่าน แล้วถูกถอยกลับ — ถ้าล้มด้วย `module is not defined` แปลว่าเผลอใส่ `type: module` เข้าไป

⚠️ **ต้องเป็น `--soft` เท่านั้น ห้ามใช้ `--hard`** — ขั้นตอนนี้รันตอนที่ `package.json` ยังมีงานค้างใน working tree `--hard` จะลบการแก้ทั้งหมดทิ้งเงียบ ๆ รวมทั้ง dependency ที่ npm เพิ่งเขียนลงไป แล้วจะไม่มีใครรู้จนกว่าจะ `npm ci` บนเครื่องใหม่แล้วพัง
(เกิดขึ้นจริงตอนรันแผนนี้รอบแรก — แผนเดิมเขียน `--hard` ไว้)

- [ ] **Step 2: ติดตั้ง**

```bash
npm install
npx wrangler --version
```

Expected: พิมพ์เวอร์ชัน wrangler ออกมาโดยไม่ error

- [ ] **Step 3: สร้าง wrangler.jsonc ฉบับ spike**

```jsonc
{
  "$schema": "./node_modules/wrangler/config-schema.json",
  "name": "keep-px-spike",
  "main": "worker/index.ts",
  "compatibility_date": "2026-08-12",
  "compatibility_flags": ["nodejs_compat"],
  "containers": [
    {
      "class_name": "Backend",
      "image": "./backend/Dockerfile",
      "image_build_context": "./backend",
      "instance_type": "basic",
      "max_instances": 1
    }
  ],
  "durable_objects": {
    "bindings": [{ "class_name": "Backend", "name": "BACKEND" }]
  },
  "migrations": [{ "tag": "v1", "new_sqlite_classes": ["Backend"] }]
}
```

- [ ] **Step 4: สร้าง worker/index.ts ฉบับ minimal**

```ts
import { Container, getContainer } from "@cloudflare/containers";
import { env } from "cloudflare:workers";

export class Backend extends Container {
  defaultPort = 8080;
  sleepAfter = "1h";
  pingEndpoint = "health";

  envVars = {
    PORT: "8080",
    ENV: "production",
    DATABASE_URL: env.DATABASE_URL,
    JWT_SECRET: env.JWT_SECRET,
  };
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    return getContainer(env.BACKEND).fetch(request);
  },
} satisfies ExportedHandler<Env>;
```

- [ ] **Step 5: สร้าง type ของ binding**

`Env` ถูกใช้ในโค้ดทุกไฟล์ของ Worker แต่ไม่ได้เขียนเอง — wrangler สร้างให้จาก `wrangler.jsonc`

```bash
npx wrangler types
```

Expected: ได้ไฟล์ `worker-configuration.d.ts` ที่มี `interface Env` ประกาศ `BACKEND` อยู่ข้างใน

เพิ่มลง `.gitignore` ที่ root เพราะเป็นไฟล์ที่ generate ใหม่ได้เสมอ:

```bash
echo "worker-configuration.d.ts" >> .gitignore
```

**ต้องรัน `npx wrangler types` ซ้ำทุกครั้งที่แก้ binding หรือ secret ใน `wrangler.jsonc`** — Task 4 และ Task 5 เพิ่ม binding ใหม่ทั้งคู่

- [ ] **Step 6: ตั้ง secret 2 ตัวที่ spike ต้องใช้**

```bash
npx wrangler secret put DATABASE_URL   # วาง connection string ของ Neon
npx wrangler secret put JWT_SECRET     # ค่าอะไรก็ได้สำหรับ spike
```

**ระวัง:** ใช้ Neon **branch สำหรับทดสอบ** ไม่ใช่ production — spike จะรัน migration ตอน boot

- [ ] **Step 7: deploy แล้วยิง health check**

```bash
npx wrangler deploy
curl -i https://keep-px-spike.<subdomain>.workers.dev/health
```

Expected: HTTP 200 และ body บอกว่าต่อ DB ได้

- [ ] **Step 8: ดู log ว่า migration รันจริง**

```bash
npx wrangler tail keep-px-spike --format pretty
```

ยิง `/health` ซ้ำอีกครั้ง แล้วดูว่า log มีบรรทัด `connected to database` และ `migrations: already up to date` (หรือ `applied successfully` ถ้ารันครั้งแรก)

- [ ] **Step 9: จดผลแล้วตัดสินใจ**

บันทึกลง `docs/superpowers/plans/2026-08-12-cloudflare-migration.md` ท้ายไฟล์ ในหัวข้อ `## ผล Spike`:
- health check ตอบ 200 หรือไม่
- เวลา cold start ของ request แรก (วัดด้วย `curl -w '%{time_total}'`)
- migration รันสำเร็จหรือไม่
- memory 1 GiB (`basic`) พอหรือไม่ — ดูว่ามี OOM ใน log ไหม

**ถ้า health check ไม่ผ่าน:** หยุดตรงนี้ กลับไปคุยกับ user ก่อน อย่าทำ Task 3 ต่อ

- [ ] **Step 10: Commit**

```bash
git add package.json package-lock.json wrangler.jsonc worker/index.ts .gitignore docs/superpowers/plans/2026-08-12-cloudflare-migration.md
git commit -m "feat(infra): spike Go backend on Cloudflare Containers

พิสูจน์ว่า Go + pgx pool + migration-on-boot รันบน Containers ได้
ก่อนลงมือย้ายจริง ผลบันทึกไว้ท้ายไฟล์แผน"
```

---

## Task 3: Worker — security headers + client IP normalization

**Files:**
- Create: `worker/headers.ts`
- Create: `worker/headers.test.ts`
- Create: `worker/client-ip.ts`
- Create: `worker/client-ip.test.ts`
- Create: `vitest.config.ts` (root)

**Interfaces:**
- Consumes: ไม่มี (pure functions)
- Produces:
  - `applySecurityHeaders(response: Response, pathname: string): Response`
  - `normalizeClientIP(request: Request): Request`
  - Task 4 เรียกทั้งสองตัวใน fetch handler

- [ ] **Step 1: เขียน test ของ headers ก่อน**

`worker/headers.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import { applySecurityHeaders } from "./headers";

function headersFor(pathname: string): Headers {
  return applySecurityHeaders(new Response("x"), pathname).headers;
}

describe("applySecurityHeaders", () => {
  it("ใส่ครบทั้ง 7 ตัวสำหรับ dashboard", () => {
    const h = headersFor("/dashboard");
    expect(h.get("X-Content-Type-Options")).toBe("nosniff");
    expect(h.get("X-Frame-Options")).toBe("DENY");
    expect(h.get("Referrer-Policy")).toBe("strict-origin-when-cross-origin");
    expect(h.get("Permissions-Policy")).toBe("camera=(), microphone=(), geolocation=()");
    expect(h.get("Strict-Transport-Security")).toBe("max-age=31536000; includeSubDomains");
    expect(h.get("Cross-Origin-Opener-Policy")).toBe("same-origin");
    expect(h.get("Content-Security-Policy")).toContain("default-src 'self'");
  });

  it("sale page ต้องไม่มี X-Frame-Options เพราะต้องฝังในหน้าอื่นได้", () => {
    const h = headersFor("/p/my-slug");
    expect(h.get("X-Frame-Options")).toBeNull();
    expect(h.get("Permissions-Policy")).toBeNull();
    expect(h.get("Cross-Origin-Opener-Policy")).toBeNull();
    expect(h.get("X-Content-Type-Options")).toBe("nosniff");
    expect(h.get("Referrer-Policy")).toBe("strict-origin-when-cross-origin");
    expect(h.get("Strict-Transport-Security")).toBe("max-age=31536000; includeSubDomains");
  });

  it("sale page CSP อนุญาต img จากทุกที่ แต่ dashboard ไม่", () => {
    expect(headersFor("/p/x").get("Content-Security-Policy")).toContain("img-src * data:");
    expect(headersFor("/dashboard").get("Content-Security-Policy")).toContain(
      "img-src 'self' data: *.googleusercontent.com *.r2.dev",
    );
  });

  it("CSP ต้องไม่มี BACKEND_URL placeholder หลงเหลือ", () => {
    for (const p of ["/dashboard", "/p/x"]) {
      expect(headersFor(p).get("Content-Security-Policy")).not.toContain("${");
    }
  });

  it("ไม่ทำลาย header เดิมของ response", () => {
    const res = new Response("x", { headers: { "Content-Type": "text/html" } });
    expect(applySecurityHeaders(res, "/").headers.get("Content-Type")).toBe("text/html");
  });
});
```

- [ ] **Step 2: สร้าง vitest.config.ts แล้วรัน test ให้ fail**

`vitest.config.ts`:

```ts
import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    include: ["worker/**/*.test.ts"],
    environment: "node",
  },
});
```

```bash
npx vitest run
```

Expected: FAIL — `Cannot find module './headers'`

- [ ] **Step 3: เขียน worker/headers.ts**

ค่าทั้งหมดคัดลอกมาจาก `frontend/nginx.conf` โดยตรง เปลี่ยนเฉพาะ `connect-src` ที่ตัด `${BACKEND_URL}` ออก

```ts
const DASHBOARD_CSP = [
  "default-src 'self'",
  "script-src 'self' 'unsafe-inline' js.stripe.com connect.facebook.net accounts.google.com",
  "style-src 'self' 'unsafe-inline' fonts.googleapis.com accounts.google.com",
  "connect-src 'self' accounts.google.com js.stripe.com",
  "frame-src js.stripe.com accounts.google.com",
  "img-src 'self' data: *.googleusercontent.com *.r2.dev",
  "font-src 'self' data: fonts.gstatic.com",
].join("; ");

const SALE_PAGE_CSP = [
  "default-src 'self'",
  "script-src 'self' 'unsafe-inline'",
  "style-src 'self' 'unsafe-inline' fonts.googleapis.com",
  "connect-src 'self'",
  "img-src * data:",
  "font-src 'self' data: fonts.gstatic.com",
].join("; ");

const HSTS = "max-age=31536000; includeSubDomains";
const REFERRER = "strict-origin-when-cross-origin";

/**
 * Sale pages are meant to be embedded on customer domains, so they
 * deliberately omit X-Frame-Options / Permissions-Policy / COOP — the same
 * split the old nginx.conf made between its `/p/` block and the server block.
 */
function isSalePage(pathname: string): boolean {
  return pathname.startsWith("/p/");
}

export function applySecurityHeaders(response: Response, pathname: string): Response {
  const res = new Response(response.body, response);

  res.headers.set("X-Content-Type-Options", "nosniff");
  res.headers.set("Referrer-Policy", REFERRER);
  res.headers.set("Strict-Transport-Security", HSTS);

  if (isSalePage(pathname)) {
    res.headers.set("Content-Security-Policy", SALE_PAGE_CSP);
    return res;
  }

  res.headers.set("X-Frame-Options", "DENY");
  res.headers.set("Permissions-Policy", "camera=(), microphone=(), geolocation=()");
  res.headers.set("Cross-Origin-Opener-Policy", "same-origin");
  res.headers.set("Content-Security-Policy", DASHBOARD_CSP);
  return res;
}
```

- [ ] **Step 4: รัน test ให้ผ่าน**

```bash
npx vitest run worker/headers.test.ts
```

Expected: PASS ทั้ง 5 เคส

- [ ] **Step 5: เขียน test ของ client IP**

`worker/client-ip.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import { normalizeClientIP } from "./client-ip";

function req(headers: Record<string, string>): Request {
  return new Request("https://example.com/api/v1/events/ingest", { headers });
}

describe("normalizeClientIP", () => {
  it("เก็บ CF-Connecting-IP ที่ Cloudflare เซ็ตไว้", () => {
    const out = normalizeClientIP(req({ "CF-Connecting-IP": "203.0.113.7" }));
    expect(out.headers.get("CF-Connecting-IP")).toBe("203.0.113.7");
  });

  it("ลบ header ที่ client ปลอมได้ทุกตัวทิ้ง", () => {
    const out = normalizeClientIP(
      req({
        "CF-Connecting-IP": "203.0.113.7",
        "X-Real-IP": "198.51.100.1",
        "X-Forwarded-For": "192.0.2.1",
        "True-Client-IP": "192.0.2.2",
        "X-Forwarded-Proto": "http",
      }),
    );
    expect(out.headers.get("X-Real-IP")).toBeNull();
    expect(out.headers.get("X-Forwarded-For")).toBeNull();
    expect(out.headers.get("True-Client-IP")).toBeNull();
    expect(out.headers.get("X-Forwarded-Proto")).toBeNull();
  });

  it("ถ้าไม่มี CF-Connecting-IP ต้องไม่หลง trust ค่าปลอม", () => {
    const out = normalizeClientIP(req({ "X-Real-IP": "198.51.100.1" }));
    expect(out.headers.get("CF-Connecting-IP")).toBeNull();
    expect(out.headers.get("X-Real-IP")).toBeNull();
  });

  it("ไม่แตะ header อื่น", () => {
    const out = normalizeClientIP(
      req({ "CF-Connecting-IP": "203.0.113.7", Authorization: "Bearer abc" }),
    );
    expect(out.headers.get("Authorization")).toBe("Bearer abc");
  });
});
```

- [ ] **Step 6: รัน test ให้ fail**

```bash
npx vitest run worker/client-ip.test.ts
```

Expected: FAIL — `Cannot find module './client-ip'`

- [ ] **Step 7: เขียน worker/client-ip.ts**

```ts
/**
 * Headers a client can set on its own request. Nginx used to overwrite these;
 * now that it is gone the Worker must strip them, otherwise a forged value
 * would reach the container and be trusted. Verified 2026-08-12: no Go code
 * reads X-Forwarded-Proto, so dropping it is safe.
 */
const CLIENT_SPOOFABLE = [
  "X-Real-IP",
  "X-Forwarded-For",
  "X-Forwarded-Proto",
  "True-Client-IP",
];

/**
 * Returns a copy of the request carrying exactly one IP header:
 * CF-Connecting-IP, which Cloudflare's edge sets and a client cannot forge.
 * The Go router reads it via chimiddleware.ClientIPFromHeader("CF-Connecting-IP").
 */
export function normalizeClientIP(request: Request): Request {
  const realIP = request.headers.get("CF-Connecting-IP");
  const out = new Request(request);

  for (const h of CLIENT_SPOOFABLE) {
    out.headers.delete(h);
  }

  if (realIP) {
    out.headers.set("CF-Connecting-IP", realIP);
  } else {
    out.headers.delete("CF-Connecting-IP");
  }

  return out;
}
```

- [ ] **Step 8: รัน test ทั้งหมดให้ผ่าน**

```bash
npx vitest run
```

Expected: PASS ทั้ง 9 เคส (headers 5 + client-ip 4)

- [ ] **Step 9: Commit**

```bash
git add worker/headers.ts worker/headers.test.ts worker/client-ip.ts worker/client-ip.test.ts vitest.config.ts
git commit -m "feat(worker): port nginx security headers and client IP guard to Worker

headers ทั้ง 7 ตัวย้ายมาจาก nginx.conf ครบ แยก 2 ชุดเหมือนเดิม
sale page ยังคงไม่มี X-Frame-Options เพราะต้องฝังในหน้าลูกค้าได้

normalizeClientIP ลบ header ที่ client ปลอมได้ทิ้งก่อนส่งเข้า container
รักษาพฤติกรรมที่ commit 57c23ef ตั้งใจไว้"
```

---

## Task 4: Worker fetch handler + Static Assets

**Files:**
- Modify: `worker/index.ts`
- Modify: `wrangler.jsonc`

**Interfaces:**
- Consumes: `applySecurityHeaders` และ `normalizeClientIP` จาก Task 3 · `Backend` container class จาก Task 2
- Produces: Worker ที่เสิร์ฟทั้ง SPA และ backend — Task 6 (CI) และ Task 8 (cutover) พึ่งพา

- [ ] **Step 1: เพิ่ม assets เข้า wrangler.jsonc**

เปลี่ยน `name` เป็น `keep-px` และเพิ่ม block `assets`:

```jsonc
{
  "$schema": "./node_modules/wrangler/config-schema.json",
  "name": "keep-px",
  "main": "worker/index.ts",
  "compatibility_date": "2026-08-12",
  "compatibility_flags": ["nodejs_compat"],
  "assets": {
    "directory": "./frontend/dist",
    "binding": "ASSETS",
    "not_found_handling": "single-page-application",
    "run_worker_first": ["/api/*", "/p/*", "/health", "/ready"]
  },
  "containers": [
    {
      "class_name": "Backend",
      "image": "./backend/Dockerfile",
      "image_build_context": "./backend",
      "instance_type": "basic",
      "max_instances": 1
    }
  ],
  "durable_objects": {
    "bindings": [{ "class_name": "Backend", "name": "BACKEND" }]
  },
  "migrations": [{ "tag": "v1", "new_sqlite_classes": ["Backend"] }]
}
```

`run_worker_first` ทำให้ 4 path นี้เข้า Worker ก่อน ส่วน path อื่นเสิร์ฟจาก static asset โดยตรง และ `not_found_handling` คืน `index.html` ให้ route ของ React

- [ ] **Step 2: เขียน worker/index.ts ฉบับเต็ม**

```ts
import { Container, getContainer } from "@cloudflare/containers";
import { env } from "cloudflare:workers";
import { applySecurityHeaders } from "./headers";
import { normalizeClientIP } from "./client-ip";

export class Backend extends Container {
  defaultPort = 8080;
  sleepAfter = "1h";
  pingEndpoint = "health";

  envVars = {
    PORT: "8080",
    ENV: "production",
    DATABASE_URL: env.DATABASE_URL,
    JWT_SECRET: env.JWT_SECRET,
    TOKEN_ENCRYPTION_KEY: env.TOKEN_ENCRYPTION_KEY,
    GOOGLE_CLIENT_ID: env.GOOGLE_CLIENT_ID,
    S3_ENDPOINT: env.S3_ENDPOINT,
    S3_BUCKET: env.S3_BUCKET,
    S3_ACCESS_KEY: env.S3_ACCESS_KEY,
    S3_SECRET_KEY: env.S3_SECRET_KEY,
    S3_PUBLIC_URL: env.S3_PUBLIC_URL,
    STRIPE_SECRET_KEY: env.STRIPE_SECRET_KEY,
    STRIPE_WEBHOOK_SECRET: env.STRIPE_WEBHOOK_SECRET,
    STRIPE_PUBLISHABLE_KEY: env.STRIPE_PUBLISHABLE_KEY,
    STRIPE_PRICE_PIXEL_SLOT: env.STRIPE_PRICE_PIXEL_SLOT,
    STRIPE_PRICE_REPLAY_SINGLE: env.STRIPE_PRICE_REPLAY_SINGLE,
    STRIPE_PRICE_REPLAY_MONTHLY: env.STRIPE_PRICE_REPLAY_MONTHLY,
    BASE_URL: env.BASE_URL,
    FRONTEND_URL: env.FRONTEND_URL,
    CORS_ALLOWED_ORIGINS: env.CORS_ALLOWED_ORIGINS,
  };

  override onError(error: unknown) {
    console.error("container error", error);
  }
}

/** Serves the stored nginx error page for a given status, falling back to plain text. */
async function errorPage(env: Env, status: 502 | 503 | 504): Promise<Response> {
  const res = await env.ASSETS.fetch(
    new Request(`https://assets.local/error-pages/${status}.html`),
  );
  if (res.ok) {
    return new Response(res.body, {
      status,
      headers: { "Content-Type": "text/html; charset=utf-8" },
    });
  }
  return new Response(`Backend unavailable (${status})`, { status });
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const pathname = new URL(request.url).pathname;

    try {
      const response = await getContainer(env.BACKEND).fetch(normalizeClientIP(request));
      return applySecurityHeaders(response, pathname);
    } catch (error) {
      console.error("failed to reach backend container", error);
      return applySecurityHeaders(await errorPage(env, 502), pathname);
    }
  },
} satisfies ExportedHandler<Env>;
```

- [ ] **Step 3: build frontend ให้มี dist สำหรับ assets**

```bash
VITE_API_URL= npm run build --prefix frontend
ls frontend/dist/index.html
```

Expected: ไฟล์มีอยู่จริง

`VITE_API_URL` ต้องเป็นค่าว่างเพราะ `frontend/src/lib/api.ts:5` เขียนว่า `import.meta.env.VITE_API_URL || ''` — ค่าว่างทำให้ยิงแบบ same-origin ซึ่งคือสิ่งที่ต้องการ

- [ ] **Step 4: คัดลอก error pages เข้า dist**

`frontend/Dockerfile` เดิมทำขั้นตอนนี้ตอน build image เมื่อลบ Dockerfile ทิ้งแล้วต้องย้ายมาไว้ใน build script แทน — แก้ `frontend/package.json` ช่อง `scripts.build`:

```json
    "build": "tsc -b && vite build && cp -r error-pages dist/error-pages",
```

รันใหม่แล้วตรวจ:

```bash
npm run build --prefix frontend && ls frontend/dist/error-pages/
```

Expected: เห็น `502.html 503.html 504.html`

- [ ] **Step 5: ทดสอบด้วย wrangler dev**

```bash
npx wrangler dev
```

เปิดอีก terminal:

```bash
curl -i http://localhost:8787/health
curl -i http://localhost:8787/ | head -20
```

Expected: `/health` ตอบ 200 จาก container · `/` ตอบ HTML ของ SPA

- [ ] **Step 6: ตรวจว่า headers ออกมาครบเหมือน nginx เดิม**

```bash
curl -sI http://localhost:8787/ | grep -iE 'x-content-type|x-frame|referrer|permissions|strict-transport|cross-origin|content-security'
curl -sI http://localhost:8787/p/anything | grep -iE 'x-frame'
```

Expected: อันแรกได้ครบ 7 บรรทัด · อันที่สองไม่มีอะไรออกมา (sale page ไม่มี `X-Frame-Options` ตามที่ตั้งใจ)

- [ ] **Step 7: Commit**

```bash
git add wrangler.jsonc worker/index.ts frontend/package.json
git commit -m "feat(worker): serve SPA and proxy backend from one Worker

run_worker_first ส่ง /api/* /p/* /health /ready เข้า container
path อื่นเสิร์ฟจาก static assets และ fallback เป็น index.html ให้ React router

error-pages ย้ายจาก frontend/Dockerfile มาอยู่ใน build script แทน"
```

---

## Task 5: Secrets + Cron keep-alive

**Files:**
- Modify: `worker/index.ts`
- Modify: `wrangler.jsonc`

**Interfaces:**
- Consumes: `Backend` class จาก Task 4
- Produces: `scheduled` handler — Task 8 ตรวจสอบว่าทำงาน

- [ ] **Step 1: ตั้ง secret ทั้ง 18 ตัว**

ค่าจริงดึงจาก Railway CLI (ตรวจแล้ว 2026-08-12 ว่ามีครบทั้ง 18 ตัว):

```bash
railway variables --project 1c9b113d-9d1d-4c3b-9296-7b6600173c0f \
  --service pixlinks-api --environment production --json > "$TMPDIR/rw.json"
chmod 600 "$TMPDIR/rw.json"
```

**MCP ของ Railway ใช้ไม่ได้กับงานนี้** — OAuth app ได้แค่ชื่อตัวแปร ค่าถูก redact ต้องใช้ CLI ที่ login แล้วเท่านั้น

**ห้ามยกไป 18 ตัวนี้ — เป็นซากที่ไม่มีโค้ดอ่านแล้ว:**
`CF_ACCOUNT_ID` `CF_API_TOKEN` `CF_CNAME_TARGET` `CF_KV_NAMESPACE_ID` `CF_ZONE_ID` (ซากฟีเจอร์ custom domain ที่ถูก drop ที่ migration 000009 — ตรวจแล้วไม่มีโค้ด Go อ่านสักตัว) และ `STRIPE_PRICE_EVENTS_1M` `STRIPE_PRICE_PIXELS_10` `STRIPE_PRICE_PIXELS_40` `STRIPE_PRICE_PLAN_LAUNCH` `STRIPE_PRICE_PLAN_SHIELD` `STRIPE_PRICE_PLAN_VAULT` `STRIPE_PRICE_REPLAY_1` `STRIPE_PRICE_REPLAY_3` `STRIPE_PRICE_REPLAY_UNLIMITED` `STRIPE_PRICE_RETENTION_180` `STRIPE_PRICE_RETENTION_365` `STRIPE_PRICE_SALE_PAGES_10` `STRIPE_PRICE_SALE_PAGES_25` (ซากราคาแบบ pack เดิม — `config.go:40-42` อ่านแค่ 3 ตัว)

**`BASE_URL` และ `FRONTEND_URL` บน Railway เป็น `https://pixlinks.xyz` อยู่แล้ว** — ยกมาตรง ๆ ไม่ต้องแก้

จากนั้นรันทีละตัว:

```bash
for s in DATABASE_URL JWT_SECRET TOKEN_ENCRYPTION_KEY GOOGLE_CLIENT_ID \
         S3_ENDPOINT S3_BUCKET S3_ACCESS_KEY S3_SECRET_KEY S3_PUBLIC_URL \
         STRIPE_SECRET_KEY STRIPE_WEBHOOK_SECRET STRIPE_PUBLISHABLE_KEY \
         STRIPE_PRICE_PIXEL_SLOT STRIPE_PRICE_REPLAY_SINGLE STRIPE_PRICE_REPLAY_MONTHLY \
         BASE_URL FRONTEND_URL CORS_ALLOWED_ORIGINS; do
  echo "=== $s ==="
  npx wrangler secret put "$s"
done
```

**ค่าที่ต้องเปลี่ยนจากเดิม ไม่ใช่คัดลอกตรง ๆ:**
- `BASE_URL` และ `FRONTEND_URL` → โดเมนใหม่ (ระหว่างทดสอบใช้ `https://keep-px.<subdomain>.workers.dev`)
- `CORS_ALLOWED_ORIGINS` → โดเมนใหม่

- [ ] **Step 2: ตรวจว่าตั้งครบ**

```bash
npx wrangler secret list
```

Expected: เห็นครบ 18 ชื่อ

- [ ] **Step 3: เพิ่ม cron trigger ใน wrangler.jsonc**

เพิ่มต่อท้าย block เดิม:

```jsonc
  "triggers": {
    "crons": ["*/30 * * * *"]
  }
```

ทุก 30 นาที สั้นกว่า `sleepAfter = "1h"` มากพอที่ container จะไม่หลับจนพลาดรอบ cleanup ที่ ticker ใน Go รันอยู่

- [ ] **Step 4: เพิ่ม scheduled handler ใน worker/index.ts**

เพิ่มเข้าไปใน `export default` ต่อจาก `fetch`:

```ts
  /**
   * Keeps the container awake. The daily retention cleanup runs as an in-process
   * ticker inside Go (main.go:112), which only fires while the container is
   * running — this ping is what guarantees that. It deliberately does not call
   * cleanup itself, so no Go code has to change.
   */
  async scheduled(_controller: ScheduledController, env: Env, ctx: ExecutionContext) {
    ctx.waitUntil(
      getContainer(env.BACKEND)
        .fetch(new Request("https://container.local/health"))
        .then((res) => console.log("keep-alive ping", res.status))
        .catch((error) => console.error("keep-alive ping failed", error)),
    );
  },
```

- [ ] **Step 5: ทดสอบ scheduled handler**

```bash
npx wrangler dev --test-scheduled
```

อีก terminal:

```bash
curl "http://localhost:8787/cdn-cgi/handler/scheduled?cron=*%2F30+*+*+*+*"
```

Expected: terminal ของ `wrangler dev` พิมพ์ `keep-alive ping 200`

- [ ] **Step 6: deploy แล้วยิงจริง**

```bash
npm run deploy
curl -sI https://keep-px.<subdomain>.workers.dev/health
```

Expected: HTTP 200

- [ ] **Step 7: Commit**

```bash
git add wrangler.jsonc worker/index.ts
git commit -m "feat(worker): cron keep-alive so the Go cleanup ticker keeps firing

cleanup รายวันเป็น ticker ใน Go ซึ่งทำงานได้เฉพาะตอน container ตื่น
cron ping /health ทุก 30 นาที สั้นกว่า sleepAfter 1h — แก้ปัญหาโดยไม่แตะ Go"
```

---

## Task 6: CI/CD

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `frontend/playwright.config.ts`

**Interfaces:**
- Consumes: `npm run deploy` จาก Task 2 · Worker ที่ deploy ได้จาก Task 5

- [ ] **Step 1: ลบ guard nginx แล้วใส่ guard ใหม่แทน**

ใน `.github/workflows/ci.yml` ลบ step `guard nginx client IP forwarding` (ราว ๆ บรรทัด 125-131) ทั้ง step แล้วเพิ่ม step ใหม่เข้าไปใน job ของ backend:

```yaml
      - name: guard client IP header
        run: |
          if ! grep -q 'ClientIPFromHeader("CF-Connecting-IP")' backend/internal/router/router.go; then
            echo "::error::router.go must resolve the client IP from CF-Connecting-IP. Any other header can be forged now that nginx is gone, and a wrong value silently breaks Meta CAPI attribution. See docs/superpowers/specs/2026-08-12-cloudflare-migration-design.md"
            exit 1
          fi
```

- [ ] **Step 2: เปลี่ยน deploy-verify ให้ deploy เอง**

แทนที่ job `deploy-verify` ทั้ง job ด้วย:

```yaml
  deploy:
    if: github.event_name == 'push' && github.ref == 'refs/heads/main' && needs.ci-gate.result == 'success'
    needs: ci-gate
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: 22

      - run: npm ci
      - run: npm ci --prefix frontend

      - name: Deploy to Cloudflare
        run: npm run deploy
        env:
          CLOUDFLARE_API_TOKEN: ${{ secrets.CLOUDFLARE_API_TOKEN }}
          CLOUDFLARE_ACCOUNT_ID: ${{ secrets.CLOUDFLARE_ACCOUNT_ID }}

      - name: Verify health
        run: |
          URL="${{ vars.PROD_URL }}/health"
          for i in $(seq 1 10); do
            STATUS=$(curl -s -o /dev/null -w '%{http_code}' --max-time 10 "$URL" || echo "000")
            if [ "$STATUS" = "200" ]; then
              echo "Backend is healthy"
              exit 0
            fi
            echo "Attempt $i/10: HTTP $STATUS — retrying in 15s..."
            sleep 15
          done
          echo "Health check failed after 10 attempts"
          exit 1

      - name: Verify security headers
        run: |
          H=$(curl -sI "${{ vars.PROD_URL }}/")
          for header in X-Content-Type-Options X-Frame-Options Referrer-Policy \
                        Permissions-Policy Strict-Transport-Security \
                        Cross-Origin-Opener-Policy Content-Security-Policy; do
            echo "$H" | grep -qi "^$header:" || { echo "::error::missing $header"; exit 1; }
          done
          echo "All 7 security headers present"
```

`wrangler deploy` จบเมื่อ deploy จริงเสร็จ จึงไม่ต้อง `sleep 60` เดาเวลาแบบเดิมอีก

- [ ] **Step 3: อัปเดต post-deploy-e2e**

ใน job `post-deploy-e2e` เปลี่ยน `needs: deploy-verify` เป็น `needs: deploy` และเงื่อนไข `needs.deploy-verify.result` เป็น `needs.deploy.result` แล้วเปลี่ยน env `E2E_BASE_URL` ให้ชี้ `${{ vars.PROD_URL }}`

- [ ] **Step 4: ตั้งค่า GitHub secrets และ variables**

ที่ Settings → Secrets and variables → Actions:
- secret `CLOUDFLARE_API_TOKEN` — token ที่มีสิทธิ์ Workers Scripts:Edit
- secret `CLOUDFLARE_ACCOUNT_ID`
- variable `PROD_URL` — โดเมนใหม่
- ลบ variable `BACKEND_PROD_URL` และ `FRONTEND_PROD_URL` ที่ไม่ใช้แล้ว

- [ ] **Step 5: ตรวจ syntax ของ workflow**

```bash
npx --yes @action-validator/cli --verbose .github/workflows/ci.yml 2>/dev/null \
  || python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/ci.yml')); print('YAML ถูกต้อง')"
```

Expected: ไม่มี error

- [ ] **Step 6: Commit แล้ว push ดูว่า CI เขียว**

```bash
git add .github/workflows/ci.yml frontend/playwright.config.ts
git commit -m "ci: deploy with wrangler instead of waiting for Railway

wrangler deploy จบเมื่อ deploy เสร็จจริง ไม่ต้อง sleep 60 เดาเวลา
guard nginx ถูกแทนด้วย guard ที่เช็กว่า router.go ใช้ CF-Connecting-IP
และเพิ่มการตรวจ security headers ครบ 7 ตัวหลัง deploy"
git push
```

Expected: CI ผ่านทุก job

---

## Task 7: Cleanup — ลบของเก่าทิ้ง

ทำหลัง Task 6 เท่านั้น เพราะต้องมั่นใจก่อนว่าของใหม่ทำงานได้จริง

**Files:**
- Delete: `frontend/Dockerfile` · `frontend/nginx.conf` · `frontend/10-set-dns-resolver.envsh` · `frontend/railway.json` · `backend/railway.json`
- Delete: `.claude/skills/railway-deploy/` · `.claude/skills/nginx-csp/`
- Modify: `backend/internal/handler/health.go:72`
- Modify: `.claude/skills/ci-pipeline/SKILL.md` · `.claude/skills/auth-flow/SKILL.md` · `.claude/skills/stripe-webhook/SKILL.md`
- Create: `.claude/skills/cloudflare-deploy/SKILL.md` · `.claude/skills/worker-headers/SKILL.md`

- [ ] **Step 1: ลบไฟล์ config ที่ตายแล้ว**

```bash
git rm frontend/Dockerfile frontend/nginx.conf frontend/10-set-dns-resolver.envsh \
       frontend/railway.json backend/railway.json
```

- [ ] **Step 2: ลบ binary ที่ค้างเครื่อง**

ไฟล์สองตัวนี้ถูก gitignore อยู่แล้ว (`backend/.gitignore` บรรทัด 3 และ 5) จึงไม่กระทบ git — ลบแค่ในเครื่อง

```bash
rm -f backend/server backend/service.test
```

- [ ] **Step 3: ตรวจว่าไม่มีอ้างอิงค้าง**

```bash
grep -ril "railway" --exclude-dir={node_modules,.git,dist,test-results,playwright-report} . | sort
```

Expected: เหลือแค่ไฟล์ใน `docs/superpowers/` (บันทึกประวัติ เก็บไว้ตั้งใจ) และ `.claude/skills/` ที่จะแก้ในขั้นถัดไป

- [ ] **Step 4: แก้ comment ใน health.go**

`backend/internal/handler/health.go` บรรทัด 72 เปลี่ยนจาก:

```go
// Use this for Railway/Kubernetes liveness probes; use /health for readiness with DB ping.
```

เป็น:

```go
// Use this for Cloudflare Containers liveness probes; use /health for readiness with DB ping.
```

- [ ] **Step 5: รัน Go test ทั้งหมดยืนยันว่าไม่พัง**

```bash
cd backend && go build ./... && go test ./internal/... 2>&1 | tail -20
```

Expected: build ผ่าน · test ผ่านทุก package

- [ ] **Step 6: เขียน skill ใหม่แทนของเก่า**

```bash
git rm -r .claude/skills/railway-deploy .claude/skills/nginx-csp
mkdir -p .claude/skills/cloudflare-deploy .claude/skills/worker-headers
```

`.claude/skills/cloudflare-deploy/SKILL.md` — เนื้อหาต้องครอบคลุม: โครงสร้าง `wrangler.jsonc`, ความหมายของ `run_worker_first`, เหตุผลที่ `max_instances` ต้องเป็น 1, วิธี deploy (`npm run deploy`), วิธีดู log (`wrangler tail`), วิธีจัดการ secret (`wrangler secret put/list`) พร้อม frontmatter:

```markdown
---
name: cloudflare-deploy
description: Deploy และ debug keep-px บน Cloudflare Workers + Containers — ใช้เมื่อ deploy พัง, deploy ไม่ขึ้น, container ไม่ตื่น, 502, wrangler error, secret หาย, cold start ช้า
---
```

`.claude/skills/worker-headers/SKILL.md` — เนื้อหาต้องครอบคลุม: headers ทั้ง 2 ชุดอยู่ที่ `worker/headers.ts`, เหตุผลที่ sale page ไม่มี `X-Frame-Options`, วิธีเพิ่มโดเมนใหม่เข้า CSP, วิธีทดสอบ (`npx vitest run worker/headers.test.ts`) พร้อม frontmatter:

```markdown
---
name: worker-headers
description: จัดการ CSP และ security headers ใน Worker — ใช้เมื่อ CSP error, blocked by CSP, script ถูก block, font ไม่ขึ้น, หน้าเว็บโหลดไม่ครบ, เพิ่มโดเมนใหม่เข้า CSP
---
```

- [ ] **Step 7: อัปเดต skill ที่อ้าง Railway/nginx**

แก้เนื้อหาใน 3 ไฟล์ให้ตรงกับสถาปัตยกรรมใหม่:
- `.claude/skills/ci-pipeline/SKILL.md` — job `deploy-verify` เปลี่ยนชื่อเป็น `deploy` และ CI เป็นคน deploy เองด้วย wrangler
- `.claude/skills/auth-flow/SKILL.md` — ที่อ้าง nginx proxy เปลี่ยนเป็น Worker routing
- `.claude/skills/stripe-webhook/SKILL.md` — ที่อ้าง Railway URL เปลี่ยนเป็นโดเมนใหม่

- [ ] **Step 8: ตรวจรอบสุดท้าย**

```bash
grep -ril "railway\|nginx" --exclude-dir={node_modules,.git,dist,test-results,playwright-report} . | sort
```

Expected: เหลือเฉพาะไฟล์ใน `docs/superpowers/plans/` และ `docs/superpowers/specs/` เท่านั้น

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "chore: remove Railway and nginx leftovers

ลบ frontend/Dockerfile, nginx.conf, 10-set-dns-resolver.envsh และ
railway.json ทั้งสองไฟล์ — ย้ายไป Worker หมดแล้ว

skill railway-deploy และ nginx-csp ถูกแทนด้วย cloudflare-deploy
และ worker-headers

เอกสารใน docs/superpowers เก็บไว้ตามเดิม เป็นบันทึกว่าทำไมถึงเคย
ตัดสินใจแบบนั้น"
```

---

## Task 8: Cutover — ย้าย DNS แล้วปิด Railway

**Files:** ไม่มีไฟล์ในโค้ด — เป็นงาน config บน dashboard

- [ ] **Step 1: เพิ่มโดเมนเข้า Cloudflare**

เพิ่ม `pixlinks.xyz` เข้าบัญชี Cloudflare แล้วเปลี่ยน nameserver ที่ registrar ตามที่ Cloudflare บอก

- [ ] **Step 2: รอ DNS propagate แล้วตรวจ**

```bash
dig +short NS pixlinks.xyz
```

Expected: เห็น nameserver ของ Cloudflare

- [ ] **Step 3: ผูก custom domain เข้า Worker**

ที่ Cloudflare dashboard → Workers & Pages → keep-px → Settings → Domains & Routes → Add custom domain → `pixlinks.xyz`

- [ ] **Step 4: อัปเดต secret ที่มีโดเมนอยู่ข้างใน**

```bash
npx wrangler secret put BASE_URL              # https://pixlinks.xyz
npx wrangler secret put FRONTEND_URL          # https://pixlinks.xyz
npx wrangler secret put CORS_ALLOWED_ORIGINS  # https://pixlinks.xyz
npm run deploy
```

- [ ] **Step 5: อัปเดต callback URL ที่ผู้ให้บริการภายนอก**

- Google Cloud Console → OAuth client → Authorized JavaScript origins และ redirect URIs → โดเมนใหม่
- Stripe Dashboard → Webhooks → endpoint URL → `https://pixlinks.xyz/api/v1/billing/webhook`

**ระวัง:** ถ้า Stripe webhook secret เปลี่ยนตอนสร้าง endpoint ใหม่ ต้อง `wrangler secret put STRIPE_WEBHOOK_SECRET` ด้วยค่าใหม่ แล้ว deploy ซ้ำ

- [ ] **Step 6: ตรวจเกณฑ์ความสำเร็จทั้ง 8 ข้อจาก spec**

```bash
# 1 + 3: เสิร์ฟจาก Cloudflare และ health/ready ตอบ
curl -sI https://pixlinks.xyz/health | head -1
curl -sI https://pixlinks.xyz/ready | head -1
curl -sI https://pixlinks.xyz/ | grep -i '^server:'

# 2: security headers ครบ
curl -sI https://pixlinks.xyz/ | grep -icE 'x-content-type-options|x-frame-options|referrer-policy|permissions-policy|strict-transport-security|cross-origin-opener-policy|content-security-policy'

# 4: E2E
E2E_BASE_URL=https://pixlinks.xyz npm run e2e --prefix frontend

# 5: Go test
cd backend && go test ./internal/... 2>&1 | tail -5
```

Expected: `/health` และ `/ready` = 200 · `server: cloudflare` · headers นับได้ 7 · E2E 31 spec ผ่าน · Go test ผ่าน

- [ ] **Step 7: ตรวจ client IP ที่บันทึกจริง (เกณฑ์ข้อ 6)**

ยิง event เข้าไปหนึ่งตัวด้วย API key จริง แล้วเปิด dashboard ดูรายการ event ล่าสุด ตรวจว่า IP ที่บันทึกคือ IP จริงของเครื่องที่ยิง (`curl -s ifconfig.me`) ไม่ใช่ IP ภายในของ Cloudflare

**นี่คือเกณฑ์ที่สำคัญที่สุด** — ถ้าผิดแปลว่า Task 1 ยังไม่ครบ อย่าปิด Railway จนกว่าข้อนี้จะผ่าน

- [ ] **Step 8: ตรวจ cron (เกณฑ์ข้อ 7)**

```bash
npx wrangler tail keep-px --format pretty
```

รอถึงรอบ 30 นาที แล้วดูว่ามี `keep-alive ping 200` โผล่ใน log

- [ ] **Step 9: ปิด Railway**

หลังทุกข้อผ่านหมดและระบบเดินได้อย่างน้อย 24 ชั่วโมง จึงลบ service ทั้งสองตัวออกจาก Railway

- [ ] **Step 10: อัปเดตบันทึกความจำของโปรเจกต์**

แก้ `/Users/jaochai/.claude/projects/-Users-jaochai-Code-keep-px/memory/` — ไฟล์ `project_local-env-points-at-production.md` และ `MEMORY.md` ให้สะท้อนว่า deploy อยู่บน Cloudflare แล้ว ไม่ใช่ Railway

---

## ผล Spike (2026-08-12) — ✅ ผ่าน

**คำถามที่ spike ตอบ: Go + pgx pool + migration-on-boot รันบน Cloudflare Containers ได้ไหม → ได้**

| วัด | ผล |
|---|---|
| `/health` cold start | HTTP 200 · **9.5 วิ** · `{"status":"ok","db":"ok","pool":{"total":4,"idle":4,"max":20}}` |
| `/health` warm | HTTP 200 · **1.09 วิ** |
| `/ready` | HTTP 200 · 1.8 วิ |
| migration | รันผ่าน — log: `migrations: already up to date` |
| memory `basic` (1 GiB) | พอ · ไม่มี OOM |
| boot ในเครื่อง (Docker) | **11 วิ** — DB connect + migration 5.2 วิ · token migration 1 วิ |

deploy: `keep-px-spike` · image `keep-px-spike-backend@sha256:087aa4a7…` · instance type `basic` (0.25 vCPU / 1 GiB / 4 GB)

### 3 อุปสรรคที่เจอ และคำตอบ — ต้องเอาเข้า Task 4 ทั้งหมด

**1. Cloudflare error 1042** — ส่ง request เดิมเข้า `container.fetch()` ตรง ๆ ไม่ได้
URL สาธารณะของ Worker ทำให้ runtime วนกลับหาตัวเอง (*"Worker tried to fetch from another Worker on the same zone"*)
→ ต้อง rewrite origin เป็น `http://container` ก่อนเสมอ (ฟังก์ชัน `toContainerRequest`)

**2. SDK ตัด timeout เร็วเกินไป** — `TIMEOUT_TO_GET_CONTAINER_MS = 8_000` แต่ boot จริงใช้ 11 วิ
อาการ: request ล้มที่ 8.5 วิ ด้วย `Failed to start container: The container is not running`
→ ต้อง override `fetch` แล้วเรียก `startAndWaitForPorts` ด้วย `instanceGetTimeoutMS: 60_000` และ `portReadyTimeoutMS: 180_000`

**3. `TOKEN_ENCRYPTION_KEY` ขาด = container ตายเงียบ** — `router.go:57` เรียก `os.Exit(1)`
Worker มองเห็นแค่ `internal error connecting to the port` ซึ่ง**ไม่บอกสาเหตุจริงเลย**
วิธีหาต้นเหตุที่ได้ผล: `docker run` image เดียวกันในเครื่องแล้วอ่าน log ตรง ๆ
→ ตรวจแล้ว: production มี `os.Exit` จุดเดียวนี้เท่านั้น

### ⚠️ คำเตือนสำหรับ Task 5 (secret จริง)

**`TOKEN_ENCRYPTION_KEY` ต้องเป็นค่าเดิมจาก Railway เท่านั้น ห้ามสร้างใหม่**
`fb_access_token` ใน DB ถูกเข้ารหัสด้วย key เดิม ถ้าใช้ key ใหม่จะถอดรหัสไม่ออก = pixel ทุกตัวส่ง CAPI ไม่ได้
(spike ใช้ key ชั่วคราวได้เพราะ `migrateTokens` ข้าม token ที่มี prefix `enc:` อยู่แล้ว ไม่ไปแตะของเดิม)

### เบ็ดเตล็ด

- ต้องมี `tsconfig.json` ที่ root ไม่งั้น editor หา `Env` / `ExportedHandler` ไม่เจอ
  typecheck ใช้ `./frontend/node_modules/.bin/tsc` ได้ ไม่ต้องลง TypeScript ที่ root
- `wrangler types` ต้องรันใหม่ทุกครั้งที่แก้ binding หรือเพิ่ม secret ใน `.dev.vars`
