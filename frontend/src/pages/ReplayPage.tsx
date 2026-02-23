import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { RotateCcw, Play, CheckCircle2, XCircle, Clock, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { usePixels } from '@/hooks/use-pixels'
import { useReplays, useReplaySession, useCreateReplay } from '@/hooks/use-replays'

const replaySchema = z.object({
  source_pixel_id: z.string().min(1, 'Source pixel is required'),
  target_pixel_id: z.string().min(1, 'Target pixel is required'),
  date_from: z.string().optional(),
  date_to: z.string().optional(),
})

type ReplayForm = z.infer<typeof replaySchema>

function statusBadge(status: string) {
  switch (status) {
    case 'completed': return <Badge variant="success"><CheckCircle2 className="h-3 w-3 mr-1" />Completed</Badge>
    case 'running': return <Badge variant="default"><Loader2 className="h-3 w-3 mr-1 animate-spin" />Running</Badge>
    case 'failed': return <Badge variant="destructive"><XCircle className="h-3 w-3 mr-1" />Failed</Badge>
    default: return <Badge variant="secondary"><Clock className="h-3 w-3 mr-1" />Pending</Badge>
  }
}

export function ReplayPage() {
  const { data: pixels } = usePixels()
  const { data: replays, isLoading } = useReplays()
  const createReplay = useCreateReplay()
  const [activeReplayId, setActiveReplayId] = useState<string | null>(null)
  const { data: activeReplay } = useReplaySession(activeReplayId)

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<ReplayForm>({
    resolver: zodResolver(replaySchema),
  })

  const onSubmit = async (formData: ReplayForm) => {
    const result = await createReplay.mutateAsync({
      source_pixel_id: formData.source_pixel_id,
      target_pixel_id: formData.target_pixel_id,
      date_from: formData.date_from ? new Date(formData.date_from).toISOString() : undefined,
      date_to: formData.date_to ? new Date(formData.date_to).toISOString() : undefined,
    })
    setActiveReplayId(result.id)
  }

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
            <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
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

              <div className="space-y-2">
                <Label>Date From (optional)</Label>
                <Input type="date" {...register('date_from')} />
              </div>

              <div className="space-y-2">
                <Label>Date To (optional)</Label>
                <Input type="date" {...register('date_to')} />
              </div>

              <Button type="submit" className="w-full" disabled={isSubmitting}>
                <RotateCcw className="h-4 w-4" />
                {isSubmitting ? 'Starting...' : 'Start Replay'}
              </Button>
            </form>
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
                          {replay.replayed_events}/{replay.total_events} events
                        </p>
                        <p className="text-xs text-neutral-500">
                          {new Date(replay.created_at).toLocaleString()}
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
    </div>
  )
}
