import { useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Collapsible } from '@/components/ui/collapsible'
import {
  useAdminCustomerDetail,
  useAdminUpdateCustomerPlan,
  useAdminSuspendCustomer,
  useAdminActivateCustomer,
  useAdminGrantCredits,
} from '@/hooks/use-admin'
import { PLAN_LABELS, formatBaht, timeAgo, daysUntil, isUnlimited } from '@/lib/utils'

interface CustomerDetailDialogProps {
  customerId: string | null
  onClose: () => void
}

const PLANS = ['sandbox', 'launch', 'shield', 'vault'] as const

export function CustomerDetailDialog({ customerId, onClose }: CustomerDetailDialogProps) {
  const { data: detail, isLoading } = useAdminCustomerDetail(customerId)
  const updatePlan = useAdminUpdateCustomerPlan()
  const suspend = useAdminSuspendCustomer()
  const activate = useAdminActivateCustomer()
  const grantCredits = useAdminGrantCredits()

  const [newPlan, setNewPlan] = useState('')
  const [creditForm, setCreditForm] = useState({
    total_replays: 1,
    max_events_per_replay: 5000,
    expires_in_days: 30,
    reason: '',
  })

  const customer = detail?.customer

  const handlePlanChange = () => {
    if (!customer || !newPlan || newPlan === customer.plan) return
    updatePlan.mutate({ customerId: customer.id, plan: newPlan })
  }

  const handleSuspendToggle = () => {
    if (!customer) return
    if (customer.suspended_at) {
      activate.mutate(customer.id)
    } else {
      suspend.mutate(customer.id)
    }
  }

  const handleGrantCredits = () => {
    if (!customer) return
    grantCredits.mutate({
      customerId: customer.id,
      pack_type: 'admin_grant',
      ...creditForm,
    })
  }

  return (
    <Dialog open={!!customerId} onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto" onClose={onClose}>
        <DialogHeader>
          <DialogTitle>รายละเอียดลูกค้า</DialogTitle>
        </DialogHeader>

        {isLoading || !customer ? (
          <div className="py-8 text-center text-muted-foreground">กำลังโหลด...</div>
        ) : (
          <div className="space-y-6 mt-4">
            {/* Account Info */}
            <div>
              <h3 className="text-sm font-semibold text-foreground mb-3">ข้อมูลบัญชี</h3>
              <div className="grid grid-cols-2 gap-3 text-sm">
                <div>
                  <span className="text-muted-foreground">อีเมล: </span>
                  <span className="text-foreground">{customer.email}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">ชื่อ: </span>
                  <span className="text-foreground">{customer.name}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">แผน: </span>
                  <Badge variant="secondary" className="text-xs">{PLAN_LABELS[customer.plan] ?? customer.plan}</Badge>
                </div>
                <div>
                  <span className="text-muted-foreground">สถานะ: </span>
                  {customer.suspended_at ? (
                    <Badge variant="destructive" className="text-xs">ระงับ</Badge>
                  ) : (
                    <Badge variant="success" className="text-xs">ใช้งาน</Badge>
                  )}
                </div>
                <div>
                  <span className="text-muted-foreground">API Key: </span>
                  <code className="text-xs bg-muted px-1 py-0.5 rounded">{customer.api_key ? `${customer.api_key.slice(0, 8)}...` : '-'}</code>
                </div>
                <div>
                  <span className="text-muted-foreground">Stripe: </span>
                  <span className="text-foreground text-xs">{customer.stripe_customer_id || '-'}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">สมัครเมื่อ: </span>
                  <span className="text-foreground">{timeAgo(customer.created_at)}</span>
                </div>
              </div>
            </div>

            {/* Usage Stats */}
            <div>
              <h3 className="text-sm font-semibold text-foreground mb-3">สถิติการใช้งาน</h3>
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                {[
                  { label: 'พิกเซล', value: detail.pixel_count },
                  { label: 'อีเวนต์', value: detail.event_count.toLocaleString() },
                  { label: 'หน้าขาย', value: detail.sale_page_count },
                  { label: 'รีเพลย์', value: detail.replay_count },
                ].map((s) => (
                  <div key={s.label} className="bg-muted rounded-lg p-3 text-center">
                    <p className="text-lg font-bold text-foreground">{s.value}</p>
                    <p className="text-xs text-muted-foreground">{s.label}</p>
                  </div>
                ))}
              </div>
            </div>

            {/* Actions */}
            <div className="space-y-4">
              <h3 className="text-sm font-semibold text-foreground">จัดการ</h3>

              {/* Change Plan */}
              <div className="flex items-end gap-3">
                <div className="flex-1">
                  <Label className="text-xs text-muted-foreground mb-1 block">เปลี่ยนแผน</Label>
                  <select
                    value={newPlan || customer.plan}
                    onChange={(e) => setNewPlan(e.target.value)}
                    className="h-9 w-full rounded-md border border-border bg-background px-3 text-sm text-foreground"
                  >
                    {PLANS.map((p) => (
                      <option key={p} value={p}>{PLAN_LABELS[p] ?? p}</option>
                    ))}
                  </select>
                </div>
                <Button
                  size="sm"
                  onClick={handlePlanChange}
                  disabled={updatePlan.isPending || !newPlan || newPlan === customer.plan}
                >
                  {updatePlan.isPending ? 'กำลังบันทึก...' : 'บันทึก'}
                </Button>
              </div>

              {/* Suspend / Activate */}
              <div>
                <Button
                  variant={customer.suspended_at ? 'default' : 'destructive'}
                  size="sm"
                  onClick={handleSuspendToggle}
                  disabled={suspend.isPending || activate.isPending}
                >
                  {customer.suspended_at ? 'เปิดใช้งานบัญชี' : 'ระงับบัญชี'}
                </Button>
              </div>

              {/* Grant Credits */}
              <div className="border border-border rounded-lg p-4 space-y-3">
                <h4 className="text-sm font-medium text-foreground">ให้เครดิตรีเพลย์</h4>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <Label className="text-xs text-muted-foreground mb-1 block">จำนวนรีเพลย์</Label>
                    <Input
                      type="number"
                      min={1}
                      value={creditForm.total_replays}
                      onChange={(e) => setCreditForm((f) => ({ ...f, total_replays: Number(e.target.value) }))}
                    />
                  </div>
                  <div>
                    <Label className="text-xs text-muted-foreground mb-1 block">อีเวนต์/รีเพลย์</Label>
                    <Input
                      type="number"
                      min={100}
                      value={creditForm.max_events_per_replay}
                      onChange={(e) => setCreditForm((f) => ({ ...f, max_events_per_replay: Number(e.target.value) }))}
                    />
                  </div>
                  <div>
                    <Label className="text-xs text-muted-foreground mb-1 block">หมดอายุ (วัน)</Label>
                    <Input
                      type="number"
                      min={1}
                      value={creditForm.expires_in_days}
                      onChange={(e) => setCreditForm((f) => ({ ...f, expires_in_days: Number(e.target.value) }))}
                    />
                  </div>
                  <div>
                    <Label className="text-xs text-muted-foreground mb-1 block">เหตุผล</Label>
                    <Input
                      value={creditForm.reason}
                      onChange={(e) => setCreditForm((f) => ({ ...f, reason: e.target.value }))}
                      placeholder="ไม่บังคับ"
                    />
                  </div>
                </div>
                <Button size="sm" onClick={handleGrantCredits} disabled={grantCredits.isPending}>
                  {grantCredits.isPending ? 'กำลังเพิ่ม...' : 'เพิ่มเครดิต'}
                </Button>
              </div>
            </div>

            {/* Collapsible Lists */}
            {detail.credits.length > 0 && (
              <Collapsible title={`เครดิตที่ใช้งาน (${detail.credits.length})`}>
                <div className="space-y-2">
                  {detail.credits.map((c) => (
                    <div key={c.id} className="flex items-center justify-between text-sm border-b border-border pb-2 last:border-0">
                      <div>
                        <span className="text-foreground">{c.pack_type}</span>
                        <span className="text-muted-foreground ml-2">
                          {c.used_replays}/{isUnlimited(c.total_replays) ? 'Unlimited' : c.total_replays} ใช้แล้ว
                        </span>
                      </div>
                      <span className="text-xs text-muted-foreground">
                        หมดอายุ {daysUntil(c.expires_at) > 0 ? `อีก ${daysUntil(c.expires_at)} วัน` : 'หมดอายุแล้ว'}
                      </span>
                    </div>
                  ))}
                </div>
              </Collapsible>
            )}

            {detail.subscriptions.length > 0 && (
              <Collapsible title={`สมาชิก (${detail.subscriptions.length})`}>
                <div className="space-y-2">
                  {detail.subscriptions.map((s) => (
                    <div key={s.id} className="flex items-center justify-between text-sm border-b border-border pb-2 last:border-0">
                      <span className="text-foreground">{s.addon_type}</span>
                      <Badge variant={s.status === 'active' ? 'success' : 'secondary'} className="text-xs">{s.status}</Badge>
                    </div>
                  ))}
                </div>
              </Collapsible>
            )}

            {detail.purchases.length > 0 && (
              <Collapsible title={`การซื้อล่าสุด (${detail.purchases.length})`}>
                <div className="space-y-2">
                  {detail.purchases.map((p) => (
                    <div key={p.id} className="flex items-center justify-between text-sm border-b border-border pb-2 last:border-0">
                      <div>
                        <span className="text-foreground">{p.pack_type}</span>
                        <span className="text-muted-foreground ml-2">{formatBaht(p.amount_satang)} THB</span>
                      </div>
                      <Badge variant={p.status === 'completed' ? 'success' : 'warning'} className="text-xs">{p.status}</Badge>
                    </div>
                  ))}
                </div>
              </Collapsible>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
