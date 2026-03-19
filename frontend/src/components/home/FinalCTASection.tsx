import { Link } from 'react-router'
import { ArrowRight, ShieldCheck } from 'lucide-react'

export function FinalCTASection() {
  return (
    <section className="bg-gradient-to-br from-blue-800 to-blue-900 py-20 sm:py-28">
      <div className="mx-auto max-w-3xl px-4 text-center sm:px-6">
        <ShieldCheck className="mx-auto h-12 w-12 text-blue-300" />

        <h2 className="mt-6 text-3xl font-bold tracking-tight text-white sm:text-4xl">
          พร้อมปกป้องข้อมูล Pixel ของคุณ?
        </h2>

        <p className="mt-4 text-lg text-blue-200">
          เริ่มต้นฟรีวันนี้ ไม่ต้องใช้บัตรเครดิต
          ปกป้องข้อมูลก่อนที่จะสายเกินไป
        </p>

        <Link
          to="/login"
          className="mt-8 inline-flex items-center gap-2 rounded-lg bg-amber-500 px-8 py-3.5 text-sm font-semibold text-slate-900 hover:bg-amber-400 transition-colors"
        >
          เริ่มต้นฟรี
          <ArrowRight className="h-4 w-4" />
        </Link>

        <p className="mt-4 text-sm text-blue-300">
          ใช้งานได้ทันที &middot; ฟรี 2 Pixel &middot; ไม่มีค่าใช้จ่ายแอบแฝง
        </p>
      </div>
    </section>
  )
}
