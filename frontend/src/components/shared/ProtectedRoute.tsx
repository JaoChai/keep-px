import { useEffect, useState } from 'react'
import { Navigate } from 'react-router'
import { getAccessToken } from '@/lib/neon-auth'
import api from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'
import type { ReactNode } from 'react'
import type { APIResponse, Customer } from '@/types'

interface ProtectedRouteProps {
  children: ReactNode
}

// Neon เก็บ session เอง (คุกกี้ของโดเมน Neon) เราจึงถามว่า "มี session ไหม"
// ด้วยการขอ JWT จาก /token — ถ้าคืนมาแปลว่า login อยู่ แล้วโหลด customer ผ่าน /auth/me
// (interceptor ใน api.ts จัดการ customer_not_provisioned ให้เอง — เรียก /auth/session + retry)
export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const [checking, setChecking] = useState(true)
  const [hasSession, setHasSession] = useState(false)

  useEffect(() => {
    getAccessToken()
      .then((token) => {
        if (!token) return null
        return api.get<APIResponse<Customer>>('/auth/me').then(({ data }) => {
          if (data.data) useAuthStore.getState().setCustomer(data.data)
          return token
        })
      })
      .then((token) => setHasSession(!!token))
      .catch((err) => {
        // มี response แปลว่า backend ตอบจริง (401/403/etc) = auth ไม่ผ่านจริง → ไป /login
        // ไม่มี response แปลว่า network error/timeout ระหว่างทาง → treat as "มี session"
        // ให้ children แสดงต่อ (page/hook อื่นจะเจอ error เดียวกันซ้ำตอนเรียก API จริง)
        setHasSession(!err.response)
      })
      .finally(() => setChecking(false))
  }, [])

  if (checking) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: '100vh' }}>
        <div style={{ color: '#666', fontSize: '14px' }}>กำลังโหลด...</div>
      </div>
    )
  }

  if (!hasSession) return <Navigate to="/login" replace />
  return <>{children}</>
}
