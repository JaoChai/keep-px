import { useState, useEffect } from 'react'
import { FileText } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { useAdminSalePages } from '@/hooks/use-admin'
import { SalePageDetailDialog } from '@/components/admin/SalePageDetailDialog'
import { timeAgo } from '@/lib/utils'

export function AdminSalePagesPage() {
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [published, setPublished] = useState('')
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

  const { data, isLoading } = useAdminSalePages(debouncedSearch, customerID, published, page, perPage)

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">เซลเพจ</h1>
          <p className="text-sm text-muted-foreground mt-1">จัดการเซลเพจทั้งหมดในระบบ</p>
        </div>
        {data && (
          <Badge variant="secondary" className="text-sm">
            <FileText className="h-3.5 w-3.5 mr-1" />
            {data.total} เพจ
          </Badge>
        )}
      </div>

      <div className="flex gap-3 mb-4">
        <Input
          placeholder="ค้นหาชื่อ/slug..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="max-w-xs"
        />
        <select
          value={published}
          onChange={(e) => { setPublished(e.target.value); setPage(1) }}
          className="h-9 rounded-md border border-border bg-background px-3 text-sm text-foreground"
        >
          <option value="">ทั้งหมด</option>
          <option value="true">เผยแพร่</option>
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
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">Slug</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">ลูกค้า</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">สถานะ</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">อีเวนต์</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">วันที่สร้าง</th>
                </tr>
              </thead>
              <tbody>
                {isLoading && !data ? (
                  <tr>
                    <td colSpan={6} className="px-4 py-8 text-center text-muted-foreground">
                      กำลังโหลด...
                    </td>
                  </tr>
                ) : data && data.data.length > 0 ? (
                  data.data.map((sp) => (
                    <tr
                      key={sp.id}
                      onClick={() => setSelectedId(sp.id)}
                      className="border-b border-border hover:bg-muted/50 cursor-pointer transition-colors"
                    >
                      <td className="px-4 py-3 text-foreground">{sp.name}</td>
                      <td className="px-4 py-3 text-muted-foreground font-mono text-xs">{sp.slug}</td>
                      <td className="px-4 py-3 text-foreground">{sp.customer_email}</td>
                      <td className="px-4 py-3">
                        {sp.is_published ? (
                          <Badge variant="success" className="text-xs">เผยแพร่</Badge>
                        ) : (
                          <Badge variant="secondary" className="text-xs">ปิด</Badge>
                        )}
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">{sp.event_count.toLocaleString()}</td>
                      <td className="px-4 py-3 text-muted-foreground">{timeAgo(sp.created_at)}</td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={6} className="px-4 py-8 text-center text-muted-foreground">
                      ไม่พบเซลเพจ
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

      <SalePageDetailDialog salePageId={selectedId} onClose={() => setSelectedId(null)} />
    </div>
  )
}
