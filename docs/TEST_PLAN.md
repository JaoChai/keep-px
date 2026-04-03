# Keep-PX E2E Test Plan

> แผนการทดสอบระบบ Keep-PX แบบ end-to-end — จำลองการใช้งานจริงเหมือน user คนหนึ่ง
>
> Last updated: 2026-04-03

---

## 1. เป้าหมาย

- ทดสอบทุก feature ที่ user ใช้ได้จริงบนหน้าเว็บ
- จำลองการใช้งานเหมือน user คนหนึ่ง login มาแล้วใช้ระบบ
- จับ bug ก่อนถึง production
- รันอัตโนมัติได้ ไม่ต้องมีคนนั่งกด

## 2. Environments

| Environment | URL | Database | ใช้ทำอะไร |
|-------------|-----|----------|----------|
| **Local** | `localhost:5173` | Neon `dev-local` | พัฒนา + debug test |
| **Production** | `https://pixlinks.xyz` | Neon production | Smoke test หลัง deploy |

## 3. Authentication

- **Local:** `POST /api/v1/auth/dev-login` → ได้ JWT tokens
- **Production:** ใช้ JWT tokens จาก Google OAuth (ดึงจาก browser localStorage)
- **วิธีใส่:** Environment variables `E2E_ACCESS_TOKEN` + `E2E_REFRESH_TOKEN`
- **Global setup** เขียน tokens ลง `e2e/.auth/user.json` ให้ทุก test ใช้ร่วมกัน

## 4. Test Data Strategy

- **สร้างเอง ลบเอง:** ทุก test สร้าง data ตอนเริ่ม + ลบตอนจบ
- **Prefix:** ใช้ `E2E-{tag}` นำหน้า เช่น `E2E-SMOKE Pixel`, `E2E-C1 SP`
- **Safety net:** `test.afterAll()` cleanup ถ้า test พังกลางทาง
- **Isolation:** แต่ละ test suite ไม่ share data กัน
- **Fake data:** Pixel ID, Access Token ใช้ค่าปลอมได้ (CAPI จะ fail แต่ event ยังเข้า DB)

## 5. External Services

| Service | ใน Test | เหตุผล |
|---------|---------|--------|
| Facebook CAPI | ❌ Mock (ใช้ fake token) | ไม่ต้อง hit FB จริง, เช็คว่า event เข้า DB พอ |
| Stripe | ❌ ไม่ test checkout จริง | เช็คแค่ redirect ไป Stripe ได้ |
| Neon DB | ✅ ใช้จริง | ต้อง test กับ DB จริง |
| Google OAuth | ❌ Bypass ด้วย JWT | ไม่ต้อง login Google จริง |

---

## 6. Test Suites — แบ่งตาม Priority

### Suite 1: `@smoke` — ระบบยังทำงาน? (5 tests, < 3 นาที)

> รัน: ทุก PR + หลัง deploy | ถ้า fail = block merge / rollback

| ID | Test | ขั้นตอน | Expected |
|----|------|---------|----------|
| SM-1 | Health check | GET `/health` | 200 OK |
| SM-2 | Frontend loads | เปิด `/` | HTML render ได้ |
| SM-3 | Login + Dashboard | Login → `/dashboard` | เห็น heading + stat cards |
| SM-4 | Navigation | คลิกทุก link ใน sidebar | ทุกหน้าโหลดได้ ไม่มี error |
| SM-5 | API responds | GET `/api/v1/auth/me` | 200 + user data |

```bash
npx playwright test --grep @smoke
```

---

### Suite 2: `@critical` — Flow ที่ทำเงินทำงาน? (15 tests, < 15 นาที)

> รัน: merge เข้า main | ถ้า fail = หยุด deploy

#### C1: Pixel CRUD (4 tests)

| ID | Test | ขั้นตอน | Expected |
|----|------|---------|----------|
| C1-1 | สร้าง pixel | กด "เพิ่มพิกเซล" → กรอก name/ID/token → บันทึก | เห็น pixel ในรายการ |
| C1-2 | แก้ไข pixel | กด edit → เปลี่ยนชื่อ → บันทึก | ชื่อเปลี่ยนในรายการ |
| C1-3 | Toggle status | กด badge → สลับ active/inactive | Badge เปลี่ยนสี + text |
| C1-4 | ลบ pixel | กด delete → ยืนยัน | หายจากรายการ |

#### C2: Sale Page Lifecycle (4 tests)

| ID | Test | ขั้นตอน | Expected |
|----|------|---------|----------|
| C2-1 | สร้าง sale page | กด "สร้างเซลเพจ" → กรอกชื่อ → เลือก pixel | เห็นในรายการ |
| C2-2 | แก้ไข + publish | เปิด editor → แก้ content → กด publish | Status = "เผยแพร่แล้ว" |
| C2-3 | เปิดหน้า public | เปิด `/p/{slug}` (ไม่ login) | เห็นหน้า sale page |
| C2-4 | ลบ sale page | กด delete → ยืนยัน | หายจากรายการ |

#### C3: Event Flow (3 tests)

| ID | Test | ขั้นตอน | Expected |
|----|------|---------|----------|
| C3-1 | Ingest events | POST `/api/v1/events/ingest` ส่ง 5 events | 200 OK |
| C3-2 | ดู event log | เปิด `/events` → mode=history | เห็น 5 events ในตาราง |
| C3-3 | ดู event detail | คลิก event แรก | เห็น JSON data ครบ |

#### C4: Replay (3 tests)

| ID | Test | ขั้นตอน | Expected |
|----|------|---------|----------|
| C4-1 | Preview replay | เลือก source → target → preview | เห็นจำนวน events + sample |
| C4-2 | Execute replay | กดยืนยัน → รอ progress | 100% สำเร็จ |
| C4-3 | Replay history | เปิดหน้า replay | เห็น replay ที่เพิ่งทำในรายการ |

#### C5: Settings (1 test)

| ID | Test | ขั้นตอน | Expected |
|----|------|---------|----------|
| C5-1 | API Key | เปิด settings → reveal key → copy | Key copied to clipboard |

```bash
npx playwright test --grep @critical
```

---

### Suite 3: `@regression` — ทุก feature ทำงาน? (30+ tests, < 30 นาที)

> รัน: ทุกคืน 02:00 | ถ้า fail = สร้าง GitHub issue

#### R-AUTH: Authentication (4 tests)

| ID | Test | Expected |
|----|------|----------|
| R-AUTH-1 | Token หมดอายุ → auto refresh | ได้ token ใหม่ + ไม่ถูก redirect ไป login |
| R-AUTH-2 | Refresh token หมดอายุ → redirect login | กลับไปหน้า login |
| R-AUTH-3 | เข้า protected route ไม่มี token | redirect ไป `/auth/login` |
| R-AUTH-4 | Regenerate API key | key เดิมใช้ไม่ได้ + key ใหม่ทำงาน |

#### R-PIXEL: Pixel Management (6 tests)

| ID | Test | Expected |
|----|------|----------|
| R-PIXEL-1 | Validation — ชื่อว่าง | เห็น error "กรุณากรอกชื่อ" |
| R-PIXEL-2 | Validation — Pixel ID ผิดรูปแบบ | เห็น error |
| R-PIXEL-3 | Test connection (fake token) | Toast error "ส่งไม่ได้" |
| R-PIXEL-4 | Set backup pixel | Backup column แสดงชื่อ backup |
| R-PIXEL-5 | ลบ backup → backup column clear | Backup column ว่าง |
| R-PIXEL-6 | Quota เต็ม → ปุ่มสร้าง disabled | ปุ่ม disabled + tooltip |

#### R-SP: Sale Pages (6 tests)

| ID | Test | Expected |
|----|------|----------|
| R-SP-1 | Block editor — เพิ่ม/ลบ block | Block count เปลี่ยน |
| R-SP-2 | Classic editor — hero + features | Preview แสดงข้อมูลถูก |
| R-SP-3 | Duplicate sale page | สำเนาปรากฏ + status = draft |
| R-SP-4 | Change pixel | Pixel ใน card เปลี่ยน |
| R-SP-5 | Copy link | Clipboard มี URL ถูกต้อง |
| R-SP-6 | Auto-save draft | ปิด → เปิดใหม่ → draft ยังอยู่ |

#### R-EVENT: Events (5 tests)

| ID | Test | Expected |
|----|------|----------|
| R-EVENT-1 | Filter by pixel | เห็นเฉพาะ events ของ pixel ที่เลือก |
| R-EVENT-2 | Filter by event type | เห็นเฉพาะ type ที่เลือก |
| R-EVENT-3 | Filter by date range | เห็นเฉพาะ events ในช่วง |
| R-EVENT-4 | Export CSV | ดาวน์โหลดไฟล์ .csv + ข้อมูลถูก |
| R-EVENT-5 | Live mode | เห็น events ใหม่ real-time |

#### R-REPLAY: Replay Edge Cases (4 tests)

| ID | Test | Expected |
|----|------|----------|
| R-REPLAY-1 | ยกเลิก replay กลางทาง | Status = cancelled + events ที่ส่งแล้วไม่ย้อนกลับ |
| R-REPLAY-2 | Retry failed events | เฉพาะ failed events ถูก retry |
| R-REPLAY-3 | ไม่มี credits → เห็น paywall | ฟอร์มถูกซ่อน + ลิงก์ไป billing |
| R-REPLAY-4 | เลือก source = target → ไม่ได้ | Validation error |

#### R-BILLING: Billing (3 tests)

| ID | Test | Expected |
|----|------|----------|
| R-BILLING-1 | Quota display ถูกต้อง | ตัวเลข match กับ API |
| R-BILLING-2 | Checkout redirect | กดชำระ → เปิด Stripe URL |
| R-BILLING-3 | Purchase history แสดง | เห็นรายการซื้อ |

#### R-DASHBOARD: Dashboard (4 tests)

| ID | Test | Expected |
|----|------|----------|
| R-DASH-1 | Stat cards + values | 5 cards แสดง + ตัวเลขไม่เป็น 0 (ถ้ามี data) |
| R-DASH-2 | Event chart + time range | เปลี่ยน 7d/14d/30d → chart update |
| R-DASH-3 | Recent activity | เห็น events ล่าสุด |
| R-DASH-4 | Onboarding wizard | user ใหม่เห็น wizard + dismiss ได้ |

#### R-UI: UI & Navigation (4 tests)

| ID | Test | Expected |
|----|------|----------|
| R-UI-1 | Responsive — mobile | Sidebar collapse + hamburger menu |
| R-UI-2 | Responsive — tablet | Layout 2 columns |
| R-UI-3 | 404 page | URL ผิด → เห็นหน้า 404 |
| R-UI-4 | Notification bell | กด bell → เห็น notifications |

#### R-GUIDE: Guide (2 tests)

| ID | Test | Expected |
|----|------|----------|
| R-GUIDE-1 | หน้า guide โหลด | เห็น heading + sections |
| R-GUIDE-2 | Search + expand | ค้นหา → expand section ถูกต้อง |

```bash
npx playwright test --grep @regression
```

---

### Suite 4: `@admin` — Admin Panel (10+ tests)

> รัน: เมื่อแก้ admin features | ต้องใช้ admin JWT

| ID | Test | Expected |
|----|------|----------|
| A-1 | Admin analytics dashboard | เห็น stats + charts |
| A-2 | Customer list + search | ค้นหาชื่อ/email → filter ทำงาน |
| A-3 | Customer detail dialog | เห็นข้อมูลครบ |
| A-4 | Change customer plan | Plan เปลี่ยนสำเร็จ |
| A-5 | Suspend + activate | Status เปลี่ยน |
| A-6 | Grant credits | Credits เพิ่มขึ้น |
| A-7 | Admin sale pages list | เห็นทุก sale pages |
| A-8 | Admin pixels list | เห็นทุก pixels |
| A-9 | Admin replays list | เห็นทุก replays |
| A-10 | Audit log | เห็น log ของ actions ที่เพิ่งทำ |

```bash
npx playwright test --grep @admin
```

---

## 7. Journey Tests — ทดสอบ flow ต่อเนื่องเหมือน user จริง

### Journey 1: First-Time User (Onboarding → ใช้งานครบ)

```
Login → เห็น onboarding wizard
  → สร้าง pixel (step 1)
  → สร้าง sale page + assign pixel (step 2)
  → ไปหน้า settings ดู API key (step 3)
  → ส่ง test event ผ่าน API (step 4)
  → ไปหน้า events เช็คว่า event มา
  → ไปหน้า dashboard เช็ค stats
  → dismiss wizard
  → cleanup
```

### Journey 2: Pixel Lifecycle (สร้าง → ใช้ → ย้าย → ลบ)

```
สร้าง pixel A (source)
  → สร้าง pixel B (target)
  → set B เป็น backup ของ A
  → สร้าง sale page + assign pixel A
  → publish sale page
  → ส่ง 5 events (PageView → Purchase)
  → เช็ค events ในหน้า event log
  → ไป replay → เลือก A→B → preview → ยืนยัน
  → รอ replay สำเร็จ
  → เช็ค event count ของ pixel B เพิ่มขึ้น
  → ลบ sale page
  → ลบ pixel B → backup column ของ A clear
  → ลบ pixel A
```

### Journey 3: Billing Flow (Upgrade → ใช้ feature)

```
เช็ค quota ปัจจุบัน (free plan)
  → pixel เต็ม quota → ปุ่มสร้าง disabled
  → ไป billing → เห็น upgrade options
  → กดซื้อ pixel slots → redirect ไป Stripe
  → (mock return) → กลับมาเว็บ
  → quota เพิ่ม → สร้าง pixel ใหม่ได้
```

### Journey 4: Error Recovery

```
เปิด URL ผิด → เห็น 404
  → กด "กลับหน้าแรก" → dashboard
  → Token expire → auto refresh
  → ส่ง event ด้วย API key ผิด → 401
  → กรอก form ไม่ครบ → เห็น validation errors
  → แก้ → submit สำเร็จ
```

---

## 8. Execution Schedule

| Trigger | Suite | เวลา | ถ้า Fail |
|---------|-------|:----:|----------|
| PR opened | `@smoke` | < 3 min | Block merge |
| Merge to main | `@smoke` + `@critical` | < 15 min | หยุด deploy, แจ้ง team |
| หลัง deploy production | `@smoke` (production URL) | < 3 min | พิจารณา rollback |
| ทุกคืน 02:00 | `@regression` + `@admin` | < 30 min | สร้าง GitHub issue |
| แก้ admin features | `@admin` | < 10 min | Fix ก่อน merge |

## 9. Flaky Test Policy

| สถานการณ์ | ทำอะไร |
|-----------|--------|
| Test flaky | Tag `@flaky` + skip ใน CI ทันที |
| สร้าง ticket | มี owner + deadline 1 สัปดาห์ |
| เกิน 2 สัปดาห์ | Fix หรือ ลบทิ้ง |
| เป้าหมาย | Flaky rate < 3% |

## 10. File Structure

```
frontend/e2e/
├── fixtures/
│   ├── auth.fixture.ts          # Auth setup (JWT storageState)
│   └── test-data.ts             # Test user config
├── pages/                       # Page Object Model
│   ├── login.page.ts
│   ├── sidebar.page.ts
│   ├── dashboard.page.ts
│   ├── pixels.page.ts
│   ├── sale-pages.page.ts
│   ├── sale-page-editor.page.ts
│   ├── event-log.page.ts
│   ├── replay.page.ts
│   ├── billing.page.ts
│   ├── settings.page.ts
│   ├── guide.page.ts
│   └── admin.page.ts
├── support/
│   └── global-setup.ts          # Token → storageState
├── tests/
│   ├── smoke/                   # @smoke (SM-1 to SM-5)
│   ├── critical/                # @critical (C1-C5)
│   ├── regression/              # @regression (R-*)
│   ├── admin/                   # @admin (A-*)
│   └── journeys/                # Full user journeys
│       ├── J1-onboarding.spec.ts
│       ├── J2-pixel-lifecycle.spec.ts
│       ├── J3-billing-flow.spec.ts
│       └── J4-error-recovery.spec.ts
└── playwright.config.ts
```

## 11. Commands

```bash
# Smoke (ทุก PR)
npx playwright test --grep @smoke

# Critical (merge to main)
npx playwright test --grep @critical

# Full regression (nightly)
npx playwright test --grep @regression

# Admin only
npx playwright test --grep @admin

# Journey tests
npx playwright test tests/journeys/

# Production smoke
E2E_BASE_URL=https://pixlinks.xyz npx playwright test --grep @smoke

# ดู report
npx playwright show-report

# Debug mode (เห็น browser)
npx playwright test --grep @smoke --headed --debug
```

## 12. Dependencies & Prerequisites

### ก่อนรัน test ต้องมี:

| สิ่งที่ต้องมี | วิธีได้มา |
|--------------|----------|
| JWT tokens | Dev login (local) หรือ Google login (prod) |
| Backend running | `cd backend && go run cmd/server/main.go` |
| Frontend running | `cd frontend && npm run dev` |
| Playwright browsers | `npx playwright install` |
| Neon DB accessible | ใช้ `dev-local` branch |

### ไม่ต้องมี:

- Facebook Pixel ID จริง
- Facebook Access Token จริง
- Stripe API keys จริง
- Google OAuth credentials

## 13. Metrics & Goals

| Metric | ปัจจุบัน | เป้าหมาย |
|--------|:-------:|:--------:|
| Smoke tests | 15 tests | 5 tests (ลด scope ให้เร็วขึ้น) |
| Critical tests | อยู่ใน journeys | 15 tests แยก tag |
| Regression tests | 30 specs (ไม่มี tag) | 30+ tests มี tag |
| Journey tests | 13 scenarios | 4 journeys (refactored) |
| Admin tests | 1 spec | 10+ tests |
| Flaky rate | ไม่ track | < 3% |
| Suite runtime | ~5 min | < 30 min (full) |
| Feature coverage | ~85% | 95%+ |
