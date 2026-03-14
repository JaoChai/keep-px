import { CreditCard } from 'lucide-react'
import { Collapsible } from '@/components/ui/collapsible'
import { Badge } from '@/components/ui/badge'
import { formatBaht, PACK_TYPE_NAMES } from '@/lib/utils'
import type { Purchase } from '@/types'

interface PurchaseHistorySectionProps {
  purchases: Purchase[]
  isLoading: boolean
}

const STATUS_CONFIG = {
  completed: { variant: 'success' as const, label: 'สำเร็จ' },
  pending: { variant: 'secondary' as const, label: 'รอดำเนินการ' },
} as const

function StatusBadge({ status }: { status: string }) {
  const { variant, label } = STATUS_CONFIG[status as keyof typeof STATUS_CONFIG] ?? { variant: 'destructive' as const, label: 'ล้มเหลว' }
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
          <div className="text-center py-6">
            <CreditCard className="h-8 w-8 text-muted-foreground mx-auto mb-2" />
            <p className="text-sm text-muted-foreground">
              ยังไม่มีประวัติการซื้อ
            </p>
          </div>
        ) : (
          <>
            {/* Desktop table */}
            <div className="hidden sm:block overflow-x-auto">
              <table className="w-full text-sm" aria-label="ประวัติการซื้อ">
                <thead className="bg-secondary">
                  <tr>
                    <th className="text-left px-4 py-2.5 text-xs font-medium text-muted-foreground">วันที่</th>
                    <th className="text-left px-4 py-2.5 text-xs font-medium text-muted-foreground">แพ็ก</th>
                    <th className="text-left px-4 py-2.5 text-xs font-medium text-muted-foreground">จำนวนเงิน</th>
                    <th className="text-left px-4 py-2.5 text-xs font-medium text-muted-foreground">สถานะ</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {purchases.map((purchase) => (
                    <tr key={purchase.id} className="hover:bg-muted/50 transition-colors">
                      <td className="px-4 py-3 text-foreground">
                        {new Date(purchase.created_at).toLocaleDateString('th-TH')}
                      </td>
                      <td className="px-4 py-3 text-foreground">{PACK_TYPE_NAMES[purchase.pack_type] ?? purchase.pack_type}</td>
                      <td className="px-4 py-3 text-foreground">
                        ฿{formatBaht(purchase.amount_satang)}
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
                <div key={purchase.id} className="flex items-center justify-between p-3 rounded-lg bg-muted/50 hover:bg-muted transition-colors">
                  <div>
                    <p className="text-sm font-medium text-foreground">{PACK_TYPE_NAMES[purchase.pack_type] ?? purchase.pack_type}</p>
                    <p className="text-xs text-muted-foreground">
                      {new Date(purchase.created_at).toLocaleDateString('th-TH')}
                    </p>
                  </div>
                  <div className="text-right">
                    <p className="text-sm font-semibold text-foreground">
                      ฿{formatBaht(purchase.amount_satang)}
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
