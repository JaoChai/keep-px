import type { ReactNode } from 'react'

interface AuthLayoutProps {
  children: ReactNode
}

export function AuthLayout({ children }: AuthLayoutProps) {
  return (
    <div className="flex min-h-screen">
      {/* Left side - Branding */}
      <div className="hidden lg:flex lg:w-1/2 items-center justify-center bg-indigo-600 p-12">
        <div className="max-w-md text-white">
          <h1 className="text-4xl font-bold mb-4">Pixlinks</h1>
          <p className="text-lg text-indigo-100">
            ปกป้องข้อมูล Facebook Pixel ของคุณ ไม่ว่าจะเกิดอะไรขึ้นกับบัญชี
          </p>
        </div>
      </div>
      {/* Right side - Form */}
      <div className="flex flex-1 items-center justify-center p-8">
        <div className="w-full max-w-md">{children}</div>
      </div>
    </div>
  )
}
