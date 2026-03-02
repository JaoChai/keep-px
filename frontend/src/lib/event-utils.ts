export function eventBadgeVariant(name: string) {
  switch (name) {
    case 'PageView':
      return 'secondary' as const
    case 'Purchase':
      return 'success' as const
    case 'Lead':
    case 'CompleteRegistration':
      return 'default' as const
    case 'AddToCart':
    case 'InitiateCheckout':
      return 'warning' as const
    default:
      return 'outline' as const
  }
}

const EVENT_TYPE_COLORS: Record<string, string> = {
  PageView: 'bg-muted-foreground',
  Purchase: 'bg-emerald-500',
  Lead: 'bg-primary',
  CompleteRegistration: 'bg-primary/80',
  AddToCart: 'bg-amber-500',
  InitiateCheckout: 'bg-amber-400',
}

export function getEventColor(name: string): string {
  return EVENT_TYPE_COLORS[name] ?? 'bg-muted-foreground'
}
