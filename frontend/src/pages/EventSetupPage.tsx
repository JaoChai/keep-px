import { useState, useRef, useEffect, useCallback } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Eye, MousePointerClick, Globe, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { usePixels } from '@/hooks/use-pixels'
import { useCreateRule } from '@/hooks/use-rules'
import api from '@/lib/api'

const urlSchema = z.object({
  url: z.string().url('Please enter a valid URL'),
  pixel_id: z.string().min(1, 'Select a pixel'),
})

type UrlForm = z.infer<typeof urlSchema>

interface SelectedElement {
  tagName: string
  text: string
  cssSelector: string
}

export function EventSetupPage() {
  const { data: pixels } = usePixels()
  const [iframeUrl, setIframeUrl] = useState('')
  const [originalUrl, setOriginalUrl] = useState('')
  const [isLoadingPage, setIsLoadingPage] = useState(false)
  const [loadError, setLoadError] = useState('')
  const [selectedElement, setSelectedElement] = useState<SelectedElement | null>(null)
  const [showRuleDialog, setShowRuleDialog] = useState(false)
  const [selectedEventName, setSelectedEventName] = useState('ViewContent')
  const [selectedTriggerType, setSelectedTriggerType] = useState('click')
  const iframeRef = useRef<HTMLIFrameElement>(null)

  const {
    register,
    handleSubmit,
    getValues,
  } = useForm<UrlForm>({
    resolver: zodResolver(urlSchema),
  })

  const { mutate: createRule, isPending } = useCreateRule()

  useEffect(() => {
    const handleMessage = (e: MessageEvent) => {
      if (e.data?.type === 'pixlinks:element-selected') {
        setSelectedElement(e.data.data)
        setShowRuleDialog(true)
      }
    }
    window.addEventListener('message', handleMessage)
    return () => window.removeEventListener('message', handleMessage)
  }, [])

  // Clean up blob URL on unmount or when URL changes
  const revokePreviousUrl = useCallback(() => {
    if (iframeUrl.startsWith('blob:')) {
      URL.revokeObjectURL(iframeUrl)
    }
  }, [iframeUrl])

  useEffect(() => {
    return revokePreviousUrl
  }, [revokePreviousUrl])

  const onLoadPage = async (data: UrlForm) => {
    setIsLoadingPage(true)
    setLoadError('')
    try {
      const response = await api.get('/proxy', {
        params: { url: data.url },
        responseType: 'text',
        transformResponse: [(data: string) => data],
      })

      let html: string = response.data

      // Inject <base> tag so relative URLs in the fetched page resolve correctly
      const targetUrl = data.url.startsWith('http') ? data.url : `https://${data.url}`
      const baseOrigin = new URL(targetUrl).origin
      html = html.replace(/<base[^>]*>/gi, '')
      html = html.replace('<head>', `<head><base href="${baseOrigin}/">`)

      revokePreviousUrl()
      const blob = new Blob([html], { type: 'text/html' })
      setIframeUrl(URL.createObjectURL(blob))
      setOriginalUrl(targetUrl)
    } catch {
      setLoadError('ไม่สามารถโหลดหน้าเว็บได้ กรุณาตรวจสอบ URL แล้วลองใหม่')
    } finally {
      setIsLoadingPage(false)
    }
  }

  const activateSetup = () => {
    iframeRef.current?.contentWindow?.postMessage(
      { type: 'pixlinks:activate-setup' },
      '*'
    )
  }

  return (
    <div className="flex flex-col h-[calc(100vh-4rem)]">
      {/* Toolbar */}
      <div className="flex items-center gap-4 p-4 border-b border-neutral-200 bg-white">
        <form onSubmit={handleSubmit(onLoadPage)} className="flex items-center gap-3 flex-1">
          <select
            className="h-9 rounded-md border border-neutral-200 px-3 text-sm"
            {...register('pixel_id')}
          >
            <option value="">Select pixel...</option>
            {pixels?.map((p) => (
              <option key={p.id} value={p.id}>{p.name}</option>
            ))}
          </select>

          <div className="flex-1 flex gap-2">
            <Input
              placeholder="https://your-salepage.com"
              {...register('url')}
            />
            <Button type="submit" disabled={isLoadingPage}>
              {isLoadingPage ? <Loader2 className="h-4 w-4 animate-spin" /> : <Globe className="h-4 w-4" />}
              {isLoadingPage ? 'Loading...' : 'Load'}
            </Button>
          </div>
        </form>

        {iframeUrl && (
          <Button variant="outline" onClick={activateSetup}>
            <MousePointerClick className="h-4 w-4" />
            Select Element
          </Button>
        )}
      </div>

      {/* Preview Area */}
      <div className="flex-1 bg-neutral-100">
        {iframeUrl ? (
          <iframe
            ref={iframeRef}
            src={iframeUrl}
            className="w-full h-full border-0"
            sandbox="allow-scripts allow-same-origin"
          />
        ) : (
          <div className="flex items-center justify-center h-full text-neutral-400">
            <div className="text-center">
              {isLoadingPage ? (
                <>
                  <Loader2 className="h-12 w-12 mx-auto mb-4 opacity-50 animate-spin" />
                  <p className="text-lg font-medium">กำลังโหลดหน้าเว็บ...</p>
                </>
              ) : loadError ? (
                <>
                  <Eye className="h-12 w-12 mx-auto mb-4 opacity-50" />
                  <p className="text-lg font-medium text-red-500">{loadError}</p>
                  <p className="text-sm mt-1">ลองตรวจสอบ URL แล้วกด Load อีกครั้ง</p>
                </>
              ) : (
                <>
                  <Eye className="h-12 w-12 mx-auto mb-4 opacity-50" />
                  <p className="text-lg font-medium">Enter a URL to preview your salepage</p>
                  <p className="text-sm mt-1">Then click elements to create event rules</p>
                </>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Rule Creation Dialog */}
      <Dialog open={showRuleDialog} onOpenChange={setShowRuleDialog}>
        <DialogContent onClose={() => setShowRuleDialog(false)}>
          <DialogHeader>
            <DialogTitle>Create Event Rule</DialogTitle>
          </DialogHeader>
          {selectedElement && (
            <div className="space-y-4 mt-4">
              <div>
                <Label className="text-xs text-neutral-500">Selected Element</Label>
                <div className="mt-1 p-3 rounded-md bg-neutral-50 border border-neutral-200">
                  <Badge variant="secondary" className="mb-2">{selectedElement.tagName}</Badge>
                  {selectedElement.text && (
                    <p className="text-sm text-neutral-700 mt-1">"{selectedElement.text}"</p>
                  )}
                  <p className="text-xs text-neutral-400 mt-1 font-mono break-all">{selectedElement.cssSelector}</p>
                </div>
              </div>

              <div className="space-y-2">
                <Label>Event Name</Label>
                <select
                  className="flex h-9 w-full rounded-md border border-neutral-200 px-3 text-sm"
                  value={selectedEventName}
                  onChange={(e) => setSelectedEventName(e.target.value)}
                >
                  <option value="ViewContent">ViewContent</option>
                  <option value="AddToCart">AddToCart</option>
                  <option value="InitiateCheckout">InitiateCheckout</option>
                  <option value="Purchase">Purchase</option>
                  <option value="Lead">Lead</option>
                  <option value="CompleteRegistration">CompleteRegistration</option>
                  <option value="AddPaymentInfo">AddPaymentInfo</option>
                  <option value="Search">Search</option>
                </select>
              </div>

              <div className="space-y-2">
                <Label>Trigger</Label>
                <select
                  className="flex h-9 w-full rounded-md border border-neutral-200 px-3 text-sm"
                  value={selectedTriggerType}
                  onChange={(e) => setSelectedTriggerType(e.target.value)}
                >
                  <option value="click">Click</option>
                  <option value="pageview">Page View</option>
                  <option value="scroll">Scroll into View</option>
                </select>
              </div>

              <DialogFooter>
                <Button variant="outline" onClick={() => setShowRuleDialog(false)}>Cancel</Button>
                <Button
                  disabled={isPending}
                  onClick={() => {
                    const pixelId = getValues('pixel_id')
                    createRule({
                      pixelId,
                      page_url: originalUrl,
                      event_name: selectedEventName,
                      trigger_type: selectedTriggerType,
                      css_selector: selectedElement.cssSelector,
                      element_text: selectedElement.text,
                    }, {
                      onSuccess: () => {
                        setShowRuleDialog(false)
                        setSelectedElement(null)
                      },
                      onError: (err) => console.error(err),
                    })
                  }}
                >
                  {isPending ? 'Saving...' : 'Save Rule'}
                </Button>
              </DialogFooter>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
