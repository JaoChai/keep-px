import { useEffect, useReducer, useState } from 'react'
import { Crosshair } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { useAdminPixels } from '@/hooks/use-admin'
import { PixelDetailDialog } from '@/components/admin/PixelDetailDialog'
import { timeAgo } from '@/lib/utils'

type FilterState = {
  search: string
  debouncedSearch: string
  active: string
  customerID: string
  page: number
}

type FilterAction =
  | { type: 'SET_SEARCH'; payload: string }
  | { type: 'COMMIT_DEBOUNCED_SEARCH'; payload: string }
  | { type: 'SET_ACTIVE'; payload: string }
  | { type: 'SET_CUSTOMER_ID'; payload: string }
  | { type: 'SET_PAGE'; payload: number }

const initialFilterState: FilterState = {
  search: '',
  debouncedSearch: '',
  active: '',
  customerID: '',
  page: 1,
}

function filterReducer(state: FilterState, action: FilterAction): FilterState {
  switch (action.type) {
    case 'SET_SEARCH':
      return { ...state, search: action.payload }
    case 'COMMIT_DEBOUNCED_SEARCH':
      return { ...state, debouncedSearch: action.payload, page: 1 }
    case 'SET_ACTIVE':
      return { ...state, active: action.payload, page: 1 }
    case 'SET_CUSTOMER_ID':
      return { ...state, customerID: action.payload, page: 1 }
    case 'SET_PAGE':
      return { ...state, page: action.payload }
    default:
      return state
  }
}

export function AdminPixelsPage() {
  const [state, dispatch] = useReducer(filterReducer, initialFilterState)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const perPage = 20

  useEffect(() => {
    const timer = setTimeout(() => {
      dispatch({ type: 'COMMIT_DEBOUNCED_SEARCH', payload: state.search })
    }, 300)
    return () => clearTimeout(timer)
  }, [state.search])

  const { data, isLoading } = useAdminPixels(state.debouncedSearch, state.customerID, state.active, state.page, perPage)

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">พิกเซล</h1>
          <p className="text-sm text-muted-foreground mt-1">จัดการพิกเซลทั้งหมดในระบบ</p>
        </div>
        {data && (
          <Badge variant="secondary" className="text-sm">
            <Crosshair className="size-3.5 mr-1" />
            {data.total} พิกเซล
          </Badge>
        )}
      </div>

      <div className="flex gap-3 mb-4">
        <Input
          placeholder="ค้นหาชื่อ/Pixel ID..."
          value={state.search}
          onChange={(e) => dispatch({ type: 'SET_SEARCH', payload: e.target.value })}
          className="max-w-xs"
        />
        <select
          value={state.active}
          onChange={(e) => dispatch({ type: 'SET_ACTIVE', payload: e.target.value })}
          className="h-9 rounded-md border border-border bg-background px-3 text-sm text-foreground"
        >
          <option value="">ทั้งหมด</option>
          <option value="true">ใช้งาน</option>
          <option value="false">ปิด</option>
        </select>
        <Input
          placeholder="Customer ID"
          value={state.customerID}
          onChange={(e) => dispatch({ type: 'SET_CUSTOMER_ID', payload: e.target.value })}
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
            <Button variant="outline" size="sm" disabled={state.page <= 1} onClick={() => dispatch({ type: 'SET_PAGE', payload: state.page - 1 })}>
              ก่อนหน้า
            </Button>
            <Button variant="outline" size="sm" disabled={state.page >= data.total_pages} onClick={() => dispatch({ type: 'SET_PAGE', payload: state.page + 1 })}>
              ถัดไป
            </Button>
          </div>
        </div>
      )}

      <PixelDetailDialog pixelId={selectedId} onClose={() => setSelectedId(null)} />
    </div>
  )
}
