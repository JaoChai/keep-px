import { useState, useEffect, useRef } from 'react'
import { useParams, useNavigate, Link } from 'react-router'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { ArrowLeft, Plus, X, Copy, Check, ExternalLink, Info, Upload, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { SalePagePreview } from '@/components/sale-pages/SalePagePreview'
import { useSalePages, useCreateSalePage, useUpdateSalePage } from '@/hooks/use-sale-pages'
import { usePixels } from '@/hooks/use-pixels'
import { useUploadImage, useUploadImages } from '@/hooks/use-upload'
import type { SalePageContent } from '@/types'

const CTA_EVENT_OPTIONS = [
  { value: 'Lead', label: 'Lead — ลูกค้าสนใจ ต้องการข้อมูลเพิ่ม' },
  { value: 'Purchase', label: 'Purchase — ลูกค้าซื้อสินค้า/บริการ' },
  { value: 'InitiateCheckout', label: 'InitiateCheckout — ลูกค้าเริ่มชำระเงิน' },
  { value: 'AddToCart', label: 'AddToCart — ลูกค้าเพิ่มสินค้าลงตะกร้า' },
  { value: 'CompleteRegistration', label: 'CompleteRegistration — ลูกค้าสมัครสมาชิกสำเร็จ' },
  { value: 'Schedule', label: 'Schedule — ลูกค้าจองนัดหมาย' },
  { value: 'SubmitApplication', label: 'SubmitApplication — ลูกค้าส่งใบสมัคร' },
] as const

const salePageSchema = z.object({
  name: z.string().min(1, 'Page name is required'),
  slug: z.string().min(1, 'URL slug is required').regex(/^[a-z0-9-]+$/, 'Only lowercase letters, numbers, and hyphens'),
  pixel_id: z.string().optional(),
  hero_title: z.string(),
  hero_subtitle: z.string(),
  hero_image_url: z.string(),
  description: z.string(),
  features: z.array(z.string()),
  cta_button_text: z.string(),
  cta_button_link: z.string(),
  contact_line_id: z.string(),
  contact_phone: z.string(),
  contact_website_url: z.string(),
  cta_event_name: z.string(),
  tracking_content_name: z.string(),
  tracking_content_value: z.number(),
  tracking_currency: z.string(),
})

type SalePageForm = z.infer<typeof salePageSchema>

function generateSlug(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9\s-]/g, '')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')
}

export function SalePageEditorPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const isEditing = !!id

  const { data: salePages } = useSalePages()
  const { data: pixels } = usePixels()
  const createSalePage = useCreateSalePage()
  const updateSalePage = useUpdateSalePage()

  const [features, setFeatures] = useState<string[]>([''])
  const [bodyImages, setBodyImages] = useState<string[]>([])
  const [slugTouched, setSlugTouched] = useState(false)
  const [publishedDialog, setPublishedDialog] = useState<{ slug: string } | null>(null)
  const [copiedUrl, setCopiedUrl] = useState(false)
  const uploadImage = useUploadImage()
  const uploadImages = useUploadImages()
  const heroFileRef = useRef<HTMLInputElement>(null)
  const bodyFileRef = useRef<HTMLInputElement>(null)

  const existingPage = isEditing ? salePages?.find((p) => p.id === id) : undefined

  const {
    register,
    handleSubmit,
    reset,
    watch,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<SalePageForm>({
    resolver: zodResolver(salePageSchema),
    defaultValues: {
      name: '',
      slug: '',
      pixel_id: '',
      hero_title: '',
      hero_subtitle: '',
      hero_image_url: '',
      description: '',
      features: [''],
      cta_button_text: '',
      cta_button_link: '',
      contact_line_id: '',
      contact_phone: '',
      contact_website_url: '',
      cta_event_name: 'Lead',
      tracking_content_name: '',
      tracking_content_value: 0,
      tracking_currency: 'THB',
    },
  })

  // Load existing data when editing
  useEffect(() => {
    if (existingPage) {
      const c = existingPage.content
      const featuresList = c.body.features.length > 0 ? c.body.features : ['']
      reset({
        name: existingPage.name,
        slug: existingPage.slug,
        pixel_id: existingPage.pixel_id ?? '',
        hero_title: c.hero.title,
        hero_subtitle: c.hero.subtitle,
        hero_image_url: c.hero.image_url,
        description: c.body.description,
        features: featuresList,
        cta_button_text: c.cta.button_text,
        cta_button_link: c.cta.button_link,
        contact_line_id: c.contact.line_id,
        contact_phone: c.contact.phone,
        contact_website_url: c.contact.website_url ?? '',
        cta_event_name: c.tracking?.cta_event_name || 'Lead',
        tracking_content_name: c.tracking?.content_name || '',
        tracking_content_value: c.tracking?.content_value || 0,
        tracking_currency: c.tracking?.currency || 'THB',
      })
      setFeatures(featuresList)
      setBodyImages(c.body.images ?? [])
      setSlugTouched(true)
    }
  }, [existingPage, reset])

  // Auto-generate slug from name
  const watchName = watch('name')
  useEffect(() => {
    if (!slugTouched && watchName) {
      const slug = generateSlug(watchName)
      setValue('slug', slug)
    }
  }, [watchName, slugTouched, setValue])

  // Sync features state to form
  useEffect(() => {
    setValue('features', features)
  }, [features, setValue])

  const watchedValues = watch()

  const addFeature = () => {
    if (features.length < 10) {
      setFeatures([...features, ''])
    }
  }

  const removeFeature = (index: number) => {
    setFeatures(features.filter((_, i) => i !== index))
  }

  const updateFeature = (index: number, value: string) => {
    const updated = [...features]
    updated[index] = value
    setFeatures(updated)
  }

  const handleHeroUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    const url = await uploadImage.mutateAsync(file)
    setValue('hero_image_url', url)
    if (heroFileRef.current) heroFileRef.current.value = ''
  }

  const handleBodyImageUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files
    if (!files || files.length === 0) return
    const urls = await uploadImages.mutateAsync(Array.from(files))
    setBodyImages((prev) => [...prev, ...urls])
    if (bodyFileRef.current) bodyFileRef.current.value = ''
  }

  const removeBodyImage = (index: number) => {
    setBodyImages((prev) => prev.filter((_, i) => i !== index))
  }

  const buildContent = (data: SalePageForm): SalePageContent => ({
    hero: {
      title: data.hero_title ?? '',
      subtitle: data.hero_subtitle ?? '',
      image_url: data.hero_image_url ?? '',
    },
    body: {
      description: data.description ?? '',
      features: features.filter((f) => f.trim()),
      images: bodyImages,
    },
    cta: {
      button_text: data.cta_button_text ?? '',
      button_link: data.cta_button_link ?? '',
    },
    contact: {
      line_id: data.contact_line_id ?? '',
      phone: data.contact_phone ?? '',
      website_url: data.contact_website_url ?? '',
    },
    tracking: {
      cta_event_name: data.cta_event_name || 'Lead',
      content_name: data.tracking_content_name || '',
      content_value: data.tracking_content_value || 0,
      currency: data.tracking_currency || 'THB',
    },
  })

  const onSubmit = async (data: SalePageForm, isPublished: boolean) => {
    const content = buildContent(data)
    const payload = {
      name: data.name,
      slug: data.slug,
      pixel_id: data.pixel_id ?? '',
      template_name: 'simple',
      content,
      is_published: isPublished,
    }

    if (isEditing) {
      await updateSalePage.mutateAsync({ id, ...payload })
      if (isPublished) {
        setPublishedDialog({ slug: data.slug })
      } else {
        navigate('/sale-pages')
      }
    } else {
      await createSalePage.mutateAsync(payload)
      if (isPublished) {
        setPublishedDialog({ slug: data.slug })
      } else {
        navigate('/sale-pages')
      }
    }
  }

  const copyUrl = (slug: string) => {
    navigator.clipboard.writeText(`${window.location.origin}/p/${slug}`)
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
          <Button
            variant="outline"
            disabled={isSubmitting}
            onClick={handleSubmit((data: SalePageForm) => onSubmit(data, false))}
          >
            {isSubmitting ? 'Saving...' : 'Save Draft'}
          </Button>
          <Button
            disabled={isSubmitting}
            onClick={handleSubmit((data: SalePageForm) => onSubmit(data, true))}
          >
            {isSubmitting ? 'Publishing...' : isEditing && existingPage?.is_published ? 'Update' : 'Publish'}
          </Button>
        </div>
      </div>

      {/* 2-Column Layout */}
      <div className="flex gap-8">
        {/* Left Column - Form */}
        <div className="flex-1 lg:w-2/3 space-y-6">
          {/* Basic Info */}
          <Card>
            <CardHeader>
              <CardTitle>Basic Info</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="name">Page Name</Label>
                <Input id="name" placeholder="My Sale Page" {...register('name')} />
                {errors.name && <p className="text-sm text-red-500">{errors.name.message}</p>}
              </div>
              <div className="space-y-2">
                <Label htmlFor="slug">URL Slug</Label>
                <div className="flex">
                  <span className="inline-flex items-center px-3 rounded-l-md border border-r-0 border-neutral-200 bg-neutral-50 text-sm text-neutral-500">
                    /p/
                  </span>
                  <Input
                    id="slug"
                    className="rounded-l-none"
                    placeholder="my-sale-page"
                    {...register('slug', {
                      onChange: () => setSlugTouched(true),
                    })}
                  />
                </div>
                {errors.slug && <p className="text-sm text-red-500">{errors.slug.message}</p>}
              </div>
              <div className="space-y-2">
                <Label htmlFor="pixel_id">Pixel (optional)</Label>
                <select
                  id="pixel_id"
                  className="flex h-9 w-full rounded-md border border-neutral-200 bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-indigo-600"
                  {...register('pixel_id')}
                >
                  <option value="">No pixel</option>
                  {pixels?.map((pixel) => (
                    <option key={pixel.id} value={pixel.id}>
                      {pixel.name} ({pixel.fb_pixel_id})
                    </option>
                  ))}
                </select>
              </div>
            </CardContent>
          </Card>

          {/* Hero Section */}
          <Card>
            <CardHeader>
              <CardTitle>Hero Section</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="hero_title">Title</Label>
                <Input id="hero_title" placeholder="Your Amazing Product" {...register('hero_title')} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="hero_subtitle">Subtitle</Label>
                <Input id="hero_subtitle" placeholder="A short description..." {...register('hero_subtitle')} />
              </div>
              <div className="space-y-2">
                <Label>Hero Image</Label>
                {watchedValues.hero_image_url && (
                  <div className="relative w-full max-w-[200px]">
                    <img src={watchedValues.hero_image_url} alt="" className="w-full h-auto rounded-lg border border-neutral-200" />
                    <button
                      type="button"
                      className="absolute -top-2 -right-2 h-5 w-5 rounded-full bg-red-500 text-white flex items-center justify-center text-xs hover:bg-red-600"
                      onClick={() => setValue('hero_image_url', '')}
                    >
                      <X className="h-3 w-3" />
                    </button>
                  </div>
                )}
                <div className="flex gap-2">
                  <input ref={heroFileRef} type="file" accept="image/*" className="hidden" onChange={handleHeroUpload} />
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    disabled={uploadImage.isPending}
                    onClick={() => heroFileRef.current?.click()}
                  >
                    {uploadImage.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Upload className="h-3.5 w-3.5" />}
                    อัพโหลดรูป
                  </Button>
                </div>
                <Input id="hero_image_url" placeholder="หรือวาง URL รูปภาพ" {...register('hero_image_url')} className="text-xs" />
              </div>
            </CardContent>
          </Card>

          {/* Description */}
          <Card>
            <CardHeader>
              <CardTitle>Description</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  placeholder="Describe your product or service..."
                  rows={4}
                  {...register('description')}
                />
              </div>
              <div className="space-y-2">
                <Label>Features</Label>
                <div className="space-y-2">
                  {features.map((feature, index) => (
                    <div key={index} className="flex items-center gap-2">
                      <Input
                        placeholder={`Feature ${index + 1}`}
                        value={feature}
                        onChange={(e) => updateFeature(index, e.target.value)}
                      />
                      {features.length > 1 && (
                        <Button
                          variant="ghost"
                          size="icon"
                          type="button"
                          onClick={() => removeFeature(index)}
                        >
                          <X className="h-4 w-4 text-neutral-400" />
                        </Button>
                      )}
                    </div>
                  ))}
                </div>
                {features.length < 10 && (
                  <Button variant="outline" size="sm" type="button" onClick={addFeature}>
                    <Plus className="h-3.5 w-3.5" />
                    Add Feature
                  </Button>
                )}
              </div>
              <div className="space-y-2">
                <Label>รูปภาพเนื้อหา</Label>
                {bodyImages.length > 0 && (
                  <div className="grid grid-cols-3 gap-2">
                    {bodyImages.map((url, index) => (
                      <div key={index} className="relative group">
                        <img src={url} alt="" className="w-full h-24 object-cover rounded-lg border border-neutral-200" />
                        <button
                          type="button"
                          className="absolute -top-2 -right-2 h-5 w-5 rounded-full bg-red-500 text-white flex items-center justify-center text-xs opacity-0 group-hover:opacity-100 transition-opacity hover:bg-red-600"
                          onClick={() => removeBodyImage(index)}
                        >
                          <X className="h-3 w-3" />
                        </button>
                      </div>
                    ))}
                  </div>
                )}
                <div>
                  <input ref={bodyFileRef} type="file" accept="image/*" multiple className="hidden" onChange={handleBodyImageUpload} />
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    disabled={uploadImages.isPending}
                    onClick={() => bodyFileRef.current?.click()}
                  >
                    {uploadImages.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Upload className="h-3.5 w-3.5" />}
                    เพิ่มรูปภาพ
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Call to Action */}
          <Card>
            <CardHeader>
              <CardTitle>Call to Action</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="cta_button_text">Button Text</Label>
                <Input id="cta_button_text" placeholder="Buy Now" {...register('cta_button_text')} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="cta_button_link">Button Link</Label>
                <Input id="cta_button_link" placeholder="https://example.com/buy" {...register('cta_button_link')} />
              </div>
            </CardContent>
          </Card>

          {/* Tracking Settings */}
          <Card>
            <CardHeader>
              <CardTitle>ตั้งค่าการติดตาม</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="cta_event_name">เมื่อกดปุ่ม CTA ให้ยิงอีเวนต์</Label>
                <select
                  id="cta_event_name"
                  className="flex h-9 w-full rounded-md border border-neutral-200 bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-indigo-600"
                  {...register('cta_event_name')}
                >
                  {CTA_EVENT_OPTIONS.map((opt) => (
                    <option key={opt.value} value={opt.value}>
                      {opt.label}
                    </option>
                  ))}
                </select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="tracking_content_name">ชื่อสินค้า (ไม่บังคับ)</Label>
                <Input
                  id="tracking_content_name"
                  placeholder="เช่น ครีมหน้าใส, คอร์สออนไลน์"
                  {...register('tracking_content_name')}
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-2">
                  <Label htmlFor="tracking_content_value">ราคาสินค้า (ไม่บังคับ)</Label>
                  <Input
                    id="tracking_content_value"
                    type="number"
                    placeholder="0"
                    {...register('tracking_content_value', { valueAsNumber: true })}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="tracking_currency">สกุลเงิน</Label>
                  <select
                    id="tracking_currency"
                    className="flex h-9 w-full rounded-md border border-neutral-200 bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-indigo-600"
                    {...register('tracking_currency')}
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
                      <li><code className="bg-blue-100 px-1 rounded">Contact</code> — ยิงเมื่อกด LINE หรือเบอร์โทร</li>
                    </ul>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Contact Info */}
          <Card>
            <CardHeader>
              <CardTitle>Contact Info</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="contact_line_id">LINE ID</Label>
                <Input id="contact_line_id" placeholder="@yourlineid" {...register('contact_line_id')} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="contact_phone">Phone Number</Label>
                <Input id="contact_phone" placeholder="08x-xxx-xxxx" {...register('contact_phone')} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="contact_website_url">Website URL</Label>
                <Input
                  id="contact_website_url"
                  placeholder="https://yourwebsite.com"
                  {...register('contact_website_url')}
                />
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Right Column - Preview */}
        <div className="hidden lg:block lg:w-1/3">
          <div className="sticky top-8">
            <p className="text-sm font-medium text-neutral-500 mb-3">Preview</p>
            <SalePagePreview
              hero={{
                title: watchedValues.hero_title ?? '',
                subtitle: watchedValues.hero_subtitle ?? '',
                image_url: watchedValues.hero_image_url ?? '',
              }}
              body={{
                description: watchedValues.description ?? '',
                features: features,
                images: bodyImages,
              }}
              cta={{
                button_text: watchedValues.cta_button_text ?? '',
                button_link: watchedValues.cta_button_link ?? '',
              }}
              contact={{
                line_id: watchedValues.contact_line_id ?? '',
                phone: watchedValues.contact_phone ?? '',
                website_url: watchedValues.contact_website_url ?? '',
              }}
              ctaEventName={watchedValues.cta_event_name || 'Lead'}
            />
          </div>
        </div>
      </div>

      {/* Published Success Dialog */}
      <Dialog open={!!publishedDialog} onOpenChange={() => setPublishedDialog(null)}>
        <DialogContent onClose={() => setPublishedDialog(null)}>
          <DialogHeader>
            <DialogTitle>Sale Page Published!</DialogTitle>
          </DialogHeader>
          <div className="mt-4 space-y-3">
            <p className="text-sm text-neutral-600">Your page is live at:</p>
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
              Share this link on your bio link, LINE, or Facebook to start tracking.
            </p>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => publishedDialog && window.open(`/p/${publishedDialog.slug}`, '_blank')}
            >
              <ExternalLink className="h-4 w-4" />
              Open Page
            </Button>
            <Button onClick={() => { setPublishedDialog(null); navigate('/sale-pages') }}>
              Back to Sale Pages
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
