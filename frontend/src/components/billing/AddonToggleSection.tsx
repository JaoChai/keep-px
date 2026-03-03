import {
  LayoutGrid,
  Radio,
  Zap,
  CalendarDays,
  Loader2,
} from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import type { Subscription } from '@/types'

const ADDONS = [
  {
    type: 'sale_pages_25',
    name: 'Sale Pages +25',
    price: 199,
    freeLabel: 'ฟรี 5',
    upgradeLabel: 'เพิ่มเป็น 30 หน้า',
    icon: LayoutGrid,
  },
  {
    type: 'pixels_40',
    name: 'Pixels +40',
    price: 149,
    freeLabel: 'ฟรี 10',
    upgradeLabel: 'เพิ่มเป็น 50',
    icon: Radio,
  },
  {
    type: 'events_1m',
    name: 'Events 1M',
    price: 490,
    freeLabel: 'ฟรี 200K',
    upgradeLabel: 'เพิ่มเป็น 1M/เดือน',
    icon: Zap,
  },
  {
    type: 'retention_180',
    name: 'Retention 180d',
    price: 390,
    freeLabel: 'ฟรี 60 วัน',
    upgradeLabel: 'เพิ่มเป็น 180 วัน',
    icon: CalendarDays,
  },
  {
    type: 'retention_365',
    name: 'Retention 365d',
    price: 690,
    freeLabel: 'ฟรี 60 วัน',
    upgradeLabel: 'เพิ่มเป็น 365 วัน',
    icon: CalendarDays,
  },
] as const

interface AddonToggleSectionProps {
  activeSubscriptions: Subscription[]
  pendingCheckoutType: string | null
  isPending: boolean
  onCheckout: (params: { addon_type: string }) => void
  onManageBilling: () => void
}

export function AddonToggleSection({
  activeSubscriptions,
  pendingCheckoutType,
  isPending,
  onCheckout,
  onManageBilling,
}: AddonToggleSectionProps) {
  return (
    <section>
      <div className="flex items-center gap-2 mb-4">
        <h2 className="text-lg font-semibold text-foreground">ส่วนเสริม</h2>
        <Badge variant="secondary">รายเดือน · ยกเลิกได้ทุกเมื่อ</Badge>
      </div>

      <Card>
        <CardContent className="p-0 divide-y divide-border">
          {ADDONS.map((addon) => {
            const activeSub = activeSubscriptions.find((s) => s.addon_type === addon.type)
            const isActive = !!activeSub
            const isToggling = pendingCheckoutType === addon.type

            return (
              <div key={addon.type} className="flex items-center gap-4 px-5 py-4">
                <div className="h-9 w-9 rounded-lg bg-muted flex items-center justify-center shrink-0">
                  <addon.icon className="h-5 w-5 text-foreground" />
                </div>

                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="text-sm font-medium text-foreground">{addon.name}</p>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    {addon.freeLabel} → {addon.upgradeLabel}
                  </p>
                  {isActive && activeSub.current_period_end && (
                    <p className="text-xs text-emerald-600 mt-0.5">
                      ต่ออายุ {new Date(activeSub.current_period_end).toLocaleDateString('th-TH')}
                    </p>
                  )}
                </div>

                <div className="text-right shrink-0 mr-2">
                  <p className="text-sm font-semibold text-foreground">
                    ฿{addon.price.toLocaleString('th-TH')}
                  </p>
                  <p className="text-xs text-muted-foreground">/เดือน</p>
                </div>

                {isToggling ? (
                  <Loader2 className="h-5 w-5 animate-spin text-muted-foreground shrink-0" />
                ) : (
                  <Switch
                    checked={isActive}
                    disabled={isPending}
                    aria-label={`เปิด/ปิด ${addon.name}`}
                    onCheckedChange={(checked) => {
                      if (checked) {
                        onCheckout({ addon_type: addon.type })
                      } else {
                        onManageBilling()
                      }
                    }}
                  />
                )}
              </div>
            )
          })}
        </CardContent>
      </Card>
    </section>
  )
}
