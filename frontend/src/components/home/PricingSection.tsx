import { Link } from 'react-router'
import { Check, X, ArrowRight } from 'lucide-react'
import { PRICING_FEATURES } from './constants'
import { useScrollReveal } from './useScrollReveal'

function CellValue({ value }: { value: string | boolean }) {
  if (typeof value === 'boolean') {
    return value ? (
      <Check className="mx-auto h-5 w-5 text-green-500" aria-label="รองรับ" />
    ) : (
      <X className="mx-auto h-5 w-5 text-slate-300" aria-label="ไม่รองรับ" />
    )
  }
  return <span>{value}</span>
}

export function PricingSection() {
  const { ref, isVisible } = useScrollReveal()

  return (
    <section id="pricing" className="bg-white py-20 sm:py-28">
      <div
        ref={ref}
        className={`mx-auto max-w-4xl px-4 sm:px-6 transition-all duration-700 ${
          isVisible ? 'translate-y-0 opacity-100' : 'translate-y-8 opacity-0'
        }`}
      >
        <div className="text-center">
          <h2 className="text-3xl font-bold tracking-tight text-slate-900 sm:text-4xl">
            ราคาที่เหมาะกับทุกขนาดธุรกิจ
          </h2>
          <p className="mt-4 text-slate-600">
            เริ่มต้นฟรี แล้วอัปเกรดทีหลังเมื่อพร้อม
          </p>
        </div>

        {/* Cards */}
        <div className="mt-14 grid gap-6 sm:grid-cols-2">
          {/* Free */}
          <div className="rounded-xl border border-slate-200 bg-white p-8">
            <div className="text-sm font-medium text-slate-500">ฟรี</div>
            <div className="mt-2 text-4xl font-bold text-slate-900">
              ฿0
              <span className="text-base font-normal text-slate-400">
                /เดือน
              </span>
            </div>
            <p className="mt-3 text-sm text-slate-600">
              เหมาะสำหรับเริ่มต้นทดลองใช้งาน
            </p>
            <Link
              to="/login"
              className="mt-6 block w-full rounded-lg border border-slate-300 bg-white py-2.5 text-center text-sm font-semibold text-slate-700 hover:bg-slate-50 transition-colors"
            >
              เริ่มต้นฟรี
            </Link>
          </div>

          {/* Paid */}
          <div className="relative rounded-xl border-2 border-blue-800 bg-white p-8">
            <div className="absolute -top-3 right-6 rounded-full bg-blue-800 px-3 py-0.5 text-xs font-semibold text-white">
              แนะนำ
            </div>
            <div className="text-sm font-medium text-blue-800">
              Pro
            </div>
            <div className="mt-2 text-4xl font-bold text-slate-900">
              Slots
              <span className="text-base font-normal text-slate-400">
                {' '}
                ตามการใช้งาน
              </span>
            </div>
            <p className="mt-3 text-sm text-slate-600">
              สำหรับธุรกิจที่ต้องการปกป้องข้อมูลอย่างจริงจัง
            </p>
            <Link
              to="/login"
              className="mt-6 flex w-full items-center justify-center gap-2 rounded-lg bg-amber-500 py-2.5 text-sm font-semibold text-slate-900 hover:bg-amber-400 transition-colors"
            >
              เริ่มต้นเลย
              <ArrowRight className="h-4 w-4" />
            </Link>
          </div>
        </div>

        {/* Comparison table */}
        <div className="mt-12 overflow-x-auto rounded-xl border border-slate-200">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-200 bg-slate-50">
                <th scope="col" className="px-6 py-3 text-left font-semibold text-slate-900">
                  ฟีเจอร์
                </th>
                <th scope="col" className="px-6 py-3 text-center font-semibold text-slate-900">
                  ฟรี
                </th>
                <th scope="col" className="px-6 py-3 text-center font-semibold text-blue-800">
                  Pro
                </th>
              </tr>
            </thead>
            <tbody>
              {PRICING_FEATURES.map((row) => (
                <tr
                  key={row.feature}
                  className="border-b border-slate-100 last:border-0"
                >
                  <td className="px-6 py-3 text-slate-700">{row.feature}</td>
                  <td className="px-6 py-3 text-center text-slate-600">
                    <CellValue value={row.free} />
                  </td>
                  <td className="px-6 py-3 text-center text-slate-900 font-medium">
                    <CellValue value={row.paid} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  )
}
