import { Image, Type, MessageCircle, Plus } from 'lucide-react'
import type { Block } from '@/types'

interface TemplateSelectorProps {
  onSelect: (blocks: Block[]) => void
}

const TEMPLATES: {
  id: string
  title: string
  description: string
  icons: typeof Image[]
  recommended?: boolean
  blocks: Omit<Block, 'id'>[]
}[] = [
  {
    id: 'image-line',
    title: 'รูป + แอดไลน์',
    description: 'อัพรูปสินค้า → ปุ่มแอดไลน์',
    icons: [Image, MessageCircle],
    recommended: true,
    blocks: [
      { type: 'image', image_url: '' },
      { type: 'button', button_style: 'line', button_text: 'แอดไลน์สั่งซื้อ', button_url: '', button_value: '' },
    ],
  },
  {
    id: 'image-text-line',
    title: 'รูป + ข้อความ + แอดไลน์',
    description: 'รูปสินค้า + รายละเอียด → ปุ่มแอดไลน์',
    icons: [Image, Type, MessageCircle],
    blocks: [
      { type: 'image', image_url: '' },
      { type: 'text', text: '' },
      { type: 'button', button_style: 'line', button_text: 'แอดไลน์สั่งซื้อ', button_url: '', button_value: '' },
    ],
  },
  {
    id: 'blank',
    title: 'เริ่มจากหน้าว่าง',
    description: 'สร้างเองตั้งแต่ต้น',
    icons: [Plus],
    blocks: [],
  },
]

export function TemplateSelector({ onSelect }: TemplateSelectorProps) {
  const handleSelect = (template: (typeof TEMPLATES)[number]) => {
    const blocks: Block[] = template.blocks.map((b) => ({ ...b, id: crypto.randomUUID() }))
    onSelect(blocks)
  }

  return (
    <div className="space-y-3">
      <div>
        <p className="text-sm font-medium text-foreground">เริ่มจากเทมเพลต</p>
        <p className="text-xs text-muted-foreground mt-1">เลือกรูปแบบเพจที่ต้องการ แล้วปรับแต่งเพิ่มได้ทีหลัง</p>
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        {TEMPLATES.map((template) => (
          <button
            key={template.id}
            type="button"
            className={`relative text-left p-4 rounded-lg border-2 transition-all hover:shadow-md ${
              template.recommended
                ? 'border-primary bg-primary/5 hover:bg-primary/10'
                : 'border-border hover:border-muted-foreground bg-card'
            }`}
            onClick={() => handleSelect(template)}
          >
            {template.recommended && (
              <span className="absolute -top-2.5 left-3 px-2 py-0.5 text-[10px] font-semibold bg-primary text-primary-foreground rounded-full">
                แนะนำ
              </span>
            )}
            <div className="flex items-center gap-1.5 mb-2">
              {template.icons.map((Icon, i) => (
                <Icon key={i} className="size-4 text-muted-foreground" />
              ))}
            </div>
            <p className="text-sm font-medium text-foreground">{template.title}</p>
            <p className="text-xs text-muted-foreground mt-0.5">{template.description}</p>
          </button>
        ))}
      </div>
    </div>
  )
}
