import { useState } from 'react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { useEvents } from '@/hooks/use-events'
import { formatDistanceToNow } from 'date-fns'

export function EventLogPage() {
  const [page, setPage] = useState(1)
  const { data, isLoading } = useEvents(page, 50)

  const events = data?.data ?? []
  const totalPages = data?.total_pages ?? 1

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-neutral-900">Event Log</h1>
          <p className="text-sm text-neutral-500 mt-1">
            {data ? `${data.total} events total` : 'Loading events...'}
          </p>
        </div>
      </div>

      {isLoading ? (
        <div className="text-center py-12 text-neutral-500">Loading events...</div>
      ) : events.length === 0 ? (
        <div className="text-center py-12 border border-dashed border-neutral-300 rounded-lg">
          <p className="text-neutral-500">No events recorded yet</p>
          <p className="text-sm text-neutral-400 mt-1">Events will appear here once your SDK starts sending data</p>
        </div>
      ) : (
        <>
          <div className="border border-neutral-200 rounded-lg overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="border-b border-neutral-200 bg-neutral-50">
                  <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">Event</th>
                  <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">Source URL</th>
                  <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">CAPI</th>
                  <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">Time</th>
                </tr>
              </thead>
              <tbody>
                {events.map((event) => (
                  <tr key={event.id} className="border-b border-neutral-200 last:border-0">
                    <td className="px-4 py-3">
                      <span className="text-sm font-medium text-neutral-900">{event.event_name}</span>
                    </td>
                    <td className="px-4 py-3 text-sm text-neutral-500 max-w-xs truncate">
                      {event.source_url || '-'}
                    </td>
                    <td className="px-4 py-3">
                      <Badge variant={event.forwarded_to_capi ? 'success' : 'secondary'}>
                        {event.forwarded_to_capi ? 'Sent' : 'Pending'}
                      </Badge>
                    </td>
                    <td className="px-4 py-3 text-sm text-neutral-500">
                      {formatDistanceToNow(new Date(event.event_time), { addSuffix: true })}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {totalPages > 1 && (
            <div className="flex items-center justify-between mt-4">
              <p className="text-sm text-neutral-500">
                Page {page} of {totalPages}
              </p>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => setPage((p) => p - 1)}
                >
                  <ChevronLeft className="h-4 w-4" />
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page >= totalPages}
                  onClick={() => setPage((p) => p + 1)}
                >
                  Next
                  <ChevronRight className="h-4 w-4" />
                </Button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  )
}
