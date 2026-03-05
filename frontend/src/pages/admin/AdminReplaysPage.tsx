import { useState } from 'react'
import { RefreshCw } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { useAdminReplays } from '@/hooks/use-admin'
import { ReplayDetailDialog } from '@/components/admin/ReplayDetailDialog'
import { timeAgo } from '@/lib/utils'

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

export function AdminReplaysPage() {
  const [status, setStatus] = useState('')
  const [customerID, setCustomerID] = useState('')
  const [page, setPage] = useState(1)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const perPage = 20

  const { data, isLoading } = useAdminReplays(status, customerID, page, perPage)

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">รีเพลย์</h1>
          <p className="text-sm text-muted-foreground mt-1">จัดการเซสชันรีเพลย์ทั้งหมด</p>
        </div>
        {data && (
          <Badge variant="secondary" className="text-sm">
            <RefreshCw className="h-3.5 w-3.5 mr-1" />
            {data.total} เซสชัน
          </Badge>
        )}
      </div>

      <div className="flex gap-3 mb-4">
        <select
          value={status}
          onChange={(e) => { setStatus(e.target.value); setPage(1) }}
          className="h-9 rounded-md border border-border bg-background px-3 text-sm text-foreground"
        >
          <option value="">ทุกสถานะ</option>
          <option value="pending">รอดำเนินการ</option>
          <option value="running">กำลังทำงาน</option>
          <option value="completed">สำเร็จ</option>
          <option value="failed">ล้มเหลว</option>
          <option value="cancelled">ยกเลิก</option>
        </select>
        <Input
          placeholder="Customer ID"
          value={customerID}
          onChange={(e) => { setCustomerID(e.target.value); setPage(1) }}
          className="max-w-[200px]"
        />
      </div>

      <Card>
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-muted/50">
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">ลูกค้า</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">Source → Target</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">สถานะ</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">ความคืบหน้า</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">วันที่สร้าง</th>
                </tr>
              </thead>
              <tbody>
                {isLoading && !data ? (
                  <tr>
                    <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">
                      กำลังโหลด...
                    </td>
                  </tr>
                ) : data && data.data.length > 0 ? (
                  data.data.map((r) => (
                    <tr
                      key={r.id}
                      onClick={() => setSelectedId(r.id)}
                      className="border-b border-border hover:bg-muted/50 cursor-pointer transition-colors"
                    >
                      <td className="px-4 py-3 text-foreground">{r.customer_email}</td>
                      <td className="px-4 py-3 text-muted-foreground text-xs">
                        {r.source_pixel_name} → {r.target_pixel_name}
                      </td>
                      <td className="px-4 py-3">
                        <Badge variant={STATUS_BADGE[r.status] ?? 'secondary'} className="text-xs">
                          {STATUS_LABELS[r.status] ?? r.status}
                        </Badge>
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {r.replayed_events}/{r.total_events}
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">{timeAgo(r.created_at)}</td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">
                      ไม่พบเซสชันรีเพลย์
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      {data && data.total_pages > 1 && (
        <div className="flex items-center justify-between mt-4">
          <p className="text-sm text-muted-foreground">
            หน้า {data.page} จาก {data.total_pages}
          </p>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
              ก่อนหน้า
            </Button>
            <Button variant="outline" size="sm" disabled={page >= data.total_pages} onClick={() => setPage((p) => p + 1)}>
              ถัดไป
            </Button>
          </div>
        </div>
      )}

      <ReplayDetailDialog replayId={selectedId} onClose={() => setSelectedId(null)} />
    </div>
  )
}
