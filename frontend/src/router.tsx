import { createBrowserRouter, Navigate } from 'react-router'
import { AppLayout } from '@/components/layout/AppLayout'
import { ProtectedRoute } from '@/components/shared/ProtectedRoute'
import { LoginPage } from '@/pages/auth/LoginPage'
import { HomePage } from '@/pages/HomePage'
import { DashboardPage } from '@/pages/DashboardPage'
import { PixelsPage } from '@/pages/PixelsPage'
import { EventsPage } from '@/pages/EventsPage'
import { ReplayPage } from '@/pages/ReplayPage'
import { SettingsPage } from '@/pages/SettingsPage'
import { SalePagesPage } from '@/pages/SalePagesPage'
import { SalePageEditorPage } from '@/pages/SalePageEditorPage'
import { BlockEditorPage } from '@/pages/BlockEditorPage'
import { BillingPage } from '@/pages/BillingPage'

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
      { path: '/sale-pages/new', element: <BlockEditorPage /> },
      { path: '/sale-pages/new-classic', element: <SalePageEditorPage /> },
      { path: '/sale-pages/:id/edit', element: <SalePageEditorPage /> },
      { path: '/sale-pages/:id/edit-blocks', element: <BlockEditorPage /> },
      { path: '/events', element: <EventsPage /> },
      { path: '/events/log', element: <Navigate to="/events?mode=history" replace /> },
      { path: '/events/realtime', element: <Navigate to="/events?mode=live" replace /> },
      { path: '/replay', element: <ReplayPage /> },
      { path: '/billing', element: <BillingPage /> },
      { path: '/settings', element: <SettingsPage /> },
    ],
  },
])
