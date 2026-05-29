import { Link } from 'react-router'
import { ArrowRight, ShieldCheck } from 'lucide-react'
import { DashboardMockup } from './DashboardMockup'
import { scrollToSection } from './constants'

export function HeroSection() {
  return (
    <section className="overflow-hidden bg-white">
      <div className="mx-auto grid max-w-6xl items-center gap-12 px-4 py-16 sm:px-6 sm:py-24 lg:grid-cols-2 lg:gap-16 lg:py-32">
        {/* Text */}
        <div className="text-center lg:text-left">
          {/* Badge */}
          <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-blue-200 bg-blue-50 px-4 py-1.5 text-sm text-blue-800">
            <ShieldCheck className="size-4" />
            <span>แพลตฟอร์มปกป้องข้อมูล Facebook Pixel</span>
          </div>

          <h1 className="text-4xl font-bold tracking-tight text-slate-900 sm:text-5xl lg:text-[3.5rem] lg:leading-[1.15]">
            บัญชีโฆษณาถูกแบน{' '}
            <span className="text-blue-800">ข้อมูล Pixel ก็ยังอยู่</span>
          </h1>

          <p className="mt-6 text-lg leading-relaxed text-slate-600">
            บันทึก สำรอง และส่งซ้ำข้อมูล Pixel Event อัตโนมัติ
            เพื่อให้ธุรกิจของคุณไม่สะดุด แม้บัญชีโฆษณาจะถูกระงับ
          </p>

          {/* CTAs */}
          <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:justify-center lg:justify-start">
            <Link
              to="/login"
              className="inline-flex items-center justify-center gap-2 rounded-lg bg-amber-500 px-6 py-3 text-sm font-semibold text-slate-900 hover:bg-amber-400 transition-colors"
            >
              เริ่มต้นฟรี
              <ArrowRight className="size-4" />
            </Link>
            <button
              type="button"
              onClick={() => scrollToSection('#how-it-works')}
              className="inline-flex items-center justify-center rounded-lg border border-slate-300 bg-white px-6 py-3 text-sm font-semibold text-slate-700 hover:bg-slate-50 transition-colors"
            >
              ดูวิธีทำงาน
            </button>
          </div>

          <p className="mt-4 text-sm text-slate-400">
            ไม่ต้องใช้บัตรเครดิต &middot; เริ่มใช้ได้ทันที
          </p>
        </div>

        {/* Mockup */}
        <div className="relative motion-safe:animate-[float_6s_ease-in-out_infinite]">
          <DashboardMockup />
        </div>
      </div>
    </section>
  )
}
