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
import {
  useAdminSalePageDetail,
  useAdminToggleSalePage,
  useAdminDeleteSalePage,
} from '@/hooks/use-admin'
import { timeAgo } from '@/lib/utils'

interface SalePageDetailDialogProps {
  salePageId: string | null
  onClose: () => void
}

export function SalePageDetailDialog({ salePageId, onClose }: SalePageDetailDialogProps) {
  const { data: detail, isLoading } = useAdminSalePageDetail(salePageId)
  const togglePublish = useAdminToggleSalePage()
  const deleteSalePage = useAdminDeleteSalePage()
  const [confirmDelete, setConfirmDelete] = useState(false)

  const sp = detail?.sale_page

  return (
    <Dialog open={!!salePageId} onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto" onClose={onClose}>
        <DialogHeader>
          <DialogTitle>รายละเอียดเซลเพจ</DialogTitle>
        </DialogHeader>

        {isLoading || !sp ? (
          <div className="py-8 text-center text-muted-foreground">กำลังโหลด...</div>
        ) : (
          <div className="space-y-6 mt-4">
            {/* Sale Page Info */}
            <div>
              <h3 className="text-sm font-semibold text-foreground mb-3">ข้อมูลเซลเพจ</h3>
              <div className="grid grid-cols-2 gap-3 text-sm">
                <div>
                  <span className="text-muted-foreground">ชื่อ: </span>
                  <span className="text-foreground">{sp.name}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">Slug: </span>
                  <code className="text-xs bg-muted px-1 py-0.5 rounded">{sp.slug}</code>
                </div>
                <div>
                  <span className="text-muted-foreground">เทมเพลต: </span>
                  <span className="text-foreground">{sp.template_name}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">สถานะ: </span>
                  {sp.is_published ? (
                    <Badge variant="success" className="text-xs">เผยแพร่</Badge>
                  ) : (
                    <Badge variant="secondary" className="text-xs">ปิด</Badge>
                  )}
                </div>
                <div>
                  <span className="text-muted-foreground">สร้างเมื่อ: </span>
                  <span className="text-foreground">{timeAgo(sp.created_at)}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">อัปเดตเมื่อ: </span>
                  <span className="text-foreground">{timeAgo(sp.updated_at)}</span>
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
                  <p className="text-lg font-bold text-foreground">{detail.linked_pixels.length}</p>
                  <p className="text-xs text-muted-foreground">พิกเซลที่เชื่อมต่อ</p>
                </div>
              </div>
            </div>

            {/* Linked Pixels */}
            {detail.linked_pixels.length > 0 && (
              <div>
                <h3 className="text-sm font-semibold text-foreground mb-3">พิกเซลที่เชื่อมต่อ</h3>
                <div className="space-y-2">
                  {detail.linked_pixels.map((px) => (
                    <div key={px.id} className="flex items-center justify-between text-sm border-b border-border pb-2 last:border-0">
                      <div>
                        <span className="text-foreground">{px.name}</span>
                        <span className="text-muted-foreground ml-2 text-xs">{px.fb_pixel_id}</span>
                      </div>
                      <Badge variant={px.is_active ? 'success' : 'secondary'} className="text-xs">
                        {px.is_active ? 'ใช้งาน' : 'ปิด'}
                      </Badge>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Actions */}
            <div className="flex gap-3">
              <Button
                variant={sp.is_published ? 'destructive' : 'default'}
                size="sm"
                onClick={() => togglePublish.mutate({ id: sp.id, enable: !sp.is_published })}
                disabled={togglePublish.isPending}
              >
                {togglePublish.isPending ? 'กำลังอัปเดต...' : sp.is_published ? 'ปิดเผยแพร่' : 'เปิดเผยแพร่'}
              </Button>
              <Button
                variant="destructive"
                size="sm"
                onClick={() => setConfirmDelete(true)}
              >
                ลบเซลเพจ
              </Button>
            </div>
          </div>
        )}
      </DialogContent>

      <Dialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>ยืนยันการลบเซลเพจ</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            คุณต้องการลบเซลเพจ <span className="font-medium text-foreground">{sp?.name}</span> ใช่หรือไม่? การกระทำนี้ไม่สามารถย้อนกลับได้
          </p>
          <DialogFooter>
            <Button variant="outline" size="sm" onClick={() => setConfirmDelete(false)}>ยกเลิก</Button>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => {
                if (!sp) return
                deleteSalePage.mutate(sp.id)
                setConfirmDelete(false)
                onClose()
              }}
              disabled={deleteSalePage.isPending}
            >
              {deleteSalePage.isPending ? 'กำลังลบ...' : 'ยืนยันลบ'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Dialog>
  )
}
