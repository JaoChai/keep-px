import { useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router'
import { toast } from 'sonner'
import { Shield, RotateCcw, Zap, Database } from 'lucide-react'
import { authClient } from '@/lib/neon-auth'

const features = [
  {
    icon: Shield,
    title: 'สำรองข้อมูล Pixel',
    desc: 'เก็บ event data ทุกรายการไว้อย่างปลอดภัย ไม่มีหาย',
  },
  {
    icon: RotateCcw,
    title: 'Replay ได้ทันที',
    desc: 'ย้ายข้อมูลไป Pixel ใหม่เมื่อบัญชีมีปัญหา',
  },
  {
    icon: Zap,
    title: 'ส่งต่อ CAPI อัตโนมัติ',
    desc: 'Forward event ผ่าน Conversions API แบบ real-time',
  },
  {
    icon: Database,
    title: 'จัดการเซลเพจ',
    desc: 'สร้างหน้าขายพร้อม tracking ในที่เดียว',
  },
]

export function LoginPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()

  // Handle Google OAuth errors from query params (one-shot)
  useEffect(() => {
    const error = searchParams.get('error')
    if (!error) return
    if (error === 'suspended') {
      toast.error('บัญชีถูกระงับ')
    } else {
      toast.error('เข้าสู่ระบบด้วย Google ไม่สำเร็จ')
    }
    // Clear error param to prevent re-firing
    navigate('/login', { replace: true })
  }, [searchParams, navigate])

  return (
    <div className="flex min-h-screen">
      {/* Left panel — branding (hidden on mobile) */}
      <div className="relative hidden w-1/2 items-center justify-center overflow-hidden bg-foreground lg:flex">
        {/* Background pattern — subtle grid */}
        <div className="login-grid-pattern pointer-events-none absolute inset-0 opacity-[0.04]" />

        {/* Gradient glow */}
        <div className="pointer-events-none absolute -top-32 -left-32 size-[28rem] rounded-full bg-white/[0.06] blur-3xl" />
        <div className="pointer-events-none absolute -right-32 -bottom-32 size-[28rem] rounded-full bg-white/[0.04] blur-3xl" />

        <div className="relative z-10 max-w-md px-12">
          {/* Logo */}
          <div className="mb-10 flex items-center gap-3">
            <div className="flex size-11 items-center justify-center rounded-xl bg-white/10 backdrop-blur-sm">
              <Shield aria-hidden="true" className="size-[22px] text-white" />
            </div>
            <h1 className="text-2xl font-bold tracking-tight text-white">
              Pixlinks
            </h1>
          </div>

          {/* Tagline */}
          <p className="text-[2rem] leading-tight font-semibold tracking-tight text-white">
            ปกป้องข้อมูล
            <br />
            Facebook Pixel
            <br />
            <span className="text-white/50">ของคุณ</span>
          </p>

          {/* Features */}
          <div className="mt-10 space-y-5">
            {features.map((f) => (
              <div key={f.title} className="flex items-start gap-3.5">
                <div className="mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-lg bg-white/[0.08] ring-1 ring-white/[0.08]">
                  <f.icon aria-hidden="true" className="size-[18px] text-white/70" />
                </div>
                <div>
                  <p className="text-sm font-medium text-white/90">{f.title}</p>
                  <p className="mt-0.5 text-[13px] leading-relaxed text-white/45">
                    {f.desc}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Right panel — login */}
      <div className="relative flex w-full flex-col items-center justify-center bg-background px-6 lg:w-1/2">
        {/* Subtle gradient bg for right panel (hidden on mobile for perf) */}
        <div className="pointer-events-none absolute inset-0 hidden overflow-hidden lg:block">
          <div className="absolute -top-40 right-0 size-80 rounded-full bg-muted/60 blur-3xl" />
          <div className="absolute bottom-0 -left-20 size-64 rounded-full bg-muted/40 blur-3xl" />
        </div>

        <div className="relative z-10 w-full max-w-sm">
          {/* Mobile-only branding */}
          <div className="mb-2 text-center lg:hidden">
            <div className="mx-auto mb-4 flex size-12 items-center justify-center rounded-2xl bg-foreground">
              <Shield aria-hidden="true" className="size-6 text-background" />
            </div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Pixlinks
            </h1>
          </div>

          {/* Desktop heading */}
          <h2 className="hidden text-2xl font-semibold tracking-tight text-foreground lg:block">
            เข้าสู่ระบบ
          </h2>

          {/* Subtitle — single element for both viewports */}
          <p className="mt-1.5 mb-8 text-center text-sm text-muted-foreground lg:text-left">
            ปกป้องข้อมูล Facebook Pixel ของคุณ
          </p>

          {/* Login card */}
          <div className="rounded-2xl border border-border bg-card p-8 shadow-[0_4px_24px_rgba(0,0,0,0.06)]">
            {/* Mobile feature pills */}
            <div className="mb-6 flex flex-wrap gap-2 lg:hidden">
              {features.map((f) => (
                <span
                  key={f.title}
                  className="inline-flex items-center gap-1.5 rounded-full bg-secondary px-3 py-1.5 text-xs font-medium text-secondary-foreground"
                >
                  <f.icon className="size-3" />
                  {f.title}
                </span>
              ))}
            </div>

            {/* Login ผ่าน Neon Auth */}
            <div className="flex justify-center">
              <button
                type="button"
                onClick={() =>
                  authClient.signIn.social({
                    provider: 'google',
                    callbackURL: window.location.origin + '/dashboard',
                  })
                }
                className="flex h-11 w-[320px] items-center justify-center gap-2 rounded-md bg-foreground text-sm font-medium text-background transition-opacity hover:opacity-90"
              >
                เข้าสู่ระบบด้วย Google
              </button>
            </div>

            <div className="my-6 flex items-center gap-3">
              <div className="h-px flex-1 bg-border" />
              <span className="text-xs text-muted-foreground">
                เข้าสู่ระบบอย่างปลอดภัย
              </span>
              <div className="h-px flex-1 bg-border" />
            </div>

            {/* Trust signals */}
            <div className="flex items-center justify-center gap-4 text-xs text-muted-foreground">
              <span className="flex items-center gap-1">
                <Shield aria-hidden="true" className="size-3" />
                ข้อมูลเข้ารหัส
              </span>
              <span className="h-3 w-px bg-border" />
              <span className="flex items-center gap-1">
                <Zap aria-hidden="true" className="size-3" />
                เริ่มใช้ฟรี
              </span>
            </div>
          </div>

          {/* Footer */}
          <p className="mt-6 text-center text-xs text-muted-foreground">
            การเข้าสู่ระบบแสดงว่าคุณยอมรับเงื่อนไขการใช้งาน
          </p>
        </div>
      </div>
    </div>
  )
}
