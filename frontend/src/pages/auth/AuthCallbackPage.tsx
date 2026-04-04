import { useEffect } from 'react'
import { useNavigate } from 'react-router'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'

export function AuthCallbackPage() {
  const navigate = useNavigate()

  useEffect(() => {
    const hash = window.location.hash.slice(1) // remove #
    const params = new URLSearchParams(hash)

    const accessToken = params.get('access_token')
    const refreshToken = params.get('refresh_token')
    const customerStr = params.get('customer')

    if (accessToken && refreshToken && customerStr) {
      try {
        const customer = JSON.parse(customerStr)
        localStorage.setItem('access_token', accessToken)
        localStorage.setItem('refresh_token', refreshToken)
        useAuthStore.getState().setCustomer(customer)

        // Clear hash from URL immediately for security
        window.history.replaceState({}, '', '/auth/callback')

        toast.success('เข้าสู่ระบบสำเร็จ')
        navigate('/dashboard', { replace: true })
      } catch {
        toast.error('เข้าสู่ระบบไม่สำเร็จ')
        navigate('/login', { replace: true })
      }
    } else {
      // Check for error in query params
      const searchParams = new URLSearchParams(window.location.search)
      const error = searchParams.get('error')
      if (error === 'suspended') {
        toast.error('บัญชีถูกระงับ')
      } else {
        toast.error('เข้าสู่ระบบไม่สำเร็จ')
      }
      navigate('/login', { replace: true })
    }
  }, [navigate])

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center">
        <div className="h-8 w-8 mx-auto animate-spin rounded-full border-4 border-muted border-t-foreground" />
        <p className="mt-4 text-sm text-muted-foreground">กำลังเข้าสู่ระบบ...</p>
      </div>
    </div>
  )
}
