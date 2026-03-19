import { STATS } from './constants'
import { useScrollReveal } from './useScrollReveal'
import { useCountUp } from './useCountUp'

function StatItem({
  value,
  suffix,
  label,
  isActive,
}: {
  value: number
  suffix: string
  label: string
  isActive: boolean
}) {
  const count = useCountUp(value, isActive)

  const display =
    value >= 1000
      ? Math.floor(count).toLocaleString()
      : value % 1 !== 0
        ? count.toFixed(1)
        : Math.floor(count).toString()

  return (
    <div className="text-center">
      <div className="text-4xl font-bold text-white sm:text-5xl">
        {display}
        <span className="text-amber-400">{suffix}</span>
      </div>
      <div className="mt-2 text-sm text-slate-400">{label}</div>
    </div>
  )
}

export function StatsSection() {
  const { ref, isVisible } = useScrollReveal()

  return (
    <section className="bg-slate-900 py-20 sm:py-28">
      <div
        ref={ref}
        className="mx-auto grid max-w-5xl gap-12 px-4 sm:grid-cols-2 sm:px-6 lg:grid-cols-4"
      >
        {STATS.map((stat) => (
          <StatItem key={stat.label} {...stat} isActive={isVisible} />
        ))}
      </div>
    </section>
  )
}
