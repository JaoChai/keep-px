import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function truncateMessage(msg: string, max = 200): string {
  if (msg.length <= max) return msg
  return msg.slice(0, max) + '...'
}
