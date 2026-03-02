import { AlertTriangle, RefreshCw } from 'lucide-react'
import { cn } from '@/lib/utils'

interface QueryErrorAlertProps {
  error: Error | null
  onRetry?: () => void
  className?: string
}

export function QueryErrorAlert({ error, onRetry, className }: QueryErrorAlertProps) {
  if (!error) return null

  return (
    <div className={cn("flex items-start gap-3 rounded-lg border border-red-200 bg-red-50 p-4", className)}>
      <AlertTriangle className="h-5 w-5 text-red-600 mt-0.5 shrink-0" />
      <div className="flex-1">
        <p className="text-sm font-medium text-red-800">
          เกิดข้อผิดพลาดในการโหลดข้อมูล
        </p>
        {error.message && (
          <p className="text-sm text-red-600 mt-1">{error.message}</p>
        )}
        {onRetry && (
          <button
            onClick={onRetry}
            className="inline-flex items-center gap-1.5 mt-3 rounded-md bg-red-100 px-3 py-1.5 text-sm font-medium text-red-700 hover:bg-red-200 transition-colors"
          >
            <RefreshCw className="h-3.5 w-3.5" />
            ลองใหม่
          </button>
        )}
      </div>
    </div>
  )
}
