import { FEATURES } from './constants'
import { useScrollReveal } from './useScrollReveal'

export function FeaturesSection() {
  const { ref, isVisible } = useScrollReveal()

  return (
    <section id="features" className="bg-white py-20 sm:py-28">
      <div ref={ref} className="mx-auto max-w-6xl px-4 sm:px-6">
        <div className="text-center">
          <h2 className="text-3xl font-bold tracking-tight text-slate-900 sm:text-4xl">
            ทุกสิ่งที่คุณต้องการเพื่อปกป้องข้อมูล Pixel
          </h2>
          <p className="mt-4 text-slate-600">
            ฟีเจอร์ครบครันสำหรับจัดการ Facebook Pixel อย่างมืออาชีพ
          </p>
        </div>

        <div className="mt-14 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {FEATURES.map((feature, i) => (
            <div
              key={feature.title}
              className={`rounded-xl border border-slate-200 bg-white p-6 hover:border-blue-200 hover:shadow-lg hover:shadow-blue-900/5 transition-all duration-700 ${
                isVisible
                  ? 'translate-y-0 opacity-100'
                  : 'translate-y-8 opacity-0'
              }`}
              style={{ transitionDelay: `${i * 100}ms` }}
            >
              <div className="mb-4 inline-flex rounded-lg bg-blue-50 p-3">
                <feature.icon className="h-6 w-6 text-blue-800" />
              </div>
              <h3 className="text-lg font-semibold text-slate-900">
                {feature.title}
              </h3>
              <p className="mt-2 text-sm leading-relaxed text-slate-600">
                {feature.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
