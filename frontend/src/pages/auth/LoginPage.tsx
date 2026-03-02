import { useState } from 'react'
import { useNavigate, Link } from 'react-router'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { toast } from 'sonner'
import { GoogleLogin } from '@react-oauth/google'
import { AuthLayout } from '@/components/layout/AuthLayout'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useLogin, useGoogleAuth } from '@/hooks/use-auth'

const loginSchema = z.object({
  email: z.string().email('Invalid email address'),
  password: z.string().min(1, 'Password is required'),
})

type LoginForm = z.infer<typeof loginSchema>

const googleClientId = import.meta.env.VITE_GOOGLE_CLIENT_ID

export function LoginPage() {
  const navigate = useNavigate()
  const login = useLogin()
  const googleAuth = useGoogleAuth()
  const [error, setError] = useState('')

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginForm>({
    resolver: zodResolver(loginSchema),
  })

  const onSubmit = async (data: LoginForm) => {
    setError('')
    try {
      await login.mutateAsync(data)
      toast.success('เข้าสู่ระบบสำเร็จ')
      navigate('/dashboard')
    } catch {
      const msg = 'อีเมลหรือรหัสผ่านไม่ถูกต้อง'
      setError(msg)
      toast.error(msg)
    }
  }

  return (
    <AuthLayout>
      <div>
        <h2 className="text-2xl font-bold text-foreground mb-2">Welcome back</h2>
        <p className="text-muted-foreground mb-8">Sign in to your Pixlinks account</p>

        {error && (
          <div className="mb-4 rounded-md bg-red-50 p-3 text-sm text-red-600">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="email">Email</Label>
            <Input
              id="email"
              type="email"
              placeholder="you@example.com"
              {...register('email')}
            />
            {errors.email && (
              <p className="text-sm text-red-500">{errors.email.message}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="password">Password</Label>
            <Input
              id="password"
              type="password"
              placeholder="Enter your password"
              {...register('password')}
            />
            {errors.password && (
              <p className="text-sm text-red-500">{errors.password.message}</p>
            )}
          </div>

          <Button type="submit" className="w-full" disabled={isSubmitting}>
            {isSubmitting ? 'Signing in...' : 'Sign in'}
          </Button>
        </form>

        {googleClientId && (
          <>
            <div className="relative my-6">
              <div className="absolute inset-0 flex items-center">
                <span className="w-full border-t" />
              </div>
              <div className="relative flex justify-center text-xs uppercase">
                <span className="bg-background px-2 text-muted-foreground">or</span>
              </div>
            </div>

            <div className="flex justify-center">
              <GoogleLogin
                onSuccess={async (response) => {
                  if (response.credential) {
                    try {
                      await googleAuth.mutateAsync(response.credential)
                      toast.success('เข้าสู่ระบบสำเร็จ')
                      navigate('/dashboard')
                    } catch {
                      const msg = 'เข้าสู่ระบบด้วย Google ไม่สำเร็จ'
                      setError(msg)
                      toast.error(msg)
                    }
                  }
                }}
                onError={() => {
                  toast.error('เข้าสู่ระบบด้วย Google ไม่สำเร็จ')
                }}
                text="signin_with"
                width="100%"
              />
            </div>
          </>
        )}

        <p className="mt-6 text-center text-sm text-muted-foreground">
          Don't have an account?{' '}
          <Link to="/register" className="text-foreground hover:text-foreground/80 hover:underline font-medium">
            Sign up
          </Link>
        </p>
      </div>
    </AuthLayout>
  )
}
