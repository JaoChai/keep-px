import { PAIN_POINTS } from './constants'
import { useScrollReveal } from './useScrollReveal'

export function PainPointsSection() {
  const { ref, isVisible } = useScrollReveal()

  return (
    <section className="bg-slate-50 py-20 sm:py-28">
      <div ref={ref} className="mx-auto max-w-6xl px-4 sm:px-6">
        <div className="text-center">
          <h2 className="text-3xl font-bold tracking-tight text-slate-900 sm:text-4xl">
            ปัญหาที่นักลงโฆษณาทุกคนเจอ
          </h2>
          <p className="mt-4 text-slate-600">
            Facebook Ads มีความเสี่ยงที่คุณควบคุมไม่ได้ — แต่เตรียมรับมือได้
          </p>
        </div>

        <div className="mt-14 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {PAIN_POINTS.map((point, i) => (
            <div
              key={point.title}
              className={`rounded-xl border border-red-100 bg-white p-6 transition-all duration-700 ${
                isVisible
                  ? 'translate-y-0 opacity-100'
                  : 'translate-y-8 opacity-0'
              }`}
              style={{ transitionDelay: `${i * 150}ms` }}
            >
              <div className="mb-4 inline-flex rounded-lg bg-red-50 p-3">
                <point.icon className="h-6 w-6 text-red-500" />
              </div>
              <h3 className="text-lg font-semibold text-slate-900">
                {point.title}
              </h3>
              <p className="mt-2 text-sm leading-relaxed text-slate-600">
                {point.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
