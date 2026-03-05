import { useState, useEffect } from 'react'
import { Crosshair } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { useAdminPixels } from '@/hooks/use-admin'
import { PixelDetailDialog } from '@/components/admin/PixelDetailDialog'
import { timeAgo } from '@/lib/utils'

export function AdminPixelsPage() {
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [active, setActive] = useState('')
  const [customerID, setCustomerID] = useState('')
  const [page, setPage] = useState(1)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const perPage = 20

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search)
      setPage(1)
    }, 300)
    return () => clearTimeout(timer)
  }, [search])

  const { data, isLoading } = useAdminPixels(debouncedSearch, customerID, active, page, perPage)

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">พิกเซล</h1>
          <p className="text-sm text-muted-foreground mt-1">จัดการพิกเซลทั้งหมดในระบบ</p>
        </div>
        {data && (
          <Badge variant="secondary" className="text-sm">
            <Crosshair className="h-3.5 w-3.5 mr-1" />
            {data.total} พิกเซล
          </Badge>
        )}
      </div>

      <div className="flex gap-3 mb-4">
        <Input
          placeholder="ค้นหาชื่อ/Pixel ID..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="max-w-xs"
        />
        <select
          value={active}
          onChange={(e) => { setActive(e.target.value); setPage(1) }}
          className="h-9 rounded-md border border-border bg-background px-3 text-sm text-foreground"
        >
          <option value="">ทั้งหมด</option>
          <option value="true">ใช้งาน</option>
          <option value="false">ปิด</option>
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
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">ชื่อ</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">FB Pixel ID</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">ลูกค้า</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">สถานะ</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">อีเวนต์</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">เซลเพจ</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">วันที่สร้าง</th>
                </tr>
              </thead>
              <tbody>
                {isLoading && !data ? (
                  <tr>
                    <td colSpan={7} className="px-4 py-8 text-center text-muted-foreground">
                      กำลังโหลด...
                    </td>
                  </tr>
                ) : data && data.data.length > 0 ? (
                  data.data.map((px) => (
                    <tr
                      key={px.id}
                      onClick={() => setSelectedId(px.id)}
                      className="border-b border-border hover:bg-muted/50 cursor-pointer transition-colors"
                    >
                      <td className="px-4 py-3 text-foreground">{px.name}</td>
                      <td className="px-4 py-3 text-muted-foreground font-mono text-xs">{px.fb_pixel_id}</td>
                      <td className="px-4 py-3 text-foreground">{px.customer_email}</td>
                      <td className="px-4 py-3">
                        {px.is_active ? (
                          <Badge variant="success" className="text-xs">ใช้งาน</Badge>
                        ) : (
                          <Badge variant="secondary" className="text-xs">ปิด</Badge>
                        )}
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">{px.event_count.toLocaleString()}</td>
                      <td className="px-4 py-3 text-muted-foreground">{px.sale_page_count}</td>
                      <td className="px-4 py-3 text-muted-foreground">{timeAgo(px.created_at)}</td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={7} className="px-4 py-8 text-center text-muted-foreground">
                      ไม่พบพิกเซล
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

      <PixelDetailDialog pixelId={selectedId} onClose={() => setSelectedId(null)} />
    </div>
  )
}
