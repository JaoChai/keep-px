import { lazy, Suspense } from 'react'
import { createBrowserRouter, Navigate } from 'react-router'
import { Loader2 } from 'lucide-react'
import { AppLayout } from '@/components/layout/AppLayout'
import { ProtectedRoute } from '@/components/shared/ProtectedRoute'
import { LoginPage } from '@/pages/auth/LoginPage'
import { HomePage } from '@/pages/HomePage'
import { DashboardPage } from '@/pages/DashboardPage'
import { PixelsPage } from '@/pages/PixelsPage'
import { EventsPage } from '@/pages/EventsPage'
import { SettingsPage } from '@/pages/SettingsPage'
import { SalePagesPage } from '@/pages/SalePagesPage'
import { NotFoundPage } from '@/pages/NotFoundPage'

const SalePageEditorPage = lazy(() => import('@/pages/SalePageEditorPage').then(m => ({ default: m.SalePageEditorPage })))
const BlockEditorPage = lazy(() => import('@/pages/BlockEditorPage').then(m => ({ default: m.BlockEditorPage })))
const BillingPage = lazy(() => import('@/pages/BillingPage').then(m => ({ default: m.BillingPage })))
const ReplayPage = lazy(() => import('@/pages/ReplayPage').then(m => ({ default: m.ReplayPage })))

const lazyFallback = <div className="flex items-center justify-center py-24"><Loader2 className="h-6 w-6 animate-spin text-muted-foreground" /></div>

export const router = createBrowserRouter([
  {
    path: '/',
    element: <HomePage />,
  },
  {
    path: '/login',
    element: <LoginPage />,
  },
  {
    path: '/register',
    element: <Navigate to="/login" replace />,
  },
  {
    element: (
      <ProtectedRoute>
        <AppLayout />
      </ProtectedRoute>
    ),
    children: [
      { path: '/dashboard', element: <DashboardPage /> },
      { path: '/pixels', element: <PixelsPage /> },
      { path: '/sale-pages', element: <SalePagesPage /> },
      { path: '/sale-pages/new', element: <Suspense fallback={lazyFallback}><BlockEditorPage /></Suspense> },
      { path: '/sale-pages/new-classic', element: <Suspense fallback={lazyFallback}><SalePageEditorPage /></Suspense> },
      { path: '/sale-pages/:id/edit', element: <Suspense fallback={lazyFallback}><SalePageEditorPage /></Suspense> },
      { path: '/sale-pages/:id/edit-blocks', element: <Suspense fallback={lazyFallback}><BlockEditorPage /></Suspense> },
      { path: '/events', element: <EventsPage /> },
      { path: '/events/log', element: <Navigate to="/events?mode=history" replace /> },
      { path: '/events/realtime', element: <Navigate to="/events?mode=live" replace /> },
      { path: '/replay', element: <Suspense fallback={lazyFallback}><ReplayPage /></Suspense> },
      { path: '/billing', element: <Suspense fallback={lazyFallback}><BillingPage /></Suspense> },
      { path: '/settings', element: <SettingsPage /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
  {
    path: '*',
    element: <NotFoundPage />,
  },
])
