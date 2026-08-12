import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

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
    vi.stubGlobal('fetch', vi.fn())
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
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: async () => ({ token }),
    } as Response)

    const first = await mod.getAccessToken()
    expect(first).toBe(token)
    expect(fetch).toHaveBeenCalledTimes(1)

    // เรียกซ้ำตอน cache ยัง valid → ต้องไม่ fetch ใหม่
    const second = await mod.getAccessToken()
    expect(second).toBe(token)
    expect(fetch).toHaveBeenCalledTimes(1)
  })

  it('refetches when token is within 60s of expiry', async () => {
    const now = Date.now()
    const token = makeJwt(now / 1000 + 600) // expiresAt = now + 600_000ms
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: async () => ({ token }),
    } as Response)

    await mod.getAccessToken() // เติม cache
    expect(fetch).toHaveBeenCalledTimes(1)

    // เดินเวลาไปจนเหลือเพดาน 60 วิ (expiresAt - 60_000 = now + 540_000)
    vi.advanceTimersByTime(540_000)

    await mod.getAccessToken()
    expect(fetch).toHaveBeenCalledTimes(2) // cache miss → fetch ใหม่
  })

  it('returns null (no throw) when /token responds not-ok', async () => {
    vi.mocked(fetch).mockResolvedValue({ ok: false, status: 401 } as Response)

    await expect(mod.getAccessToken()).resolves.toBeNull()
    expect(fetch).toHaveBeenCalledTimes(1)
  })

  it('returns null (no throw) when fetch rejects', async () => {
    // network/CORS failure — getAccessToken ต้องไม่ throw ต่อ (สัญญา Promise<string|null>)
    vi.mocked(fetch).mockRejectedValue(new Error('network down'))

    await expect(mod.getAccessToken()).resolves.toBeNull()
    expect(fetch).toHaveBeenCalledTimes(1)
  })
})
