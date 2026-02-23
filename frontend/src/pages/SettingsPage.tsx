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
      <h1 className="text-2xl font-bold text-neutral-900 mb-6">Settings</h1>

      <div className="space-y-6 max-w-2xl">
        {/* Profile */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Profile</CardTitle>
            <CardDescription>Your account information</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label>Name</Label>
              <Input value={customer?.name || ''} readOnly />
            </div>
            <div className="space-y-2">
              <Label>Email</Label>
              <Input value={customer?.email || ''} readOnly />
            </div>
            <div className="space-y-2">
              <Label>Plan</Label>
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
            <CardDescription>Use this key in your SDK integration</CardDescription>
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
            <p className="text-xs text-neutral-400 mt-2">
              Go to <a href="/pixels" className="text-blue-500 hover:underline">Pixels</a> and click the <code className="bg-neutral-100 px-1 rounded">&lt;/&gt;</code> button to get a ready-to-use snippet for each pixel.
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
