interface PixlinksConfig {
  apiKey: string
  pixelId: string
  endpoint?: string
  debug?: boolean
}

interface EventPayload {
  event_name: string
  pixel_id: string
  event_data?: Record<string, unknown>
  user_data?: Record<string, unknown>
  source_url?: string
  event_time?: string
  event_id?: string
}

interface QueuedEvent {
  payload: EventPayload
  retries: number
}

function generateEventId(): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID()
  }
  // Fallback for older browsers
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0
    const v = c === 'x' ? r : (r & 0x3) | 0x8
    return v.toString(16)
  })
}

const DEFAULT_ENDPOINT = 'https://pixlinks.xyz'
const MAX_RETRIES = 3
const BATCH_INTERVAL = 2000
const MAX_BATCH_SIZE = 10

export class PixlinksTracker {
  private config: Required<PixlinksConfig>
  private queue: QueuedEvent[] = []
  private timer: ReturnType<typeof setInterval> | null = null
  private initialized = false

  constructor(config: PixlinksConfig) {
    if (!config.apiKey) {
      throw new Error('[Pixlinks] apiKey is required')
    }

    if (!config.pixelId) {
      throw new Error('[Pixlinks] pixelId is required')
    }

    this.config = {
      apiKey: config.apiKey,
      pixelId: config.pixelId,
      endpoint: config.endpoint || DEFAULT_ENDPOINT,
      debug: config.debug || false,
    }

    this.startBatchTimer()
    this.initialized = true
    this.log('Initialized')
  }

  track(eventName: string, eventData?: Record<string, unknown>, userData?: Record<string, unknown>, eventId?: string): void {
    if (!this.initialized) {
      console.warn('[Pixlinks] Not initialized')
      return
    }

    const payload: EventPayload = {
      event_name: eventName,
      pixel_id: this.config.pixelId,
      event_data: eventData || {},
      user_data: userData,
      source_url: window.location.href,
      event_time: new Date().toISOString(),
      event_id: eventId || generateEventId(),
    }

    this.queue.push({ payload, retries: 0 })
    this.log(`Queued event: ${eventName}`)

    if (this.queue.length >= MAX_BATCH_SIZE) {
      this.flush()
    }
  }

  trackPageView(userData?: Record<string, unknown>): void {
    this.track('PageView', {
      title: document.title,
      referrer: document.referrer,
    }, userData)
  }

  async flush(): Promise<void> {
    if (this.queue.length === 0) return

    const batch = this.queue.splice(0, MAX_BATCH_SIZE)
    const events = batch.map((item) => item.payload)

    try {
      const response = await fetch(`${this.config.endpoint}/api/v1/events/ingest`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-API-Key': this.config.apiKey,
        },
        body: JSON.stringify({ events }),
        keepalive: true,
      })

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`)
      }

      this.log(`Flushed ${events.length} events`)
    } catch (error) {
      this.log(`Flush failed: ${error}`, true)
      // Re-queue failed events with incremented retry count
      for (const item of batch) {
        if (item.retries < MAX_RETRIES) {
          this.queue.push({ ...item, retries: item.retries + 1 })
        } else {
          this.log(`Dropped event after ${MAX_RETRIES} retries: ${item.payload.event_name}`, true)
        }
      }
    }
  }

  destroy(): void {
    if (this.timer) {
      clearInterval(this.timer)
      this.timer = null
    }
    this.flush()
    this.initialized = false
    this.log('Destroyed')
  }

  private startBatchTimer(): void {
    this.timer = setInterval(() => {
      this.flush()
    }, BATCH_INTERVAL)
  }

  private log(message: string, isError = false): void {
    if (!this.config.debug) return
    const method = isError ? 'error' : 'log'
    console[method](`[Pixlinks] ${message}`)
  }
}
