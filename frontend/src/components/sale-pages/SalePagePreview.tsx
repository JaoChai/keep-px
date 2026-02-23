import { CheckCircle, Phone, MessageCircle } from 'lucide-react'

interface SalePagePreviewProps {
  hero: { title: string; subtitle: string; image_url: string }
  body: { description: string; features: string[] }
  cta: { button_text: string; button_link: string }
  contact: { line_id: string; phone: string }
}

export function SalePagePreview({ hero, body, cta, contact }: SalePagePreviewProps) {
  return (
    <div className="max-w-[375px] mx-auto rounded-2xl border border-neutral-200 shadow-lg overflow-hidden bg-white">
      {/* Hero Section */}
      <div className="bg-gradient-to-br from-indigo-600 to-purple-600 px-6 py-10 text-center text-white">
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
      <div className="px-6 py-6">
        {body.description ? (
          <p className="text-sm text-neutral-600 leading-relaxed">{body.description}</p>
        ) : (
          <p className="text-sm text-neutral-300">Description text will appear here...</p>
        )}

        {/* Features */}
        {body.features.length > 0 ? (
          <ul className="mt-4 space-y-2">
            {body.features.filter(f => f.trim()).map((feature, i) => (
              <li key={i} className="flex items-start gap-2 text-sm text-neutral-700">
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
      </div>

      {/* CTA Section */}
      <div className="px-6 pb-4">
        <button className="w-full py-3 rounded-lg bg-gradient-to-r from-indigo-600 to-purple-600 text-white font-semibold text-sm shadow-md">
          {cta.button_text || 'Call to Action'}
        </button>
      </div>

      {/* Contact Section */}
      {(contact.line_id || contact.phone) && (
        <div className="px-6 py-4 border-t border-neutral-100">
          <p className="text-xs font-medium text-neutral-500 mb-2 text-center">Contact Us</p>
          <div className="flex items-center justify-center gap-4">
            {contact.line_id && (
              <div className="flex items-center gap-1.5 text-xs text-neutral-600">
                <MessageCircle className="h-3.5 w-3.5 text-emerald-500" />
                <span>{contact.line_id}</span>
              </div>
            )}
            {contact.phone && (
              <div className="flex items-center gap-1.5 text-xs text-neutral-600">
                <Phone className="h-3.5 w-3.5 text-indigo-500" />
                <span>{contact.phone}</span>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Footer */}
      <div className="px-6 py-3 bg-neutral-50 text-center">
        <p className="text-[10px] text-neutral-400">Powered by Pixlinks</p>
      </div>
    </div>
  )
}
