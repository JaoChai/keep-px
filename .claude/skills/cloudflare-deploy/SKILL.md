---
name: cloudflare-deploy
description: Deploy และ debug keep-px บน Cloudflare Workers + Containers — ใช้เมื่อ deploy พัง, deploy ไม่ขึ้น, container ไม่ตื่น, 502, wrangler error, secret หาย, cold start ช้า
---

# Cloudflare Deploy

## When to Activate

- "Deploy failed" / "deploy ไม่ขึ้น" / "wrangler error"
- "502" / "container ไม่ตื่น" / "Backend unavailable"
- "Cold start ช้า" / "request แรกพัง แล้วดีเอง"
- "Secret หาย" / "env var ไม่ถึง container"
- "Worker tried to fetch from another Worker" (Cloudflare error 1042)

## Architecture

keep-px รันบน Cloudflare เดียวจบ: **Worker** เป็นประตูรับทุก request, **Container** รัน Go backend

```
Request → Worker (worker/index.ts)
          ├─ backend path (/api/, /p/, /health, /ready) → Container (Go)
          └─ path อื่น → static asset (frontend/dist) ผ่าน ASSETS binding
         ทุก response ผ่าน applySecurityHeaders ก่อนออก
```

| ไฟล์ | หน้าที่ |
|------|---------|
| `wrangler.jsonc` | config หลัก — Worker + Container + cron + assets |
| `worker/index.ts` | routing + Backend container class + keep-alive cron |
| `worker/headers.ts` | security headers (ดู skill `worker-headers`) |
| `backend/Dockerfile` | image ของ Container |

## โครงสร้าง wrangler.jsonc

- `"main": "worker/index.ts"` — entry point ของ Worker
- `"compatibility_date"` + `"compatibility_flags": ["nodejs_compat"]` — runtime compat ของ Worker
- `"assets"` — static SPA:
  - `"directory": "./frontend/dist"` — build output ของ frontend
  - `"binding": "ASSETS"` — เรียกในโค้ดผ่าน `env.ASSETS.fetch(...)`
  - `"not_found_handling": "single-page-application"` — fallback ไป index.html เพื่อ client-side routing
  - **`"run_worker_first": true`** — ทุก request รวม static asset ต้องวนผ่าน Worker ก่อน (ดีเทลด้านล่าง)
- `"containers"` — Container ของ Go backend:
  - `"class_name": "Backend"` — คลาสใน `worker/index.ts` ที่ extends `Container`
  - `"image": "./backend/Dockerfile"` — build จาก Dockerfile
  - `"max_instances": 1` — ต้องเป็น 1 เสมอ (ดีเทลด้านล่าง)
- `"durable_objects"` + `"migrations"` — `Backend` ถูก bind เป็น Durable Object ด้วย
- `"triggers": { "crons": ["*/30 * * * *"] }` — keep-alive ping ทุก 30 นาที

## run_worker_first คืออะไร

ปกติ Cloudflare เสิร์ฟ static asset ตรงจาก edge เร็วสุด แต่จะ **ข้าม Worker ทั้ง Worker** — หน้า `/` จะได้ security headers 0 จาก 7 ตัว (วัดจริง 2026-08-12)

`run_worker_first: true` บังคับให้ทุก request วนเข้า Worker ก่อน เพื่อให้ `applySecurityHeaders` ใส่ header ครบทุก response รวม static asset นี่คือเหตุผลที่การแยก backend/SPA อยู่ในโค้ด (`isBackendPath`) ไม่ใช่ใน config

ถ้าปิด flag นี้ หน้า dashboard จะไม่มี CSP / X-Frame-Options / HSTS เลย

## ทำไม max_instances ต้องเป็น 1

1. **`Backend` คือ Durable Object** — DO เป็น singleton by design มี instance เดียวต่อ ID (ตั้งใน `durable_objects.bindings`)
2. **Go process รัน in-process cleanup ticker** ที่ `cmd/server/main.go:112` — ticker ไฟขณะที่ container เปิดอยู่เท่านั้น ถ้ามี 2 instance = cleanup รันซ้อน 2 รอบ
3. **ไม่มี sharding logic** — `getContainer(env.BACKEND)` เรียก "the backend" ไม่ได้แยก ID ตาม request

ถ้าตั้ง >1 cleanup cron จะทำงานซ้อนกันและ DO binding จะ ambiguous

## Deploy

```bash
npm run deploy
```

รัน `npm run build && wrangler deploy` — build frontend ก่อน แล้ว deploy Worker + Container image ขึ้น Cloudflare

CI เป็นคน deploy เองหลัง merge main (ดู skill `ci-pipeline`)

## ดู log

```bash
npx wrangler tail          # real-time log ของ Worker
```

log จาก `worker/index.ts` ที่จะเห็น: `keep-alive ping 200`, `failed to reach backend container`, `container error`

## จัดการ secret

Container env var ทั้งหมดมาจาก `env.*` ของ Worker (ดู `envVars` ใน `worker/index.ts`) ดังนั้นต้องตั้ง secret ที่ **Worker** ไม่ใช่ที่ container:

```bash
npx wrangler secret put DATABASE_URL
npx wrangler secret put JWT_SECRET
npx wrangler secret list                    # ดู secret ทั้งหมดที่ตั้งไว้
```

secret ที่ envVars อ้างอิง (ต้องครบ): `DATABASE_URL`, `JWT_SECRET`, `TOKEN_ENCRYPTION_KEY`, `GOOGLE_CLIENT_ID`, `S3_ENDPOINT`, `S3_BUCKET`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_PUBLIC_URL`, `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET`, `STRIPE_PUBLISHABLE_KEY`, `STRIPE_PRICE_PIXEL_SLOT`, `STRIPE_PRICE_REPLAY_SINGLE`, `STRIPE_PRICE_REPLAY_MONTHLY`, `BASE_URL`, `FRONTEND_URL`, `CORS_ALLOWED_ORIGINS`

⚠️ `TOKEN_ENCRYPTION_KEY` ถ้าหาย container จะ `os.Exit(1)` ทันทีตอน boot (router.go:57) — error จะออกมาเป็น "port connection error" จาก Worker ไม่ใช่สาเหตุจริง

## Debug: 502 / container ไม่ตื่น

```
502 "Backend unavailable"?
  → wrangler tail ดู "failed to reach backend container"
  → cold start ช้าเกิน timeout ไหม (ดูด้านล่าง)
  → secret ครบไหม (wrangler secret list) — โดยเฉพาะ TOKEN_ENCRYPTION_KEY

Cold start พัง?
  → SDK default ให้ container 8s ตื่น + 20s port แต่ Go image ต้องรัน 27 migrations
    + ping Neon + recover replay sessions + re-encrypt tokens ก่อน ListenAndServe
  → ในโค้ดตั้ง instanceGetTimeoutMS=60s, portReadyTimeoutMS=180s ไว้แล้ว
    (worker/index.ts fetch override) — ถ้ายังช้า ต้องเช็ค migration หรือ Neon latency

Cloudflare error 1042 ("Worker tried to fetch from another Worker on the same zone")?
  → runtime resolve public hostname แล้ว loop กลับเข้า Worker เอง
  → แก้ในโค้ด: toContainerRequest() เขียน origin เป็น http://container ไม่ใช่ public URL
```

## Related

- `worker-headers` — CSP และ security headers
- `ci-pipeline` — CI deploy step
