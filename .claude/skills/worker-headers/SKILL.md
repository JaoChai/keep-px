---
name: worker-headers
description: จัดการ CSP และ security headers ใน Worker — ใช้เมื่อ CSP error, blocked by CSP, script ถูก block, font ไม่ขึ้น, หน้าเว็บโหลดไม่ครบ, เพิ่มโดเมนใหม่เข้า CSP
---

# Worker Headers

## When to Activate

- "CSP error" / "blocked by CSP" / "Refused to load"
- "Script ถูก block" / "font ไม่ขึ้น" / "หน้าเว็บโหลดไม่ครบ"
- "เพิ่มโดเมนใหม่เข้า CSP" (Stripe, Google, Facebook, R2 ใหม่)
- "sale page ฝังใน iframe ไม่ได้" / "X-Frame-Options block"

## ที่อยู่ของ headers

headers ทั้งหมดอยู่ใน **`worker/headers.ts`**

Worker เรียก `applySecurityHeaders(response, pathname)` ในทุก response — ทั้ง static asset และ backend proxy (ดู `worker/index.ts`) ต้องเปิด `run_worker_first: true` ใน `wrangler.jsonc` ไม่งั้น asset จะข้าม Worker และไม่มี header เลย (ดู skill `cloudflare-deploy`)

## headers 2 ชุด

แยกตาม path ว่าเป็น dashboard หรือ sale page — เช็คด้วย `isSalePage()` ที่ดู `pathname.startsWith("/p/")`

### Dashboard (path อื่นทั้งหมด) — `DASHBOARD_CSP`

ใส่ครบ 7 ตัว:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Permissions-Policy: camera=(), microphone=(), geolocation=()`
- `Strict-Transport-Security: max-age=31536000; includeSubDomains`
- `Cross-Origin-Opener-Policy: same-origin`
- `Content-Security-Policy: DASHBOARD_CSP`

CSP directives สำคัญ:
- `script-src 'self' 'unsafe-inline' js.stripe.com connect.facebook.net accounts.google.com`
- `style-src 'self' 'unsafe-inline' fonts.googleapis.com accounts.google.com`
- `connect-src 'self' accounts.google.com js.stripe.com`
- `frame-src js.stripe.com accounts.google.com`
- `img-src 'self' data: *.googleusercontent.com *.r2.dev`
- `font-src 'self' data: fonts.gstatic.com`

### Sale page (`/p/*`) — `SALE_PAGE_CSP`

ใส่แค่ 4 ตัว (ละ framing headers เพื่อให้ฝังได้):
- `X-Content-Type-Options: nosniff`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Strict-Transport-Security: max-age=31536000; includeSubDomains`
- `Content-Security-Policy: SALE_PAGE_CSP`

**ไม่มี** `X-Frame-Options`, `Permissions-Policy`, `Cross-Origin-Opener-Policy`

CSP ของ sale page เปิด `img-src * data:` กว้างกว่า เพราะลูกค้าใส่รูปจากโดเมนอะไรก็ได้

## ทำไม sale page ไม่มี X-Frame-Options

sale page (`/p/<slug>`) ออกแบบให้ **ฝังในหน้าเว็บของลูกค้าผ่าน iframe** ได้ ถ้าใส่ `X-Frame-Options: DENY` หรือ `Permissions-Policy` ที่เข้มงวด จะฝังไม่ได้

การแยกชุดนี้สืบทอดจากระบบเดิมที่แยก `/p/` กับ path อื่น — คนละชุด header คนละ path

## เพิ่มโดเมนใหม่เข้า CSP

1. เปิด `worker/headers.ts`
2. ดูใน console error ว่าโดเมนถูก block ที่ directive ไหน (`script-src`, `connect-src`, `img-src`, `frame-src`, `font-src`)
3. เพิ่มโดเมนเข้า array ที่ถูกต้อง:
   - ใช้กับ dashboard → `DASHBOARD_CSP`
   - ใช้กับ sale page → `SALE_PAGE_CSP`
4. เช่น เพิ่ม provider ใหม่ใน dashboard: เพิ่มโดเมนเข้า `script-src` และ `connect-src`
5. ระวัง syntax: `'self'` มี quote, โดเมนธรรมดาไม่มี (`js.stripe.com`)

⚠️ อย่าใส่ `${VAR}` placeholder — CSP ต้องเป็น literal string เพราะ headers อยู่ในโค้ด Worker ไม่ใช่ envsubst อีก ถ้ามี placeholder หลงเหลือ test จะตรวจจับได้

## ทดสอบ

```bash
npx vitest run worker/headers.test.ts
```

test ครอบคลุม 5 กรณี:
- dashboard ได้ header ครบ 7 ตัว
- sale page ไม่มี `X-Frame-Options` (ฝังใน iframe ได้)
- sale page CSP เปิด `img-src * data:` แต่ dashboard ไม่
- CSP ไม่มี `${` placeholder หลงเหลือ
- header เดิมของ response ไม่ถูกทำลาย

หลังแก้ `headers.ts` รัน test นี้ก่อน deploy เสมอ

## Related

- `cloudflare-deploy` — ภาพรวม Worker + Container
- `auth-flow` — COOP header ที่กระทบ Google Sign-In
