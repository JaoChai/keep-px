import { useRef, useState } from 'react'
import { ChevronUp, ChevronDown, Trash2, Image, Type, MessageCircle, Globe, Link, Upload, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { useUploadImage, useUploadImages } from '@/hooks/use-upload'
import type { Block, ButtonStyle } from '@/types'

interface BlockEditorProps {
  blocks: Block[]
  onChange: (blocks: Block[]) => void
}

export function BlockEditor({ blocks, onChange }: BlockEditorProps) {
  const uploadImage = useUploadImage()
  const uploadImages = useUploadImages()
  const imageFileRef = useRef<HTMLInputElement>(null)
  const [deleteBlockConfirm, setDeleteBlockConfirm] = useState<number | null>(null)

  const addBlock = (block: Block) => onChange([...blocks, block])

  const updateBlock = (index: number, updates: Partial<Block>) => {
    const updated = blocks.map((b, i) => i === index ? { ...b, ...updates } as Block : b)
    onChange(updated)
  }

  const removeBlock = (index: number) => onChange(blocks.filter((_, i) => i !== index))

  const moveBlock = (index: number, direction: -1 | 1) => {
    const newIndex = index + direction
    if (newIndex < 0 || newIndex >= blocks.length) return
    const updated = blocks.map((b, i) => {
      if (i === index) return blocks[newIndex]!
      if (i === newIndex) return blocks[index]!
      return b
    })
    onChange(updated)
  }

  const handleImageUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files
    if (!files || files.length === 0) return
    const urls = await uploadImages.mutateAsync(Array.from(files))
    const newBlocks: Block[] = urls.map(url => ({
      id: crypto.randomUUID(),
      type: 'image' as const,
      image_url: url,
    }))
    onChange([...blocks, ...newBlocks])
    if (imageFileRef.current) imageFileRef.current.value = ''
  }

  const handleSingleImageUpload = async (index: number, e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    const url = await uploadImage.mutateAsync(file)
    updateBlock(index, { image_url: url })
  }

  const addTextBlock = () => addBlock({ id: crypto.randomUUID(), type: 'text', text: '' })
  const addButtonBlock = (style: ButtonStyle) => addBlock({
    id: crypto.randomUUID(),
    type: 'button',
    button_style: style,
    button_text: style === 'line' ? 'แอดไลน์' : style === 'website' ? 'เยี่ยมชมเว็บไซต์' : 'คลิกที่นี่',
    button_url: '',
    button_value: '',
  })

  const getTypeBadge = (type: string) => {
    switch (type) {
      case 'image': return <Badge variant="secondary">รูปภาพ</Badge>
      case 'text': return <Badge variant="secondary">ข้อความ</Badge>
      case 'button': return <Badge variant="secondary">ปุ่ม</Badge>
      default: return null
    }
  }

  return (
    <div className="space-y-3">
      {blocks.map((block, index) => (
        <div key={block.id} className="border border-border rounded-lg p-4 bg-card">
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-2">
              {getTypeBadge(block.type)}
            </div>
            <div className="flex items-center gap-1">
              <Button variant="ghost" size="icon" className="h-7 w-7" disabled={index === 0} onClick={() => moveBlock(index, -1)}>
                <ChevronUp className="h-4 w-4" />
              </Button>
              <Button variant="ghost" size="icon" className="h-7 w-7" disabled={index === blocks.length - 1} onClick={() => moveBlock(index, 1)}>
                <ChevronDown className="h-4 w-4" />
              </Button>
              <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => setDeleteBlockConfirm(index)}>
                <Trash2 className="h-4 w-4 text-red-500" />
              </Button>
            </div>
          </div>

          {/* Type-specific fields */}
          {block.type === 'image' && (
            <div className="space-y-2">
              {block.image_url ? (
                <img src={block.image_url} alt="" className="w-full max-h-48 object-cover rounded-md border border-border" />
              ) : (
                <div className="w-full h-32 bg-secondary rounded-md flex items-center justify-center text-muted-foreground text-sm">
                  ยังไม่มีรูป
                </div>
              )}
              <div className="flex gap-2">
                <input
                  type="file"
                  accept="image/*"
                  className="hidden"
                  id={`img-upload-${block.id}`}
                  onChange={(e) => handleSingleImageUpload(index, e)}
                />
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={uploadImage.isPending}
                  onClick={() => document.getElementById(`img-upload-${block.id}`)?.click()}
                >
                  {uploadImage.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Upload className="h-3.5 w-3.5" />}
                  {uploadImage.isPending && uploadImage.progress > 0 ? `${uploadImage.progress}%` : block.image_url ? 'เปลี่ยนรูป' : 'อัพโหลดรูป'}
                </Button>
              </div>
              <Input
                placeholder="หรือวาง URL รูปภาพ"
                value={block.image_url || ''}
                onChange={(e) => updateBlock(index, { image_url: e.target.value })}
                className="text-xs"
              />
              <Input
                placeholder="ลิงก์เมื่อกดรูป (ไม่บังคับ) เช่น https://line.me/ti/p/~@shop"
                value={block.link_url || ''}
                onChange={(e) => updateBlock(index, { link_url: e.target.value })}
                className="text-xs"
              />
            </div>
          )}

          {block.type === 'text' && (
            <Textarea
              placeholder="พิมพ์ข้อความ..."
              value={block.text || ''}
              onChange={(e) => updateBlock(index, { text: e.target.value })}
              rows={3}
            />
          )}

          {block.type === 'button' && (
            <div className="space-y-3">
              <div className="space-y-1.5">
                <Label className="text-xs">ประเภทปุ่ม</Label>
                <select
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                  value={block.button_style || 'line'}
                  onChange={(e) => updateBlock(index, { button_style: e.target.value as ButtonStyle })}
                >
                  <option value="line">LINE</option>
                  <option value="website">เว็บไซต์</option>
                  <option value="custom">ลิงก์ทั่วไป</option>
                </select>
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs">ข้อความบนปุ่ม</Label>
                <Input
                  placeholder="เช่น แอดไลน์, ซื้อเลย"
                  value={block.button_text || ''}
                  onChange={(e) => updateBlock(index, { button_text: e.target.value })}
                />
              </div>
              {block.button_style === 'line' ? (
                <div className="space-y-1.5">
                  <Label className="text-xs">LINE ID</Label>
                  <Input
                    placeholder="@yourlineid"
                    value={block.button_value || ''}
                    onChange={(e) => updateBlock(index, { button_value: e.target.value })}
                  />
                </div>
              ) : (
                <div className="space-y-1.5">
                  <Label className="text-xs">URL</Label>
                  <Input
                    placeholder="https://..."
                    value={block.button_url || ''}
                    onChange={(e) => updateBlock(index, { button_url: e.target.value })}
                  />
                </div>
              )}
            </div>
          )}
        </div>
      ))}

      {/* Add Block Buttons */}
      <div className="border-2 border-dashed border-border rounded-lg p-4">
        <p className="text-xs font-medium text-muted-foreground mb-3 text-center">เพิ่มบล็อก</p>
        <div className="flex flex-wrap justify-center gap-2">
          <input ref={imageFileRef} type="file" accept="image/*" multiple className="hidden" onChange={handleImageUpload} />
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={uploadImages.isPending}
            onClick={() => imageFileRef.current?.click()}
          >
            {uploadImages.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Image className="h-3.5 w-3.5" />}
            {uploadImages.isPending && uploadImages.progress > 0 ? `${uploadImages.progress}%` : 'รูปภาพ'}
          </Button>
          <Button type="button" variant="outline" size="sm" onClick={addTextBlock}>
            <Type className="h-3.5 w-3.5" />
            ข้อความ
          </Button>
          <Button type="button" variant="outline" size="sm" onClick={() => addButtonBlock('line')}>
            <MessageCircle className="h-3.5 w-3.5" />
            LINE
          </Button>
          <Button type="button" variant="outline" size="sm" onClick={() => addButtonBlock('website')}>
            <Globe className="h-3.5 w-3.5" />
            เว็บไซต์
          </Button>
          <Button type="button" variant="outline" size="sm" onClick={() => addButtonBlock('custom')}>
            <Link className="h-3.5 w-3.5" />
            ลิงก์
          </Button>
        </div>
      </div>

      {/* Block Delete Confirmation */}
      <Dialog open={deleteBlockConfirm !== null} onOpenChange={() => setDeleteBlockConfirm(null)}>
        <DialogContent onClose={() => setDeleteBlockConfirm(null)}>
          <DialogHeader>
            <DialogTitle>ลบบล็อก</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground mt-2">
            คุณแน่ใจหรือไม่ว่าต้องการลบบล็อกนี้?
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteBlockConfirm(null)}>ยกเลิก</Button>
            <Button variant="destructive" onClick={() => { if (deleteBlockConfirm !== null) { removeBlock(deleteBlockConfirm); setDeleteBlockConfirm(null) } }}>ลบ</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
