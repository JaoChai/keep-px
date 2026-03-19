import { useId, useState } from 'react'
import { ChevronDown } from 'lucide-react'
import { FAQ_ITEMS } from './constants'
import { useScrollReveal } from './useScrollReveal'

function FAQItem({
  question,
  answer,
  isOpen,
  onToggle,
  id,
}: {
  question: string
  answer: string
  isOpen: boolean
  onToggle: () => void
  id: string
}) {
  const answerId = `${id}-answer`

  return (
    <div className="border-b border-slate-200 last:border-0">
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={isOpen}
        aria-controls={answerId}
        className="flex w-full items-center justify-between py-5 text-left"
      >
        <span className="text-base font-medium text-slate-900">
          {question}
        </span>
        <ChevronDown
          className={`ml-4 h-5 w-5 shrink-0 text-slate-400 transition-transform duration-200 ${
            isOpen ? 'rotate-180' : ''
          }`}
        />
      </button>
      <div
        id={answerId}
        role="region"
        className={`grid transition-all duration-200 ${
          isOpen ? 'grid-rows-[1fr] pb-5' : 'grid-rows-[0fr]'
        }`}
      >
        <div className="overflow-hidden">
          <p className="text-sm leading-relaxed text-slate-600">{answer}</p>
        </div>
      </div>
    </div>
  )
}

export function FAQSection() {
  const { ref, isVisible } = useScrollReveal()
  const [openIndex, setOpenIndex] = useState<number | null>(null)
  const baseId = useId()

  return (
    <section id="faq" className="bg-slate-50 py-20 sm:py-28">
      <div
        ref={ref}
        className={`mx-auto max-w-3xl px-4 sm:px-6 transition-all duration-700 ${
          isVisible ? 'translate-y-0 opacity-100' : 'translate-y-8 opacity-0'
        }`}
      >
        <div className="text-center">
          <h2 className="text-3xl font-bold tracking-tight text-slate-900 sm:text-4xl">
            คำถามที่พบบ่อย
          </h2>
          <p className="mt-4 text-slate-600">
            มีคำถาม? เรามีคำตอบให้คุณ
          </p>
        </div>

        <div className="mt-12 rounded-xl border border-slate-200 bg-white px-6">
          {FAQ_ITEMS.map((item, i) => (
            <FAQItem
              key={item.question}
              question={item.question}
              answer={item.answer}
              isOpen={openIndex === i}
              onToggle={() => setOpenIndex(openIndex === i ? null : i)}
              id={`${baseId}-faq-${i}`}
            />
          ))}
        </div>
      </div>
    </section>
  )
}
