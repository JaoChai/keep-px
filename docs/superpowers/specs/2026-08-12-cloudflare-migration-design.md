# ย้าย stack ไป Cloudflare — Design

วันที่: 2026-08-12
สถานะ: รอ review

## 1. เป้าหมายและขอบเขต

ย้ายทุกอย่างที่รันอยู่บน Railway ไป Cloudflare ก่อนเปิดใช้งานโปรเจกต์จริง

เป้าหมายที่ผู้ใช้ระบุ: รวม vendor ให้เหลือเจ้าเดียว · หนีปัญหา Railway · ลดค่าใช้จ่าย · ได้ประโยชน์จาก edge

**อยู่ในขอบเขต:** frontend, backend, งานเบื้องหลัง, secrets, CI/CD, DNS, และการล้างไฟล์เก่าที่ตายแล้ว

**ไม่อยู่ในขอบเขต:** ฐานข้อมูล — Neon Postgres (`us-east-1`) คงเดิม ตามการตัดสินใจของผู้ใช้

### ทำไมไม่ย้าย DB ไป D1

Cloudflare มี DB ตัวเดียวคือ D1 (SQLite) ซึ่งบังคับห่วงโซ่ทั้งหมด: D1 ต่อได้จาก Worker binding เท่านั้น → backend ต้องเป็น Worker → Worker รัน Go ไม่ได้ → ต้องเขียนใหม่ทั้ง 12,491 บรรทัด

และ D1 ชนกับตัวสินค้าโดยตรง:
- เพดาน 10 GB ต่อ database — `pixel_events` มี JSONB สองคอลัมน์ต่อแถว
- D1 เขียนแบบ single-threaded และ docs ระบุว่าเหมาะกับงาน read-heavy — แต่ event ingest คืองานเขียนล้วน
- D1 ไม่มี interactive transaction (มีแค่ `batch()`) — โค้ดตัดเครดิตใช้ `FOR UPDATE` อยู่ 3 จุด

สรุป: ต้นทุนสูง ความเสี่ยงสูง ประโยชน์ต่ำ → ไม่ทำ

## 2. สถาปัตยกรรม

### ก่อน
```
Railway ├─ frontend service (Nginx + React SPA, proxy /api และ /p ไป backend)
        └─ backend service (Go)
Neon Postgres (us-east-1) · Cloudflare R2
```

### หลัง
```
              pixlinks.xyz — DNS + CDN + WAF ของ Cloudflare
                            │
                    ┌───────▼────────┐
                    │  Worker (TS)   │  router + security headers + scheduled()
                    └───┬────────┬───┘
             /  /assets/│        │/api/*  /p/*  /health  /ready
                        │        │
            ┌───────────▼──┐  ┌──▼────────────────────────┐
            │ Static Assets│  │ Container: Go เดิม 100%   │
            │  React SPA   │  │ (backend/Dockerfile เดิม) │
            └──────────────┘  └──┬─────────────────┬──────┘
   Cron Trigger ─────────────────┘                 │
   (cleanup รายวัน)                        ┌───────▼─────┐  ┌────────┐
                                           │Neon Postgres│  │   R2   │
                                           └─────────────┘  └────────┘
```

โค้ด Go ไม่ต้องแก้ ยกเว้นจุดเดียวในข้อ 4.3

## 3. การตัดสินใจที่ตรวจสอบกับเอกสารแล้ว

ทุกข้อยืนยันจาก Cloudflare docs ผ่าน Context7 แล้ว

| เรื่อง | สิ่งที่ยืนยันได้ |
|---|---|
| Worker เสิร์ฟทั้ง static และ container ได้ | `assets.run_worker_first` รับ array ของ path pattern + `not_found_handling: "single-page-application"` |
| ส่งทุก request เข้า container ตัวเดียว | `getContainer(env.BACKEND)` เรียกโดยไม่ใส่ id = singleton |
| ตั้งค่า container | `defaultPort`, `sleepAfter`, `envVars`, `pingEndpoint`, hooks `onStart`/`onStop`/`onError` |
| ส่ง secret เข้า container | `import { env } from "cloudflare:workers"` แล้ว `envVars = { JWT_SECRET: env.JWT_SECRET }` |
| cron | `triggers.crons` ใน wrangler + `scheduled(controller, env, ctx)` handler |
| ข้อจำกัด container | `max_instances` default 20 · instance types: lite / basic / standard-1..4 |

### ค่าที่เลือก

- `instance_type = "basic"` (1/4 vCPU, 1 GiB, 4 GB disk) — พอสำหรับ pgx pool 20 conn ขยับขึ้นได้ถ้าไม่พอ
- `max_instances = 1` — **บังคับ** เพราะสามเหตุผล: migration รันตอน boot (`main.go:78`), rate limiter เก็บใน memory (`middleware/ratelimit.go`), sale page cache เก็บใน memory (`service/sale_page_cache.go`)
- `pingEndpoint = "health"` — backend มี `/health` อยู่แล้ว (`router.go:107`)
- `sleepAfter = "1h"` — ต้องยาวกว่ารอบ replay ที่นานที่สุด ปรับได้หลังวัดจริง

### ค่าใช้จ่ายที่ประเมินไว้

Workers Paid $5 + Container `basic` ประมาณ $7-8 = **~$12-13/เดือน**

คำนวณจากเรทใน docs (memory $0.0000025/GiB-s, CPU $0.00002/vCPU-s, disk $0.00000007/GB-s) สมมติรัน 24/7 หักส่วนที่แถมแล้ว — **ยังไม่ได้ยืนยันกับบิลจริง**

## 4. งานที่ต้องทำ

### 4.1 Frontend → Workers Static Assets

`frontend/src/lib/api.ts:5` เขียนว่า `import.meta.env.VITE_API_URL || ''` → เมื่อทุกอย่างอยู่ origin เดียวกัน ตั้ง `VITE_API_URL` เป็นค่าว่าง แล้ว **โค้ด frontend ไม่ต้องแก้เลยสักบรรทัด**

ย้าย security headers จาก `nginx.conf` ไป Worker — 7 ตัว (`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Permissions-Policy`, `Strict-Transport-Security`, `Cross-Origin-Opener-Policy`, `Content-Security-Policy`) แยกเป็น 2 ชุดเหมือนเดิม: dashboard และ sale page

CSP แก้จุดเดียว: `connect-src 'self' ${BACKEND_URL} ...` → `connect-src 'self' ...` เพราะ backend กลายเป็น origin เดียวกับ frontend

### 4.2 Backend → Cloudflare Container

`backend/Dockerfile` ใช้ได้ตามเดิม — `EXPOSE 8080` อยู่แล้ว, multi-stage build, non-root user, `COPY db/migrations` ครบ

เขียนใหม่: `worker/index.ts` — Container class + fetch handler + scheduled handler

### 4.3 แก้ client IP — จุดเสี่ยงสูงสุดของงานนี้

`backend/internal/router/router.go:30` ปัจจุบัน:
```go
r.Use(chimiddleware.ClientIPFromHeader("X-Real-IP"))
```

`X-Real-IP` ถูกเซ็ตโดย Nginx (`nginx.conf` บรรทัด proxy_set_header) ซึ่งรับค่าต่อจาก edge proxy ของ Railway พอทั้งสองอย่างหายไป header นี้จะไม่มีใครเซ็ตให้

**ต้องเปลี่ยนเป็น `CF-Connecting-IP`** ซึ่ง Cloudflare เซ็ตให้เสมอและ client ปลอมไม่ได้

ทำไมถึงสำคัญ: client IP ถูกส่งต่อเข้า Meta CAPI เพื่อจับคู่ผู้ใช้ ถ้าค่าผิดจะไม่มี error ให้เห็น แต่ attribution จะพังเงียบ ๆ — และนั่นคือหัวใจของสินค้า

Worker ต้องลบ header ที่ client ส่งมาเองทิ้งก่อนส่งต่อเข้า container ด้วย เพื่อรักษาพฤติกรรมที่ commit `57c23ef` ตั้งใจไว้ (`fix(security): stop trusting client-supplied IP headers`)

ต้องแก้ `backend/internal/router/clientip_test.go` ตามด้วย

### 4.4 งานเบื้องหลัง 3 ตัว

| งาน | ตอนนี้ | หลังย้าย |
|---|---|---|
| cleanup รายวัน | goroutine + ticker (`main.go:112`) | **คงโค้ด Go เดิมไว้** · Cron Trigger ทำหน้าที่ปลุก container ให้ตื่น ไม่ได้เรียก cleanup เอง |
| replay | goroutine ยาว batch ละ 1,000 events (`replay_service.go:250`) | `sleepAfter = "1h"` · มี `RecoverOrphanedSessions` (`main.go:85`) กู้ให้อยู่แล้วถ้าถูกฆ่า |
| rate limiter + sale page cache | in-memory | ปลอดภัยเพราะ `max_instances = 1` |

เหตุผลที่เลือกให้ Cron แค่ปลุก container แทนที่จะย้าย cleanup ออกมาเป็น endpoint: ticker ใน Go จะทำงานได้ก็ต่อเมื่อ container ตื่นอยู่ การให้ Cron ping `/health` ทุกช่วงสั้นกว่า `sleepAfter` แก้ปัญหาได้ตรงจุดโดย**ไม่ต้องแตะโค้ด Go เลย** ถ้าวัดแล้วพบว่า container ยังหลับจนพลาดรอบ cleanup ค่อยเปลี่ยนไปทำเป็น endpoint แยก

### 4.5 Secrets

ย้ายจาก Railway ไป Worker secrets แล้วส่งต่อเข้า container ผ่าน `envVars`

รายการจาก `backend/internal/config/config.go`: `DATABASE_URL`, `JWT_SECRET`, `TOKEN_ENCRYPTION_KEY`, `GOOGLE_CLIENT_ID`, `S3_ENDPOINT`, `S3_BUCKET`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_PUBLIC_URL`, `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET`, `STRIPE_PUBLISHABLE_KEY`, `STRIPE_PRICE_PIXEL_SLOT`, `STRIPE_PRICE_REPLAY_SINGLE`, `STRIPE_PRICE_REPLAY_MONTHLY`, `BASE_URL`, `FRONTEND_URL`, `CORS_ALLOWED_ORIGINS`

`CORS_ALLOWED_ORIGINS` อาจไม่จำเป็นอีกเมื่อเป็น same-origin แต่ยังต้องตั้งไว้สำหรับ sale page ที่ฝังบนโดเมนลูกค้า — ตรวจตอนลงมือ

### 4.6 CI/CD

`.github/workflows/ci.yml`:
- ลบ guard `nginx client IP forwarding` (บรรทัด ~125-131) — ไฟล์ที่มันเช็กจะไม่มีแล้ว **แทนที่ด้วย guard ใหม่ที่เช็กว่า `router.go` ใช้ `CF-Connecting-IP`** เพื่อกันการถอยกลับ
- `deploy-verify` (บรรทัด ~158) — ตอนนี้ `sleep 60` รอ Railway auto-deploy · เปลี่ยนเป็น `wrangler deploy` ที่ CI สั่งเอง แล้วเช็ก health หลัง deploy จบจริง ไม่ต้องเดาเวลา
- `post-deploy-e2e` (บรรทัด ~203) — ปรับ URL

`frontend/playwright.config.ts:23` ใช้ `E2E_BASE_URL` — ต้องชี้โดเมนใหม่

### 4.7 DNS

ย้าย nameserver ของ `pixlinks.xyz` มา Cloudflare

## 5. Cleanup — ของเก่าที่ต้องล้าง

ตรวจจาก repo จริงแล้ว git tree สะอาด ไม่มีไฟล์ค้าง

### ลบทิ้ง — ตาย 100%

| ไฟล์ | ขนาด | เหตุผล |
|---|---|---|
| `frontend/Dockerfile` | 23 บรรทัด | Nginx image ไม่ใช้แล้ว |
| `frontend/nginx.conf` | 100 บรรทัด | ย้ายไป Worker ทั้งไฟล์ |
| `frontend/10-set-dns-resolver.envsh` | 6 บรรทัด | hack แก้ปัญหา DNS ของ Railway โดยเฉพาะ |
| `frontend/railway.json` | 11 บรรทัด | Railway config |
| `backend/railway.json` | 11 บรรทัด | Railway config |
| `backend/server` | 39 MB | binary ค้างเครื่อง (gitignore แล้ว) |
| `backend/service.test` | 12 MB | binary ค้างเครื่อง (gitignore แล้ว) |

### เก็บไว้ แต่ต้องแก้

- `backend/Dockerfile` — **เก็บ** Cloudflare Containers ใช้ไฟล์นี้
- `frontend/error-pages/{502,503,504}.html` — **เก็บ** เดิม nginx เสิร์ฟผ่าน `error_page` ตอนนี้ให้ Worker เสิร์ฟแทนเมื่อ container ไม่ตอบ
- `backend/internal/handler/health.go:72` — comment อ้าง Railway ต้องอัปเดตข้อความ

### Skills ที่ตายแล้ว (`.claude/skills/`)

- `railway-deploy/` — ตายสนิท → เขียนใหม่เป็น `cloudflare-deploy`
- `nginx-csp/` — ตายสนิท → เขียนใหม่เป็น `worker-headers` (CSP ยังอยู่ แค่ย้ายที่อยู่)
- `ci-pipeline/SKILL.md`, `auth-flow/SKILL.md`, `stripe-webhook/SKILL.md` — อ้าง Railway/nginx ต้องอัปเดตเนื้อหา

### เก็บไว้ ไม่แตะ

`docs/superpowers/plans/` และ `docs/superpowers/specs/` 3 ไฟล์ที่อ้าง Railway เป็นบันทึกประวัติว่าทำไมถึงตัดสินใจแบบนั้นในอดีต การลบทิ้งจะทำให้เสียเหตุผลเบื้องหลัง

## 6. จุดเสี่ยง

| ความเสี่ยง | ผลถ้าเกิด | รับมือ |
|---|---|---|
| ลืมแก้ client IP | Meta CAPI attribution พังเงียบ ไม่มี error | guard ใน CI + test |
| Go + pgx pool + migration-on-boot บน Containers | ยังไม่เคยพิสูจน์ | ทำ spike ก้อน 4.2 ก่อนงานอื่น |
| container หลับแล้ว replay ตาย | replay session ค้าง | `sleepAfter` ยาว + `RecoverOrphanedSessions` ที่มีอยู่แล้ว |
| cold start | request แรกช้า | วัดจริงหลัง spike |
| ค่าใช้จ่ายจริงต่างจากที่ประเมิน | บิลเกินคาด | ดูบิลจริงเดือนแรก |

## 7. เกณฑ์ความสำเร็จ

1. `pixlinks.xyz` เสิร์ฟจาก Cloudflare ทั้งหมด ไม่มี service ไหนเหลือบน Railway
2. security headers ทั้ง 7 ตัวและ CSP ทั้ง 2 ชุดออกมาเหมือนเดิม — ตรวจด้วย `curl -I` เทียบกับ `nginx.conf` เดิม
3. `/health` และ `/ready` ตอบ 200 ผ่าน Worker
4. E2E 31 spec ผ่านบนโดเมนใหม่
5. Go test 29 ไฟล์ 201 function ยังผ่านหมด
6. event ที่ ingest เข้ามาบันทึก IP ผู้ใช้จริง ไม่ใช่ IP ของ proxy — ยืนยันด้วย test
7. cleanup cron ทำงานตามกำหนด
8. ไฟล์ในข้อ 5 ถูกลบครบ ไม่มีอ้างอิงค้าง
