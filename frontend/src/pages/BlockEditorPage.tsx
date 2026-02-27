import { useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router'
import { ArrowLeft, Copy, Check, ExternalLink, Info, Eye, Pencil } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Collapsible } from '@/components/ui/collapsible'
import { BlockEditor } from '@/components/sale-pages/BlockEditor'
import { BlockPreview } from '@/components/sale-pages/BlockPreview'
import { useSalePages, useCreateSalePage, useUpdateSalePage } from '@/hooks/use-sale-pages'
import { usePixels } from '@/hooks/use-pixels'
import type { Block, SalePage, SalePageContentV2 } from '@/types'

const CTA_EVENT_OPTIONS = [
  { value: 'Lead', label: 'Lead — ลูกค้าสนใจ ต้องการข้อมูลเพิ่ม' },
  { value: 'Purchase', label: 'Purchase — ลูกค้าซื้อสินค้า/บริการ' },
  { value: 'InitiateCheckout', label: 'InitiateCheckout — ลูกค้าเริ่มชำระเงิน' },
  { value: 'AddToCart', label: 'AddToCart — ลูกค้าเพิ่มสินค้าลงตะกร้า' },
  { value: 'CompleteRegistration', label: 'CompleteRegistration — ลูกค้าสมัครสมาชิกสำเร็จ' },
  { value: 'Schedule', label: 'Schedule — ลูกค้าจองนัดหมาย' },
  { value: 'SubmitApplication', label: 'SubmitApplication — ลูกค้าส่งใบสมัคร' },
] as const

function generateSlug(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9\s-]/g, '')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')
}

// Wrapper: handles data loading and v1→v2 redirect
export function BlockEditorPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const isEditing = !!id
  const { data: salePages, isLoading } = useSalePages()

  if (isEditing) {
    if (isLoading) {
      return <div className="text-center py-12 text-neutral-500">Loading...</div>
    }
    const page = salePages?.find((p) => p.id === id)
    if (!page) {
      return <div className="text-center py-12 text-neutral-500">Page not found</div>
    }
    const content = page.content as SalePageContentV2
    if (content.version !== 2) {
      // v1 page — redirect to classic editor
      navigate(`/sale-pages/${id}/edit`, { replace: true })
      return null
    }
    return <BlockEditorInner key={id} existingPage={page} />
  }

  return <BlockEditorInner key="new" />
}

// Inner: pure editor component with stable initial state
function BlockEditorInner({ existingPage }: { existingPage?: SalePage }) {
  const navigate = useNavigate()
  const isEditing = !!existingPage

  const { data: pixels } = usePixels()
  const createSalePage = useCreateSalePage()
  const updateSalePage = useUpdateSalePage()

  // Parse existing v2 content for initial values
  const v2 = existingPage ? (existingPage.content as SalePageContentV2) : null

  // Page settings
  const [name, setName] = useState(existingPage?.name ?? '')
  const [slug, setSlug] = useState(existingPage?.slug ?? '')
  const [slugTouched, setSlugTouched] = useState(isEditing)
  const [pixelId, setPixelId] = useState(existingPage?.pixel_id ?? '')

  // Blocks
  const [blocks, setBlocks] = useState<Block[]>(v2?.blocks ?? [])

  // Tracking
  const [ctaEventName, setCtaEventName] = useState(v2?.tracking?.cta_event_name || 'Lead')
  const [trackingContentName, setTrackingContentName] = useState(v2?.tracking?.content_name || '')
  const [trackingContentValue, setTrackingContentValue] = useState(v2?.tracking?.content_value || 0)
  const [trackingCurrency, setTrackingCurrency] = useState(v2?.tracking?.currency || 'THB')

  // UI state
  const [mobileView, setMobileView] = useState<'edit' | 'preview'>('edit')
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [publishedDialog, setPublishedDialog] = useState<{ slug: string } | null>(null)
  const [copiedUrl, setCopiedUrl] = useState(false)

  const handleNameChange = (value: string) => {
    setName(value)
    if (!slugTouched) {
      setSlug(generateSlug(value))
    }
  }

  const buildContent = (): SalePageContentV2 => ({
    version: 2,
    blocks,
    tracking: {
      cta_event_name: ctaEventName || 'Lead',
      content_name: trackingContentName || '',
      content_value: trackingContentValue || 0,
      currency: trackingCurrency || 'THB',
    },
  })

  const isSubmitting = createSalePage.isPending || updateSalePage.isPending

  const onSubmit = async (isPublished: boolean) => {
    setSubmitError(null)
    if (!name.trim() || !slug.trim()) {
      setSubmitError('กรุณากรอกชื่อหน้าเพจและ URL')
      return
    }
    if (blocks.length === 0) {
      setSubmitError('กรุณาเพิ่มบล็อกอย่างน้อย 1 อัน')
      return
    }
    try {
      const content = buildContent()
      const payload = {
        name,
        slug,
        pixel_id: pixelId || undefined,
        template_name: 'blocks',
        content,
        is_published: isPublished,
      }

      if (isEditing && existingPage) {
        await updateSalePage.mutateAsync({ id: existingPage.id, ...payload })
      } else {
        await createSalePage.mutateAsync(payload)
      }

      if (isPublished) {
        setPublishedDialog({ slug })
      } else {
        navigate('/sale-pages')
      }
    } catch {
      setSubmitError('บันทึกไม่สำเร็จ กรุณาลองใหม่อีกครั้ง')
    }
  }

  const copyUrl = (pageSlug: string) => {
    navigator.clipboard.writeText(`${window.location.origin}/p/${pageSlug}`)
    setCopiedUrl(true)
    setTimeout(() => setCopiedUrl(false), 2000)
  }

  return (
    <div>
      {/* Top Bar */}
      <div className="flex items-center justify-between mb-6">
        <Link
          to="/sale-pages"
          className="flex items-center gap-1.5 text-sm text-neutral-600 hover:text-neutral-900 transition-colors"
        >
          <ArrowLeft className="h-4 w-4" />
          Sale Pages
        </Link>
        <div className="flex items-center gap-2">
          {/* Mobile toggle */}
          <div className="flex lg:hidden border border-neutral-200 rounded-md overflow-hidden">
            <button
              type="button"
              className={`px-3 py-1.5 text-xs font-medium ${mobileView === 'edit' ? 'bg-neutral-900 text-white' : 'text-neutral-600'}`}
              onClick={() => setMobileView('edit')}
            >
              <Pencil className="h-3 w-3 inline mr-1" />
              แก้ไข
            </button>
            <button
              type="button"
              className={`px-3 py-1.5 text-xs font-medium ${mobileView === 'preview' ? 'bg-neutral-900 text-white' : 'text-neutral-600'}`}
              onClick={() => setMobileView('preview')}
            >
              <Eye className="h-3 w-3 inline mr-1" />
              พรีวิว
            </button>
          </div>
          <Button
            variant="outline"
            disabled={isSubmitting}
            onClick={() => onSubmit(false)}
          >
            {isSubmitting ? 'กำลังบันทึก...' : 'บันทึกแบบร่าง'}
          </Button>
          <Button
            disabled={isSubmitting}
            onClick={() => onSubmit(true)}
          >
            {isSubmitting ? 'กำลังเผยแพร่...' : isEditing && existingPage?.is_published ? 'อัพเดท' : 'เผยแพร่'}
          </Button>
        </div>
      </div>

      {/* Error Message */}
      {submitError && (
        <div className="mb-4 p-3 rounded-md bg-red-50 border border-red-200 text-sm text-red-700">
          {submitError}
        </div>
      )}

      {/* 2-Column Layout */}
      <div className="flex gap-8">
        {/* Left Column - Editor */}
        <div className={`flex-1 lg:w-2/3 space-y-4 ${mobileView === 'preview' ? 'hidden lg:block' : ''}`}>
          {/* Settings */}
          <Collapsible title="ตั้งค่าหน้าเพจ" defaultOpen={!isEditing}>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="page-name">ชื่อหน้าเพจ</Label>
                <Input
                  id="page-name"
                  placeholder="เช่น โปรโมชั่นครีมหน้าใส"
                  value={name}
                  onChange={(e) => handleNameChange(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="page-slug">URL</Label>
                <div className="flex">
                  <span className="inline-flex items-center px-3 rounded-l-md border border-r-0 border-neutral-200 bg-neutral-50 text-sm text-neutral-500">
                    /p/
                  </span>
                  <Input
                    id="page-slug"
                    className="rounded-l-none"
                    placeholder="my-page"
                    value={slug}
                    onChange={(e) => { setSlug(e.target.value); setSlugTouched(true) }}
                  />
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="pixel-select">Pixel (ไม่บังคับ)</Label>
                <select
                  id="pixel-select"
                  className="flex h-9 w-full rounded-md border border-neutral-200 bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-indigo-600"
                  value={pixelId}
                  onChange={(e) => setPixelId(e.target.value)}
                >
                  <option value="">ไม่เลือก pixel</option>
                  {pixels?.map((pixel) => (
                    <option key={pixel.id} value={pixel.id}>
                      {pixel.name} ({pixel.fb_pixel_id})
                    </option>
                  ))}
                </select>
              </div>
            </div>
          </Collapsible>

          {/* Block Editor */}
          <BlockEditor blocks={blocks} onChange={setBlocks} />

          {/* Tracking Settings */}
          <Collapsible title="ตั้งค่าการติดตาม">
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="cta-event">เมื่อกดปุ่ม CTA ให้ยิงอีเวนต์</Label>
                <select
                  id="cta-event"
                  className="flex h-9 w-full rounded-md border border-neutral-200 bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-indigo-600"
                  value={ctaEventName}
                  onChange={(e) => setCtaEventName(e.target.value)}
                >
                  {CTA_EVENT_OPTIONS.map((opt) => (
                    <option key={opt.value} value={opt.value}>{opt.label}</option>
                  ))}
                </select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="content-name">ชื่อสินค้า (ไม่บังคับ)</Label>
                <Input
                  id="content-name"
                  placeholder="เช่น ครีมหน้าใส, คอร์สออนไลน์"
                  value={trackingContentName}
                  onChange={(e) => setTrackingContentName(e.target.value)}
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-2">
                  <Label htmlFor="content-value">ราคาสินค้า (ไม่บังคับ)</Label>
                  <Input
                    id="content-value"
                    type="number"
                    placeholder="0"
                    value={trackingContentValue}
                    onChange={(e) => setTrackingContentValue(Number(e.target.value) || 0)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="currency">สกุลเงิน</Label>
                  <select
                    id="currency"
                    className="flex h-9 w-full rounded-md border border-neutral-200 bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-indigo-600"
                    value={trackingCurrency}
                    onChange={(e) => setTrackingCurrency(e.target.value)}
                  >
                    <option value="THB">THB (บาท)</option>
                    <option value="USD">USD (ดอลลาร์)</option>
                  </select>
                </div>
              </div>
              <div className="rounded-md bg-blue-50 border border-blue-200 p-3">
                <div className="flex items-start gap-2">
                  <Info className="h-4 w-4 text-blue-500 mt-0.5 shrink-0" />
                  <div className="text-xs text-blue-700 space-y-1">
                    <p className="font-medium">อีเวนต์ที่ยิงอัตโนมัติ:</p>
                    <ul className="space-y-0.5 ml-1">
                      <li><code className="bg-blue-100 px-1 rounded">PageView</code> — ยิงเมื่อเปิดหน้าเพจ</li>
                      <li><code className="bg-blue-100 px-1 rounded">ViewContent</code> — ยิงเมื่อเปิดหน้าเพจ</li>
                    </ul>
                  </div>
                </div>
              </div>
            </div>
          </Collapsible>
        </div>

        {/* Right Column - Preview */}
        <div className={`lg:w-1/3 ${mobileView === 'edit' ? 'hidden lg:block' : ''}`}>
          <div className="sticky top-8">
            <p className="text-sm font-medium text-neutral-500 mb-3">Preview</p>
            <BlockPreview blocks={blocks} ctaEventName={ctaEventName} />
          </div>
        </div>
      </div>

      {/* Published Success Dialog */}
      <Dialog open={!!publishedDialog} onOpenChange={() => setPublishedDialog(null)}>
        <DialogContent onClose={() => setPublishedDialog(null)}>
          <DialogHeader>
            <DialogTitle>เผยแพร่สำเร็จ!</DialogTitle>
          </DialogHeader>
          <div className="mt-4 space-y-3">
            <p className="text-sm text-neutral-600">หน้าเพจของคุณพร้อมใช้งานแล้วที่:</p>
            <div className="flex items-center gap-2 p-2 bg-neutral-50 rounded-md border border-neutral-200">
              <code className="text-sm text-indigo-600 flex-1 truncate">
                {publishedDialog && `${window.location.origin}/p/${publishedDialog.slug}`}
              </code>
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8"
                onClick={() => publishedDialog && copyUrl(publishedDialog.slug)}
              >
                {copiedUrl ? <Check className="h-4 w-4 text-emerald-500" /> : <Copy className="h-4 w-4" />}
              </Button>
            </div>
            <p className="text-xs text-neutral-500">
              แชร์ลิงก์นี้ใน bio link, LINE, หรือ Facebook เพื่อเริ่มเก็บข้อมูล
            </p>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => publishedDialog && window.open(`/p/${publishedDialog.slug}`, '_blank')}
            >
              <ExternalLink className="h-4 w-4" />
              เปิดหน้าเพจ
            </Button>
            <Button onClick={() => { setPublishedDialog(null); navigate('/sale-pages') }}>
              กลับไปหน้ารายการ
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
