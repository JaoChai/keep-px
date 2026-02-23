import { Navigate } from 'react-router'
import { useAuthStore } from '@/stores/auth-store'
import type { ReactNode } from 'react'

interface ProtectedRouteProps {
  children: ReactNode
}

export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const hasToken = localStorage.getItem('access_token')

  if (!isAuthenticated && !hasToken) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}
