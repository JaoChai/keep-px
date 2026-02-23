import { Outlet } from 'react-router'
import { Sidebar } from './Sidebar'

export function AppLayout() {
  return (
    <div className="min-h-screen bg-neutral-50">
      <Sidebar />
      <main className="pl-[260px]">
        <div className="p-8">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
