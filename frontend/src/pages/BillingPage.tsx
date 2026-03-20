import { useCallback, useEffect, useState } from 'react'
import { useSearchParams } from 'react-router'
import { toast } from 'sonner'
import { Check, X } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { useBillingOverview, useQuota, useCreateCheckout, useCreatePortalSession, useUpdateSlots } from '@/hooks/use-billing'
import { AccountStatusCard } from '@/components/billing/AccountStatusCard'
import { PixelSlotSection } from '@/components/billing/PixelSlotSection'
import { ReplaySection } from '@/components/billing/ReplaySection'
import { PurchaseHistorySection } from '@/components/billing/PurchaseHistorySection'

const PRICING_FEATURES = [
  { feature: 'จำนวนพิกเซล', free: '2', paid: 'ปรับได้ตาม Slots' },
  { feature: 'จำนวนเซลเพจ', free: '2', paid: 'ปรับได้ตาม Slots' },
  { feature: 'อีเวนต์ต่อเดือน', free: '1,000', paid: 'ตาม Slots' },
  { feature: 'เก็บข้อมูล', free: '7 วัน', paid: '90 วัน' },
  { feature: 'รีเพลย์', free: false, paid: true },
  { feature: 'Replay Credits', free: false, paid: true },
  { feature: 'CAPI Forwarding', free: true, paid: true },
  { feature: 'Analytics Dashboard', free: true, paid: true },
] as const

function PricingComparisonSection() {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">เปรียบเทียบแพ็กเกจ</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border">
                <th className="text-left py-3 px-4 font-medium text-muted-foreground">ฟีเจอร์</th>
                <th className="text-center py-3 px-4 font-medium text-muted-foreground">
                  <Badge variant="secondary">Free</Badge>
                </th>
                <th className="text-center py-3 px-4 font-medium text-muted-foreground bg-primary/5 rounded-t-lg">
                  <Badge variant="default">Paid</Badge>
                </th>
              </tr>
            </thead>
            <tbody>
              {PRICING_FEATURES.map((row) => (
                <tr key={row.feature} className="border-b border-border last:border-0">
                  <td className="py-3 px-4 text-foreground">{row.feature}</td>
                  <td className="py-3 px-4 text-center">
                    {typeof row.free === 'boolean' ? (
                      row.free ? (
                        <Check className="h-4 w-4 text-emerald-600 mx-auto" />
                      ) : (
                        <X className="h-4 w-4 text-muted-foreground mx-auto" />
                      )
                    ) : (
                      <span className="text-muted-foreground">{row.free}</span>
                    )}
                  </td>
                  <td className="py-3 px-4 text-center bg-primary/5">
                    {typeof row.paid === 'boolean' ? (
                      row.paid ? (
                        <Check className="h-4 w-4 text-emerald-600 mx-auto" />
                      ) : (
                        <X className="h-4 w-4 text-muted-foreground mx-auto" />
                      )
                    ) : (
                      <span className="font-medium text-foreground">{row.paid}</span>
                    )}
                  </td>
                </tr>
              ))}
              <tr>
                <td className="py-4 px-4" />
                <td className="py-4 px-4" />
                <td className="py-4 px-4 bg-primary/5 rounded-b-lg">
                  <div className="flex flex-col items-center gap-1">
                    <Button
                      size="lg"
                      onClick={() => document.getElementById('pixel-slots')?.scrollIntoView({ behavior: 'smooth' })}
                    >
                      อัปเกรดเลย
                    </Button>
                    <span className="text-xs text-muted-foreground">ดู Pixel Slots ด้านล่าง</span>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  )
}

export function BillingPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const { data: overview, isLoading: overviewLoading } = useBillingOverview()
  const { data: quota } = useQuota()
  const checkout = useCreateCheckout()
  const portal = useCreatePortalSession()
  const updateSlots = useUpdateSlots()
  const [pendingType, setPendingType] = useState<string | null>(null)

  const handleCheckout = (params: { type: string; quantity?: number }) => {
    setPendingType(params.type)
    checkout.mutate(params, {
      onSettled: () => setPendingType(null),
    })
  }

  useEffect(() => {
    const status = searchParams.get('status')
    if (status === 'success') {
      toast.success('ชำระเงินสำเร็จ!')
      setSearchParams({}, { replace: true })
    } else if (status === 'cancel') {
      toast.info('การชำระเงินถูกยกเลิก')
      setSearchParams({}, { replace: true })
    }
  }, [searchParams, setSearchParams])

  const activeCredits = overview?.credits ?? []
  const scrollToSlots = useCallback(() => {
    document.getElementById('pixel-slots')?.scrollIntoView({ behavior: 'smooth' })
  }, [])

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-foreground">การเงิน</h1>
        <p className="text-sm text-muted-foreground mt-1">
          จัดการ Pixel Slots และแพ็กรีเพลย์
        </p>
      </div>

      {/* Account Status */}
      {quota && (
        <AccountStatusCard
          quota={quota}
          credits={activeCredits}
          onUpgrade={scrollToSlots}
          onManageBilling={() => portal.mutate()}
          isPortalPending={portal.isPending}
        />
      )}

      {/* Pixel Slots + Replay side-by-side */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div id="pixel-slots">
          <PixelSlotSection
            currentSlots={quota?.pixel_slots ?? 0}
            isPending={checkout.isPending}
            pendingType={pendingType}
            onCheckout={handleCheckout}
            onUpdateSlots={(q) => updateSlots.mutate(q)}
            isUpdating={updateSlots.isPending}
          />
        </div>

        <ReplaySection
          credits={activeCredits}
          pendingCheckoutType={pendingType}
          isPending={checkout.isPending}
          onCheckout={handleCheckout}
        />
      </div>

      {/* Pricing Comparison */}
      <PricingComparisonSection />

      {/* Purchase History */}
      <PurchaseHistorySection
        purchases={overview?.purchases ?? []}
        isLoading={overviewLoading}
      />
    </div>
  )
}
