import { createAuthClient } from '@neondatabase/neon-js/auth'

const authURL = import.meta.env.VITE_NEON_AUTH_URL
if (!authURL) {
  throw new Error('ไม่ได้ตั้งค่า VITE_NEON_AUTH_URL')
}

export const authClient = createAuthClient(authURL)

// ---------------------------------------------------------------------------
// JWT ที่ backend ตรวจได้ **ไม่ใช่** ค่าที่ getSession() คืนมา
// getSession().session.token เป็น session token ทึบ ๆ ที่ตรวจกับ JWKS ไม่ได้
// ตัว JWT จริงต้องขอจาก endpoint /token แยกต่างหาก (spike ยืนยันแล้ว)
// และมันอายุแค่ 15 นาที จึงต้อง cache ไว้แล้วขอใหม่ก่อนหมดอายุ
// ---------------------------------------------------------------------------
let cached: { token: string; expiresAt: number } | null = null

export async function getAccessToken(): Promise<string | null> {
  // เผื่อเวลาไว้ 60 วินาที กัน token หมดอายุระหว่างทางไปถึง backend
  if (cached && Date.now() < cached.expiresAt - 60_000) {
    return cached.token
  }

  try {
    const res = await fetch(`${authURL}/token`, { credentials: 'include' })
    if (!res.ok) {
      cached = null
      return null
    }
    const { token } = (await res.json()) as { token: string }
    const payload = JSON.parse(atob(token.split('.')[1]!)) as { exp: number }
    cached = { token, expiresAt: payload.exp * 1000 }
    return token
  } catch {
    cached = null
    return null
  }
}

export function clearAccessToken() {
  cached = null
}
