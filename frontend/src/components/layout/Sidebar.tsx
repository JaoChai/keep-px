import { useEffect } from 'react'
import { NavLink, useNavigate } from 'react-router'
import {
  LayoutDashboard,
  Radio,
  Activity,
  FileText,
  RotateCcw,
  CreditCard,
  Settings,
  LogOut,
  X,
  Shield,
  Users,
  BarChart3,
  Receipt,
} from 'lucide-react'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { NotificationBell } from './NotificationBell'

const navItems = [
  { to: '/dashboard', icon: LayoutDashboard, label: 'แดชบอร์ด' },
  { to: '/pixels', icon: Radio, label: 'พิกเซล' },
  { to: '/sale-pages', icon: FileText, label: 'หน้าขาย' },
  { to: '/events', icon: Activity, label: 'อีเวนต์' },
  { to: '/replay', icon: RotateCcw, label: 'รีเพลย์' },
  { to: '/billing', icon: CreditCard, label: 'การเงิน' },
  { to: '/settings', icon: Settings, label: 'ตั้งค่า' },
]

interface SidebarProps {
  open: boolean
  onClose: () => void
}

const adminNavItems = [
  { to: '/admin/customers', icon: Users, label: 'ลูกค้า' },
  { to: '/admin/analytics', icon: BarChart3, label: 'สถิติแพลตฟอร์ม' },
  { to: '/admin/billing', icon: Receipt, label: 'การเงินทั้งหมด' },
]

export function Sidebar({ open, onClose }: SidebarProps) {
  const logout = useAuthStore((s) => s.logout)
  const customer = useAuthStore((s) => s.customer)
  const navigate = useNavigate()

  const handleLogout = () => {
    logout()
    toast.success('ออกจากระบบสำเร็จ')
    navigate('/login', { replace: true })
  }

  // Escape key + body scroll lock for mobile overlay
  useEffect(() => {
    if (!open) return
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handleKeyDown)
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      document.body.style.overflow = ''
    }
  }, [open, onClose])

  const sidebarContent = (
    <aside className="flex h-full w-[260px] flex-col border-r border-border bg-card">
      <div className="flex h-16 items-center justify-between px-6 border-b border-border">
        <h1 className="text-xl font-bold text-foreground">Pixlinks</h1>
        <div className="flex items-center gap-1">
          <NotificationBell />
          <button
            onClick={onClose}
            aria-label="Close sidebar"
            className="lg:hidden rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
      </div>

      <nav className="flex-1 overflow-y-auto py-4 px-3">
        <ul className="space-y-1">
          {navItems.map((item) => (
            <li key={item.to}>
              <NavLink
                to={item.to}
                onClick={onClose}
                className={({ isActive }) =>
                  `flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors ${
                    isActive
                      ? 'bg-accent text-accent-foreground'
                      : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
                  }`
                }
              >
                <item.icon className="h-5 w-5" />
                {item.label}
              </NavLink>
            </li>
          ))}
        </ul>

        {customer?.is_admin && (
          <>
            <div className="my-4 border-t border-border" />
            <div className="flex items-center gap-2 px-3 mb-2">
              <Shield className="h-4 w-4 text-muted-foreground" />
              <span className="text-xs font-semibold uppercase text-muted-foreground tracking-wider">แอดมิน</span>
            </div>
            <ul className="space-y-1">
              {adminNavItems.map((item) => (
                <li key={item.to}>
                  <NavLink
                    to={item.to}
                    onClick={onClose}
                    className={({ isActive }) =>
                      `flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors ${
                        isActive
                          ? 'bg-accent text-accent-foreground'
                          : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
                      }`
                    }
                  >
                    <item.icon className="h-5 w-5" />
                    {item.label}
                  </NavLink>
                </li>
              ))}
            </ul>
          </>
        )}
      </nav>

      <div className="border-t border-border p-3">
        <button
          onClick={handleLogout}
          className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-muted-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
        >
          <LogOut className="h-5 w-5" />
          ออกจากระบบ
        </button>
      </div>
    </aside>
  )

  return (
    <>
      {/* Desktop sidebar */}
      <div className="hidden lg:block fixed inset-y-0 left-0 z-50">
        {sidebarContent}
      </div>

      {/* Mobile overlay sidebar */}
      {open && (
        <div className="fixed inset-0 z-50 lg:hidden" role="dialog" aria-modal="true">
          <div
            className="fixed inset-0 bg-black/50"
            onClick={onClose}
            aria-hidden="true"
          />
          <div className="fixed inset-y-0 left-0 z-50">
            {sidebarContent}
          </div>
        </div>
      )}
    </>
  )
}
