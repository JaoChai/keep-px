import { useState } from 'react'
import { Copy, Check, Key, Eye, EyeOff, RefreshCw } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { useAuthStore } from '@/stores/auth-store'
import { useRegenerateAPIKey } from '@/hooks/use-auth'

export function SettingsPage() {
  const customer = useAuthStore((s) => s.customer)
  const [copied, setCopied] = useState(false)
  const [showKey, setShowKey] = useState(false)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const regenerateMutation = useRegenerateAPIKey()

  const copyAPIKey = () => {
    if (customer?.api_key) {
      navigator.clipboard.writeText(customer.api_key)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  const handleRegenerate = () => {
    regenerateMutation.mutate(undefined, {
      onSuccess: () => {
        setConfirmOpen(false)
        toast.success('สร้าง API Key ใหม่สำเร็จ')
      },
      onError: () => {
        toast.error('ไม่สามารถสร้าง API Key ใหม่ได้')
      },
    })
  }

  return (
    <div>
      <h1 className="text-2xl font-bold text-foreground mb-6">ตั้งค่า</h1>

      <div className="space-y-6 max-w-2xl">
        {/* Profile */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">โปรไฟล์</CardTitle>
            <CardDescription>ข้อมูลบัญชีของคุณ</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label>ชื่อ</Label>
              <Input value={customer?.name || ''} readOnly />
            </div>
            <div className="space-y-2">
              <Label>อีเมล</Label>
              <Input value={customer?.email || ''} readOnly />
            </div>
            <div className="space-y-2">
              <Label>แพลน</Label>
              <Input value={customer?.plan || 'free'} readOnly />
            </div>
          </CardContent>
        </Card>

        {/* API Key */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Key className="h-4 w-4" />
              API Key
            </CardTitle>
            <CardDescription>API Key สำหรับรับข้อมูลอีเวนต์</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex gap-2">
              <Input
                value={
                  showKey
                    ? customer?.api_key || ''
                    : customer?.api_key
                      ? `${'•'.repeat(12)}${customer.api_key.slice(-4)}`
                      : ''
                }
                readOnly
                className="font-mono text-sm"
              />
              <Button variant="outline" onClick={() => setShowKey(!showKey)}>
                {showKey ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </Button>
              <Button variant="outline" onClick={copyAPIKey}>
                {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
              </Button>
            </div>
            <div className="flex items-center justify-between mt-3">
              <p className="text-xs text-muted-foreground">
                คีย์นี้ใช้สำหรับเทมเพลตเซลเพจเพื่อส่งอีเวนต์ไปยังพิกเซลของคุณ
              </p>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setConfirmOpen(true)}
                disabled={regenerateMutation.isPending}
              >
                <RefreshCw className={`h-4 w-4 mr-1.5 ${regenerateMutation.isPending ? 'animate-spin' : ''}`} />
                สร้างคีย์ใหม่
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>

      <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>สร้าง API Key ใหม่</DialogTitle>
            <DialogDescription>
              คีย์เดิมจะใช้ไม่ได้ทันที ต้องการสร้างคีย์ใหม่หรือไม่?
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirmOpen(false)} disabled={regenerateMutation.isPending}>
              ยกเลิก
            </Button>
            <Button variant="destructive" onClick={handleRegenerate} disabled={regenerateMutation.isPending}>
              {regenerateMutation.isPending ? (
                <RefreshCw className="h-4 w-4 mr-1.5 animate-spin" />
              ) : null}
              ยืนยัน
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
