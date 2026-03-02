import { Component } from 'react'
import type { ReactNode, ErrorInfo } from 'react'
import { Button } from '@/components/ui/button'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError(): State {
    return { hasError: true }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('ErrorBoundary caught:', error, errorInfo)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="min-h-screen flex items-center justify-center bg-background">
          <div className="rounded-lg border border-border bg-card p-8 text-center max-w-md shadow-sm">
            <h2 className="text-lg font-semibold text-foreground mb-2">
              เกิดข้อผิดพลาดที่ไม่คาดคิด
            </h2>
            <p className="text-sm text-muted-foreground mb-4">
              กรุณาลองโหลดหน้าเว็บใหม่อีกครั้ง
            </p>
            <Button onClick={() => window.location.reload()}>
              โหลดใหม่
            </Button>
          </div>
        </div>
      )
    }

    return this.props.children
  }
}
