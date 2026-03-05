import { useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useAdminReplayDetail, useAdminCancelReplay } from '@/hooks/use-admin'
import { timeAgo } from '@/lib/utils'

interface ReplayDetailDialogProps {
  replayId: string | null
  onClose: () => void
}

const STATUS_BADGE: Record<string, 'warning' | 'default' | 'success' | 'destructive' | 'secondary'> = {
  pending: 'warning',
  running: 'default',
  completed: 'success',
  failed: 'destructive',
  cancelled: 'secondary',
}

const STATUS_LABELS: Record<string, string> = {
  pending: 'รอดำเนินการ',
  running: 'กำลังทำงาน',
  completed: 'สำเร็จ',
  failed: 'ล้มเหลว',
  cancelled: 'ยกเลิก',
}

export function ReplayDetailDialog({ replayId, onClose }: ReplayDetailDialogProps) {
  const { data: detail, isLoading } = useAdminReplayDetail(replayId)
  const cancelReplay = useAdminCancelReplay()
  const [confirmCancel, setConfirmCancel] = useState(false)

  const session = detail?.session

  const progress = session && session.total_events > 0
    ? Math.round((session.replayed_events / session.total_events) * 100)
    : 0

  const canCancel = session && (session.status === 'pending' || session.status === 'running')

  return (
    <Dialog open={!!replayId} onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto" onClose={onClose}>
        <DialogHeader>
          <DialogTitle>รายละเอียดรีเพลย์</DialogTitle>
        </DialogHeader>

        {isLoading || !session ? (
          <div className="py-8 text-center text-muted-foreground">กำลังโหลด...</div>
        ) : (
          <div className="space-y-6 mt-4">
            {/* Session Info */}
            <div>
              <h3 className="text-sm font-semibold text-foreground mb-3">ข้อมูลเซสชัน</h3>
              <div className="grid grid-cols-2 gap-3 text-sm">
                <div>
                  <span className="text-muted-foreground">สถานะ: </span>
                  <Badge variant={STATUS_BADGE[session.status] ?? 'secondary'} className="text-xs">
                    {STATUS_LABELS[session.status] ?? session.status}
                  </Badge>
                </div>
                <div>
                  <span className="text-muted-foreground">สร้างเมื่อ: </span>
                  <span className="text-foreground">{timeAgo(session.created_at)}</span>
                </div>
                {session.started_at && (
                  <div>
                    <span className="text-muted-foreground">เริ่มเมื่อ: </span>
                    <span className="text-foreground">{timeAgo(session.started_at)}</span>
                  </div>
                )}
                {session.completed_at && (
                  <div>
                    <span className="text-muted-foreground">เสร็จเมื่อ: </span>
                    <span className="text-foreground">{timeAgo(session.completed_at)}</span>
                  </div>
                )}
                {session.event_types && session.event_types.length > 0 && (
                  <div className="col-span-2">
                    <span className="text-muted-foreground">ประเภทอีเวนต์: </span>
                    <span className="text-foreground">{session.event_types.join(', ')}</span>
                  </div>
                )}
                {session.error_message && (
                  <div className="col-span-2">
                    <span className="text-muted-foreground">ข้อผิดพลาด: </span>
                    <span className="text-destructive">{session.error_message}</span>
                  </div>
                )}
              </div>
            </div>

            {/* Progress Bar */}
            <div>
              <h3 className="text-sm font-semibold text-foreground mb-3">ความคืบหน้า</h3>
              <div className="w-full bg-muted rounded-full h-3 mb-2">
                <div
                  className="bg-foreground h-3 rounded-full transition-all"
                  style={{ width: `${progress}%` }}
                />
              </div>
              <div className="flex justify-between text-xs text-muted-foreground">
                <span>{session.replayed_events.toLocaleString()} / {session.total_events.toLocaleString()} อีเวนต์</span>
                <span>{progress}%</span>
              </div>
              {session.failed_events > 0 && (
                <p className="text-xs text-destructive mt-1">ล้มเหลว {session.failed_events.toLocaleString()} อีเวนต์</p>
              )}
            </div>

            {/* Customer Info */}
            <div>
              <h3 className="text-sm font-semibold text-foreground mb-3">ลูกค้า</h3>
              <div className="grid grid-cols-2 gap-3 text-sm">
                <div>
                  <span className="text-muted-foreground">อีเมล: </span>
                  <span className="text-foreground">{detail.customer_email}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">ชื่อ: </span>
                  <span className="text-foreground">{detail.customer_name}</span>
                </div>
              </div>
            </div>

            {/* Source / Target Pixels */}
            <div>
              <h3 className="text-sm font-semibold text-foreground mb-3">พิกเซล</h3>
              <div className="grid grid-cols-2 gap-4">
                <div className="bg-muted rounded-lg p-3">
                  <p className="text-xs text-muted-foreground mb-1">Source</p>
                  <p className="text-sm font-medium text-foreground">{detail.source_pixel.name}</p>
                  <p className="text-xs text-muted-foreground">{detail.source_pixel.fb_pixel_id}</p>
                  <Badge variant={detail.source_pixel.is_active ? 'success' : 'secondary'} className="text-xs mt-1">
                    {detail.source_pixel.is_active ? 'ใช้งาน' : 'ปิด'}
                  </Badge>
                </div>
                <div className="bg-muted rounded-lg p-3">
                  <p className="text-xs text-muted-foreground mb-1">Target</p>
                  <p className="text-sm font-medium text-foreground">{detail.target_pixel.name}</p>
                  <p className="text-xs text-muted-foreground">{detail.target_pixel.fb_pixel_id}</p>
                  <Badge variant={detail.target_pixel.is_active ? 'success' : 'secondary'} className="text-xs mt-1">
                    {detail.target_pixel.is_active ? 'ใช้งาน' : 'ปิด'}
                  </Badge>
                </div>
              </div>
            </div>

            {/* Cancel Action */}
            {canCancel && (
              <div>
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => setConfirmCancel(true)}
                >
                  ยกเลิกรีเพลย์
                </Button>
              </div>
            )}
          </div>
        )}
      </DialogContent>

      <Dialog open={confirmCancel} onOpenChange={setConfirmCancel}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>ยืนยันการยกเลิกรีเพลย์</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            คุณต้องการยกเลิกเซสชันรีเพลย์นี้ใช่หรือไม่?
          </p>
          <DialogFooter>
            <Button variant="outline" size="sm" onClick={() => setConfirmCancel(false)}>ปิด</Button>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => {
                if (!session) return
                cancelReplay.mutate(session.id)
                setConfirmCancel(false)
              }}
              disabled={cancelReplay.isPending}
            >
              {cancelReplay.isPending ? 'กำลังยกเลิก...' : 'ยืนยันยกเลิก'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Dialog>
  )
}
