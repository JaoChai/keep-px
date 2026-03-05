import { useState } from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { cn, formatBaht, timeAgo, PACK_TYPE_NAMES } from '@/lib/utils'
import {
  useAdminPurchases,
  useAdminSubscriptions,
  useAdminCreditGrants,
} from '@/hooks/use-admin'

type BillingTab = 'purchases' | 'subscriptions' | 'credits'

const TABS: { key: BillingTab; label: string }[] = [
  { key: 'purchases', label: 'การซื้อ' },
  { key: 'subscriptions', label: 'สมาชิก' },
  { key: 'credits', label: 'เครดิตที่ให้' },
]

const statusBadgeVariant = (status: string): 'default' | 'success' | 'warning' | 'destructive' | 'secondary' => {
  switch (status) {
    case 'completed': case 'active': return 'success'
    case 'pending': return 'warning'
    case 'failed': case 'canceled': case 'cancelled': return 'destructive'
    default: return 'secondary'
  }
}

function Pagination({ page, totalPages, onPageChange }: { page: number; totalPages: number; onPageChange: (p: number) => void }) {
  if (totalPages <= 1) return null
  return (
    <div className="flex items-center justify-between mt-4">
      <p className="text-sm text-muted-foreground">หน้า {page} จาก {totalPages}</p>
      <div className="flex gap-2">
        <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => onPageChange(page - 1)}>ก่อนหน้า</Button>
        <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => onPageChange(page + 1)}>ถัดไป</Button>
      </div>
    </div>
  )
}

function PurchasesTab() {
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const { data, isLoading } = useAdminPurchases(status, page, 20)

  return (
    <div>
      <div className="mb-4">
        <select
          value={status}
          onChange={(e) => { setStatus(e.target.value); setPage(1) }}
          className="h-9 rounded-md border border-border bg-background px-3 text-sm text-foreground"
        >
          <option value="">ทุกสถานะ</option>
          <option value="completed">สำเร็จ</option>
          <option value="pending">รอดำเนินการ</option>
          <option value="failed">ล้มเหลว</option>
        </select>
      </div>
      <Card>
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-muted/50">
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">ลูกค้า</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">แพ็ก</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">จำนวน</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">สถานะ</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">วันที่</th>
                </tr>
              </thead>
              <tbody>
                {isLoading && !data ? (
                  <tr><td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">กำลังโหลด...</td></tr>
                ) : data && data.data.length > 0 ? (
                  data.data.map((p) => (
                    <tr key={p.id} className="border-b border-border">
                      <td className="px-4 py-3">
                        <div className="text-foreground">{p.customer_name}</div>
                        <div className="text-xs text-muted-foreground">{p.customer_email}</div>
                      </td>
                      <td className="px-4 py-3 text-foreground">{PACK_TYPE_NAMES[p.pack_type] ?? p.pack_type}</td>
                      <td className="px-4 py-3 text-foreground">{formatBaht(p.amount_satang)} THB</td>
                      <td className="px-4 py-3">
                        <Badge variant={statusBadgeVariant(p.status)} className="text-xs">{p.status}</Badge>
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">{timeAgo(p.created_at)}</td>
                    </tr>
                  ))
                ) : (
                  <tr><td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">ไม่พบข้อมูล</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
      {data && <Pagination page={data.page} totalPages={data.total_pages} onPageChange={setPage} />}
    </div>
  )
}

function SubscriptionsTab() {
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const { data, isLoading } = useAdminSubscriptions(status, page, 20)

  return (
    <div>
      <div className="mb-4">
        <select
          value={status}
          onChange={(e) => { setStatus(e.target.value); setPage(1) }}
          className="h-9 rounded-md border border-border bg-background px-3 text-sm text-foreground"
        >
          <option value="">ทุกสถานะ</option>
          <option value="active">ใช้งาน</option>
          <option value="canceled">ยกเลิก</option>
        </select>
      </div>
      <Card>
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-muted/50">
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">ลูกค้า</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">ส่วนเสริม</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">สถานะ</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">สิ้นสุด</th>
                </tr>
              </thead>
              <tbody>
                {isLoading && !data ? (
                  <tr><td colSpan={4} className="px-4 py-8 text-center text-muted-foreground">กำลังโหลด...</td></tr>
                ) : data && data.data.length > 0 ? (
                  data.data.map((s) => (
                    <tr key={s.id} className="border-b border-border">
                      <td className="px-4 py-3">
                        <div className="text-foreground">{s.customer_name}</div>
                        <div className="text-xs text-muted-foreground">{s.customer_email}</div>
                      </td>
                      <td className="px-4 py-3 text-foreground">{PACK_TYPE_NAMES[s.addon_type] ?? s.addon_type}</td>
                      <td className="px-4 py-3">
                        <Badge variant={statusBadgeVariant(s.status)} className="text-xs">
                          {s.status}
                          {s.cancel_at_period_end && ' (จะยกเลิก)'}
                        </Badge>
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {s.current_period_end ? new Date(s.current_period_end).toLocaleDateString('th-TH') : '-'}
                      </td>
                    </tr>
                  ))
                ) : (
                  <tr><td colSpan={4} className="px-4 py-8 text-center text-muted-foreground">ไม่พบข้อมูล</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
      {data && <Pagination page={data.page} totalPages={data.total_pages} onPageChange={setPage} />}
    </div>
  )
}

function CreditGrantsTab() {
  const [page, setPage] = useState(1)
  const { data, isLoading } = useAdminCreditGrants(page, 20)

  return (
    <div>
      <Card>
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-muted/50">
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">ลูกค้า</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">แพ็ก</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">รีเพลย์</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">เหตุผล</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">หมดอายุ</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">วันที่ให้</th>
                </tr>
              </thead>
              <tbody>
                {isLoading && !data ? (
                  <tr><td colSpan={6} className="px-4 py-8 text-center text-muted-foreground">กำลังโหลด...</td></tr>
                ) : data && data.data.length > 0 ? (
                  data.data.map((g) => (
                    <tr key={g.id} className="border-b border-border">
                      <td className="px-4 py-3">
                        <div className="text-foreground">{g.customer_name}</div>
                        <div className="text-xs text-muted-foreground">{g.customer_email}</div>
                      </td>
                      <td className="px-4 py-3 text-foreground">{PACK_TYPE_NAMES[g.pack_type] ?? g.pack_type}</td>
                      <td className="px-4 py-3 text-foreground">{g.total_replays}</td>
                      <td className="px-4 py-3 text-muted-foreground">{g.reason || '-'}</td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {new Date(g.expires_at).toLocaleDateString('th-TH')}
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">{timeAgo(g.created_at)}</td>
                    </tr>
                  ))
                ) : (
                  <tr><td colSpan={6} className="px-4 py-8 text-center text-muted-foreground">ไม่พบข้อมูล</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
      {data && <Pagination page={data.page} totalPages={data.total_pages} onPageChange={setPage} />}
    </div>
  )
}

export function AdminBillingPage() {
  const [activeTab, setActiveTab] = useState<BillingTab>('purchases')

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-foreground">การเงินทั้งหมด</h1>
        <p className="text-sm text-muted-foreground mt-1">ดูรายการซื้อ สมาชิก และเครดิตทั้งระบบ</p>
      </div>

      {/* Tab Bar */}
      <div className="flex gap-1 bg-muted p-1 rounded-lg w-fit mb-6">
        {TABS.map((tab) => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.key)}
            className={cn(
              'px-4 py-1.5 text-sm font-medium rounded-md transition-colors',
              activeTab === tab.key
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground',
            )}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {activeTab === 'purchases' && <PurchasesTab />}
      {activeTab === 'subscriptions' && <SubscriptionsTab />}
      {activeTab === 'credits' && <CreditGrantsTab />}
    </div>
  )
}
