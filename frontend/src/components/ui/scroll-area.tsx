import * as React from 'react'
import { cn } from '@/lib/utils'

interface ScrollAreaProps extends React.HTMLAttributes<HTMLDivElement> {
  maxHeight?: string
}

export function ScrollArea({ children, className, maxHeight = '400px', style, ...props }: ScrollAreaProps) {
  return (
    <div
      className={cn('overflow-y-auto', className)}
      style={{ maxHeight, ...style }}
      {...props}
    >
      {children}
    </div>
  )
}
