import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useAdminPixelDetail, useAdminTogglePixel } from '@/hooks/use-admin'
import { timeAgo } from '@/lib/utils'

interface PixelDetailDialogProps {
  pixelId: string | null
  onClose: () => void
}

export function PixelDetailDialog({ pixelId, onClose }: PixelDetailDialogProps) {
  const { data: detail, isLoading } = useAdminPixelDetail(pixelId)
  const toggleActive = useAdminTogglePixel()

  const px = detail?.pixel

  return (
    <Dialog open={!!pixelId} onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto" onClose={onClose}>
        <DialogHeader>
          <DialogTitle>รายละเอียดพิกเซล</DialogTitle>
        </DialogHeader>

        {isLoading || !px ? (
          <div className="py-8 text-center text-muted-foreground">กำลังโหลด...</div>
        ) : (
          <div className="space-y-6 mt-4">
            {/* Pixel Info */}
            <div>
              <h3 className="text-sm font-semibold text-foreground mb-3">ข้อมูลพิกเซล</h3>
              <div className="grid grid-cols-2 gap-3 text-sm">
                <div>
                  <span className="text-muted-foreground">ชื่อ: </span>
                  <span className="text-foreground">{px.name}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">FB Pixel ID: </span>
                  <code className="text-xs bg-muted px-1 py-0.5 rounded">{px.fb_pixel_id}</code>
                </div>
                <div>
                  <span className="text-muted-foreground">สถานะ: </span>
                  {px.is_active ? (
                    <Badge variant="success" className="text-xs">ใช้งาน</Badge>
                  ) : (
                    <Badge variant="secondary" className="text-xs">ปิด</Badge>
                  )}
                </div>
                <div>
                  <span className="text-muted-foreground">Status: </span>
                  <span className="text-foreground">{px.status}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">สร้างเมื่อ: </span>
                  <span className="text-foreground">{timeAgo(px.created_at)}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">อัปเดตเมื่อ: </span>
                  <span className="text-foreground">{timeAgo(px.updated_at)}</span>
                </div>
              </div>
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

            {/* Stats */}
            <div>
              <h3 className="text-sm font-semibold text-foreground mb-3">สถิติ</h3>
              <div className="grid grid-cols-2 gap-3">
                <div className="bg-muted rounded-lg p-3 text-center">
                  <p className="text-lg font-bold text-foreground">{detail.event_count.toLocaleString()}</p>
                  <p className="text-xs text-muted-foreground">อีเวนต์</p>
                </div>
                <div className="bg-muted rounded-lg p-3 text-center">
                  <p className="text-lg font-bold text-foreground">{detail.linked_sale_pages.length}</p>
                  <p className="text-xs text-muted-foreground">เซลเพจที่เชื่อมต่อ</p>
                </div>
              </div>
            </div>

            {/* Linked Sale Pages */}
            {detail.linked_sale_pages.length > 0 && (
              <div>
                <h3 className="text-sm font-semibold text-foreground mb-3">เซลเพจที่เชื่อมต่อ</h3>
                <div className="space-y-2">
                  {detail.linked_sale_pages.map((sp) => (
                    <div key={sp.id} className="flex items-center justify-between text-sm border-b border-border pb-2 last:border-0">
                      <div>
                        <span className="text-foreground">{sp.name}</span>
                        <span className="text-muted-foreground ml-2 text-xs">/{sp.slug}</span>
                      </div>
                      <Badge variant={sp.is_published ? 'success' : 'secondary'} className="text-xs">
                        {sp.is_published ? 'เผยแพร่' : 'ปิด'}
                      </Badge>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Actions */}
            <div>
              <Button
                variant={px.is_active ? 'destructive' : 'default'}
                size="sm"
                onClick={() => toggleActive.mutate({ id: px.id, enable: !px.is_active })}
                disabled={toggleActive.isPending}
              >
                {toggleActive.isPending ? 'กำลังอัปเดต...' : px.is_active ? 'ปิดการใช้งาน' : 'เปิดการใช้งาน'}
              </Button>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
