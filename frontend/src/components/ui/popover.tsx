import * as React from 'react'
import { useEffect, useRef, useState } from 'react'
import { cn } from '@/lib/utils'

interface PopoverContextValue {
  open: boolean
  setOpen: (open: boolean) => void
  triggerRef: React.RefObject<HTMLButtonElement | null>
}

const PopoverContext = React.createContext<PopoverContextValue | null>(null)

function usePopover() {
  const ctx = React.useContext(PopoverContext)
  if (!ctx) throw new Error('usePopover must be used within Popover')
  return ctx
}

export function Popover({
  children,
  onOpenChange,
}: {
  children: React.ReactNode
  onOpenChange?: (open: boolean) => void
}) {
  const [open, setOpenInternal] = useState(false)
  const triggerRef = useRef<HTMLButtonElement | null>(null)

  const setOpen = (next: boolean) => {
    setOpenInternal(next)
    onOpenChange?.(next)
  }

  return (
    <PopoverContext.Provider value={{ open, setOpen, triggerRef }}>
      <div className="relative">{children}</div>
    </PopoverContext.Provider>
  )
}

export function PopoverTrigger({ children, className, ...props }: React.ButtonHTMLAttributes<HTMLButtonElement>) {
  const { open, setOpen, triggerRef } = usePopover()
  return (
    <button
      ref={triggerRef}
      type="button"
      aria-expanded={open}
      className={className}
      {...props}
      onClick={(e) => {
        setOpen(!open)
        props.onClick?.(e)
      }}
    >
      {children}
    </button>
  )
}

const alignClasses = {
  start: 'left-0',
  center: 'left-1/2 -translate-x-1/2',
  end: 'right-0',
} as const

export function PopoverContent({
  children,
  className,
  align = 'end',
  ...props
}: React.HTMLAttributes<HTMLDivElement> & { align?: 'start' | 'center' | 'end' }) {
  const { open, setOpen, triggerRef } = usePopover()
  const contentRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const handleClickOutside = (e: MouseEvent) => {
      const target = e.target as Node
      if (
        contentRef.current && !contentRef.current.contains(target) &&
        triggerRef.current && !triggerRef.current.contains(target)
      ) {
        setOpen(false)
      }
    }
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', handleClickOutside)
    document.addEventListener('keydown', handleEscape)
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
      document.removeEventListener('keydown', handleEscape)
    }
  }, [open, setOpen, triggerRef])

  if (!open) return null

  return (
    <div
      ref={contentRef}
      className={cn(
        'absolute top-full z-50 mt-2 w-80 rounded-lg border border-border bg-card shadow-lg animate-in fade-in-0 zoom-in-95',
        alignClasses[align],
        className
      )}
      {...props}
    >
      {children}
    </div>
  )
}
