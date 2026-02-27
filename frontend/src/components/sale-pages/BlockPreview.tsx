import { MessageCircle, Globe, Link } from 'lucide-react'
import type { Block, PageStyle } from '@/types'

interface BlockPreviewProps {
  blocks: Block[]
  ctaEventName?: string
  style?: PageStyle
}

export function BlockPreview({ blocks, ctaEventName, style }: BlockPreviewProps) {
  const accentColor = style?.accent_color
  const bgColor = style?.bg_color
  const textColor = style?.text_color
  const bgImageUrl = style?.bg_image_url

  return (
    <div
      className="max-w-[375px] mx-auto rounded-2xl border border-neutral-200 shadow-lg overflow-hidden bg-white"
      style={{
        ...(bgColor ? { backgroundColor: bgColor } : {}),
        ...(bgImageUrl ? { backgroundImage: `url(${bgImageUrl})`, backgroundSize: 'cover', backgroundPosition: 'center' } : {}),
      }}
    >
      {/* Blocks */}
      <div>
        {blocks.length === 0 && (
          <div className="px-6 py-16 text-center text-neutral-300 text-sm">
            เพิ่มบล็อกเพื่อเริ่มสร้างหน้าเพจ
          </div>
        )}
        {blocks.map((block) => {
          if (block.type === 'image') {
            return block.image_url ? (
              <img
                key={block.id}
                src={block.image_url}
                alt=""
                className="w-full block object-cover"
                style={{ aspectRatio: '1/1' }}
              />
            ) : (
              <div key={block.id} className="w-full bg-neutral-100 flex items-center justify-center text-neutral-400 text-xs" style={{ aspectRatio: '1/1' }}>
                รูปภาพ
              </div>
            )
          }

          if (block.type === 'text') {
            return (
              <div key={block.id} className="px-5 py-4 text-center">
                <p className="text-sm whitespace-pre-line leading-relaxed" style={{ color: textColor || undefined }}>
                  {block.text || <span className="text-neutral-300">ข้อความ...</span>}
                </p>
              </div>
            )
          }

          if (block.type === 'button') {
            const btnColor = block.button_style === 'line' ? '#06C755' : block.button_style === 'website' ? '#7c3aed' : (accentColor || '#4f46e5')
            const Icon = block.button_style === 'line' ? MessageCircle : block.button_style === 'website' ? Globe : Link
            return (
              <div key={block.id} className="px-5 py-1.5">
                <div className="relative">
                  <div
                    className="w-full flex items-center justify-center gap-2 py-3 rounded-xl text-white text-sm font-semibold"
                    style={{ backgroundColor: btnColor }}
                  >
                    <Icon className="h-4 w-4" />
                    <span>{block.button_text || 'ปุ่ม'}</span>
                  </div>
                  {ctaEventName && (
                    <span className="absolute -top-1.5 -right-1.5 px-1.5 py-0.5 text-[9px] font-medium bg-amber-100 text-amber-700 rounded-full border border-amber-200">
                      {ctaEventName}
                    </span>
                  )}
                </div>
              </div>
            )
          }

          return null
        })}
      </div>

      {/* Footer */}
      <div className="px-6 py-3 bg-neutral-50 text-center mt-2">
        <p className="text-[10px] text-neutral-400">Powered by Pixlinks</p>
      </div>
    </div>
  )
}
