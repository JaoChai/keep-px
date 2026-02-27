import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Pause, Play, Trash2, Check, X } from 'lucide-react'
import { useRealtimeEvents } from '@/hooks/use-realtime-events'
import { usePixels } from '@/hooks/use-pixels'
import { formatDistanceToNow } from 'date-fns'

const eventBadgeVariant = (name: string) => {
  switch (name) {
    case 'PageView':
      return 'secondary'
    case 'Purchase':
      return 'success'
    case 'Lead':
    case 'CompleteRegistration':
      return 'default'
    case 'AddToCart':
    case 'InitiateCheckout':
      return 'warning'
    default:
      return 'outline'
  }
}

export function RealtimeEventsPage() {
  const { events, isLive, isPaused, togglePause, clear, pixelId, setPixelId } =
    useRealtimeEvents()
  const { data: pixels } = usePixels()

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <div>
            <h1 className="text-2xl font-bold text-neutral-900">Realtime Events</h1>
            <p className="text-sm text-neutral-500 mt-1">{events.length} events</p>
          </div>
          <Badge variant={isLive ? 'success' : 'warning'}>{isLive ? 'Live' : 'Paused'}</Badge>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={togglePause}>
            {isPaused ? <Play className="h-4 w-4 mr-1" /> : <Pause className="h-4 w-4 mr-1" />}
            {isPaused ? 'Resume' : 'Pause'}
          </Button>
          <Button variant="outline" size="sm" onClick={clear}>
            <Trash2 className="h-4 w-4 mr-1" />
            Clear
          </Button>
        </div>
      </div>

      <div className="mb-4">
        <select
          className="border border-neutral-200 rounded-lg px-3 py-2 text-sm bg-white text-neutral-900"
          value={pixelId ?? ''}
          onChange={(e) => setPixelId(e.target.value || null)}
        >
          <option value="">All Pixels</option>
          {pixels?.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </select>
      </div>

      {events.length === 0 ? (
        <div className="text-center py-16 border border-dashed border-neutral-300 rounded-lg">
          <div className="animate-pulse mb-3">
            <div className="inline-block h-3 w-3 rounded-full bg-emerald-400" />
          </div>
          <p className="text-neutral-500">Waiting for events...</p>
          <p className="text-sm text-neutral-400 mt-1">
            New events will appear here in realtime
          </p>
        </div>
      ) : (
        <div className="border border-neutral-200 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-neutral-200 bg-neutral-50">
                <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">Event</th>
                <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">Pixel</th>
                <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">
                  Source URL
                </th>
                <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">CAPI</th>
                <th className="text-left text-sm font-medium text-neutral-500 px-4 py-3">Time</th>
              </tr>
            </thead>
            <tbody>
              {events.map((event) => (
                <tr
                  key={event.id}
                  className="border-b border-neutral-200 last:border-0 animate-[fadeIn_0.3s_ease-in]"
                >
                  <td className="px-4 py-3">
                    <Badge variant={eventBadgeVariant(event.event_name)}>
                      {event.event_name}
                    </Badge>
                  </td>
                  <td className="px-4 py-3 text-sm text-neutral-700">{event.pixel_name}</td>
                  <td
                    className="px-4 py-3 text-sm text-neutral-500 max-w-xs truncate"
                    title={event.source_url}
                  >
                    {event.source_url || '-'}
                  </td>
                  <td className="px-4 py-3">
                    {event.forwarded_to_capi ? (
                      <Check className="h-4 w-4 text-emerald-600" />
                    ) : (
                      <X className="h-4 w-4 text-red-400" />
                    )}
                  </td>
                  <td className="px-4 py-3 text-sm text-neutral-500">
                    {formatDistanceToNow(new Date(event.event_time), { addSuffix: true })}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
