import { useState, useEffect, useMemo } from 'react'
import { useParams, useNavigate, Link } from 'react-router'
import { ArrowLeft, Copy, Check, ExternalLink, Eye, Pencil, ChevronDown, X, Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Collapsible } from '@/components/ui/collapsible'
import { Popover, PopoverTrigger, PopoverContent } from '@/components/ui/popover'
import { BlockEditor } from '@/components/sale-pages/BlockEditor'
import { BlockPreview } from '@/components/sale-pages/BlockPreview'
import { StyleEditor } from '@/components/sale-pages/StyleEditor'
import { TemplateSelector } from '@/components/sale-pages/TemplateSelector'
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
  const [templateSelected, setTemplateSelected] = useState(false)
  const unsaved = useUnsavedChanges(hasChanges)
  const showTemplateSelector = !isEditing && blocks.length === 0 && !templateSelected

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
      setSubmitError('กรุณาเพิ่มเนื้อหาอย่างน้อย 1 อัน')
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
      unsaved.allowNavigation()

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

  const handleTemplateSelect = (templateBlocks: Block[]) => {
    setBlocks(templateBlocks)
    setTemplateSelected(true)
    if (templateBlocks.length > 0) setHasChanges(true)
  }

  const removePixel = (pixelId: string) => {
    setSelectedPixelIds(prev => prev.filter(id => id !== pixelId))
    setHasChanges(true)
  }

  const addPixel = (pixelId: string) => {
    setSelectedPixelIds(prev => [...prev, pixelId])
    setHasChanges(true)
  }

  const { selectedPixels, unselectedPixels } = useMemo(() => {
    if (!pixels) return { selectedPixels: [], unselectedPixels: [] }
    return {
      selectedPixels: pixels.filter(p => selectedPixelIds.includes(p.id)),
      unselectedPixels: pixels.filter(p => !selectedPixelIds.includes(p.id)),
    }
  }, [pixels, selectedPixelIds])

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

          {/* Section 1: Page Name */}
          <div className="border border-border rounded-lg p-4 bg-card space-y-3">
            <div className="space-y-2">
              <Label htmlFor="page-name">ชื่อเพจ</Label>
              <Input
                id="page-name"
                placeholder="เช่น โปรโมชั่นครีมหน้าใส"
                value={name}
                onChange={(e) => { setName(e.target.value); setHasChanges(true) }}
              />
            </div>
            <div>
              <button
                type="button"
                className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
                onClick={() => setShowCustomSlug(!showCustomSlug)}
              >
                <ChevronDown className={`h-3 w-3 transition-transform ${showCustomSlug ? 'rotate-0' : '-rotate-90'}`} />
                ตั้ง URL เอง
              </button>
              {showCustomSlug && (
                <div className="flex mt-2">
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
            </div>
          </div>

          {/* Section 2: Pixel Selection (Chips) */}
          <div className="border border-border rounded-lg p-4 bg-card space-y-2">
            <Label>Pixel ที่ใช้เก็บข้อมูล</Label>
            <div className="flex flex-wrap items-center gap-2">
              {selectedPixels.map((pixel) => (
                <Badge key={pixel.id} variant="default" className="gap-1 pr-1">
                  {pixel.name}
                  <button
                    type="button"
                    className="ml-0.5 rounded-full hover:bg-primary-foreground/20 p-0.5"
                    onClick={() => removePixel(pixel.id)}
                  >
                    <X className="h-3 w-3" />
                  </button>
                </Badge>
              ))}
              {(!pixels || pixels.length === 0) ? (
                <Link to="/pixels" className="text-xs text-primary hover:underline">
                  ยังไม่มี Pixel — สร้าง Pixel
                </Link>
              ) : unselectedPixels.length > 0 ? (
                <Popover>
                  <PopoverTrigger className="inline-flex items-center gap-1 px-2.5 py-1 text-xs font-medium border border-dashed border-border rounded-md text-muted-foreground hover:text-foreground hover:border-foreground transition-colors">
                    <Plus className="h-3 w-3" />
                    เพิ่ม
                  </PopoverTrigger>
                  <PopoverContent align="start" className="p-2 w-64">
                    <div className="space-y-0.5">
                      {unselectedPixels.map((pixel) => (
                        <button
                          key={pixel.id}
                          type="button"
                          className="w-full text-left px-3 py-2 text-sm rounded-md hover:bg-accent transition-colors"
                          onClick={() => addPixel(pixel.id)}
                        >
                          <span className="font-medium">{pixel.name}</span>
                          <span className="text-muted-foreground ml-1.5 text-xs">{pixel.fb_pixel_id}</span>
                        </button>
                      ))}
                    </div>
                  </PopoverContent>
                </Popover>
              ) : selectedPixels.length === 0 ? (
                <p className="text-xs text-muted-foreground">เลือก Pixel เพื่อเก็บข้อมูลลูกค้า</p>
              ) : null}
            </div>
          </div>

          {/* Section 3: Tracking (always visible, compact) */}
          <div className="border border-border rounded-lg p-4 bg-card space-y-3">
            <Label>เมื่อลูกค้ากดปุ่ม</Label>
            <div className="grid grid-cols-1 sm:grid-cols-4 gap-3">
              <div className="sm:col-span-1">
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
              <div className="sm:col-span-1">
                <Input
                  id="content-name"
                  placeholder="ชื่อสินค้า"
                  value={trackingContentName}
                  onChange={(e) => { setTrackingContentName(e.target.value); setHasChanges(true) }}
                />
              </div>
              <div className="sm:col-span-1">
                <Input
                  id="content-value"
                  type="number"
                  placeholder="ราคา"
                  value={trackingContentValue || ''}
                  onChange={(e) => { setTrackingContentValue(Number(e.target.value) || 0); setHasChanges(true) }}
                />
              </div>
              <div className="sm:col-span-1">
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
            <p className="text-[11px] text-muted-foreground">
              <code className="bg-muted px-1 rounded text-[10px]">PageView</code> + <code className="bg-muted px-1 rounded text-[10px]">ViewContent</code> ยิงอัตโนมัติเมื่อเปิดเพจ
            </p>
          </div>

          {/* Section 4: Template Selector or Block Editor */}
          {showTemplateSelector ? (
            <TemplateSelector onSelect={handleTemplateSelect} />
          ) : (
            <BlockEditor blocks={blocks} onChange={(b) => { setBlocks(b); setHasChanges(true) }} />
          )}

          {/* Section 5: Page Style (Collapsible — secondary) */}
          <Collapsible title="รูปแบบหน้าเพจ">
            <StyleEditor style={pageStyle} onChange={(s) => { setPageStyle(s); setHasChanges(true) }} />
          </Collapsible>
        </div>

        {/* Right Column - Preview */}
        <div className={`lg:w-1/3 ${mobileView === 'edit' ? 'hidden lg:block' : ''}`}>
          <div className="sticky top-8">
            <p className="text-sm font-medium text-muted-foreground mb-3">Preview</p>
            <BlockPreview
              blocks={blocks}
              ctaEventName={ctaEventName}
              style={pageStyle}
              emptyText={showTemplateSelector ? 'เลือกเทมเพลตด้านซ้ายเพื่อเริ่มต้น' : undefined}
            />
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
