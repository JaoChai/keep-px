import { CheckCircle, Phone, MessageCircle, Globe } from 'lucide-react'
import type { PageStyle } from '@/types'

interface SalePagePreviewProps {
  hero: { title: string; subtitle: string; image_url: string }
  body: { description: string; features: string[]; images?: string[] }
  cta: { button_text: string; button_link: string }
  contact: { line_id: string; phone: string; website_url?: string }
  ctaEventName?: string
  style?: PageStyle
}

export function SalePagePreview({ hero, body, cta, contact, ctaEventName, style }: SalePagePreviewProps) {
  const accentColor = style?.accent_color || '#667eea'
  const bgColor = style?.bg_color
  const textColor = style?.text_color
  const bgImageUrl = style?.bg_image_url

  return (
    <div
      className="max-w-[375px] mx-auto rounded-2xl border border-border shadow-lg overflow-hidden bg-white"
      style={{
        ...(bgColor ? { backgroundColor: bgColor } : {}),
        ...(bgImageUrl ? { backgroundImage: `url(${bgImageUrl})`, backgroundSize: 'cover', backgroundPosition: 'center' } : {}),
      }}
    >
      {/* Hero Section */}
      <div
        className={!style?.accent_color ? 'bg-gradient-to-br from-zinc-800 to-zinc-900 px-6 py-10 text-center text-white' : 'px-6 py-10 text-center text-white'}
        style={style?.accent_color ? { background: `linear-gradient(135deg, ${accentColor}, ${accentColor}dd)` } : undefined}
      >
        {hero.image_url ? (
          <img
            src={hero.image_url}
            alt=""
            className="w-24 h-24 mx-auto rounded-full object-cover mb-4 border-2 border-white/30"
          />
        ) : (
          <div className="w-24 h-24 mx-auto rounded-full bg-white/20 mb-4 flex items-center justify-center">
            <span className="text-white/50 text-xs">Image</span>
          </div>
        )}
        <h1 className="text-xl font-bold leading-tight">
          {hero.title || <span className="text-white/40">Page Title</span>}
        </h1>
        {(hero.subtitle || !hero.title) && (
          <p className="mt-2 text-sm text-white/80">
            {hero.subtitle || <span className="text-white/30">Subtitle text</span>}
          </p>
        )}
      </div>

      {/* Body Section */}
      <div className="px-6 py-6" style={textColor ? { color: textColor } : undefined}>
        {body.description ? (
          <p className="text-sm leading-relaxed" style={{ color: textColor || undefined }}>{body.description}</p>
        ) : (
          <p className="text-sm text-neutral-300">Description text will appear here...</p>
        )}

        {/* Features */}
        {body.features.length > 0 ? (
          <ul className="mt-4 space-y-2">
            {body.features.filter(f => f.trim()).map((feature, i) => (
              <li key={i} className="flex items-start gap-2 text-sm" style={{ color: textColor || undefined }}>
                <CheckCircle className="h-4 w-4 text-emerald-500 mt-0.5 shrink-0" />
                <span>{feature}</span>
              </li>
            ))}
          </ul>
        ) : (
          <ul className="mt-4 space-y-2">
            <li className="flex items-start gap-2 text-sm text-neutral-300">
              <CheckCircle className="h-4 w-4 text-neutral-300 mt-0.5 shrink-0" />
              <span>Feature item</span>
            </li>
          </ul>
        )}

        {/* Body Images */}
        {body.images && body.images.length > 0 && (
          <div className="mt-4 space-y-2">
            {body.images.map((url, i) => (
              <img key={i} src={url} alt="" className="w-full h-auto rounded-lg" />
            ))}
          </div>
        )}
      </div>

      {/* CTA Section */}
      <div className="px-6 pb-4">
        <div className="relative">
          <button
            className={!style?.accent_color ? 'w-full py-3 rounded-lg bg-gradient-to-r from-zinc-800 to-zinc-900 text-white font-semibold text-sm shadow-md' : 'w-full py-3 rounded-lg text-white font-semibold text-sm shadow-md'}
            style={style?.accent_color ? { background: `linear-gradient(to right, ${accentColor}, ${accentColor}cc)` } : undefined}
          >
            {cta.button_text || 'Call to Action'}
          </button>
          {ctaEventName && (
            <span className="absolute -top-2 -right-2 px-1.5 py-0.5 text-[10px] font-medium bg-amber-100 text-amber-700 rounded-full border border-amber-200">
              {ctaEventName}
            </span>
          )}
        </div>
      </div>

      {/* Contact Section */}
      {(contact.line_id || contact.phone || contact.website_url) && (
        <div className="px-6 py-4 border-t border-border">
          <p className="text-xs font-medium text-muted-foreground mb-3 text-center">ติดต่อเรา</p>
          <div className="space-y-2">
            {contact.line_id && (
              <div className="w-full flex items-center justify-center gap-2 py-2.5 rounded-lg text-white text-sm font-semibold" style={{ backgroundColor: '#06C755' }}>
                <MessageCircle className="h-4 w-4" />
                <span>LINE: {contact.line_id}</span>
              </div>
            )}
            {contact.phone && (
              <div className="w-full flex items-center justify-center gap-2 py-2.5 rounded-lg bg-blue-600 text-white text-sm font-semibold">
                <Phone className="h-4 w-4" />
                <span>โทร: {contact.phone}</span>
              </div>
            )}
            {contact.website_url && (
              <div className="w-full flex items-center justify-center gap-2 py-2.5 rounded-lg bg-purple-600 text-white text-sm font-semibold">
                <Globe className="h-4 w-4" />
                <span>เยี่ยมชมเว็บไซต์</span>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Footer */}
      <div className="px-6 py-3 bg-muted text-center">
        <p className="text-[10px] text-muted-foreground">Powered by Pixlinks</p>
      </div>
    </div>
  )
}
