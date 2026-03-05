import { useState } from 'react'
import { ScrollText } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { useAdminAuditLog } from '@/hooks/use-admin'
import { timeAgo } from '@/lib/utils'

const ACTION_LABELS: Record<string, string> = {
  suspend_customer: 'ระงับบัญชี',
  activate_customer: 'เปิดใช้งาน',
  change_plan: 'เปลี่ยนแผน',
  grant_credits: 'เพิ่มเครดิต',
  disable_sale_page: 'ปิดเซลเพจ',
  enable_sale_page: 'เปิดเซลเพจ',
  delete_sale_page: 'ลบเซลเพจ',
  disable_pixel: 'ปิดพิกเซล',
  enable_pixel: 'เปิดพิกเซล',
  cancel_replay: 'ยกเลิกรีเพลย์',
}

const ACTION_BADGE: Record<string, 'destructive' | 'success' | 'warning' | 'default' | 'secondary'> = {
  suspend_customer: 'destructive',
  activate_customer: 'success',
  change_plan: 'warning',
  grant_credits: 'success',
  disable_sale_page: 'destructive',
  enable_sale_page: 'success',
  delete_sale_page: 'destructive',
  disable_pixel: 'destructive',
  enable_pixel: 'success',
  cancel_replay: 'warning',
}

const ACTIONS = [
  'suspend_customer',
  'activate_customer',
  'change_plan',
  'grant_credits',
  'disable_sale_page',
  'enable_sale_page',
  'delete_sale_page',
  'disable_pixel',
  'enable_pixel',
  'cancel_replay',
]

export function AdminAuditLogPage() {
  const [action, setAction] = useState('')
  const [adminID, setAdminID] = useState('')
  const [targetCustomerID, setTargetCustomerID] = useState('')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [page, setPage] = useState(1)
  const perPage = 20

  const { data, isLoading } = useAdminAuditLog(adminID, action, targetCustomerID, from, to, page, perPage)

  const formatDetails = (details: unknown): string => {
    if (!details) return '-'
    if (typeof details === 'string') return details
    try {
      return JSON.stringify(details)
    } catch {
      return '-'
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground">บันทึกกิจกรรม</h1>
          <p className="text-sm text-muted-foreground mt-1">ติดตามการกระทำของแอดมินทั้งหมด</p>
        </div>
        {data && (
          <Badge variant="secondary" className="text-sm">
            <ScrollText className="h-3.5 w-3.5 mr-1" />
            {data.total} รายการ
          </Badge>
        )}
      </div>

      <div className="flex flex-wrap gap-3 mb-4">
        <select
          value={action}
          onChange={(e) => { setAction(e.target.value); setPage(1) }}
          className="h-9 rounded-md border border-border bg-background px-3 text-sm text-foreground"
        >
          <option value="">ทุกการกระทำ</option>
          {ACTIONS.map((a) => (
            <option key={a} value={a}>{ACTION_LABELS[a] ?? a}</option>
          ))}
        </select>
        <Input
          placeholder="Admin ID"
          value={adminID}
          onChange={(e) => { setAdminID(e.target.value); setPage(1) }}
          className="max-w-[180px]"
        />
        <Input
          placeholder="Customer ID เป้าหมาย"
          value={targetCustomerID}
          onChange={(e) => { setTargetCustomerID(e.target.value); setPage(1) }}
          className="max-w-[200px]"
        />
        <Input
          type="datetime-local"
          value={from}
          onChange={(e) => { setFrom(e.target.value); setPage(1) }}
          className="max-w-[200px]"
        />
        <Input
          type="datetime-local"
          value={to}
          onChange={(e) => { setTo(e.target.value); setPage(1) }}
          className="max-w-[200px]"
        />
      </div>

      <Card>
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-muted/50">
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">เวลา</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">แอดมิน</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">การกระทำ</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">ประเภท</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">ลูกค้าเป้าหมาย</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">รายละเอียด</th>
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
                  data.data.map((entry) => (
                    <tr key={entry.id} className="border-b border-border hover:bg-muted/50 transition-colors">
                      <td className="px-4 py-3 text-muted-foreground">{timeAgo(entry.created_at)}</td>
                      <td className="px-4 py-3 text-foreground">{entry.admin_email}</td>
                      <td className="px-4 py-3">
                        <Badge variant={ACTION_BADGE[entry.action] ?? 'secondary'} className="text-xs">
                          {ACTION_LABELS[entry.action] ?? entry.action}
                        </Badge>
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">{entry.target_type}</td>
                      <td className="px-4 py-3 text-foreground">{entry.customer_email ?? '-'}</td>
                      <td className="px-4 py-3 text-muted-foreground text-xs max-w-[200px] truncate">
                        {formatDetails(entry.details)}
                      </td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={6} className="px-4 py-8 text-center text-muted-foreground">
                      ไม่พบบันทึกกิจกรรม
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
    </div>
  )
}
