import { STEPS } from './constants'
import { useScrollReveal } from './useScrollReveal'

export function HowItWorksSection() {
  const { ref, isVisible } = useScrollReveal()

  return (
    <section id="how-it-works" className="bg-slate-50 py-20 sm:py-28">
      <div ref={ref} className="mx-auto max-w-6xl px-4 sm:px-6">
        <div className="text-center">
          <h2 className="text-3xl font-bold tracking-tight text-slate-900 sm:text-4xl">
            เริ่มต้นใช้งานได้ง่ายๆ
          </h2>
          <p className="mt-4 text-slate-600">
            4 ขั้นตอนสู่การปกป้องข้อมูล Pixel ของคุณ
          </p>
        </div>

        <div className="mt-14 grid gap-8 sm:grid-cols-2 lg:grid-cols-4">
          {STEPS.map((step, i) => (
            <div
              key={step.number}
              className={`relative text-center transition-all duration-700 ${
                isVisible
                  ? 'translate-y-0 opacity-100'
                  : 'translate-y-8 opacity-0'
              }`}
              style={{ transitionDelay: `${i * 150}ms` }}
            >
              {/* Connector line (desktop only, between steps) */}
              {i < STEPS.length - 1 && (
                <div className="absolute top-6 left-[calc(50%+28px)] right-[calc(-50%+28px)] hidden lg:block">
                  <div className="h-0.5 w-full bg-blue-200" />
                </div>
              )}

              <div className="relative mx-auto mb-4 flex size-12 items-center justify-center rounded-full bg-blue-800 text-lg font-bold text-white">
                {step.number}
              </div>
              <h3 className="text-lg font-semibold text-slate-900">
                {step.title}
              </h3>
              <p className="mt-2 text-sm leading-relaxed text-slate-600">
                {step.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
