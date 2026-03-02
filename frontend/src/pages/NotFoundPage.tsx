import { Link } from 'react-router'
import { buttonVariants } from '@/components/ui/button'

export function NotFoundPage() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <div className="text-center">
        <h1 className="text-6xl font-bold text-foreground">404</h1>
        <p className="text-lg text-muted-foreground mt-2">ไม่พบหน้านี้</p>
        <Link to="/dashboard" className={buttonVariants({ className: 'mt-6' })}>
          กลับหน้าหลัก
        </Link>
      </div>
    </div>
  )
}
