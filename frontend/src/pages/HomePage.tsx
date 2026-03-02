import { Link } from 'react-router'
import { Radio, RotateCcw, FileText, ArrowRight } from 'lucide-react'
import { Button, buttonVariants } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { cn } from '@/lib/utils'

const features = [
  {
    icon: Radio,
    title: 'Event Tracking',
    description:
      'บันทึกทุก Event จาก Facebook Pixel โดยอัตโนมัติ ไม่พลาดแม้แต่ข้อมูลเดียว',
  },
  {
    icon: RotateCcw,
    title: 'Backup & Replay',
    description:
      'สำรองข้อมูลและส่งซ้ำไปยัง Pixel ใหม่ เมื่อบัญชีถูกระงับ',
  },
  {
    icon: FileText,
    title: 'Sale Pages',
    description:
      'สร้างหน้าขายที่รองรับหลาย Pixel พร้อมติดตาม Event อัตโนมัติ',
  },
]

const steps = [
  {
    number: '1',
    title: 'สร้าง Pixel',
    description: 'เพิ่ม Facebook Pixel ของคุณเข้าสู่ระบบ',
  },
  {
    number: '2',
    title: 'เชื่อมต่อ Sale Page',
    description: 'สร้างหน้าขายและเชื่อมต่อกับ Pixel',
  },
  {
    number: '3',
    title: 'สำรองข้อมูลอัตโนมัติ',
    description: 'ระบบจะบันทึกทุก Event และพร้อมส่งซ้ำเมื่อต้องการ',
  },
]

function scrollToSection(id: string) {
  const el = document.getElementById(id)
  if (el) {
    el.scrollIntoView({ behavior: 'smooth' })
  }
}

export function HomePage() {
  return (
    <div className="min-h-screen bg-background text-foreground">
      {/* Navbar */}
      <nav className="sticky top-0 z-50 border-b border-border bg-card/95 backdrop-blur supports-[backdrop-filter]:bg-card/80">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4 sm:px-6">
          <span className="text-xl font-bold text-foreground">Pixlinks</span>

          <div className="hidden sm:flex items-center gap-6">
            <button
              onClick={() => scrollToSection('features')}
              className="text-sm text-muted-foreground hover:text-foreground transition-colors"
            >
              Features
            </button>
            <button
              onClick={() => scrollToSection('how-it-works')}
              className="text-sm text-muted-foreground hover:text-foreground transition-colors"
            >
              How It Works
            </button>
          </div>

          <Link
            to="/login"
            className={buttonVariants({ size: 'sm' })}
          >
            Get Started
          </Link>
        </div>
      </nav>

      {/* Hero Section */}
      <section className="mx-auto max-w-4xl px-4 py-20 sm:px-6 sm:py-28 md:py-36 text-center">
        <h1 className="text-4xl font-bold tracking-tight sm:text-5xl lg:text-6xl">
          ปกป้องข้อมูล Facebook Pixel
          <br className="hidden sm:block" />
          ของคุณ
        </h1>
        <p className="mx-auto mt-6 max-w-2xl text-lg text-muted-foreground">
          บันทึก สำรอง และส่งซ้ำข้อมูล Pixel Event อัตโนมัติ
          เพื่อให้ธุรกิจของคุณไม่สะดุด แม้บัญชีโฆษณาจะถูกระงับ
        </p>
        <div className="mt-10 flex flex-col sm:flex-row items-center justify-center gap-4">
          <Link
            to="/login"
            className={cn(buttonVariants({ size: 'lg' }), 'gap-2')}
          >
            เริ่มต้นใช้งาน
            <ArrowRight className="h-4 w-4" />
          </Link>
          <Button
            variant="outline"
            size="lg"
            onClick={() => scrollToSection('features')}
          >
            เรียนรู้เพิ่มเติม
          </Button>
        </div>
      </section>

      {/* Features Section */}
      <section id="features" className="border-t border-border bg-muted/40 py-20 sm:py-28">
        <div className="mx-auto max-w-6xl px-4 sm:px-6">
          <div className="text-center">
            <h2 className="text-3xl font-bold tracking-tight sm:text-4xl">
              ฟีเจอร์หลัก
            </h2>
            <p className="mt-4 text-muted-foreground">
              ทุกสิ่งที่คุณต้องการเพื่อปกป้องข้อมูล Pixel
            </p>
          </div>

          <div className="mt-14 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
            {features.map((feature) => (
              <Card key={feature.title} className="border-border">
                <CardContent className="pt-6">
                  <div className="mb-4 inline-flex rounded-lg bg-muted p-3">
                    <feature.icon className="h-6 w-6 text-foreground" />
                  </div>
                  <h3 className="text-lg font-semibold">{feature.title}</h3>
                  <p className="mt-2 text-sm text-muted-foreground leading-relaxed">
                    {feature.description}
                  </p>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      </section>

      {/* How It Works Section */}
      <section id="how-it-works" className="py-20 sm:py-28">
        <div className="mx-auto max-w-6xl px-4 sm:px-6">
          <div className="text-center">
            <h2 className="text-3xl font-bold tracking-tight sm:text-4xl">
              วิธีการใช้งาน
            </h2>
            <p className="mt-4 text-muted-foreground">
              เริ่มต้นใช้งานได้ง่ายๆ ใน 3 ขั้นตอน
            </p>
          </div>

          <div className="mt-14 grid gap-10 sm:grid-cols-3">
            {steps.map((step) => (
              <div key={step.number} className="text-center">
                <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-primary text-primary-foreground text-lg font-bold">
                  {step.number}
                </div>
                <h3 className="text-lg font-semibold">{step.title}</h3>
                <p className="mt-2 text-sm text-muted-foreground leading-relaxed">
                  {step.description}
                </p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-border py-10">
        <div className="mx-auto flex max-w-6xl flex-col items-center gap-6 px-4 sm:flex-row sm:justify-between sm:px-6">
          <div className="flex items-center gap-2">
            <span className="font-bold text-foreground">Pixlinks</span>
            <span className="text-sm text-muted-foreground">
              &copy; {new Date().getFullYear()}
            </span>
          </div>
          <div className="flex items-center gap-6 text-sm text-muted-foreground">
            <button
              onClick={() => scrollToSection('features')}
              className="hover:text-foreground transition-colors"
            >
              Features
            </button>
            <button
              onClick={() => scrollToSection('how-it-works')}
              className="hover:text-foreground transition-colors"
            >
              How It Works
            </button>
            <Link to="/login" className="hover:text-foreground transition-colors">
              Login
            </Link>
          </div>
        </div>
      </footer>
    </div>
  )
}
