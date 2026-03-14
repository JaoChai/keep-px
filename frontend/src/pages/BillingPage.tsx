import { useCallback, useEffect, useState } from 'react'
import { useSearchParams } from 'react-router'
import { toast } from 'sonner'
import { useBillingOverview, useQuota, useCreateCheckout, useCreatePortalSession, useUpdateSlots } from '@/hooks/use-billing'
import { AccountStatusCard } from '@/components/billing/AccountStatusCard'
import { PixelSlotSection } from '@/components/billing/PixelSlotSection'
import { ReplaySection } from '@/components/billing/ReplaySection'
import { PurchaseHistorySection } from '@/components/billing/PurchaseHistorySection'

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

      {/* Purchase History */}
      <PurchaseHistorySection
        purchases={overview?.purchases ?? []}
        isLoading={overviewLoading}
      />
    </div>
  )
}
