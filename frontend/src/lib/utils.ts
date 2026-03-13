import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'
import { formatDistanceToNow } from 'date-fns'
import { th } from 'date-fns/locale'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function truncateMessage(msg: string, max = 200): string {
  if (msg.length <= max) return msg
  return msg.slice(0, max) + '...'
}

export function formatBaht(satang: number): string {
  return `${(satang / 100).toLocaleString('th-TH')}`
}

const UNLIMITED_REPLAYS = -1

export function isUnlimited(value: number): boolean {
  return value === UNLIMITED_REPLAYS
}

export function timeAgo(date: string | Date): string {
  return formatDistanceToNow(new Date(date), { addSuffix: true, locale: th })
}

export function daysUntil(date: string | Date): number {
  const diff = new Date(date).getTime() - Date.now()
  return Math.ceil(diff / (1000 * 60 * 60 * 24))
}

export const PLAN_LABELS: Record<string, string> = {
  sandbox: 'Free',
  paid: 'Paid',
}

export const PACK_TYPE_NAMES: Record<string, string> = {
  replay_single: 'Replay Single (1 ครั้ง)',
  replay_monthly: 'Replay Unlimited (รายเดือน)',
  pixel_slots: 'Pixel Slots',
  plan_grant: 'Admin Grant',
}
