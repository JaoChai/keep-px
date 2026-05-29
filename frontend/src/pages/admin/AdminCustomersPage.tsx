import { useEffect, useReducer, useState } from 'react'
import { Users } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { useAdminCustomers } from '@/hooks/use-admin'
import { CustomerDetailDialog } from '@/components/admin/CustomerDetailDialog'
import { PLAN_LABELS, timeAgo } from '@/lib/utils'

const PLANS = ['', 'sandbox', 'launch', 'shield', 'vault'] as const

type FilterState = {
  search: string
  debouncedSearch: string
  plan: string
  status: string
  page: number
}

type FilterAction =
  | { type: 'SET_SEARCH'; payload: string }
  | { type: 'COMMIT_DEBOUNCED_SEARCH'; payload: string }
  | { type: 'SET_PLAN'; payload: string }
  | { type: 'SET_STATUS'; payload: string }
  | { type: 'SET_PAGE'; payload: number }

const initialFilterState: FilterState = {
  search: '',
  debouncedSearch: '',
  plan: '',
  status: '',
  page: 1,
}

function filterReducer(state: FilterState, action: FilterAction): FilterState {
  switch (action.type) {
    case 'SET_SEARCH':
      return { ...state, search: action.payload }
    case 'COMMIT_DEBOUNCED_SEARCH':
      return { ...state, debouncedSearch: action.payload, page: 1 }
    case 'SET_PLAN':
      return { ...state, plan: action.payload, page: 1 }
    case 'SET_STATUS':
      return { ...state, status: action.payload, page: 1 }
    case 'SET_PAGE':
      return { ...state, page: action.payload }
    default:
      return state
  }
}

export function AdminCustomersPage() {
  const [state, dispatch] = useReducer(filterReducer, initialFilterState)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const perPage = 20

  // Debounce search
  useEffect(() => {
    const timer = setTimeout(() => {
      dispatch({ type: 'COMMIT_DEBOUNCED_SEARCH', payload: state.search })
    }, 300)
    return () => clearTimeout(timer)
  }, [state.search])

  const { data, isLoading } = useAdminCustomers(state.debouncedSearch, state.plan, state.status, state.page, perPage)

  const planBadgeVariant = (p: string): 'default' | 'secondary' | 'success' | 'warning' => {
    switch (p) {
      case 'vault': return 'default'
      case 'shield': return 'success'
      case 'launch': return 'warning'
      default: return 'secondary'
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">ลูกค้า</h1>
          <p className="text-sm text-muted-foreground mt-1">จัดการบัญชีลูกค้าทั้งหมด</p>
        </div>
        {data && (
          <Badge variant="secondary" className="text-sm">
            <Users className="size-3.5 mr-1" />
            {data.total} คน
          </Badge>
        )}
      </div>

      {/* Filters */}
      <div className="flex gap-3 mb-4">
        <Input
          placeholder="ค้นหาอีเมลหรือชื่อ..."
          value={state.search}
          onChange={(e) => dispatch({ type: 'SET_SEARCH', payload: e.target.value })}
          className="max-w-xs"
        />
        <select
          value={state.plan}
          onChange={(e) => dispatch({ type: 'SET_PLAN', payload: e.target.value })}
          className="h-9 rounded-md border border-border bg-background px-3 text-sm text-foreground"
        >
          <option value="">ทุกแผน</option>
          {PLANS.filter(Boolean).map((p) => (
            <option key={p} value={p}>{PLAN_LABELS[p] ?? p}</option>
          ))}
        </select>
        <select
          value={state.status}
          onChange={(e) => dispatch({ type: 'SET_STATUS', payload: e.target.value })}
          className="h-9 rounded-md border border-border bg-background px-3 text-sm text-foreground"
        >
          <option value="">ทุกสถานะ</option>
          <option value="active">ใช้งาน</option>
          <option value="suspended">ระงับ</option>
        </select>
      </div>

      {/* Table */}
      <Card>
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-muted/50">
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">อีเมล</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">ชื่อ</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">แผน</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">สถานะ</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">สมัครเมื่อ</th>
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
                  data.data.map((c) => (
                    <tr
                      key={c.id}
                      onClick={() => setSelectedId(c.id)}
                      className="border-b border-border hover:bg-muted/50 cursor-pointer transition-colors"
                    >
                      <td className="px-4 py-3 text-foreground">{c.email}</td>
                      <td className="px-4 py-3 text-foreground">{c.name}</td>
                      <td className="px-4 py-3">
                        <Badge variant={planBadgeVariant(c.plan)} className="text-xs">
                          {PLAN_LABELS[c.plan] ?? c.plan}
                        </Badge>
                      </td>
                      <td className="px-4 py-3">
                        {c.suspended_at ? (
                          <Badge variant="destructive" className="text-xs">ระงับ</Badge>
                        ) : (
                          <Badge variant="success" className="text-xs">ใช้งาน</Badge>
                        )}
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">{timeAgo(c.created_at)}</td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">
                      ไม่พบลูกค้า
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      {/* Pagination */}
      {data && data.total_pages > 1 && (
        <div className="flex items-center justify-between mt-4">
          <p className="text-sm text-muted-foreground">
            หน้า {data.page} จาก {data.total_pages}
          </p>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={state.page <= 1}
              onClick={() => dispatch({ type: 'SET_PAGE', payload: state.page - 1 })}
            >
              ก่อนหน้า
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={state.page >= data.total_pages}
              onClick={() => dispatch({ type: 'SET_PAGE', payload: state.page + 1 })}
            >
              ถัดไป
            </Button>
          </div>
        </div>
      )}

      {/* Detail Dialog */}
      <CustomerDetailDialog
        customerId={selectedId}
        onClose={() => setSelectedId(null)}
      />
    </div>
  )
}
