import { createInternalNeonAuth } from '@neondatabase/neon-js/auth'

const authURL = import.meta.env.VITE_NEON_AUTH_URL
if (!authURL) {
  throw new Error('ไม่ได้ตั้งค่า VITE_NEON_AUTH_URL')
}

const neon = createInternalNeonAuth(authURL)

// authClient คือ Better Auth adapter ตัวเดิม — signIn.social/signOut ฯลฯ ใช้งานเหมือนเดิมทุกอย่าง
export const authClient = neon.adapter

// ---------------------------------------------------------------------------
// getJWTToken() คือ API ทางการของ SDK สำหรับเอา JWT ไปยิง backend ด้วย Authorization: Bearer
// (ยืนยันแล้วด้วยการ login จริงในเครื่อง + decode + verify กับ JWKS endpoint — signature ผ่าน)
// ภายในมันเรียก getSession() ซึ่งดูเหมือนคืน session token ทึบ ๆ แต่จริง ๆ แล้ว SDK มี onSuccess
// hook ที่อ่าน header `set-auth-jwt` จาก response แล้ว inject ทับ session.token — ค่าที่ได้จึงเป็น
// JWT จริงเสมอ ไม่ใช่ opaque token (เดิมเข้าใจผิดว่าต้องยิง endpoint /token เองแยกต่างหาก ซึ่งพัง
// เพราะ /token ไม่ใช่ endpoint ของ SDK เวอร์ชันนี้เลย ใช้ authClient.$fetch('/token') ไม่ได้)
// อายุ token 15 นาที จึงต้อง cache ไว้แล้วขอใหม่ก่อนหมดอายุ
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
    const token = await neon.getJWTToken()
    if (!token) {
      cached = null
      return null
    }
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
