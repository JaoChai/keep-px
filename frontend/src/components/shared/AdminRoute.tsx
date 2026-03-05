import { Navigate } from 'react-router'
import { useAuthStore } from '@/stores/auth-store'

export function AdminRoute({ children }: { children: React.ReactNode }) {
  const customer = useAuthStore((s) => s.customer)
  if (!customer?.is_admin) return <Navigate to="/dashboard" replace />
  return <>{children}</>
}
