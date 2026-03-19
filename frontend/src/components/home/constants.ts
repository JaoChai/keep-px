import {
  Radio,
  RotateCcw,
  FileText,
  BarChart3,
  Layers,
  Send,
  ShieldOff,
  DatabaseZap,
  RefreshCw,
  type LucideIcon,
} from 'lucide-react'

// --- Nav ---
export const NAV_LINKS = [
  { label: 'ฟีเจอร์', href: '#features' },
  { label: 'วิธีใช้งาน', href: '#how-it-works' },
  { label: 'ราคา', href: '#pricing' },
  { label: 'คำถามที่พบบ่อย', href: '#faq' },
] as const

// --- Pain Points ---
export interface PainPoint {
  icon: LucideIcon
  title: string
  description: string
}

export const PAIN_POINTS: PainPoint[] = [
  {
    icon: ShieldOff,
    title: 'บัญชีโฆษณาถูกแบน',
    description:
      'Facebook สามารถแบนบัญชีโฆษณาได้ทุกเมื่อ โดยไม่แจ้งล่วงหน้า คุณอาจเสียทุกอย่างภายในไม่กี่วินาที',
  },
  {
    icon: DatabaseZap,
    title: 'ข้อมูล Pixel หายไปทั้งหมด',
    description:
      'เมื่อถูกแบน ข้อมูลที่ Pixel เรียนรู้มาทั้งหมด — Custom Audience, Conversion Data ทุกอย่างจะหายไปในทันที',
  },
  {
    icon: RefreshCw,
    title: 'ต้องเริ่มต้นจากศูนย์',
    description:
      'สร้างบัญชีใหม่ Pixel ใหม่ ยิงโฆษณาใหม่ รอเรียนรู้ใหม่ เสียเงินและเวลาไปอีกหลายสัปดาห์',
  },
]

// --- Features ---
export interface Feature {
  icon: LucideIcon
  title: string
  description: string
}

export const FEATURES: Feature[] = [
  {
    icon: Radio,
    title: 'Event Tracking',
    description:
      'บันทึกทุก Event จาก Facebook Pixel แบบ Real-time ไม่พลาดแม้แต่ข้อมูลเดียว',
  },
  {
    icon: Send,
    title: 'CAPI Forwarding',
    description:
      'ส่งต่อข้อมูลไปยัง Facebook Conversions API โดยตรง เพิ่ม Event Match Quality',
  },
  {
    icon: RotateCcw,
    title: 'Backup & Replay',
    description:
      'สำรองข้อมูลทุก Event และส่งซ้ำไปยัง Pixel ใหม่ได้ทันทีเมื่อถูกแบน',
  },
  {
    icon: FileText,
    title: 'Sale Pages',
    description:
      'สร้างเซลเพจที่รองรับหลาย Pixel พร้อมติดตาม Event อัตโนมัติ',
  },
  {
    icon: BarChart3,
    title: 'Analytics Dashboard',
    description:
      'ดูสถิติ Event แบบ Real-time พร้อมกราฟและรายงานละเอียด',
  },
  {
    icon: Layers,
    title: 'Multi-Pixel Support',
    description:
      'จัดการหลาย Pixel ในที่เดียว สลับใช้งานได้ทันที ไม่จำกัดจำนวน',
  },
]

// --- How It Works ---
export interface Step {
  number: string
  title: string
  description: string
}

export const STEPS: Step[] = [
  {
    number: '1',
    title: 'สร้าง Pixel',
    description: 'เพิ่ม Facebook Pixel ของคุณเข้าสู่ระบบ Pixlinks',
  },
  {
    number: '2',
    title: 'สร้าง Sale Page',
    description: 'สร้างเซลเพจและเชื่อมต่อกับ Pixel ที่ต้องการ',
  },
  {
    number: '3',
    title: 'เก็บ Event อัตโนมัติ',
    description: 'ระบบจะบันทึกทุก Event ที่เกิดขึ้นบน Sale Page',
  },
  {
    number: '4',
    title: 'Replay เมื่อต้องการ',
    description: 'ส่งข้อมูลซ้ำไปยัง Pixel ใหม่ได้ทุกเมื่อ',
  },
]

// --- Stats ---
export interface Stat {
  value: number
  suffix: string
  label: string
}

export const STATS: Stat[] = [
  { value: 500000, suffix: '+', label: 'Events ที่บันทึกไว้' },
  { value: 99.9, suffix: '%', label: 'Uptime' },
  { value: 1200, suffix: '+', label: 'Pixels ที่ปกป้อง' },
  { value: 3, suffix: ' นาที', label: 'เวลาเฉลี่ย Replay' },
]

// --- Pricing ---
export interface PricingFeature {
  feature: string
  free: string | boolean
  paid: string | boolean
}

export const PRICING_FEATURES: PricingFeature[] = [
  { feature: 'จำนวนพิกเซล', free: '2', paid: 'ปรับได้ตาม Slots' },
  { feature: 'จำนวนเซลเพจ', free: '2', paid: 'ปรับได้ตาม Slots' },
  { feature: 'อีเวนต์ต่อเดือน', free: '1,000', paid: 'ตาม Slots' },
  { feature: 'เก็บข้อมูล', free: '7 วัน', paid: '90 วัน' },
  { feature: 'รีเพลย์', free: false, paid: true },
  { feature: 'Replay Credits', free: false, paid: true },
  { feature: 'CAPI Forwarding', free: true, paid: true },
  { feature: 'Analytics Dashboard', free: true, paid: true },
]

// --- FAQ ---
export interface FAQItem {
  question: string
  answer: string
}

export const FAQ_ITEMS: FAQItem[] = [
  {
    question: 'Pixlinks คืออะไร?',
    answer:
      'Pixlinks คือแพลตฟอร์มที่ช่วยปกป้องข้อมูล Facebook Pixel ของคุณ โดยบันทึก Event ทุกรายการ สำรองข้อมูล และส่งซ้ำไปยัง Pixel ใหม่ได้ทันทีเมื่อบัญชีโฆษณาถูกแบน',
  },
  {
    question: 'ข้อมูลของฉันปลอดภัยไหม?',
    answer:
      'ข้อมูลทั้งหมดถูกเข้ารหัสและจัดเก็บอย่างปลอดภัยบนเซิร์ฟเวอร์ที่ได้มาตรฐาน เราไม่แชร์ข้อมูลของคุณกับบุคคลที่สาม',
  },
  {
    question: 'ถ้าบัญชีโฆษณาถูกแบน ฉันต้องทำอย่างไร?',
    answer:
      'เพียงสร้าง Pixel ใหม่ใน Pixlinks จากนั้นใช้ฟีเจอร์ Replay เพื่อส่งข้อมูล Event ทั้งหมดไปยัง Pixel ใหม่ ระบบจะทำงานอัตโนมัติ',
  },
  {
    question: 'Replay ทำงานอย่างไร?',
    answer:
      'Replay จะส่งข้อมูล Event ที่บันทึกไว้ไปยัง Facebook Conversions API ของ Pixel ใหม่ โดยรักษาข้อมูลเดิมทั้งหมดไว้ ทำให้ Pixel ใหม่มีข้อมูลเรียนรู้เหมือนเดิม',
  },
  {
    question: 'ใช้ฟรีได้กี่ Pixel?',
    answer:
      'แพ็กเกจฟรีรองรับ 2 Pixel และ 2 Sale Page พร้อม 1,000 Event ต่อเดือน เหมาะสำหรับเริ่มต้นทดลองใช้งาน',
  },
  {
    question: 'CAPI คืออะไร?',
    answer:
      'CAPI (Conversions API) คือ API ของ Facebook ที่ใช้ส่งข้อมูล Event จากเซิร์ฟเวอร์โดยตรง ช่วยเพิ่มความแม่นยำของการติดตามและ Event Match Quality',
  },
  {
    question: 'สามารถยกเลิกได้ทุกเมื่อไหม?',
    answer:
      'ได้ครับ คุณสามารถยกเลิกแพ็กเกจได้ทุกเมื่อโดยไม่มีค่าใช้จ่ายเพิ่มเติม ข้อมูลจะยังคงอยู่จนกว่าจะหมดอายุตามแพ็กเกจ',
  },
  {
    question: 'รองรับการชำระเงินแบบใด?',
    answer:
      'เรารองรับการชำระเงินผ่าน Stripe ซึ่งรองรับบัตรเครดิต/เดบิตทุกชนิดและ PromptPay',
  },
]

// --- Footer ---
export const FOOTER_PRODUCT_LINKS = [
  { label: 'ฟีเจอร์', href: '#features' },
  { label: 'ราคา', href: '#pricing' },
  { label: 'วิธีใช้งาน', href: '#how-it-works' },
  { label: 'คำถามที่พบบ่อย', href: '#faq' },
] as const

export const FOOTER_COMPANY_LINKS = [
  { label: 'เงื่อนไขการใช้งาน', href: '' },
  { label: 'นโยบายความเป็นส่วนตัว', href: '' },
] as const

// --- Shared utility ---
export function scrollToSection(href: string) {
  if (!href.startsWith('#') || href.length < 2) return
  const el = document.querySelector(href)
  if (el) el.scrollIntoView({ behavior: 'smooth' })
}
