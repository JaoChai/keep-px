import { useState } from 'react'
import { Minus, Plus, CreditCard, Loader2, CheckCircle2 } from 'lucide-react'
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
    <Card>
      <CardContent className="p-6 space-y-5">
        <div>
          <h3 className="text-lg font-bold text-foreground">Pixel Slots</h3>
          <p className="text-sm text-muted-foreground mt-1">
            แต่ละ slot ได้รับ: 1 pixel + 1 sale page + 100K events/เดือน + เก็บข้อมูล 180 วัน
          </p>
        </div>

        {/* Quantity selector */}
        <div className="flex items-center justify-center gap-4">
          <Button
            variant="outline"
            size="icon"
            onClick={() => setQuantity(Math.max(1, quantity - 1))}
            disabled={quantity <= 1}
          >
            <Minus className="h-4 w-4" />
          </Button>
          <div className="text-center min-w-[120px]">
            <p className="text-4xl font-bold text-foreground">{quantity}</p>
            <p className="text-sm text-muted-foreground">pixel slots</p>
          </div>
          <Button
            variant="outline"
            size="icon"
            onClick={() => setQuantity(quantity + 1)}
          >
            <Plus className="h-4 w-4" />
          </Button>
        </div>

        {/* Price display */}
        <div className="text-center space-y-1">
          <p className="text-2xl font-bold text-foreground">
            ฿{totalPrice.toLocaleString('th-TH')}<span className="text-sm font-normal text-muted-foreground">/เดือน</span>
          </p>
          <p className="text-xs text-muted-foreground">
            ฿{pricePerSlot}/pixel/เดือน
          </p>
        </div>

        {/* What's included */}
        <ul className="space-y-2 text-sm text-muted-foreground">
          <li className="flex items-center gap-2">
            <CheckCircle2 className="h-4 w-4 text-emerald-500 shrink-0" />
            {quantity} pixel{quantity > 1 ? 's' : ''} + {quantity} sale page{quantity > 1 ? 's' : ''}
          </li>
          <li className="flex items-center gap-2">
            <CheckCircle2 className="h-4 w-4 text-emerald-500 shrink-0" />
            {(quantity * 100000).toLocaleString()} events/เดือน (pooled)
          </li>
          <li className="flex items-center gap-2">
            <CheckCircle2 className="h-4 w-4 text-emerald-500 shrink-0" />
            เก็บข้อมูล 180 วัน
          </li>
        </ul>

        {/* CTA button */}
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
      </CardContent>
    </Card>
  )
}
