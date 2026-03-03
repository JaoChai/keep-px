import { Collapsible } from '@/components/ui/collapsible'
import { Badge } from '@/components/ui/badge'
import { formatBaht } from '@/lib/utils'
import type { Purchase } from '@/types'

interface PurchaseHistorySectionProps {
  purchases: Purchase[]
  isLoading: boolean
}

function StatusBadge({ status }: { status: string }) {
  const config = {
    completed: { variant: 'success' as const, label: 'สำเร็จ' },
    pending: { variant: 'secondary' as const, label: 'รอดำเนินการ' },
  }
  const { variant, label } = config[status as keyof typeof config] ?? { variant: 'destructive' as const, label: 'ล้มเหลว' }
  return <Badge variant={variant}>{label}</Badge>
}

export function PurchaseHistorySection({ purchases, isLoading }: PurchaseHistorySectionProps) {
  if (isLoading) {
    return (
      <section>
        <Collapsible title="ประวัติการซื้อ (กำลังโหลด...)">
          <p className="text-sm text-muted-foreground text-center py-4">กำลังโหลด...</p>
        </Collapsible>
      </section>
    )
  }

  const count = purchases.length
  const title = count > 0
    ? `ประวัติการซื้อ (${count} รายการ)`
    : 'ประวัติการซื้อ'

  return (
    <section>
      <Collapsible title={title}>
        {count === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-4">
            ยังไม่มีประวัติการซื้อ
          </p>
        ) : (
          <>
            {/* Desktop table */}
            <div className="hidden sm:block overflow-x-auto">
              <table className="w-full text-sm" aria-label="ประวัติการซื้อ">
                <thead className="bg-muted">
                  <tr>
                    <th className="text-left px-4 py-3 font-medium text-muted-foreground">วันที่</th>
                    <th className="text-left px-4 py-3 font-medium text-muted-foreground">แพ็ก</th>
                    <th className="text-left px-4 py-3 font-medium text-muted-foreground">จำนวนเงิน</th>
                    <th className="text-left px-4 py-3 font-medium text-muted-foreground">สถานะ</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {purchases.map((purchase) => (
                    <tr key={purchase.id}>
                      <td className="px-4 py-3 text-foreground">
                        {new Date(purchase.created_at).toLocaleDateString('th-TH')}
                      </td>
                      <td className="px-4 py-3 text-foreground capitalize">{purchase.pack_type}</td>
                      <td className="px-4 py-3 text-foreground">
                        {formatBaht(purchase.amount_satang)} {purchase.currency}
                      </td>
                      <td className="px-4 py-3">
                        <StatusBadge status={purchase.status} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Mobile card list */}
            <div className="sm:hidden space-y-3">
              {purchases.map((purchase) => (
                <div key={purchase.id} className="flex items-center justify-between p-3 rounded-lg bg-muted/50">
                  <div>
                    <p className="text-sm font-medium text-foreground capitalize">{purchase.pack_type}</p>
                    <p className="text-xs text-muted-foreground">
                      {new Date(purchase.created_at).toLocaleDateString('th-TH')}
                    </p>
                  </div>
                  <div className="text-right">
                    <p className="text-sm font-semibold text-foreground">
                      {formatBaht(purchase.amount_satang)} {purchase.currency}
                    </p>
                    <StatusBadge status={purchase.status} />
                  </div>
                </div>
              ))}
            </div>
          </>
        )}
      </Collapsible>
    </section>
  )
}
