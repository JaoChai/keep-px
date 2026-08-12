# ย้ายระบบ login ไป Neon Auth

**วันที่:** 2026-08-12
**สถานะ:** design อนุมัติแล้ว รอเขียน implementation plan

## ปัญหา

ตอนนี้เราถือระบบยืนยันตัวตนไว้เองทั้งหมด — ตรวจ Google ID token เอง ออก JWT เอง หมุน refresh
token เอง เก็บ token ลงตารางของเราเอง เท่ากับต้องรับผิดชอบความปลอดภัยของ login ทั้งเส้น
ทั้งที่ไม่ใช่คุณค่าหลักของสินค้า

**เป้าหมาย:** ลดภาระฝั่งเรา ไม่ใช่เพิ่มช่องทาง login เจ้าของยืนยันว่าใช้ Google อย่างเดียวต่อไปก็พอ

## ทางเลือกที่พิจารณาแล้วไม่เอา

**Logto / Clerk / WorkOS** — รองรับ social ได้กว้างกว่า (Facebook, LINE, Apple) แต่ต้องเพิ่ม
vendor ใหม่เข้า stack เป้าหมายคือลดภาระ ไม่ใช่เพิ่มช่องทาง จึงไม่คุ้ม

**Neon Auth แบบยามหน้าประตู** — ให้ Neon พิสูจน์ตัวตนอย่างเดียว แล้วเรายังออก JWT + refresh
token ของเราเองต่อ เปลี่ยนโค้ดน้อยที่สุดจริง แต่กลายเป็นถือระบบ token สองชั้น = ภาระ*มากกว่า*เดิม
ผิดเป้าหมาย

**อ่าน session จากตาราง `neon_auth` ใน Postgres ตรง ๆ** — ทำได้เพราะ Neon เก็บ session ไว้ใน
ฐานข้อมูลตัวเดียวกับที่ Go ต่ออยู่แล้ว แต่เท่ากับผูกตัวเองกับโครงสร้างตารางภายในของ Neon
ซึ่งไม่ใช่สัญญาที่เขารับประกัน เปลี่ยนเมื่อไหร่เราพังเงียบ ๆ

## ทางที่เลือก

ให้ Neon Auth ถือ session ทั้งหมด แล้วรื้อระบบ token ของเราทิ้ง

## ข้อเท็จจริงที่ตรวจแล้ว

- Neon Auth วันนี้คือ **Managed Better Auth** (Stack Auth เป็นของเดิมที่กำลังเลิกใช้)
- เก็บ user/session ลง schema `neon_auth` ในฐานข้อมูล Neon ของเราเอง
- ฟรีถึง 60,000 MAU · แพลน Launch/Scale ถึง 1M MAU
- มี JWKS endpoint ให้ backend ภาษาไหนก็ตรวจ JWT ได้ตามมาตรฐาน
- **รองรับ social แค่ Google · GitHub · Vercel** ไม่มี Facebook / Apple / LINE
  (รับทราบและยอมรับแล้ว เพราะเป้าหมายไม่ใช่การเพิ่มช่องทาง)
- แต่ละ branch ของฐานข้อมูลได้ auth environment แยกกัน → spike บน branch ทดสอบได้
  โดยไม่แตะ production

**ยังไม่ได้พิสูจน์:** ยังไม่เคยรัน Neon Auth จริงกับ Go container ที่อยู่หลัง Cloudflare Worker
จึงต้องมี spike เป็นด่านแรกก่อนแตะโค้ดจริง

## สภาพปัจจุบัน

```
LoginPage.tsx:168 (@react-oauth/google, redirect)
  → auth_handler.go:151 GoogleAuthCallback   ตรวจ CSRF token ของ Google
  → auth_service.go:66  GoogleAuth           idtoken.Validate ตรวจกับ Google
  → หา/สร้าง customer (จับคู่ด้วย google_id แล้วค่อย email)
  → generateTokens()                         ออก JWT HS256 + refresh token 32 ไบต์
  → redirect กลับ /auth/callback#access_token=...&refresh_token=...&customer=...
```

จุดตรวจ JWT มีที่เดียวคือ `middleware.JWTAuth` (`router.go:156`) ซึ่งอ่านทั้ง `sub`
(= id ของ customer) และ `is_admin` **จากตัว claim ในโทเค็นเอง** ไม่ได้แตะฐานข้อมูลเลย

## สถาปัตยกรรมใหม่

**การแบ่งความรับผิดชอบ**

| ใคร | ถืออะไร |
|---|---|
| Neon Auth | ตัวตนและ session ทั้งหมด · คุยกับ Google เอง · เก็บใน schema `neon_auth` |
| `customers` ของเรา | ของทางธุรกิจล้วน ๆ — `api_key` `plan` `retention_days` `is_admin` `suspended_at` เครดิต pixel |

ผูกกันด้วยคอลัมน์ใหม่ตัวเดียว `customers.auth_user_id`

**เส้นทางตอน login**

```
1. หน้า login → ปุ่มของ Neon Auth
2. Neon Auth ↔ Google ↔ Neon Auth          เราไม่แตะขั้นนี้
3. Neon ออก JWT ให้ frontend
4. ทุก request แนบ  Authorization: Bearer <JWT ของ Neon>
5. middleware ตรวจลายเซ็นกับ JWKS ของ Neon (cache กุญแจในหน่วยความจำ)
6. sub → หา customer จาก auth_user_id → ใส่ customer id + is_admin ลง context
7. handler ทั้งหมดไม่ต้องแก้
```

**การสร้าง customer ครั้งแรก**

เพิ่ม endpoint เดียว `POST /api/v1/auth/session` — frontend เรียกครั้งเดียวหลัง login สำเร็จ
→ สร้าง `customer` (API key + plan sandbox เหมือนเดิม) → คืนข้อมูล customer กลับ

เหตุผลที่ไม่ให้ `/auth/me` สร้างให้เอง: `/auth/me` เป็นการอ่าน ถ้าปล่อยให้เขียนข้อมูลด้วยจะ
ตามไม่ได้ว่า record นี้เกิดตอนไหน และ middleware จะกลายเป็นจุดเขียนฐานข้อมูลทุก request

## ผลกระทบต่อ middleware (จุดที่ต้องระวังที่สุด)

`middleware/auth.go:55` วันนี้อ่าน `is_admin` จาก claim และใช้ `sub` เป็น customer id ตรง ๆ
พอเปลี่ยนไปใช้ JWT ของ Neon ทั้งสองอย่างใช้ไม่ได้ — `sub` กลายเป็น id ของ Neon และไม่มีข้อมูล admin

→ middleware ต้องอ่านฐานข้อมูลเพิ่ม **1 query ต่อ request** เพื่อแปลง Neon id → customer
พร้อมดึง `is_admin` และ `suspended_at`

- **ต้นทุน:** ค้นด้วย unique index คอลัมน์เดียว และทุก endpoint ก็แตะ DB อยู่แล้ว
  **ยังไม่ทำ cache** จนกว่าจะวัดแล้วพบว่าช้าจริง (CLAUDE.md §2)
- **ของแถม:** เดิมคนถูกระงับบัญชียังใช้งานต่อได้จนกว่า token หมดอายุ (15 นาที) เพราะเช็คแค่ตอน
  ต่ออายุ แบบใหม่เช็คทุก request

## เคสขอบ

| เคส | ทำอะไร |
|---|---|
| JWT ผ่าน แต่ยังไม่มี `customer` | ตอบ 401 พร้อมรหัส `customer_not_provisioned` → frontend เรียก `/auth/session` ก่อน ไม่ใช่เตะออกหน้า login |
| คนเดิมที่มี `customer` อยู่แล้ว login เข้ามา | `/auth/session` จับคู่ด้วย email ถ้าเจอ customer ที่ยังไม่มี `auth_user_id` ให้ผูกเข้าด้วยกัน ไม่สร้างใหม่ · ผูกได้เฉพาะเมื่อ Neon ยืนยันว่า email ผ่านการตรวจสอบแล้ว |
| เปิด 2 แท็บพร้อมกันตอน login ครั้งแรก | unique index บน `auth_user_id` + upsert → แท็บช้ากว่าได้ record เดิม ไม่เกิดบัญชีซ้ำ |
| กุญแจของ Neon หมุน | cache JWKS ในหน่วยความจำ · เจอกุญแจไม่รู้จักให้ดึงใหม่ แต่จำกัดความถี่ กันคนยิง token ปลอมรัวให้เราวิ่งหา Neon ไม่หยุด |
| Neon Auth ล่ม | คนที่ login อยู่แล้วใช้ต่อได้จนกว่า token หมดอายุ (ตรวจด้วยกุญแจที่ cache ไว้) · คนใหม่ login ไม่ได้ — ข้อแลกเปลี่ยนที่ยอมรับ |
| Logout | frontend สั่ง Neon Auth ออกจากระบบ ฝั่งเราไม่มีอะไรต้องทำ |

## ของที่ต้องลบ

**Go**

| ที่ | ลบอะไร |
|---|---|
| `service/auth_service.go` | `GoogleAuth()` · `DevLogin()` · `RefreshTokens()` · `generateTokens()` · `hashToken()` — เหลือ `GetCustomerByID` + `RegenerateAPIKey` |
| `handler/auth_handler.go` | `GoogleAuth` · `GoogleAuthCallback` (รวมโค้ดกัน CSRF) · `DevLogin` · `Refresh` · `Logout` |
| `repository/postgres/refresh_token_repo.go` | ทั้งไฟล์ + interface + mock |
| `router.go` | `/auth/google` · `/auth/google/callback` · `/auth/refresh` · `/auth/dev-login` · `/auth/logout` |
| migration `000028` | `DROP TABLE refresh_tokens` · `DROP COLUMN customers.google_id` · `ADD COLUMN customers.auth_user_id` (unique) |
| `config.go` | `JWT_SECRET` · `JWT_ACCESS_TTL` · `JWT_REFRESH_TTL` · `GOOGLE_CLIENT_ID` |
| `go.mod` | ถอด `google.golang.org/api` |
| `middleware/auth.go` | **แก้ ไม่ใช่ลบ** — HS256 ด้วยรหัสลับของเราเอง → ตรวจกับ JWKS ของ Neon |

**React**

| ที่ | ลบอะไร |
|---|---|
| `lib/api.ts` (177 บรรทัด) | logic ต่ออายุ token ทั้งชุด — หมุน refresh token, `_last_refresh_attempt`, ซิงก์ระหว่างแท็บ · เหลือแค่แนบ token ใส่ header |
| `pages/auth/AuthCallbackPage.tsx` | ทั้งไฟล์ — Neon รับ callback เอง |
| `stores/auth-store.ts` | การเซฟ/ลบ token ใน localStorage |
| `package.json` | ถอด `@react-oauth/google` |
| env | `VITE_GOOGLE_CLIENT_ID` |

**Cloudflare**

- ลบ secret `JWT_SECRET` · `GOOGLE_CLIENT_ID` (18 → 16)
- เพิ่มค่าตั้งต้นของ Neon Auth เป็น `vars` ใน `wrangler.jsonc` **ไม่ใช่ secret** เพราะเป็น URL
  สาธารณะไม่ใช่ความลับ และควรอยู่ใน version control ให้เห็นเวลามันเปลี่ยน:
  `NEON_AUTH_URL` ฝั่ง Go และ `VITE_NEON_AUTH_URL` ฝั่ง frontend (ค่าเดียวกัน รูปแบบ
  `https://ep-xxx.neon.tech/neondb/auth`)
  ที่อยู่ของกุญแจคือ `{NEON_AUTH_URL}/.well-known/jwks.json` และผู้ออกโทเค็น (issuer) คือ
  origin ของ URL นั้น — ทั้งสองค่าคำนวณจากตัวแปรเดียว ไม่ต้องตั้งเพิ่ม
  ค่าจริงของ `ep-xxx` จะได้จาก spike ขั้น 0
- `worker/headers.ts` — CSP เพิ่มโดเมนของ Neon Auth เข้า `connect-src` และเอา `accounts.google.com`
  ออกจาก `script-src`/`frame-src` ของหน้า dashboard
  (เก็บ `connect.facebook.net` ไว้ เป็นของ sale page คนละเรื่อง)
- `wrangler.jsonc:19` `JWT_REFRESH_TTL: "168h"` จะถูกลบ — ค่านี้เพิ่งกู้กลับมาจาก Railway ใน
  PR #221 วันเดียวกัน อายุ session จะไปขึ้นกับการตั้งค่าฝั่ง Neon Auth แทน

ประมาณการ: ลบ ~400 บรรทัด เขียนใหม่ ~120 บรรทัด

## การ login ตอนพัฒนาในเครื่อง

`/auth/dev-login` (login ด้วย email เปล่า ๆ เปิดเฉพาะโหมด development) ถูกลบไปด้วย เพราะมันออก
JWT ด้วยระบบเก่าที่กำลังรื้อ

แทนที่ด้วยการ login ด้วย Google จริงในเครื่อง — Neon Auth ให้ credential กลางสำหรับทดสอบมาให้อยู่
แล้วโดยไม่ต้องตั้งค่าอะไร และแต่ละ branch ของฐานข้อมูลมี auth environment แยกกัน จึงทดสอบใน
เครื่องได้โดยไม่แตะผู้ใช้จริง

**ข้อแลกเปลี่ยน:** การ login ในเครื่องจะต้องต่อเน็ตและกดผ่านหน้าจอ Google จริง ช้ากว่าเดิมที่
ยิง API ตัวเดียวจบ ยอมรับได้เพราะเป็นงานที่ทำวันละไม่กี่ครั้ง และแลกมากับการไม่ต้องดูแลทางลัดที่
เป็นช่องโหว่ถ้าหลุดไปโผล่บน production

## การย้ายข้อมูล

เจ้าของยืนยันว่ายังไม่มีผู้ใช้จริงหรือมีน้อยมาก → **ตัดสวิตช์ทีเดียว ไม่ต้องรันสองระบบคู่กัน**
คนที่ login ค้างอยู่จะถูกเตะออกและต้องกด login ใหม่หนึ่งครั้ง ยอมรับได้
บัญชีเดิมที่มีอยู่จะถูกจับคู่ด้วย email ตามกติกาในตารางเคสขอบ เครดิต/pixel/API key ไม่หาย

## แผนทดสอบ

**ขั้น 0 — spike (ด่านตัดสิน)**
เปิด Neon Auth บน branch ทดสอบของ Neon (**ไม่ใช่ฐานข้อมูล production**) → login ด้วย Google จริง
→ เอา JWT ที่ได้ให้โปรแกรม Go เล็ก ๆ ตรวจกับ JWKS ต้องผ่านจริง
**ถ้าไม่ผ่าน หยุดทั้งแผน กลับมาคุยกับเจ้าของ**

**ขั้นถัดไป — เขียนเทสต์ก่อนโค้ดทุกงาน**

- `middleware/auth_test.go` — token ถูกต้อง / หมดอายุ / ลายเซ็นผิด / ผู้ออกผิด / กุญแจไม่รู้จัก /
  บัญชีถูกระงับต้องโดนปฏิเสธ
- `/auth/session` — สร้างใหม่ / ผูกกับ email เดิม / เรียกซ้ำไม่สร้างซ้ำ / email ที่ยังไม่ยืนยันต้องไม่ผูก
- ตรวจว่าไม่มีอะไรหลงเหลือ: grep ทั้ง repo ต้องไม่เจอ `JWT_SECRET` · `idtoken` · `refresh_token`
  · `@react-oauth/google`

**ช่องโหว่ที่ยอมรับ:** ชุดทดสอบ E2E ถูกลบใน PR #223 (ยังไม่ merge ณ วันเขียน) → จะไม่มีการทดสอบ
อัตโนมัติที่ login จริงผ่านเบราว์เซอร์อีก ขั้นสุดท้ายจึงต้องเป็นการทดสอบด้วยมือบน production
โดยเจ้าของ login จริงหนึ่งครั้ง แล้วยืนยันผลจาก log — Claude ทำแทนไม่ได้เพราะต้องใช้บัญชี
Google ของเจ้าของ

## เกณฑ์ว่าสำเร็จ

1. spike ผ่าน — Go ตรวจ JWT ของ Neon ได้จริง
2. login ด้วย Google บน production สำเร็จ และเข้าหน้า dashboard ได้
3. บัญชีที่ถูกระงับใช้งานไม่ได้ทันที ไม่ต้องรอ token หมดอายุ
4. grep แล้วไม่เหลือร่องรอยของระบบเดิมทั้ง 4 คำ
5. `go test ./internal/...` และ `npm test` เขียวทั้งหมด
6. security header ยังครบ 7 ตัวบนหน้า `/` (CSP ที่แก้ไม่ทำของเดิมพัง)
