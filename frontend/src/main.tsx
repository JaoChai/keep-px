import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from 'sonner'
import { ErrorBoundary } from './components/shared/ErrorBoundary'
import { router } from './router'
import './index.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000,
      retry: 1,
    },
  },
})

const content = (
  <QueryClientProvider client={queryClient}>
    <RouterProvider router={router} />
    <Toaster richColors position="top-right" />
  </QueryClientProvider>
)

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ErrorBoundary>
      {content}
    </ErrorBoundary>
  </StrictMode>
)
