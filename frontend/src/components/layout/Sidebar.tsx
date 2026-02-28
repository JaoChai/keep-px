import { NavLink, useNavigate } from 'react-router'
import {
  LayoutDashboard,
  Radio,
  Activity,
  FileText,
  Globe,
  RotateCcw,
  Settings,
  LogOut,
} from 'lucide-react'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'

const navItems = [
  { to: '/dashboard', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/pixels', icon: Radio, label: 'Pixels' },
  { to: '/sale-pages', icon: FileText, label: 'Sale Pages' },
  { to: '/domains', icon: Globe, label: 'Custom Domains' },
  { to: '/events', icon: Activity, label: 'Events' },
  { to: '/replay', icon: RotateCcw, label: 'Replay Center' },
  { to: '/settings', icon: Settings, label: 'Settings' },
]

export function Sidebar() {
  const logout = useAuthStore((s) => s.logout)
  const navigate = useNavigate()

  const handleLogout = () => {
    logout()
    toast.success('ออกจากระบบสำเร็จ')
    navigate('/login', { replace: true })
  }

  return (
    <aside className="fixed inset-y-0 left-0 z-50 w-[260px] border-r border-neutral-200 bg-white flex flex-col">
      <div className="flex h-16 items-center px-6 border-b border-neutral-200">
        <h1 className="text-xl font-bold text-indigo-600">Pixlinks</h1>
      </div>

      <nav className="flex-1 overflow-y-auto py-4 px-3">
        <ul className="space-y-1">
          {navItems.map((item) => (
            <li key={item.to}>
              <NavLink
                to={item.to}
                className={({ isActive }) =>
                  `flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors ${
                    isActive
                      ? 'bg-indigo-50 text-indigo-600'
                      : 'text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900'
                  }`
                }
              >
                <item.icon className="h-5 w-5" />
                {item.label}
              </NavLink>
            </li>
          ))}
        </ul>
      </nav>

      <div className="border-t border-neutral-200 p-3">
        <button
          onClick={handleLogout}
          className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900 transition-colors"
        >
          <LogOut className="h-5 w-5" />
          Logout
        </button>
      </div>
    </aside>
  )
}
