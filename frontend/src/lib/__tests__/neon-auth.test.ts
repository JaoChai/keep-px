import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

// mock ที่ระดับ SDK แทนที่จะ mock fetch ตรง ๆ — เพราะตอนนี้ getAccessToken() ไม่ได้ยิง
// fetch เอง แต่เรียกผ่าน neon.getJWTToken() ของ SDK (ดูคอมเมนต์ใน neon-auth.ts)
const getJWTToken = vi.fn()
vi.mock('@neondatabase/neon-js/auth', () => ({
  createInternalNeonAuth: vi.fn(() => ({
    adapter: {},
    getJWTToken,
  })),
}))

// โหลด module ใหม่ทุก test (reset cache `cached` ของ getAccessToken) หลัง stub env
// เพราะ neon-auth.ts อ่าน VITE_NEON_AUTH_URL ตอน module-load แล้ว throw ถ้าไม่ได้ตั้ง
let mod: typeof import('@/lib/neon-auth')

// base64url จริง (ตัด padding + แทน +/ ด้วย -_) — atob() ธรรมดารับไม่ได้ ต้องแปลงก่อน
// (สาเหตุของบั๊กที่ neon-auth.ts เคย throw กับ payload จริงราว 45% ของกรณีสุ่ม)
// btoa() เองรับได้แค่ Latin1 — ต้องแปลงเป็น byte string ผ่าน TextEncoder ก่อนเพื่อให้ใส่ข้อความไทยได้
function base64url(json: string): string {
  const bytes = new TextEncoder().encode(json)
  const binary = Array.from(bytes, (b) => String.fromCharCode(b)).join('')
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

function makeJwt(expSeconds: number): string {
  const header = base64url(JSON.stringify({ alg: 'EdDSA', typ: 'JWT' }))
  const payload = base64url(JSON.stringify({ exp: expSeconds, name: 'ทดสอบ???>>>' }))
  return `${header}.${payload}.sig`
}

describe('getAccessToken', () => {
  beforeEach(async () => {
    vi.stubEnv('VITE_NEON_AUTH_URL', 'https://auth.test')
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-01-01T00:00:00Z'))
    getJWTToken.mockReset()
    vi.resetModules()
    mod = await import('@/lib/neon-auth')
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
    vi.unstubAllEnvs()
  })

  it('returns cached token without refetching while valid', async () => {
    const now = Date.now()
    const token = makeJwt(now / 1000 + 600) // หมดอายุใน 600 วิ
    getJWTToken.mockResolvedValue(token)

    const first = await mod.getAccessToken()
    expect(first).toBe(token)
    expect(getJWTToken).toHaveBeenCalledTimes(1)

    // เรียกซ้ำตอน cache ยัง valid → ต้องไม่เรียก getJWTToken ใหม่
    const second = await mod.getAccessToken()
    expect(second).toBe(token)
    expect(getJWTToken).toHaveBeenCalledTimes(1)
  })

  it('refetches when token is within 60s of expiry', async () => {
    const now = Date.now()
    const token = makeJwt(now / 1000 + 600) // expiresAt = now + 600_000ms
    getJWTToken.mockResolvedValue(token)

    await mod.getAccessToken() // เติม cache
    expect(getJWTToken).toHaveBeenCalledTimes(1)

    // เดินเวลาไปจนเหลือเพดาน 60 วิ (expiresAt - 60_000 = now + 540_000)
    vi.advanceTimersByTime(540_000)

    await mod.getAccessToken()
    expect(getJWTToken).toHaveBeenCalledTimes(2) // cache miss → เรียกใหม่
  })

  it('returns null (no throw) when getJWTToken resolves null', async () => {
    // ไม่มี session ที่ยัง valid — SDK คืน null ตามสัญญาของมันเอง (ไม่ throw)
    getJWTToken.mockResolvedValue(null)

    await expect(mod.getAccessToken()).resolves.toBeNull()
    expect(getJWTToken).toHaveBeenCalledTimes(1)
  })

  it('returns null (no throw) when getJWTToken rejects', async () => {
    // network/CORS failure — getAccessToken ต้องไม่ throw ต่อ (สัญญา Promise<string|null>)
    getJWTToken.mockRejectedValue(new Error('network down'))

    await expect(mod.getAccessToken()).resolves.toBeNull()
    expect(getJWTToken).toHaveBeenCalledTimes(1)
  })

  // กันถอยหลัง: ถ้าใครเปลี่ยน getAccessToken() กลับไปยิง authClient.$fetch('/token') เอง
  // (แบบที่ PR #226 เคยทำ ซึ่งพังเพราะ /token ไม่ใช่ endpoint จริงของ SDK เวอร์ชันนี้ — ดูคอมเมนต์
  // ใน neon-auth.ts) test นี้ต้องแดง เพราะ mock ของ authClient ด้านบนไม่มีเมธอด $fetch ให้เรียก
  // และไม่ควรมี fetch ตรง ๆ เกิดขึ้นเลยในเส้นทางนี้
  it('never calls fetch directly — must go through SDK getJWTToken()', async () => {
    const fetchSpy = vi.fn()
    vi.stubGlobal('fetch', fetchSpy)
    getJWTToken.mockResolvedValue(makeJwt(Date.now() / 1000 + 600))

    await mod.getAccessToken()

    expect(fetchSpy).not.toHaveBeenCalled()
    expect(getJWTToken).toHaveBeenCalledTimes(1)
  })
})
