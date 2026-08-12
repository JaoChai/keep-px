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

// JWT ใช้ base64url (ตัวอักษร -/_ แทน +// และไม่มี padding) ซึ่ง atob() ทำงานด้วยไม่ได้
// (atob รับแค่ base64 มาตรฐาน) — ต้องแปลงก่อนเสมอ ไม่งั้น payload ที่มี -/_ จะ throw
function decodeJwtPayload(token: string): { exp: number } {
  const base64 = token
    .split('.')[1]!
    .replace(/-/g, '+')
    .replace(/_/g, '/')
  const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), '=')
  return JSON.parse(atob(padded)) as { exp: number }
}

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
    const payload = decodeJwtPayload(token)
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
