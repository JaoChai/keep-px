import { useState } from 'react'
import { Bell } from 'lucide-react'
import { Popover, PopoverTrigger, PopoverContent } from '@/components/ui/popover'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useNotifications, useUnreadCount, useMarkRead, useMarkAllRead } from '@/hooks/use-notifications'
import type { AppNotification } from '@/types'

function timeAgo(dateStr: string): string {
  const seconds = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000)
  if (seconds < 60) return 'เมื่อสักครู่'
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes} นาทีที่แล้ว`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours} ชั่วโมงที่แล้ว`
  const days = Math.floor(hours / 24)
  return `${days} วันที่แล้ว`
}

function NotificationItem({
  notification,
  onMarkRead,
}: {
  notification: AppNotification
  onMarkRead: (id: string) => void
}) {
  return (
    <button
      className="flex w-full items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-accent"
      onClick={() => {
        if (!notification.is_read) onMarkRead(notification.id)
      }}
    >
      <div className="mt-1.5 flex-shrink-0">
        {!notification.is_read ? (
          <span className="block h-2 w-2 rounded-full bg-primary" />
        ) : (
          <span className="block h-2 w-2" />
        )}
      </div>
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium text-foreground truncate">{notification.title}</p>
        <p className="text-xs text-muted-foreground line-clamp-2">{notification.body}</p>
        <p className="mt-1 text-xs text-muted-foreground/70">{timeAgo(notification.created_at)}</p>
      </div>
    </button>
  )
}

export function NotificationBell() {
  const [open, setOpen] = useState(false)
  const { data: unreadCount = 0 } = useUnreadCount()
  const { data: notifData } = useNotifications(open)
  const markRead = useMarkRead()
  const markAllRead = useMarkAllRead()

  const notifications = notifData?.notifications ?? []
  const displayCount = unreadCount > 9 ? '9+' : unreadCount

  return (
    <Popover>
      <PopoverTrigger
        className="relative rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
        aria-label="Notifications"
        onClick={() => setOpen(!open)}
      >
        <Bell className="h-5 w-5" />
        {unreadCount > 0 && (
          <span className="absolute -right-1 -top-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-bold text-primary-foreground">
            {displayCount}
          </span>
        )}
      </PopoverTrigger>

      <PopoverContent align="end" className="w-80 p-0">
        <div className="flex items-center justify-between border-b border-border px-4 py-3">
          <h3 className="text-sm font-semibold text-foreground">Notifications</h3>
          {unreadCount > 0 && (
            <button
              className="text-xs text-muted-foreground hover:text-foreground transition-colors"
              onClick={() => markAllRead.mutate()}
            >
              Mark all read
            </button>
          )}
        </div>

        <ScrollArea maxHeight="400px">
          {notifications.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
              <Bell className="mb-2 h-8 w-8 opacity-30" />
              <p className="text-sm">No notifications</p>
            </div>
          ) : (
            <div className="divide-y divide-border">
              {notifications.map((n) => (
                <NotificationItem
                  key={n.id}
                  notification={n}
                  onMarkRead={(id) => markRead.mutate(id)}
                />
              ))}
            </div>
          )}
        </ScrollArea>
      </PopoverContent>
    </Popover>
  )
}
