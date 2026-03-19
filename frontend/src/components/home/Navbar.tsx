import { useState } from 'react'
import { Link } from 'react-router'
import { Shield, Menu, X } from 'lucide-react'
import { NAV_LINKS, scrollToSection } from './constants'

export function Navbar() {
  const [open, setOpen] = useState(false)

  return (
    <nav className="sticky top-0 z-50 border-b border-slate-200 bg-white/90 backdrop-blur">
      <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4 sm:px-6">
        {/* Logo */}
        <Link to="/" className="flex items-center gap-2" aria-label="Pixlinks — กลับหน้าหลัก">
          <Shield className="h-6 w-6 text-blue-800" />
          <span className="text-xl font-bold text-slate-900">Pixlinks</span>
        </Link>

        {/* Desktop menu */}
        <div className="hidden md:flex items-center gap-8">
          {NAV_LINKS.map((link) => (
            <button
              key={link.href}
              onClick={() => scrollToSection(link.href)}
              className="text-sm text-slate-600 hover:text-slate-900 transition-colors"
            >
              {link.label}
            </button>
          ))}
        </div>

        {/* Desktop CTA */}
        <Link
          to="/login"
          className="hidden md:inline-flex rounded-lg bg-amber-500 px-5 py-2 text-sm font-semibold text-slate-900 hover:bg-amber-400 transition-colors"
        >
          เริ่มต้นฟรี
        </Link>

        {/* Mobile toggle */}
        <button
          onClick={() => setOpen(!open)}
          className="md:hidden p-2 text-slate-600"
          aria-label={open ? 'ปิดเมนู' : 'เปิดเมนู'}
        >
          {open ? <X className="h-6 w-6" /> : <Menu className="h-6 w-6" />}
        </button>
      </div>

      {/* Mobile menu */}
      {open && (
        <div className="md:hidden border-t border-slate-200 bg-white px-4 pb-4 pt-2">
          {NAV_LINKS.map((link) => (
            <button
              key={link.href}
              onClick={() => {
                scrollToSection(link.href)
                setOpen(false)
              }}
              className="block w-full py-3 text-left text-sm text-slate-600 hover:text-slate-900"
            >
              {link.label}
            </button>
          ))}
          <Link
            to="/login"
            onClick={() => setOpen(false)}
            className="mt-2 block w-full rounded-lg bg-amber-500 py-2.5 text-center text-sm font-semibold text-slate-900 hover:bg-amber-400 transition-colors"
          >
            เริ่มต้นฟรี
          </Link>
        </div>
      )}
    </nav>
  )
}
