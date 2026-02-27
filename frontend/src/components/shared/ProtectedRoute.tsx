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
  const setCustomer = useAuthStore((s) => s.setCustomer)
  const logout = useAuthStore((s) => s.logout)
  const [isVerifying, setIsVerifying] = useState(true)
  const [isValid, setIsValid] = useState(false)

  const token = localStorage.getItem('access_token')

  useEffect(() => {
    if (!hasHydrated || !token) return

    // Verify token with server and fetch fresh customer data
    api
      .get<APIResponse<Customer>>('/auth/me')
      .then(({ data }) => {
        if (data.data) {
          setCustomer(data.data)
          setIsValid(true)
        } else {
          logout()
        }
      })
      .catch(() => {
        logout()
      })
      .finally(() => setIsVerifying(false))
  }, [hasHydrated, token]) // eslint-disable-line react-hooks/exhaustive-deps

  if (!hasHydrated) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: '100vh' }}>
        <div style={{ color: '#666', fontSize: '14px' }}>Loading...</div>
      </div>
    )
  }

  if (!token) {
    return <Navigate to="/login" replace />
  }

  if (isVerifying) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: '100vh' }}>
        <div style={{ color: '#666', fontSize: '14px' }}>Loading...</div>
      </div>
    )
  }

  if (!isValid) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}
