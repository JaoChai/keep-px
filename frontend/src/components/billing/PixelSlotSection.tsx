import { useState } from 'react'
import { Minus, Plus, CreditCard, Loader2, CheckCircle2, Radio } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'

interface PixelSlotSectionProps {
  currentSlots: number  // 0 = free user
  isPending: boolean
  pendingType: string | null
  onCheckout: (params: { type: string; quantity?: number }) => void
  onUpdateSlots: (quantity: number) => void
  isUpdating: boolean
}

export function PixelSlotSection({
  currentSlots,
  isPending,
  pendingType,
  onCheckout,
  onUpdateSlots,
  isUpdating,
}: PixelSlotSectionProps) {
  const [quantity, setQuantity] = useState(Math.max(currentSlots, 1))
  const pricePerSlot = 199
  const totalPrice = quantity * pricePerSlot
  const isExistingSubscriber = currentSlots > 0
  const quantityChanged = quantity !== currentSlots

  return (
    <Card className="h-full flex flex-col">
      <CardContent className="p-5 flex flex-col flex-1">
        {/* Header with icon */}
        <div className="flex items-center gap-3 mb-4">
          <div className="h-9 w-9 rounded-lg bg-primary/5 flex items-center justify-center">
            <Radio className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h3 className="text-lg font-bold text-foreground">Pixel Slots</h3>
            <p className="text-xs text-muted-foreground">฿{pricePerSlot}/slot/เดือน</p>
          </div>
        </div>

        {/* Quantity selector — horizontal row */}
        <div className="flex items-center justify-between bg-secondary/50 rounded-lg p-3 mb-4">
          <div className="flex items-center gap-3">
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              onClick={() => setQuantity(Math.max(1, quantity - 1))}
              disabled={quantity <= 1}
            >
              <Minus className="h-3.5 w-3.5" />
            </Button>
            <div className="text-center min-w-[60px]">
              <p className="text-2xl font-bold text-foreground">{quantity}</p>
              <p className="text-xs text-muted-foreground">slots</p>
            </div>
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              onClick={() => setQuantity(quantity + 1)}
            >
              <Plus className="h-3.5 w-3.5" />
            </Button>
          </div>
          <div className="text-right">
            <p className="text-xl font-bold text-foreground">
              ฿{totalPrice.toLocaleString('th-TH')}
            </p>
            <p className="text-xs text-muted-foreground">/เดือน</p>
          </div>
        </div>

        {/* What's included */}
        <ul className="space-y-1.5 text-sm text-muted-foreground mb-4">
          <li className="flex items-center gap-2">
            <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500 shrink-0" />
            {quantity} pixel{quantity > 1 ? 's' : ''} + {quantity} sale page{quantity > 1 ? 's' : ''}
          </li>
          <li className="flex items-center gap-2">
            <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500 shrink-0" />
            {(quantity * 100000).toLocaleString()} events/เดือน (pooled)
          </li>
          <li className="flex items-center gap-2">
            <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500 shrink-0" />
            เก็บข้อมูล 180 วัน
          </li>
        </ul>

        {/* CTA button — pushed to bottom */}
        <div className="mt-auto pt-2">
          {isExistingSubscriber ? (
            <Button
              className="w-full"
              onClick={() => onUpdateSlots(quantity)}
              disabled={!quantityChanged || isUpdating || isPending}
            >
              {isUpdating ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <CreditCard className="h-4 w-4" />
              )}
              {quantityChanged ? 'อัพเดทจำนวน' : 'จำนวนปัจจุบัน'}
            </Button>
          ) : (
            <Button
              className="w-full"
              onClick={() => onCheckout({ type: 'pixel_slots', quantity })}
              disabled={isPending}
            >
              {pendingType === 'pixel_slots' ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <CreditCard className="h-4 w-4" />
              )}
              สมัครสมาชิก
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
