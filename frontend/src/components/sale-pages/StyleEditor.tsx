import { useRef } from 'react'
import { X, Upload, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { useUploadImage } from '@/hooks/use-upload'
import type { PageStyle, PresetTheme } from '@/types'

const PRESET_THEMES: PresetTheme[] = [
  { name: 'Default Indigo', style: { bg_color: '#f8f9fa', accent_color: '#667eea', text_color: '#1a1a2e' } },
  { name: 'Rose', style: { bg_color: '#fff1f2', accent_color: '#e11d48', text_color: '#1a1a2e' } },
  { name: 'Emerald', style: { bg_color: '#f0fdf4', accent_color: '#059669', text_color: '#1a1a2e' } },
  { name: 'Amber', style: { bg_color: '#fffbeb', accent_color: '#d97706', text_color: '#1a1a2e' } },
  { name: 'Slate', style: { bg_color: '#f1f5f9', accent_color: '#475569', text_color: '#1e293b' } },
  { name: 'Dark', style: { bg_color: '#1a1a2e', accent_color: '#667eea', text_color: '#f8f9fa' } },
]

interface StyleEditorProps {
  style: PageStyle
  onChange: (style: PageStyle) => void
}

export function StyleEditor({ style, onChange }: StyleEditorProps) {
  const uploadImage = useUploadImage()
  const fileRef = useRef<HTMLInputElement>(null)

  const update = (patch: Partial<PageStyle>) => {
    onChange({ ...style, ...patch })
  }

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    const url = await uploadImage.mutateAsync(file)
    update({ bg_image_url: url })
    if (fileRef.current) fileRef.current.value = ''
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>รูปแบบหน้าเพจ</CardTitle>
      </CardHeader>
      <CardContent className="space-y-5">
        {/* Preset Themes */}
        <div className="space-y-2">
          <p className="text-sm font-medium text-foreground">ธีมสำเร็จรูป</p>
          <div className="flex flex-wrap gap-2">
            {PRESET_THEMES.map((theme) => {
              const isActive =
                style.bg_color === theme.style.bg_color &&
                style.accent_color === theme.style.accent_color &&
                style.text_color === theme.style.text_color
              return (
                <button
                  key={theme.name}
                  type="button"
                  title={theme.name}
                  className={`relative h-8 w-8 rounded-full border-2 transition-all ${isActive ? 'border-primary ring-2 ring-ring/20' : 'border-border hover:border-muted-foreground'}`}
                  style={{ background: `linear-gradient(135deg, ${theme.style.bg_color} 50%, ${theme.style.accent_color} 50%)` }}
                  onClick={() => onChange({ ...style, bg_color: theme.style.bg_color, accent_color: theme.style.accent_color, text_color: theme.style.text_color })}
                />
              )
            })}
          </div>
        </div>

        {/* Color Pickers */}
        <div className="space-y-3">
          <div className="flex items-center gap-3">
            <input
              type="color"
              value={style.bg_color || '#f8f9fa'}
              onChange={(e) => update({ bg_color: e.target.value })}
              className="h-8 w-8 rounded border border-border cursor-pointer p-0"
            />
            <div className="flex-1 space-y-1">
              <label className="text-xs font-medium text-muted-foreground">สีพื้นหลัง</label>
              <Input
                value={style.bg_color || ''}
                placeholder="#f8f9fa"
                onChange={(e) => update({ bg_color: e.target.value })}
                className="h-7 text-xs"
              />
            </div>
          </div>
          <div className="flex items-center gap-3">
            <input
              type="color"
              value={style.accent_color || '#667eea'}
              onChange={(e) => update({ accent_color: e.target.value })}
              className="h-8 w-8 rounded border border-border cursor-pointer p-0"
            />
            <div className="flex-1 space-y-1">
              <label className="text-xs font-medium text-muted-foreground">สีปุ่ม/Accent</label>
              <Input
                value={style.accent_color || ''}
                placeholder="#667eea"
                onChange={(e) => update({ accent_color: e.target.value })}
                className="h-7 text-xs"
              />
            </div>
          </div>
          <div className="flex items-center gap-3">
            <input
              type="color"
              value={style.text_color || '#1a1a2e'}
              onChange={(e) => update({ text_color: e.target.value })}
              className="h-8 w-8 rounded border border-border cursor-pointer p-0"
            />
            <div className="flex-1 space-y-1">
              <label className="text-xs font-medium text-muted-foreground">สีตัวอักษร</label>
              <Input
                value={style.text_color || ''}
                placeholder="#1a1a2e"
                onChange={(e) => update({ text_color: e.target.value })}
                className="h-7 text-xs"
              />
            </div>
          </div>
        </div>

        {/* Background Image */}
        <div className="space-y-2">
          <p className="text-sm font-medium text-foreground">รูปพื้นหลัง</p>
          {style.bg_image_url && (
            <div className="relative w-full max-w-[200px]">
              <img
                src={style.bg_image_url}
                alt=""
                className="w-full h-auto rounded-lg border border-border"
              />
              <button
                type="button"
                className="absolute -top-2 -right-2 h-5 w-5 rounded-full bg-red-500 text-white flex items-center justify-center text-xs hover:bg-red-600"
                onClick={() => update({ bg_image_url: '' })}
              >
                <X className="h-3 w-3" />
              </button>
            </div>
          )}
          <div className="flex gap-2">
            <input ref={fileRef} type="file" accept="image/*" className="hidden" onChange={handleUpload} />
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={uploadImage.isPending}
              onClick={() => fileRef.current?.click()}
            >
              {uploadImage.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Upload className="h-3.5 w-3.5" />}
              {uploadImage.isPending && uploadImage.progress > 0 ? `${uploadImage.progress}%` : 'อัพโหลดรูปพื้นหลัง'}
            </Button>
          </div>
          <Input
            placeholder="หรือวาง URL รูปภาพ"
            value={style.bg_image_url || ''}
            onChange={(e) => update({ bg_image_url: e.target.value })}
            className="text-xs"
          />
        </div>
      </CardContent>
    </Card>
  )
}
