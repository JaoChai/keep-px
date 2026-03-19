import { useId } from 'react'

export function DashboardMockup() {
  const gradientId = useId()

  return (
    <div className="relative w-full max-w-lg mx-auto">
      {/* Window frame */}
      <div className="rounded-xl border border-slate-200 bg-white shadow-2xl shadow-blue-900/10 overflow-hidden">
        {/* Title bar */}
        <div className="flex items-center gap-2 border-b border-slate-100 bg-slate-50 px-4 py-2.5">
          <div className="flex gap-1.5">
            <div className="h-3 w-3 rounded-full bg-red-400" />
            <div className="h-3 w-3 rounded-full bg-amber-400" />
            <div className="h-3 w-3 rounded-full bg-green-400" />
          </div>
          <div className="mx-auto rounded bg-slate-200 px-8 py-1 text-xs text-slate-400">
            pixlinks.app/dashboard
          </div>
        </div>

        {/* Dashboard content */}
        <div className="p-4 space-y-4">
          {/* Stats row */}
          <div className="grid grid-cols-3 gap-3">
            {[
              { label: 'Events Today', value: '2,847', color: 'text-blue-700' },
              { label: 'Active Pixels', value: '6', color: 'text-green-600' },
              { label: 'CAPI Status', value: 'Active', color: 'text-emerald-600' },
            ].map((stat) => (
              <div
                key={stat.label}
                className="rounded-lg bg-slate-50 p-3 text-center"
              >
                <div className={`text-lg font-bold ${stat.color}`}>
                  {stat.value}
                </div>
                <div className="text-[10px] text-slate-400 mt-0.5">
                  {stat.label}
                </div>
              </div>
            ))}
          </div>

          {/* Chart mockup */}
          <div className="rounded-lg bg-slate-50 p-3">
            <div className="text-xs font-medium text-slate-500 mb-2">
              Event Volume (7 days)
            </div>
            <svg
              viewBox="0 0 280 80"
              className="w-full h-auto"
              aria-hidden="true"
            >
              {/* Grid lines */}
              {[0, 20, 40, 60].map((y) => (
                <line
                  key={y}
                  x1="0"
                  y1={y}
                  x2="280"
                  y2={y}
                  stroke="#e2e8f0"
                  strokeWidth="0.5"
                />
              ))}
              {/* Area */}
              <path
                d="M0,60 L40,45 L80,50 L120,30 L160,35 L200,15 L240,20 L280,10 L280,80 L0,80 Z"
                fill={`url(#${gradientId})`}
              />
              {/* Line */}
              <path
                d="M0,60 L40,45 L80,50 L120,30 L160,35 L200,15 L240,20 L280,10"
                fill="none"
                stroke="#1e40af"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
              {/* Dots */}
              {[
                [0, 60],
                [40, 45],
                [80, 50],
                [120, 30],
                [160, 35],
                [200, 15],
                [240, 20],
                [280, 10],
              ].map(([cx, cy]) => (
                <circle
                  key={`${cx}-${cy}`}
                  cx={cx}
                  cy={cy}
                  r="3"
                  fill="#1e40af"
                />
              ))}
              <defs>
                <linearGradient
                  id={gradientId}
                  x1="0"
                  y1="0"
                  x2="0"
                  y2="1"
                >
                  <stop offset="0%" stopColor="#1e40af" stopOpacity="0.2" />
                  <stop offset="100%" stopColor="#1e40af" stopOpacity="0" />
                </linearGradient>
              </defs>
            </svg>
          </div>

          {/* Event list */}
          <div className="space-y-2">
            {[
              { name: 'PageView', count: '1,203', time: '2s ago' },
              { name: 'Purchase', count: '47', time: '15s ago' },
              { name: 'AddToCart', count: '312', time: '8s ago' },
            ].map((event) => (
              <div
                key={event.name}
                className="flex items-center justify-between rounded-lg bg-slate-50 px-3 py-2"
              >
                <div className="flex items-center gap-2">
                  <div className="h-2 w-2 rounded-full bg-green-500 motion-safe:animate-pulse" />
                  <span className="text-xs font-medium text-slate-700">
                    {event.name}
                  </span>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-xs font-semibold text-slate-800">
                    {event.count}
                  </span>
                  <span className="text-[10px] text-slate-400">
                    {event.time}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Floating decoration */}
      <div className="absolute -top-4 -right-4 h-24 w-24 rounded-full bg-amber-400/20 blur-2xl" />
      <div className="absolute -bottom-6 -left-6 h-32 w-32 rounded-full bg-blue-600/10 blur-2xl" />
    </div>
  )
}
