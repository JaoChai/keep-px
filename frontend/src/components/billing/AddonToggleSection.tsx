import {
  LayoutGrid,
  Radio,
  Zap,
  Loader2,
} from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Switch } from '@/components/ui/switch'
import type { Subscription } from '@/types'

const ADDONS = [
  {
    type: 'pixels_10',
    name: 'Pixels +10',
    price: 290,
    description: 'เพิ่มพิกเซลอีก 10',
    icon: Radio,
  },
  {
    type: 'sale_pages_10',
    name: 'Sale Pages +10',
    price: 190,
    description: 'เพิ่ม Sale Pages อีก 10 หน้า',
    icon: LayoutGrid,
  },
  {
    type: 'events_1m',
    name: 'Events +1M',
    price: 490,
    description: 'เพิ่ม 1,000,000 events/เดือน',
    icon: Zap,
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
    <div>
      <p className="text-xs text-muted-foreground mb-3">
        เพิ่มความสามารถแบบรายเดือน · ยกเลิกได้ทุกเมื่อ
      </p>

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
                    {addon.description}
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
    </div>
  )
}
