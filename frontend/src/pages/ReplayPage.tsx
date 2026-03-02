import { useEffect, useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { RotateCcw, Play, CheckCircle2, XCircle, Clock, Loader2, AlertTriangle, StopCircle, Eye } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { toast } from 'sonner'
import { usePixels } from '@/hooks/use-pixels'
import { useReplays, useReplaySession, useCreateReplay, useCancelReplay, useRetryReplay, useReplayPreview, useEventTypes } from '@/hooks/use-replays'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import type { ReplayPreview } from '@/types'

const replaySchema = z.object({
  source_pixel_id: z.string().min(1, 'Source pixel is required'),
  target_pixel_id: z.string().min(1, 'Target pixel is required'),
  date_from: z.string().optional(),
  date_to: z.string().optional(),
  time_mode: z.enum(['original', 'current']),
  batch_delay_ms: z.number().min(0).max(60000),
})

type ReplayForm = z.infer<typeof replaySchema>

function statusBadge(status: string) {
  switch (status) {
    case 'completed': return <Badge variant="success"><CheckCircle2 className="h-3 w-3 mr-1" />Completed</Badge>
    case 'running': return <Badge variant="default"><Loader2 className="h-3 w-3 mr-1 animate-spin" />Running</Badge>
    case 'failed': return <Badge variant="destructive"><XCircle className="h-3 w-3 mr-1" />Failed</Badge>
    case 'cancelled': return <Badge variant="outline"><StopCircle className="h-3 w-3 mr-1" />Cancelled</Badge>
    default: return <Badge variant="secondary"><Clock className="h-3 w-3 mr-1" />Pending</Badge>
  }
}

function toStartOfDay(dateStr: string): string {
  return new Date(dateStr + 'T00:00:00').toISOString()
}

function toEndOfDay(dateStr: string): string {
  return new Date(dateStr + 'T23:59:59').toISOString()
}

export function ReplayPage() {
  const { data: pixels } = usePixels()
  const { data: replays, isLoading } = useReplays()
  const createReplay = useCreateReplay()
  const cancelReplay = useCancelReplay()
  const retryReplay = useRetryReplay()
  const previewReplay = useReplayPreview()
  const [activeReplayId, setActiveReplayId] = useState<string | null>(null)
  const { data: activeReplay } = useReplaySession(activeReplayId)
  const [preview, setPreview] = useState<ReplayPreview | null>(null)
  const [pendingFormData, setPendingFormData] = useState<ReplayForm | null>(null)
  const [selectedEventTypes, setSelectedEventTypes] = useState<string[]>([])
  const [cancelConfirm, setCancelConfirm] = useState<string | null>(null)

  const pixelMap = useMemo(() => {
    const map = new Map<string, string>()
    pixels?.forEach((p) => map.set(p.id, p.name))
    return map
  }, [pixels])

  const {
    register,
    handleSubmit,
    getValues,
    watch,
    formState: { errors },
  } = useForm<ReplayForm>({
    resolver: zodResolver(replaySchema),
    defaultValues: {
      time_mode: 'original',
      batch_delay_ms: 0,
    },
  })

  const watchedSourcePixelId = watch('source_pixel_id')
  const { data: eventTypes } = useEventTypes(watchedSourcePixelId)

  useEffect(() => {
    setSelectedEventTypes([])
  }, [watchedSourcePixelId])

  const onPreview = async (formData: ReplayForm) => {
    try {
      const result = await previewReplay.mutateAsync({
        source_pixel_id: formData.source_pixel_id,
        target_pixel_id: formData.target_pixel_id,
        event_types: selectedEventTypes.length > 0 ? selectedEventTypes : undefined,
        date_from: formData.date_from ? toStartOfDay(formData.date_from) : undefined,
        date_to: formData.date_to ? toEndOfDay(formData.date_to) : undefined,
      })
      setPreview(result)
      setPendingFormData(formData)
    } catch {
      toast.error('Failed to load preview')
    }
  }

  const onConfirmReplay = async () => {
    if (!pendingFormData) return
    try {
      const result = await createReplay.mutateAsync({
        source_pixel_id: pendingFormData.source_pixel_id,
        target_pixel_id: pendingFormData.target_pixel_id,
        event_types: selectedEventTypes.length > 0 ? selectedEventTypes : undefined,
        date_from: pendingFormData.date_from ? toStartOfDay(pendingFormData.date_from) : undefined,
        date_to: pendingFormData.date_to ? toEndOfDay(pendingFormData.date_to) : undefined,
        time_mode: pendingFormData.time_mode,
        batch_delay_ms: pendingFormData.batch_delay_ms || undefined,
      })
      setActiveReplayId(result.session.id)
      setPreview(null)
      setPendingFormData(null)
      if (result.warning) {
        toast.warning(result.warning)
      }
    } catch {
      toast.error('Failed to start replay')
    }
  }

  const handleCancel = async (id: string) => {
    try {
      await cancelReplay.mutateAsync(id)
      toast.success('Replay cancelled')
    } catch {
      toast.error('Failed to cancel replay')
    }
  }

  const handleRetry = async (id: string) => {
    try {
      const result = await retryReplay.mutateAsync(id)
      setActiveReplayId(result.id)
      toast.success('Retry started')
    } catch {
      toast.error('Failed to retry replay')
    }
  }

  const canCancel = activeReplay && (activeReplay.status === 'running' || activeReplay.status === 'pending')
  const canRetry = activeReplay && (activeReplay.status === 'failed' || activeReplay.status === 'cancelled') && activeReplay.failed_events > 0

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-neutral-900">Replay Center</h1>
        <p className="text-sm text-neutral-500 mt-1">Replay events from one pixel to another</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Replay Form */}
        <Card className="lg:col-span-1">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Play className="h-4 w-4" />
              New Replay
            </CardTitle>
          </CardHeader>
          <CardContent>
            {!preview ? (
              <form onSubmit={handleSubmit(onPreview)} className="space-y-4">
                <div className="space-y-2">
                  <Label>Source Pixel</Label>
                  <select
                    className="flex h-9 w-full rounded-md border border-neutral-200 bg-transparent px-3 py-1 text-sm"
                    {...register('source_pixel_id')}
                  >
                    <option value="">Select source...</option>
                    {pixels?.map((p) => (
                      <option key={p.id} value={p.id}>{p.name} ({p.fb_pixel_id})</option>
                    ))}
                  </select>
                  {errors.source_pixel_id && <p className="text-sm text-red-500">{errors.source_pixel_id.message}</p>}
                </div>

                <div className="space-y-2">
                  <Label>Target Pixel</Label>
                  <select
                    className="flex h-9 w-full rounded-md border border-neutral-200 bg-transparent px-3 py-1 text-sm"
                    {...register('target_pixel_id')}
                  >
                    <option value="">Select target...</option>
                    {pixels?.map((p) => (
                      <option key={p.id} value={p.id}>{p.name} ({p.fb_pixel_id})</option>
                    ))}
                  </select>
                  {errors.target_pixel_id && <p className="text-sm text-red-500">{errors.target_pixel_id.message}</p>}
                </div>

                {eventTypes && eventTypes.length > 0 && (
                  <div className="space-y-2">
                    <Label>Event Types (optional)</Label>
                    <div className="space-y-1.5 max-h-32 overflow-y-auto rounded-md border border-neutral-200 p-2">
                      {eventTypes.map((type) => (
                        <label key={type} className="flex items-center gap-2 text-sm cursor-pointer">
                          <input
                            type="checkbox"
                            checked={selectedEventTypes.includes(type)}
                            onChange={(e) => {
                              setSelectedEventTypes(prev =>
                                e.target.checked
                                  ? [...prev, type]
                                  : prev.filter(t => t !== type)
                              )
                            }}
                            className="rounded border-neutral-300"
                          />
                          {type}
                        </label>
                      ))}
                    </div>
                    <p className="text-xs text-neutral-400">Leave unchecked to include all event types</p>
                  </div>
                )}

                <div className="space-y-2">
                  <Label>Date From (optional)</Label>
                  <Input type="date" {...register('date_from')} />
                </div>

                <div className="space-y-2">
                  <Label>Date To (optional)</Label>
                  <Input type="date" {...register('date_to')} />
                </div>

                <div className="space-y-2">
                  <Label>Time Mode</Label>
                  <select
                    className="flex h-9 w-full rounded-md border border-neutral-200 bg-transparent px-3 py-1 text-sm"
                    {...register('time_mode')}
                  >
                    <option value="original">Original (use original event timestamps)</option>
                    <option value="current">Current (use current time for all events)</option>
                  </select>
                  <p className="text-xs text-neutral-400">Use "Current" if events are older than 7 days to avoid Facebook rejection</p>
                </div>

                <div className="space-y-2">
                  <Label>Batch Delay (ms)</Label>
                  <Input
                    type="number"
                    min={0}
                    max={60000}
                    placeholder="0"
                    {...register('batch_delay_ms', { valueAsNumber: true })}
                  />
                  <p className="text-xs text-neutral-400">Delay between batches (0-60000ms). Use for warm-up on new pixels.</p>
                </div>

                <Button type="submit" className="w-full" disabled={previewReplay.isPending}>
                  <Eye className="h-4 w-4" />
                  {previewReplay.isPending ? 'Loading Preview...' : 'Preview'}
                </Button>
              </form>
            ) : (
              <div className="space-y-4">
                <div className="rounded-lg border border-neutral-200 p-3 space-y-2">
                  <p className="text-sm font-medium text-neutral-900">Preview Summary</p>
                  <p className="text-sm text-neutral-600">
                    <span className="font-semibold">{preview.total_events}</span> events will be replayed
                  </p>
                  <p className="text-xs text-neutral-500">
                    From: {pixelMap.get(getValues('source_pixel_id')) || 'Unknown'} → To: {pixelMap.get(getValues('target_pixel_id')) || 'Unknown'}
                  </p>
                </div>

                {preview.warning && (
                  <div className="flex items-start gap-2 rounded-lg border border-amber-200 bg-amber-50 p-3">
                    <AlertTriangle className="h-4 w-4 text-amber-600 mt-0.5 shrink-0" />
                    <p className="text-sm text-amber-700">{preview.warning}</p>
                  </div>
                )}

                {(preview.sample_events?.length ?? 0) > 0 && (
                  <div className="space-y-2">
                    <p className="text-xs font-medium text-neutral-500">Sample Events</p>
                    <div className="max-h-48 overflow-y-auto rounded-lg border border-neutral-200">
                      <table className="w-full text-xs">
                        <thead className="bg-neutral-50 sticky top-0">
                          <tr>
                            <th className="text-left px-2 py-1.5 font-medium text-neutral-600">Event</th>
                            <th className="text-left px-2 py-1.5 font-medium text-neutral-600">Time</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-neutral-100">
                          {preview.sample_events.map((event) => (
                            <tr key={event.id}>
                              <td className="px-2 py-1.5 text-neutral-900">{event.event_name}</td>
                              <td className="px-2 py-1.5 text-neutral-500">{new Date(event.event_time).toLocaleString()}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </div>
                )}

                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    className="flex-1"
                    onClick={() => { setPreview(null); setPendingFormData(null) }}
                  >
                    Back
                  </Button>
                  <Button
                    className="flex-1"
                    onClick={onConfirmReplay}
                    disabled={createReplay.isPending}
                  >
                    <RotateCcw className="h-4 w-4" />
                    {createReplay.isPending ? 'Starting...' : 'Confirm Replay'}
                  </Button>
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Active Replay Progress */}
        {activeReplay && (
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle className="flex items-center justify-between text-base">
                <span>Replay Progress</span>
                {statusBadge(activeReplay.status)}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {/* Error message */}
                {activeReplay.status === 'failed' && activeReplay.error_message && (
                  <div className="flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 p-3">
                    <AlertTriangle className="h-4 w-4 text-red-600 mt-0.5 shrink-0" />
                    <div>
                      <p className="text-sm font-medium text-red-800">Replay Failed</p>
                      <p className="text-sm text-red-600 mt-1">{activeReplay.error_message}</p>
                    </div>
                  </div>
                )}

                <div className="w-full bg-neutral-200 rounded-full h-3">
                  <div
                    className="bg-indigo-600 h-3 rounded-full transition-all duration-500"
                    style={{
                      width: `${activeReplay.total_events > 0
                        ? ((activeReplay.replayed_events + activeReplay.failed_events) / activeReplay.total_events) * 100
                        : 0}%`
                    }}
                  />
                </div>
                <div className="grid grid-cols-3 gap-4 text-center">
                  <div>
                    <p className="text-2xl font-bold text-neutral-900">{activeReplay.total_events}</p>
                    <p className="text-xs text-neutral-500">Total</p>
                  </div>
                  <div>
                    <p className="text-2xl font-bold text-emerald-600">{activeReplay.replayed_events}</p>
                    <p className="text-xs text-neutral-500">Replayed</p>
                  </div>
                  <div>
                    <p className="text-2xl font-bold text-red-600">{activeReplay.failed_events}</p>
                    <p className="text-xs text-neutral-500">Failed</p>
                  </div>
                </div>

                {/* Action buttons */}
                <div className="flex gap-2">
                  {canCancel && (
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => setCancelConfirm(activeReplay.id)}
                      disabled={cancelReplay.isPending}
                    >
                      <StopCircle className="h-4 w-4" />
                      {cancelReplay.isPending ? 'Cancelling...' : 'Cancel'}
                    </Button>
                  )}
                  {canRetry && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleRetry(activeReplay.id)}
                      disabled={retryReplay.isPending}
                    >
                      <RotateCcw className="h-4 w-4" />
                      {retryReplay.isPending ? 'Retrying...' : 'Retry Failed'}
                    </Button>
                  )}
                </div>

                {/* Replay config info */}
                <div className="flex gap-3 text-xs text-neutral-400 border-t border-neutral-100 pt-3">
                  <span>Mode: {activeReplay.time_mode}</span>
                  {activeReplay.batch_delay_ms > 0 && <span>Delay: {activeReplay.batch_delay_ms}ms</span>}
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Replay History */}
        {!activeReplay && (
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle className="text-base">Replay History</CardTitle>
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <p className="text-neutral-500 text-sm">Loading...</p>
              ) : !replays || replays.length === 0 ? (
                <p className="text-neutral-500 text-sm">No replays yet</p>
              ) : (
                <div className="space-y-3">
                  {replays.map((replay) => (
                    <div
                      key={replay.id}
                      className="flex items-center justify-between p-3 rounded-lg border border-neutral-200 cursor-pointer hover:bg-neutral-50"
                      onClick={() => setActiveReplayId(replay.id)}
                    >
                      <div>
                        <p className="text-sm font-medium text-neutral-900">
                          {pixelMap.get(replay.source_pixel_id) || replay.source_pixel_id.slice(0, 8)} → {pixelMap.get(replay.target_pixel_id) || replay.target_pixel_id.slice(0, 8)}
                        </p>
                        <p className="text-xs text-neutral-500">
                          {replay.replayed_events}/{replay.total_events} events &middot; {new Date(replay.created_at).toLocaleString()}
                          {replay.error_message && (
                            <span className="text-red-500 ml-2">{replay.error_message}</span>
                          )}
                        </p>
                      </div>
                      {statusBadge(replay.status)}
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        )}
      </div>

      <Dialog open={cancelConfirm !== null} onOpenChange={(open) => !open && setCancelConfirm(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Cancel Replay?</DialogTitle>
            <DialogDescription>
              Are you sure you want to cancel this replay? Events already replayed will not be rolled back.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCancelConfirm(null)}>
              Keep Running
            </Button>
            <Button
              variant="destructive"
              onClick={() => {
                if (cancelConfirm) {
                  handleCancel(cancelConfirm)
                  setCancelConfirm(null)
                }
              }}
              disabled={cancelReplay.isPending}
            >
              {cancelReplay.isPending ? 'Cancelling...' : 'Yes, Cancel Replay'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
