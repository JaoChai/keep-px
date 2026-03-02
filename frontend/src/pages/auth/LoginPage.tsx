import { useNavigate } from 'react-router'
import { toast } from 'sonner'
import { GoogleLogin } from '@react-oauth/google'
import { useGoogleAuth } from '@/hooks/use-auth'

export function LoginPage() {
  const navigate = useNavigate()
  const googleAuth = useGoogleAuth()

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm text-center">
        <h1 className="text-3xl font-bold tracking-tight text-foreground">
          Pixlinks
        </h1>
        <p className="mt-2 text-sm text-muted-foreground">
          ปกป้องข้อมูล Facebook Pixel ของคุณ
        </p>

        <div className="my-8 border-t border-border" />

        <div className="flex justify-center">
          <GoogleLogin
            onSuccess={async (response) => {
              if (response.credential) {
                try {
                  await googleAuth.mutateAsync(response.credential)
                  toast.success('เข้าสู่ระบบสำเร็จ')
                  navigate('/dashboard')
                } catch {
                  toast.error('เข้าสู่ระบบด้วย Google ไม่สำเร็จ')
                }
              }
            }}
            onError={() => {
              toast.error('เข้าสู่ระบบด้วย Google ไม่สำเร็จ')
            }}
            theme="filled_black"
            size="large"
            shape="rectangular"
            width={320}
            text="continue_with"
          />
        </div>

        <p className="mt-8 text-xs text-muted-foreground">
          การเข้าสู่ระบบแสดงว่าคุณยอมรับเงื่อนไขการใช้งาน
        </p>
      </div>
    </div>
  )
}
