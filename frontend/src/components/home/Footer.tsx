import { Shield, Mail } from 'lucide-react'
import { FOOTER_PRODUCT_LINKS, FOOTER_COMPANY_LINKS, scrollToSection } from './constants'

export function Footer() {
  return (
    <footer className="bg-slate-900 py-16">
      <div className="mx-auto max-w-6xl px-4 sm:px-6">
        <div className="grid gap-12 sm:grid-cols-2 lg:grid-cols-4">
          {/* Brand */}
          <div className="sm:col-span-2 lg:col-span-1">
            <div className="flex items-center gap-2">
              <Shield className="h-6 w-6 text-blue-400" />
              <span className="text-lg font-bold text-white">Pixlinks</span>
            </div>
            <p className="mt-3 text-sm leading-relaxed text-slate-400">
              แพลตฟอร์มปกป้องข้อมูล Facebook Pixel
              สำหรับนักลงโฆษณามืออาชีพ
            </p>
          </div>

          {/* Product */}
          <div>
            <h4 className="text-sm font-semibold text-white">ผลิตภัณฑ์</h4>
            <ul className="mt-4 space-y-3">
              {FOOTER_PRODUCT_LINKS.map((link) => (
                <li key={link.label}>
                  <button
                    onClick={() => scrollToSection(link.href)}
                    className="text-sm text-slate-400 hover:text-white transition-colors"
                  >
                    {link.label}
                  </button>
                </li>
              ))}
            </ul>
          </div>

          {/* Company */}
          <div>
            <h4 className="text-sm font-semibold text-white">บริษัท</h4>
            <ul className="mt-4 space-y-3">
              {FOOTER_COMPANY_LINKS.map((link) => (
                <li key={link.label}>
                  <span className="text-sm text-slate-500">
                    {link.label}
                  </span>
                </li>
              ))}
            </ul>
          </div>

          {/* Contact */}
          <div>
            <h4 className="text-sm font-semibold text-white">ติดต่อ</h4>
            <ul className="mt-4 space-y-3">
              <li>
                <a
                  href="mailto:support@pixlinks.app"
                  className="flex items-center gap-2 text-sm text-slate-400 hover:text-white transition-colors"
                >
                  <Mail className="h-4 w-4" />
                  support@pixlinks.app
                </a>
              </li>
            </ul>
          </div>
        </div>

        {/* Bottom */}
        <div className="mt-12 flex flex-col items-center justify-between gap-4 border-t border-slate-800 pt-8 sm:flex-row">
          <p className="text-sm text-slate-500">
            &copy; {new Date().getFullYear()} Pixlinks. All rights reserved.
          </p>
          <p className="text-sm text-slate-500">
            Made with care for Thai advertisers
          </p>
        </div>
      </div>
    </footer>
  )
}
