import { createBrowserRouter, Navigate } from 'react-router'
import { AppLayout } from '@/components/layout/AppLayout'
import { ProtectedRoute } from '@/components/shared/ProtectedRoute'
import { LoginPage } from '@/pages/auth/LoginPage'
import { RegisterPage } from '@/pages/auth/RegisterPage'
import { DashboardPage } from '@/pages/DashboardPage'
import { PixelsPage } from '@/pages/PixelsPage'
import { EventLogPage } from '@/pages/EventLogPage'
import { ReplayPage } from '@/pages/ReplayPage'
import { SettingsPage } from '@/pages/SettingsPage'
import { SalePagesPage } from '@/pages/SalePagesPage'
import { SalePageEditorPage } from '@/pages/SalePageEditorPage'
import { BlockEditorPage } from '@/pages/BlockEditorPage'

export const router = createBrowserRouter([
  {
    path: '/login',
    element: <LoginPage />,
  },
  {
    path: '/register',
    element: <RegisterPage />,
  },
  {
    path: '/',
    element: (
      <ProtectedRoute>
        <AppLayout />
      </ProtectedRoute>
    ),
    children: [
      { index: true, element: <Navigate to="/dashboard" replace /> },
      { path: 'dashboard', element: <DashboardPage /> },
      { path: 'pixels', element: <PixelsPage /> },
      { path: 'sale-pages', element: <SalePagesPage /> },
      { path: 'sale-pages/new', element: <BlockEditorPage /> },
      { path: 'sale-pages/new-classic', element: <SalePageEditorPage /> },
      { path: 'sale-pages/:id/edit', element: <SalePageEditorPage /> },
      { path: 'sale-pages/:id/edit-blocks', element: <BlockEditorPage /> },
      { path: 'events/log', element: <EventLogPage /> },
      { path: 'replay', element: <ReplayPage /> },
      { path: 'settings', element: <SettingsPage /> },
    ],
  },
])
