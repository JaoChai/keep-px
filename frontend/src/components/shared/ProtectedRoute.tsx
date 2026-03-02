import { useEffect, useState } from 'react'
import { Navigate } from 'react-router'
import { useAuthStore } from '@/stores/auth-store'
import api from '@/lib/api'
import type { ReactNode } from 'react'
import type { APIResponse } from '@/types'
import type { Customer } from '@/types'

interface ProtectedRouteProps {
  children: ReactNode
}

export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const hasHydrated = useAuthStore((s) => s._hasHydrated)
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const setCustomer = useAuthStore((s) => s.setCustomer)
  const logout = useAuthStore((s) => s.logout)
  const [isVerifying, setIsVerifying] = useState(
    () => !!localStorage.getItem('access_token')
  )
  const [isValid, setIsValid] = useState(false)

  useEffect(() => {
    if (!hasHydrated) return

    const token = localStorage.getItem('access_token')
    if (!token) return

    const controller = new AbortController()

    api
      .get<APIResponse<Customer>>('/auth/me', { signal: controller.signal })
      .then(({ data }) => {
        if (controller.signal.aborted) return
        if (data.data) {
          setCustomer(data.data)
          setIsValid(true)
        } else {
          logout()
          setIsValid(false)
        }
      })
      .catch((err) => {
        if (controller.signal.aborted) return
        if (err.code === 'ERR_CANCELED') return
        logout()
        setIsValid(false)
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          setIsVerifying(false)
        }
      })

    return () => controller.abort()
  }, [hasHydrated, setCustomer, logout])

  if (!hasHydrated || isVerifying) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: '100vh' }}>
        <div style={{ color: '#666', fontSize: '14px' }}>กำลังโหลด...</div>
      </div>
    )
  }

  if (!isValid || !isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}
