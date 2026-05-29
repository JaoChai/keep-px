import { useEffect, useReducer, useState } from 'react'
import { FileText } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { useAdminSalePages } from '@/hooks/use-admin'
import { SalePageDetailDialog } from '@/components/admin/SalePageDetailDialog'
import { timeAgo } from '@/lib/utils'

type FilterState = {
  search: string
  debouncedSearch: string
  published: string
  customerID: string
  page: number
}

type FilterAction =
  | { type: 'SET_SEARCH'; payload: string }
  | { type: 'COMMIT_DEBOUNCED_SEARCH'; payload: string }
  | { type: 'SET_PUBLISHED'; payload: string }
  | { type: 'SET_CUSTOMER_ID'; payload: string }
  | { type: 'SET_PAGE'; payload: number }

const initialFilterState: FilterState = {
  search: '',
  debouncedSearch: '',
  published: '',
  customerID: '',
  page: 1,
}

function filterReducer(state: FilterState, action: FilterAction): FilterState {
  switch (action.type) {
    case 'SET_SEARCH':
      return { ...state, search: action.payload }
    case 'COMMIT_DEBOUNCED_SEARCH':
      return { ...state, debouncedSearch: action.payload, page: 1 }
    case 'SET_PUBLISHED':
      return { ...state, published: action.payload, page: 1 }
    case 'SET_CUSTOMER_ID':
      return { ...state, customerID: action.payload, page: 1 }
    case 'SET_PAGE':
      return { ...state, page: action.payload }
    default:
      return state
  }
}

export function AdminSalePagesPage() {
  const [state, dispatch] = useReducer(filterReducer, initialFilterState)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const perPage = 20

  useEffect(() => {
    const timer = setTimeout(() => {
      dispatch({ type: 'COMMIT_DEBOUNCED_SEARCH', payload: state.search })
    }, 300)
    return () => clearTimeout(timer)
  }, [state.search])

  const { data, isLoading } = useAdminSalePages(state.debouncedSearch, state.customerID, state.published, state.page, perPage)

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">เซลเพจ</h1>
          <p className="text-sm text-muted-foreground mt-1">จัดการเซลเพจทั้งหมดในระบบ</p>
        </div>
        {data && (
          <Badge variant="secondary" className="text-sm">
            <FileText className="size-3.5 mr-1" />
            {data.total} เพจ
          </Badge>
        )}
      </div>

      <div className="flex gap-3 mb-4">
        <Input
          placeholder="ค้นหาชื่อ/slug..."
          value={state.search}
          onChange={(e) => dispatch({ type: 'SET_SEARCH', payload: e.target.value })}
          className="max-w-xs"
        />
        <select
          value={state.published}
          onChange={(e) => dispatch({ type: 'SET_PUBLISHED', payload: e.target.value })}
          className="h-9 rounded-md border border-border bg-background px-3 text-sm text-foreground"
        >
          <option value="">ทั้งหมด</option>
          <option value="true">เผยแพร่</option>
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
            <Button variant="outline" size="sm" disabled={state.page <= 1} onClick={() => dispatch({ type: 'SET_PAGE', payload: state.page - 1 })}>
              ก่อนหน้า
            </Button>
            <Button variant="outline" size="sm" disabled={state.page >= data.total_pages} onClick={() => dispatch({ type: 'SET_PAGE', payload: state.page + 1 })}>
              ถัดไป
            </Button>
          </div>
        </div>
      )}

      <SalePageDetailDialog salePageId={selectedId} onClose={() => setSelectedId(null)} />
    </div>
  )
}
