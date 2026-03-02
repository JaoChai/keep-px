import { useEffect, useRef } from 'react'
import { Navigate } from 'react-router'
import { useAuthStore } from '@/stores/auth-store'
import api from '@/lib/api'
import type { ReactNode } from 'react'
import type { APIResponse } from '@/types'
import type { Customer } from '@/types'

let hasVerifiedThisSession = false

// Reset verification flag on logout
useAuthStore.subscribe((state, prevState) => {
  if (prevState.isAuthenticated && !state.isAuthenticated) {
    hasVerifiedThisSession = false
  }
})

interface ProtectedRouteProps {
  children: ReactNode
}

export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const hasHydrated = useAuthStore((s) => s._hasHydrated)
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const setCustomer = useAuthStore((s) => s.setCustomer)
  const logout = useAuthStore((s) => s.logout)
  const verifyingRef = useRef(false)

  useEffect(() => {
    if (!hasHydrated || !isAuthenticated || hasVerifiedThisSession || verifyingRef.current) return

    verifyingRef.current = true
    const controller = new AbortController()

    api
      .get<APIResponse<Customer>>('/auth/me', { signal: controller.signal })
      .then(({ data }) => {
        if (controller.signal.aborted) return
        if (data.data) {
          setCustomer(data.data)
        } else {
          logout()
        }
      })
      .catch((err) => {
        if (controller.signal.aborted) return
        if (err.code === 'ERR_CANCELED') return
        // Only logout on auth errors (401/403), not network errors
        if (err.response?.status === 401 || err.response?.status === 403) {
          logout()
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          hasVerifiedThisSession = true
          verifyingRef.current = false
        }
      })

    return () => controller.abort()
  }, [hasHydrated, isAuthenticated, setCustomer, logout])

  if (!hasHydrated) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: '100vh' }}>
        <div style={{ color: '#666', fontSize: '14px' }}>กำลังโหลด...</div>
      </div>
    )
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}
