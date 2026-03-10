import { useState, useMemo, useCallback } from 'react'
import {
  Search,
  BookOpen,
  Radio,
  Activity,
  RotateCcw,
  FileText,
  CreditCard,
  Settings,
  ChevronDown,
  ChevronRight,
  Zap,
  Globe,
  Eye,
  Play,
  AlertTriangle,
  CheckCircle2,
  LogIn,
} from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface GuideSection {
  id: string
  icon: React.ElementType
  title: string
  badge?: string
  subsections: GuideSubsection[]
}

interface GuideSubsection {
  id: string
  title: string
  content: React.ReactNode
}

// ---------------------------------------------------------------------------
// Flow Step Component
// ---------------------------------------------------------------------------

function FlowStep({ step, label, last }: { step: number; label: string; last?: boolean }) {
  return (
    <div className="flex items-start gap-3">
      <div className="flex flex-col items-center">
        <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-bold text-primary-foreground">
          {step}
        </div>
        {!last && <div className="w-px grow bg-border mt-1 min-h-[20px]" />}
      </div>
      <p className="text-sm text-foreground pt-1">{label}</p>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Info Box Component
// ---------------------------------------------------------------------------

function InfoBox({ type, children }: { type: 'tip' | 'warning' | 'important'; children: React.ReactNode }) {
  const styles = {
    tip: 'border-emerald-500/30 bg-emerald-500/5 text-emerald-700 dark:text-emerald-400',
    warning: 'border-amber-500/30 bg-amber-500/5 text-amber-700 dark:text-amber-400',
    important: 'border-blue-500/30 bg-blue-500/5 text-blue-700 dark:text-blue-400',
  }
  const icons = {
    tip: CheckCircle2,
    warning: AlertTriangle,
    important: Zap,
  }
  const labels = { tip: 'เคล็ดลับ', warning: 'คำเตือน', important: 'สำคัญ' }
  const Icon = icons[type]
  return (
    <div className={cn('flex gap-3 rounded-lg border p-3 text-sm', styles[type])}>
      <Icon className="h-4 w-4 mt-0.5 shrink-0" />
      <div>
        <span className="font-semibold">{labels[type]}:</span> {children}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Table Component
// ---------------------------------------------------------------------------

function GuideTable({ headers, rows }: { headers: string[]; rows: string[][] }) {
  return (
    <div className="overflow-x-auto rounded-lg border border-border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border bg-muted/50">
            {headers.map((h, i) => (
              <th key={i} className="px-4 py-2 text-left font-medium text-muted-foreground">{h}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, i) => (
            <tr key={i} className="border-b border-border last:border-b-0">
              {row.map((cell, j) => (
                <td key={j} className="px-4 py-2 text-foreground">{cell}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Guide Content Data
// ---------------------------------------------------------------------------

const guideSections: GuideSection[] = [
  {
    id: 'getting-started',
    icon: LogIn,
    title: 'เริ่มต้นใช้งาน',
    badge: 'เริ่มที่นี่',
    subsections: [
      {
        id: 'login',
        title: 'สมัครและเข้าสู่ระบบ',
        content: (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground leading-relaxed">
              เข้าเว็บ Keep-PX แล้วกด <strong className="text-foreground">"เข้าสู่ระบบด้วย Google"</strong> ระบบจะสร้างบัญชีให้อัตโนมัติจาก Google Account เมื่อเข้าสู่ระบบสำเร็จจะถูกพาไปที่หน้าแดชบอร์ดทันที
            </p>
          </div>
        ),
      },
      {
        id: 'first-steps',
        title: 'หลังเข้าสู่ระบบครั้งแรก',
        content: (
          <div className="space-y-2">
            <p className="text-sm text-muted-foreground mb-3">ขั้นตอนแนะนำหลังล็อกอินครั้งแรก:</p>
            <FlowStep step={1} label="ไปที่ Pixels → สร้าง Pixel ตัวแรก" />
            <FlowStep step={2} label="ไปที่เซลเพจ → สร้างเซลเพจแรก เชื่อม Pixel ที่สร้างไว้" />
            <FlowStep step={3} label="เผยแพร่เซลเพจ → แชร์ลิงก์ให้ลูกค้า" />
            <FlowStep step={4} label="กลับมาดูข้อมูลที่หน้า Events และ แดชบอร์ด" last />
          </div>
        ),
      },
    ],
  },
  {
    id: 'dashboard',
    icon: Eye,
    title: 'แดชบอร์ด',
    subsections: [
      {
        id: 'dashboard-overview',
        title: 'ตัวเลขสรุป',
        content: (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground leading-relaxed">หน้าแดชบอร์ดแสดงภาพรวมทั้งหมดผ่าน 5 การ์ดหลัก:</p>
            <GuideTable
              headers={['การ์ด', 'ความหมาย']}
              rows={[
                ['Active Pixels', 'จำนวน Pixel ที่เปิดใช้งาน / ทั้งหมด'],
                ['Events Today', 'จำนวน Event วันนี้ (แสดงแนวโน้มเทียบวันก่อน)'],
                ['CAPI Rate', 'อัตราส่ง Event ไป Facebook สำเร็จ'],
                ['Events This Week', 'จำนวน Event สัปดาห์นี้'],
                ['Active Replays', 'จำนวน Replay ที่กำลังทำงาน'],
              ]}
            />
            <InfoBox type="tip">
              การ์ด CAPI Rate แสดงสีตามสถานะ: เขียว = ดี, เหลือง = ปานกลาง, แดง = มีปัญหาควรตรวจสอบ
            </InfoBox>
          </div>
        ),
      },
      {
        id: 'dashboard-quota',
        title: 'โควตา Event รายเดือน',
        content: (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground leading-relaxed">
              แถบแสดงจำนวน Event ที่ใช้ไปเทียบกับ Limit ของแพ็กเกจ ถ้าใกล้เต็มให้พิจารณาอัปเกรดแพ็กเกจหรือซื้อ Add-on เพิ่ม
            </p>
          </div>
        ),
      },
      {
        id: 'dashboard-chart',
        title: 'กราฟและข้อมูล',
        content: (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground leading-relaxed">
              กราฟ Event Volume แสดงแนวโน้มตามเวลา เลือกดูได้ 7 วัน, 14 วัน, 30 วัน, 90 วัน นอกจากนี้ยังมีกิจกรรมล่าสุด, สถานะ Pixel ทั้งหมด และ Event ยอดนิยม
            </p>
          </div>
        ),
      },
    ],
  },
  {
    id: 'pixels',
    icon: Radio,
    title: 'จัดการ Pixel',
    subsections: [
      {
        id: 'pixel-create',
        title: 'สร้าง Pixel ใหม่',
        content: (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground leading-relaxed">กดปุ่ม "สร้าง Pixel" แล้วกรอกข้อมูลดังนี้:</p>
            <GuideTable
              headers={['ช่อง', 'คำอธิบาย', 'ตัวอย่าง']}
              rows={[
                ['ชื่อ Pixel', 'ชื่อที่ตั้งเอง ไว้จำง่าย ๆ', 'Pixel โฆษณาเสื้อผ้า'],
                ['Facebook Pixel ID', 'รหัส Pixel จาก Events Manager', '123456789012345'],
                ['Access Token', 'โทเค็นสำหรับส่ง CAPI', 'EAAxxxxxxxx...'],
                ['Test Event Code', '(ไม่บังคับ) โค้ดทดสอบ', 'TEST12345'],
                ['Backup Pixel', '(ไม่บังคับ) Pixel สำรอง', 'เลือกจากรายการ'],
              ]}
            />
          </div>
        ),
      },
      {
        id: 'pixel-find-credentials',
        title: 'หา Pixel ID และ Access Token',
        content: (
          <div className="space-y-2">
            <p className="text-sm text-muted-foreground mb-3">ค้นหาข้อมูลได้จาก Facebook Events Manager:</p>
            <FlowStep step={1} label="เข้า Facebook Events Manager → เลือก Pixel ของคุณ" />
            <FlowStep step={2} label="Pixel ID: แสดงอยู่ใต้ชื่อ Pixel (เลข 15-16 หลัก)" />
            <FlowStep step={3} label="Access Token: ไปที่ Settings → Generate Access Token" last />
          </div>
        ),
      },
      {
        id: 'pixel-actions',
        title: 'การจัดการ Pixel',
        content: (
          <div className="space-y-4">
            <GuideTable
              headers={['ปุ่ม', 'ทำอะไร']}
              rows={[
                ['Test Connection', 'ทดสอบเชื่อมต่อ Facebook CAPI'],
                ['แก้ไข', 'แก้ไขข้อมูล Pixel'],
                ['ลบ', 'ลบ Pixel (Event ที่เก็บแล้วยังอยู่)'],
                ['สลับสถานะ', 'เปิด/ปิด Pixel ชั่วคราว'],
              ]}
            />
          </div>
        ),
      },
      {
        id: 'pixel-backup',
        title: 'Backup Pixel',
        content: (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground leading-relaxed">
              เมื่อเปิด Backup Pixel ระบบจะส่ง Event ไปยัง Pixel สำรอง <strong className="text-foreground">พร้อมกัน</strong> กับ Pixel หลักผ่าน CAPI ถ้า Pixel หลักโดนแบน ยังมีข้อมูลอยู่ใน Pixel สำรอง
            </p>
            <InfoBox type="important">
              ต้องสร้าง Pixel ที่ 2 ก่อน แล้วค่อยแก้ไข Pixel ที่ 1 เพื่อเลือก Pixel ที่ 2 เป็น Backup
            </InfoBox>
          </div>
        ),
      },
    ],
  },
  {
    id: 'events',
    icon: Activity,
    title: 'ดู Events',
    subsections: [
      {
        id: 'events-live',
        title: 'โหมด Live (เรียลไทม์)',
        content: (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground leading-relaxed">
              ดู Event ที่เข้ามาแบบสด ๆ ขณะที่ลูกค้าเข้าเซลเพจ
            </p>
            <div className="flex flex-wrap gap-2">
              <Badge variant="secondary" className="gap-1"><Play className="h-3 w-3" /> Play/Pause</Badge>
              <Badge variant="secondary" className="gap-1">Refresh</Badge>
              <Badge variant="secondary" className="gap-1">Clear</Badge>
            </div>
            <GuideTable
              headers={['คอลัมน์', 'ตัวอย่าง']}
              rows={[
                ['Event Name', 'PageView, Purchase, Lead, ViewContent'],
                ['Pixel', 'ชื่อ Pixel ที่รับ Event'],
                ['Source URL', 'หน้าเว็บที่ Event เกิด'],
                ['CAPI', 'ส่งสำเร็จ / ส่งไม่สำเร็จ'],
                ['เวลา', '2 นาทีที่แล้ว'],
              ]}
            />
          </div>
        ),
      },
      {
        id: 'events-history',
        title: 'โหมด History (ย้อนหลัง)',
        content: (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground leading-relaxed">
              ดู Event ย้อนหลังทั้งหมด มี Filter ตาม Pixel และแบ่งหน้า (50 Event/หน้า)
            </p>
          </div>
        ),
      },
    ],
  },
  {
    id: 'replay',
    icon: RotateCcw,
    title: 'Replay Center',
    badge: 'สำคัญ',
    subsections: [
      {
        id: 'replay-what',
        title: 'Replay คืออะไร?',
        content: (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground leading-relaxed">
              Replay คือการนำ Event ที่เก็บไว้ในระบบ ส่งซ้ำไปยัง Pixel ตัวใหม่ผ่าน Facebook CAPI
            </p>
            <p className="text-sm font-medium text-foreground">ใช้เมื่อไหร่?</p>
            <ul className="text-sm text-muted-foreground space-y-1 ml-4 list-disc">
              <li>แอดเคาท์โดนแบน ต้องย้ายข้อมูลไปยัง Pixel ใหม่</li>
              <li>อยากส่ง Event ซ้ำเพื่อให้ Facebook เรียนรู้ข้อมูลได้เร็วขึ้น</li>
            </ul>
          </div>
        ),
      },
      {
        id: 'replay-create',
        title: 'วิธีสร้าง Replay',
        content: (
          <div className="space-y-4">
            <GuideTable
              headers={['ช่อง', 'คำอธิบาย']}
              rows={[
                ['Source Pixel', 'Pixel ต้นทาง (ดึง Event จาก Pixel นี้)'],
                ['Target Pixel', 'Pixel ปลายทาง (ส่ง Event ไปยัง Pixel นี้)'],
                ['Event Type', '(ไม่บังคับ) เลือก Replay เฉพาะบางประเภท'],
                ['Date Range', '(ไม่บังคับ) กำหนดช่วงเวลา'],
                ['Time Mode', 'Original = เวลาเดิม / Current = เวลาปัจจุบัน'],
                ['Batch Delay', 'หน่วงเวลาระหว่างชุด (0-60,000 ms)'],
              ]}
            />
            <InfoBox type="warning">
              Event เก่ากว่า 7 วัน ควรเลือก Time Mode = "Current" เพราะ Facebook อาจปฏิเสธ Event ที่เก่าเกินไป
            </InfoBox>
          </div>
        ),
      },
      {
        id: 'replay-status',
        title: 'สถานะ Replay',
        content: (
          <div className="space-y-4">
            <GuideTable
              headers={['สถานะ', 'ความหมาย']}
              rows={[
                ['Pending', 'รอเริ่ม'],
                ['Running', 'กำลังส่ง Event อยู่'],
                ['Completed', 'ส่งเสร็จเรียบร้อย'],
                ['Failed', 'ส่งไม่สำเร็จ (ดูข้อผิดพลาด)'],
                ['Cancelled', 'ถูกยกเลิก'],
              ]}
            />
            <p className="text-sm text-muted-foreground">
              ปุ่ม <strong className="text-foreground">Cancel</strong> ยกเลิก Replay ที่กำลังทำงาน,{' '}
              <strong className="text-foreground">Retry Failed</strong> ส่ง Event ที่ล้มเหลวซ้ำ
            </p>
          </div>
        ),
      },
      {
        id: 'replay-credit',
        title: 'Replay Credit',
        content: (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground leading-relaxed">
              การ Replay แต่ละครั้งใช้ <strong className="text-foreground">1 Credit</strong> ดูจำนวนคงเหลือได้ที่หน้า Replay หรือหน้า Billing ซื้อเพิ่มได้ที่ Billing &rarr; แท็บ Replays
            </p>
          </div>
        ),
      },
    ],
  },
  {
    id: 'sale-pages',
    icon: FileText,
    title: 'เซลเพจ',
    subsections: [
      {
        id: 'salepage-what',
        title: 'เซลเพจคืออะไร?',
        content: (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground leading-relaxed">
              เซลเพจคือหน้าเว็บที่ Keep-PX สร้างและโฮสต์ให้ ใช้สำหรับแสดงสินค้า/บริการ และ <strong className="text-foreground">เก็บข้อมูล Pixel อัตโนมัติ</strong> เมื่อมีคนเข้าชม แชร์ลิงก์ไปยัง Social Media หรืออีเมลได้เลย
            </p>
          </div>
        ),
      },
      {
        id: 'salepage-templates',
        title: 'เลือก Template',
        content: (
          <div className="space-y-4">
            <GuideTable
              headers={['Template', 'คำอธิบาย']}
              rows={[
                ['Classic', 'แบบตายตัว กรอกข้อมูลตามช่อง เรียบง่าย'],
                ['Blocks', 'แบบลาก-วาง ปรับแต่งอิสระ ยืดหยุ่นกว่า'],
              ]}
            />
          </div>
        ),
      },
      {
        id: 'salepage-settings',
        title: 'ตั้งค่าเซลเพจ',
        content: (
          <div className="space-y-4">
            <p className="text-sm font-medium text-foreground">ข้อมูลพื้นฐาน</p>
            <GuideTable
              headers={['ช่อง', 'คำอธิบาย']}
              rows={[
                ['ชื่อหน้า', 'ชื่อภายใน ไว้จำ (ลูกค้าไม่เห็น)'],
                ['URL Slug', 'ส่วนท้ายของลิงก์ เช่น my-product → /p/my-product'],
                ['เชื่อม Pixel', 'เลือก Pixel ที่ต้องการเก็บข้อมูล (เลือกได้หลายตัว)'],
              ]}
            />
            <p className="text-sm font-medium text-foreground mt-4">การติดตาม (Tracking)</p>
            <GuideTable
              headers={['ช่อง', 'คำอธิบาย']}
              rows={[
                ['CTA Event', 'Event ที่ยิงเมื่อกดปุ่ม: Lead / Purchase / Contact / CompleteRegistration'],
                ['Content Name', 'ชื่อสินค้า (ส่งให้ Facebook)'],
                ['Content Value', 'ราคาสินค้า (ส่งให้ Facebook)'],
                ['Currency', 'สกุลเงิน: THB, USD'],
              ]}
            />
          </div>
        ),
      },
      {
        id: 'salepage-auto-events',
        title: 'Event ที่ยิงอัตโนมัติ',
        content: (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground leading-relaxed">เมื่อลูกค้าเข้าชมเซลเพจ ระบบจะยิง Event อัตโนมัติ:</p>
            <FlowStep step={1} label="ลูกค้าเปิดหน้า → ยิง PageView + ViewContent" />
            <FlowStep step={2} label="ลูกค้ากดปุ่ม CTA → ยิง Purchase / Lead / Contact" />
            <FlowStep step={3} label="ลูกค้ากดลิงก์ LINE / โทรศัพท์ → ยิง Contact" last />
            <InfoBox type="tip">
              ทุก Event ถูกส่งผ่าน Facebook CAPI โดยอัตโนมัติ ไม่ต้องเขียนโค้ดเพิ่ม
            </InfoBox>
          </div>
        ),
      },
      {
        id: 'salepage-publish',
        title: 'เผยแพร่และแชร์',
        content: (
          <div className="space-y-3">
            <GuideTable
              headers={['ปุ่ม', 'ความหมาย']}
              rows={[
                ['Save as Draft', 'บันทึกไว้ก่อน ยังไม่เผยแพร่'],
                ['Publish', 'เผยแพร่ทันที — ลูกค้าเข้าถึงได้ผ่านลิงก์'],
              ]}
            />
          </div>
        ),
      },
    ],
  },
  {
    id: 'billing',
    icon: CreditCard,
    title: 'การเงิน',
    subsections: [
      {
        id: 'billing-plans',
        title: 'แพ็กเกจ',
        content: (
          <div className="space-y-4">
            <GuideTable
              headers={['แพ็กเกจ', 'Event/เดือน', 'คำอธิบาย']}
              rows={[
                ['Sandbox (ฟรี)', 'จำกัด', 'ทดลองใช้งาน'],
                ['Launch', '1M', 'สำหรับเริ่มต้น'],
                ['Shield', '5M', 'สำหรับธุรกิจขนาดกลาง'],
                ['Vault', 'ไม่จำกัด', 'สำหรับธุรกิจขนาดใหญ่'],
              ]}
            />
          </div>
        ),
      },
      {
        id: 'billing-replay-packs',
        title: 'ซื้อ Replay Credit',
        content: (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground leading-relaxed">
              ซื้อ Replay Credit เพิ่มเติมได้ที่แท็บ Replays ในหน้าการเงิน มีแพ็ก 1 ครั้ง, 3 ครั้ง, และ Unlimited กดปุ่ม "Buy Pack" แล้วชำระเงินผ่าน Stripe
            </p>
          </div>
        ),
      },
      {
        id: 'billing-addons',
        title: 'Add-ons',
        content: (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground leading-relaxed">ซื้อเพิ่มเติมเป็น Subscription รายเดือน:</p>
            <GuideTable
              headers={['Add-on', 'ได้อะไร']}
              rows={[
                ['Events +1M', 'เพิ่มโควตา Event อีก 1 ล้าน/เดือน'],
                ['Sale Pages +10', 'เพิ่มเซลเพจอีก 10 หน้า'],
                ['Pixels +10', 'เพิ่ม Pixel อีก 10 ตัว'],
              ]}
            />
          </div>
        ),
      },
      {
        id: 'billing-manage',
        title: 'จัดการ Billing',
        content: (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground leading-relaxed">
              กดปุ่ม "Manage Billing" เพื่อเปิดหน้า Stripe Customer Portal สำหรับเปลี่ยนบัตรเครดิต, ดูใบเสร็จ, หรือยกเลิก Subscription
            </p>
          </div>
        ),
      },
    ],
  },
  {
    id: 'settings',
    icon: Settings,
    title: 'ตั้งค่าบัญชี',
    subsections: [
      {
        id: 'settings-profile',
        title: 'ข้อมูลโปรไฟล์',
        content: (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground leading-relaxed">
              แสดงชื่อ, อีเมล จาก Google Account และแพ็กเกจปัจจุบัน (อ่านอย่างเดียว)
            </p>
          </div>
        ),
      },
      {
        id: 'settings-api-key',
        title: 'API Key',
        content: (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground leading-relaxed">
              API Key ใช้สำหรับให้เซลเพจส่ง Event เข้าระบบ
            </p>
            <GuideTable
              headers={['ปุ่ม', 'ทำอะไร']}
              rows={[
                ['Show/Hide', 'แสดง/ซ่อน API Key'],
                ['Copy', 'คัดลอก API Key'],
                ['Regenerate', 'สร้าง Key ใหม่ (คีย์เก่าจะใช้ไม่ได้ทันที)'],
              ]}
            />
            <InfoBox type="warning">
              ถ้า Regenerate API Key เซลเพจที่ใช้คีย์เก่าจะส่ง Event ไม่ได้ ระบบจัดการให้อัตโนมัติสำหรับเซลเพจที่สร้างในระบบ
            </InfoBox>
          </div>
        ),
      },
    ],
  },
  {
    id: 'glossary',
    icon: BookOpen,
    title: 'คำศัพท์สำคัญ',
    subsections: [
      {
        id: 'glossary-pixel',
        title: 'Facebook Pixel',
        content: (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground leading-relaxed">
              โค้ดติดตามจาก Facebook สำหรับเก็บข้อมูลพฤติกรรมลูกค้า ประกอบด้วย <strong className="text-foreground">Pixel ID</strong> (ตัวเลข 15-16 หลัก) และ <strong className="text-foreground">Access Token</strong> (กุญแจสำหรับส่งข้อมูลไป Facebook)
            </p>
          </div>
        ),
      },
      {
        id: 'glossary-events',
        title: 'Event (เหตุการณ์)',
        content: (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground leading-relaxed">การกระทำที่ลูกค้าทำบนเว็บไซต์:</p>
            <GuideTable
              headers={['Event', 'ความหมาย']}
              rows={[
                ['PageView', 'เปิดหน้าเว็บ'],
                ['ViewContent', 'ดูเนื้อหาสินค้า'],
                ['Lead', 'สนใจสินค้า (กรอกฟอร์ม, กดดูรายละเอียด)'],
                ['Purchase', 'ซื้อสินค้า'],
                ['Contact', 'ติดต่อร้าน (กด LINE, กดโทร)'],
                ['CompleteRegistration', 'สมัครสมาชิกสำเร็จ'],
              ]}
            />
          </div>
        ),
      },
      {
        id: 'glossary-capi',
        title: 'Conversions API (CAPI)',
        content: (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground leading-relaxed">
              วิธีส่งข้อมูล Event ไป Facebook ผ่านเซิร์ฟเวอร์ (Server-to-Server) เสถียรกว่า Pixel บนเว็บ ไม่โดน Ad Blocker บล็อก ข้อมูลแม่นยำกว่า Keep-PX ใช้ CAPI เป็นวิธีหลักในการส่ง Event
            </p>
          </div>
        ),
      },
      {
        id: 'glossary-other',
        title: 'คำอื่น ๆ',
        content: (
          <div className="space-y-4">
            <GuideTable
              headers={['คำ', 'ความหมาย']}
              rows={[
                ['Backup Pixel', 'Pixel สำรองที่รับ Event พร้อม Pixel หลักผ่าน CAPI'],
                ['Replay', 'ส่ง Event ซ้ำไปยัง Pixel ตัวใหม่'],
                ['Replay Credit', 'หน่วยนับจำนวนครั้งที่ Replay ได้'],
                ['Event Quota', 'จำนวน Event สูงสุดต่อเดือน ขึ้นกับแพ็กเกจ'],
              ]}
            />
          </div>
        ),
      },
    ],
  },
  {
    id: 'scenarios',
    icon: Globe,
    title: 'สถานการณ์จริง',
    subsections: [
      {
        id: 'scenario-new',
        title: 'เริ่มต้นใช้งานครั้งแรก',
        content: (
          <div className="space-y-2">
            <FlowStep step={1} label="เข้าสู่ระบบด้วย Google" />
            <FlowStep step={2} label="ไปที่ Pixels → สร้าง Pixel (กรอก Pixel ID + Access Token)" />
            <FlowStep step={3} label="กด Test Connection ยืนยันว่าเชื่อมต่อ Facebook ได้" />
            <FlowStep step={4} label="ไปที่เซลเพจ → สร้างเซลเพจ → เชื่อม Pixel" />
            <FlowStep step={5} label="กด Publish → คัดลอกลิงก์ /p/xxx" />
            <FlowStep step={6} label="แชร์ลิงก์ไปยัง Facebook, LINE, อีเมล" />
            <FlowStep step={7} label="กลับมาดู Events → เปิดโหมด Live → ดู Event เข้ามาสด" last />
          </div>
        ),
      },
      {
        id: 'scenario-banned',
        title: 'แอดเคาท์โดนแบน — กู้คืนข้อมูล',
        content: (
          <div className="space-y-2">
            <FlowStep step={1} label="สร้างแอดเคาท์ Facebook ใหม่ + สร้าง Pixel ใหม่" />
            <FlowStep step={2} label="ไปที่ Pixels → สร้าง Pixel ใหม่ในระบบ" />
            <FlowStep step={3} label="ไปที่ Replay Center" />
            <FlowStep step={4} label="เลือก Source = Pixel เก่า, Target = Pixel ใหม่" />
            <FlowStep step={5} label="(ถ้า Event เก่ากว่า 7 วัน) เลือก Time Mode = Current" />
            <FlowStep step={6} label="กด Preview → ตรวจดูจำนวน Event → กดยืนยัน" />
            <FlowStep step={7} label="รอ Replay ทำงาน → ดูสถานะที่แผงขวา → เสร็จ!" last />
            <InfoBox type="tip">
              ถ้าอยากกรองเฉพาะ Event บางประเภท เช่น Purchase ให้เลือก Event Type Filter ก่อนกด Preview
            </InfoBox>
          </div>
        ),
      },
      {
        id: 'scenario-backup',
        title: 'ป้องกันล่วงหน้าด้วย Backup Pixel',
        content: (
          <div className="space-y-2">
            <FlowStep step={1} label="สร้าง Pixel หลัก (Pixel A)" />
            <FlowStep step={2} label="สร้าง Pixel สำรอง (Pixel B) ในแอดเคาท์อื่น" />
            <FlowStep step={3} label="แก้ไข Pixel A → เลือก Pixel B เป็น Backup" />
            <FlowStep step={4} label="ทุก Event จะถูกส่งไปทั้ง Pixel A และ B พร้อมกัน" />
            <FlowStep step={5} label="ถ้า Pixel A โดนแบน → Pixel B ยังมีข้อมูลครบ" last />
          </div>
        ),
      },
    ],
  },
]

// ---------------------------------------------------------------------------
// Main Component
// ---------------------------------------------------------------------------

export function GuidePage() {
  const [searchQuery, setSearchQuery] = useState('')
  const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set(['getting-started']))
  const [expandedSubsections, setExpandedSubsections] = useState<Set<string>>(new Set(['login', 'first-steps']))

  // Filter sections by search query
  const filteredSections = useMemo(() => {
    if (!searchQuery.trim()) return guideSections
    const q = searchQuery.toLowerCase()
    return guideSections
      .map((section) => {
        const sectionMatch = section.title.toLowerCase().includes(q)
        const matchedSubs = section.subsections.filter((sub) => sub.title.toLowerCase().includes(q))
        if (sectionMatch) return section
        if (matchedSubs.length > 0) return { ...section, subsections: matchedSubs }
        return null
      })
      .filter(Boolean) as GuideSection[]
  }, [searchQuery])

  const handleSearchChange = useCallback((value: string) => {
    setSearchQuery(value)
    if (value.trim()) {
      const q = value.toLowerCase()
      const matched = guideSections
        .map((section) => {
          const sectionMatch = section.title.toLowerCase().includes(q)
          const matchedSubs = section.subsections.filter((sub) => sub.title.toLowerCase().includes(q))
          if (sectionMatch || matchedSubs.length > 0) return section
          return null
        })
        .filter(Boolean) as GuideSection[]
      setExpandedSections(new Set(matched.map((s) => s.id)))
      setExpandedSubsections(new Set(matched.flatMap((s) => s.subsections.map((sub) => sub.id))))
    }
  }, [])

  const toggleSection = useCallback((id: string) => {
    setExpandedSections((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }, [])

  const toggleSubsection = useCallback((id: string) => {
    setExpandedSubsections((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }, [])

  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
        <div className="max-w-3xl mx-auto px-4 sm:px-6 py-6 sm:py-8">
          {/* Header */}
          <div className="mb-8">
            <div className="flex items-center gap-3 mb-2">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
                <BookOpen className="h-5 w-5 text-primary" />
              </div>
              <div>
                <h1 className="text-2xl font-bold text-foreground">คู่มือการใช้งาน</h1>
                <p className="text-sm text-muted-foreground">ทุกสิ่งที่ต้องรู้เกี่ยวกับ Keep-PX</p>
              </div>
            </div>
          </div>

          {/* Search */}
          <div className="relative mb-8">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              value={searchQuery}
              onChange={(e) => handleSearchChange(e.target.value)}
              placeholder="ค้นหาหัวข้อ... เช่น Pixel, Replay, เซลเพจ"
              className="pl-9 h-10"
            />
            {searchQuery && (
              <span className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">
                พบ {filteredSections.length} หัวข้อ
              </span>
            )}
          </div>

          {/* No Results */}
          {filteredSections.length === 0 && (
            <div className="text-center py-12">
              <Search className="h-10 w-10 text-muted-foreground/30 mx-auto mb-3" />
              <p className="text-sm text-muted-foreground">ไม่พบหัวข้อที่ค้นหา</p>
              <button
                onClick={() => setSearchQuery('')}
                className="text-sm text-primary hover:underline mt-1 cursor-pointer"
              >
                ล้างการค้นหา
              </button>
            </div>
          )}

          {/* Sections */}
          <div className="space-y-4">
            {filteredSections.map((section) => {
              const isExpanded = expandedSections.has(section.id)
              return (
                <div
                  key={section.id}
                  id={`section-${section.id}`}
                  data-section-id={section.id}
                  className="rounded-xl border border-border bg-card overflow-hidden"
                >
                  {/* Section Header */}
                  <button
                    onClick={() => toggleSection(section.id)}
                    className="flex items-center justify-between w-full px-5 py-4 hover:bg-accent/30 transition-colors cursor-pointer"
                  >
                    <div className="flex items-center gap-3">
                      <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-muted">
                        <section.icon className="h-4 w-4 text-foreground" />
                      </div>
                      <span className="text-base font-semibold text-foreground">{section.title}</span>
                      {section.badge && (
                        <Badge variant="secondary" className="text-xs">{section.badge}</Badge>
                      )}
                    </div>
                    <ChevronDown
                      className={cn(
                        'h-4 w-4 text-muted-foreground transition-transform duration-200',
                        isExpanded && 'rotate-180'
                      )}
                    />
                  </button>

                  {/* Subsections */}
                  {isExpanded && (
                    <div className="border-t border-border">
                      {section.subsections.map((sub, subIdx) => {
                        const isSubExpanded = expandedSubsections.has(sub.id)
                        return (
                          <div
                            key={sub.id}
                            className={cn(
                              subIdx < section.subsections.length - 1 && 'border-b border-border/50'
                            )}
                          >
                            <button
                              onClick={() => toggleSubsection(sub.id)}
                              className="flex items-center gap-2 w-full px-5 py-3 text-sm hover:bg-accent/20 transition-colors cursor-pointer"
                            >
                              <ChevronRight
                                className={cn(
                                  'h-3.5 w-3.5 text-muted-foreground transition-transform duration-200 shrink-0',
                                  isSubExpanded && 'rotate-90'
                                )}
                              />
                              <span className={cn(
                                'text-left font-medium',
                                isSubExpanded ? 'text-foreground' : 'text-muted-foreground'
                              )}>
                                {sub.title}
                              </span>
                            </button>
                            {isSubExpanded && (
                              <div className="px-5 pb-4 pl-10">
                                {sub.content}
                              </div>
                            )}
                          </div>
                        )
                      })}
                    </div>
                  )}
                </div>
              )
            })}
          </div>

          {/* Footer */}
          <div className="mt-8 mb-4 text-center">
            <p className="text-xs text-muted-foreground">
              ต้องการความช่วยเหลือเพิ่มเติม? ติดต่อทีมซัพพอร์ตได้ตลอดเวลา
            </p>
          </div>
        </div>
    </div>
  )
}

