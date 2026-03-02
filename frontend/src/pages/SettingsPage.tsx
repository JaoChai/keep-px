import { useState } from 'react'
import { Copy, Check, Key } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { useAuthStore } from '@/stores/auth-store'

export function SettingsPage() {
  const customer = useAuthStore((s) => s.customer)
  const [copied, setCopied] = useState(false)

  const copyAPIKey = () => {
    if (customer?.api_key) {
      navigator.clipboard.writeText(customer.api_key)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
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
                value={customer?.api_key || ''}
                readOnly
                className="font-mono text-sm"
              />
              <Button variant="outline" onClick={copyAPIKey}>
                {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
              </Button>
            </div>
            <p className="text-xs text-muted-foreground mt-2">
              คีย์นี้ใช้สำหรับเทมเพลตหน้าขายเพื่อส่งอีเวนต์ไปยังพิกเซลของคุณ
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
