import { useState, useEffect, useMemo } from 'react'
import { useParams, useNavigate, Link } from 'react-router'
import { ArrowLeft, Copy, Check, ExternalLink, Info, Eye, Pencil, ChevronDown } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Collapsible } from '@/components/ui/collapsible'
import { BlockEditor } from '@/components/sale-pages/BlockEditor'
import { BlockPreview } from '@/components/sale-pages/BlockPreview'
import { StyleEditor } from '@/components/sale-pages/StyleEditor'
import { UnsavedChangesDialog } from '@/components/shared/UnsavedChangesDialog'
import { useSalePages, useCreateSalePage, useUpdateSalePage } from '@/hooks/use-sale-pages'
import { usePixels } from '@/hooks/use-pixels'
import { useUnsavedChanges } from '@/hooks/use-unsaved-changes'
import { useAutoSaveDraft, loadDraft } from '@/hooks/use-auto-save-draft'
import { CTA_EVENT_OPTIONS } from '@/lib/utils'
import { toast } from 'sonner'
import type { Block, SalePage, SalePageContentV2, PageStyle } from '@/types'


// Wrapper: handles data loading and v1→v2 redirect
export function BlockEditorPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const isEditing = !!id
  const { data: salePages, isLoading } = useSalePages()

  if (isEditing) {
    if (isLoading) {
      return <div className="text-center py-12 text-muted-foreground">Loading...</div>
    }
    const page = salePages?.find((p) => p.id === id)
    if (!page) {
      return <div className="text-center py-12 text-muted-foreground">Page not found</div>
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
  const [showCustomSlug, setShowCustomSlug] = useState(isEditing)
  const [selectedPixelIds, setSelectedPixelIds] = useState<string[]>(existingPage?.pixel_ids ?? [])

  // Page style
  const [pageStyle, setPageStyle] = useState<PageStyle>(v2?.style ?? {})

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
  const [hasChanges, setHasChanges] = useState(false)
  const unsaved = useUnsavedChanges(hasChanges)

  const draftKey = isEditing && existingPage ? `sale-page:${existingPage.id}` : 'sale-page:new'

  // Auto-save draft
  const draftData = useMemo(
    () => ({ name, slug, selectedPixelIds, blocks, pageStyle, ctaEventName, trackingContentName, trackingContentValue, trackingCurrency }),
    [name, slug, selectedPixelIds, blocks, pageStyle, ctaEventName, trackingContentName, trackingContentValue, trackingCurrency]
  )
  const { clearDraft } = useAutoSaveDraft(draftKey, draftData)

  // Restore draft
  useEffect(() => {
    const draft = loadDraft<typeof draftData>(draftKey)
    if (!draft) return
    // For editing: only restore if draft is different from server data
    if (isEditing && existingPage) {
      if (JSON.stringify(draft.blocks) === JSON.stringify(v2?.blocks)) return
    }
    setName(draft.name ?? '')
    setSlug(draft.slug ?? '')
    setSelectedPixelIds(draft.selectedPixelIds ?? [])
    setBlocks(draft.blocks ?? [])
    setPageStyle(draft.pageStyle ?? {})
    setCtaEventName(draft.ctaEventName ?? 'Lead')
    setTrackingContentName(draft.trackingContentName ?? '')
    setTrackingContentValue(draft.trackingContentValue ?? 0)
    setTrackingCurrency(draft.trackingCurrency ?? 'THB')
    toast.info('กู้คืนแบบร่างจากการบันทึกอัตโนมัติ')
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const buildContent = (): SalePageContentV2 => ({
    version: 2,
    blocks,
    tracking: {
      cta_event_name: ctaEventName || 'Lead',
      content_name: trackingContentName || '',
      content_value: trackingContentValue || 0,
      currency: trackingCurrency || 'THB',
    },
    style: pageStyle,
  })

  const isSubmitting = createSalePage.isPending || updateSalePage.isPending

  const onSubmit = async (isPublished: boolean) => {
    setSubmitError(null)
    if (!name.trim()) {
      setSubmitError('กรุณากรอกชื่อหน้าเพจ')
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
        slug: slug || undefined,
        pixel_ids: selectedPixelIds,
        template_name: 'blocks',
        content,
        is_published: isPublished,
      }

      let resultSlug = slug
      if (isEditing && existingPage) {
        const result = await updateSalePage.mutateAsync({ id: existingPage.id, ...payload })
        resultSlug = result.slug
      } else {
        const result = await createSalePage.mutateAsync(payload)
        resultSlug = result.slug
      }

      clearDraft()
      setHasChanges(false)

      if (isPublished) {
        setPublishedDialog({ slug: resultSlug })
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
          className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
        >
          <ArrowLeft className="h-4 w-4" />
          Sale Pages
        </Link>
        <div className="flex items-center gap-2">
          {/* Mobile toggle */}
          <div className="flex lg:hidden border border-border rounded-md overflow-hidden">
            <button
              type="button"
              className={`px-3 py-1.5 text-xs font-medium ${mobileView === 'edit' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground'}`}
              onClick={() => setMobileView('edit')}
            >
              <Pencil className="h-3 w-3 inline mr-1" />
              แก้ไข
            </button>
            <button
              type="button"
              className={`px-3 py-1.5 text-xs font-medium ${mobileView === 'preview' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground'}`}
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
                  onChange={(e) => { setName(e.target.value); setHasChanges(true) }}
                />
              </div>
              <div className="space-y-2">
                <button
                  type="button"
                  className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
                  onClick={() => setShowCustomSlug(!showCustomSlug)}
                >
                  <ChevronDown className={`h-3.5 w-3.5 transition-transform ${showCustomSlug ? 'rotate-0' : '-rotate-90'}`} />
                  ตั้ง URL เอง (ไม่บังคับ)
                </button>
                {showCustomSlug && (
                  <div className="flex">
                    <span className="inline-flex items-center px-3 rounded-l-md border border-r-0 border-border bg-muted text-sm text-muted-foreground">
                      /p/
                    </span>
                    <Input
                      id="page-slug"
                      className="rounded-l-none"
                      placeholder="my-page"
                      value={slug}
                      onChange={(e) => { setSlug(e.target.value); setHasChanges(true) }}
                    />
                  </div>
                )}
                {!showCustomSlug && (
                  <p className="text-xs text-muted-foreground">ระบบจะสร้าง URL ให้อัตโนมัติ</p>
                )}
              </div>
              <div className="space-y-2">
                <Label>Pixels (ไม่บังคับ)</Label>
                <div className="max-h-40 overflow-y-auto border border-border rounded-md p-2 space-y-1">
                  {(!pixels || pixels.length === 0) && (
                    <p className="text-xs text-muted-foreground">No pixels available</p>
                  )}
                  {pixels?.map((pixel) => (
                    <label key={pixel.id} className="flex items-center gap-2 text-sm py-1 px-1 rounded hover:bg-accent cursor-pointer">
                      <input
                        type="checkbox"
                        checked={selectedPixelIds.includes(pixel.id)}
                        onChange={(e) => {
                          if (e.target.checked) {
                            setSelectedPixelIds(prev => [...prev, pixel.id])
                          } else {
                            setSelectedPixelIds(prev => prev.filter(id => id !== pixel.id))
                          }
                          setHasChanges(true)
                        }}
                        className="rounded border-border"
                      />
                      {pixel.name} ({pixel.fb_pixel_id})
                    </label>
                  ))}
                </div>
              </div>
            </div>
          </Collapsible>

          {/* Block Editor */}
          <BlockEditor blocks={blocks} onChange={(b) => { setBlocks(b); setHasChanges(true) }} />

          {/* Page Style */}
          <Collapsible title="รูปแบบหน้าเพจ">
            <StyleEditor style={pageStyle} onChange={(s) => { setPageStyle(s); setHasChanges(true) }} />
          </Collapsible>

          {/* Tracking Settings */}
          <Collapsible title="ตั้งค่าการติดตาม">
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="cta-event">เมื่อกดปุ่ม CTA ให้ยิงอีเวนต์</Label>
                <select
                  id="cta-event"
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                  value={ctaEventName}
                  onChange={(e) => { setCtaEventName(e.target.value); setHasChanges(true) }}
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
                  onChange={(e) => { setTrackingContentName(e.target.value); setHasChanges(true) }}
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
                    onChange={(e) => { setTrackingContentValue(Number(e.target.value) || 0); setHasChanges(true) }}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="currency">สกุลเงิน</Label>
                  <select
                    id="currency"
                    className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                    value={trackingCurrency}
                    onChange={(e) => { setTrackingCurrency(e.target.value); setHasChanges(true) }}
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
            <p className="text-sm font-medium text-muted-foreground mb-3">Preview</p>
            <BlockPreview blocks={blocks} ctaEventName={ctaEventName} style={pageStyle} />
          </div>
        </div>
      </div>

      <UnsavedChangesDialog isBlocked={unsaved.isBlocked} onStay={unsaved.cancelLeave} onLeave={unsaved.confirmLeave} />

      {/* Published Success Dialog */}
      <Dialog open={!!publishedDialog} onOpenChange={() => setPublishedDialog(null)}>
        <DialogContent onClose={() => setPublishedDialog(null)}>
          <DialogHeader>
            <DialogTitle>เผยแพร่สำเร็จ!</DialogTitle>
          </DialogHeader>
          <div className="mt-4 space-y-3">
            <p className="text-sm text-muted-foreground">หน้าเพจของคุณพร้อมใช้งานแล้วที่:</p>
            <div className="flex items-center gap-2 p-2 bg-muted rounded-md border border-border">
              <code className="text-sm text-foreground flex-1 truncate">
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
            <p className="text-xs text-muted-foreground">
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
