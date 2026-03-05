export interface Customer {
  id: string
  email: string
  name: string
  api_key: string
  plan: string
  is_admin?: boolean
  suspended_at?: string
  created_at: string
  updated_at: string
}

export interface Pixel {
  id: string
  customer_id: string
  fb_pixel_id: string
  name: string
  is_active: boolean
  status: string
  backup_pixel_id?: string
  test_event_code?: string
  created_at: string
  updated_at: string
}

export interface PixelEvent {
  id: string
  pixel_id: string
  event_name: string
  event_data: Record<string, unknown>
  user_data?: Record<string, unknown>
  source_url?: string
  event_time: string
  forwarded_to_capi: boolean
  capi_response_code?: number
  created_at: string
}

export interface ReplaySession {
  id: string
  customer_id: string
  source_pixel_id: string
  target_pixel_id: string
  status: string
  total_events: number
  replayed_events: number
  failed_events: number
  event_types?: string[]
  date_from?: string
  date_to?: string
  time_mode: string
  batch_delay_ms: number
  error_message?: string
  started_at?: string
  completed_at?: string
  cancelled_at?: string
  failed_batch_ranges?: Array<{ start: number; end: number }>
  created_at: string
}

export interface ReplayPreview {
  total_events: number
  sample_events: PixelEvent[]
  warning?: string
}

export interface PageStyle {
  bg_color?: string
  accent_color?: string
  text_color?: string
  bg_image_url?: string
}

export interface PresetTheme {
  name: string
  style: PageStyle
}

export interface SalePageContent {
  hero: {
    title: string
    subtitle: string
    image_url: string
  }
  body: {
    description: string
    features: string[]
    images?: string[]
  }
  cta: {
    button_text: string
    button_link: string
  }
  contact: {
    line_id: string
    phone: string
    website_url?: string
  }
  tracking?: {
    cta_event_name: string
    content_name: string
    content_value: number
    currency: string
  }
  style?: PageStyle
}

// Block-based content (v2)
export type BlockType = 'image' | 'text' | 'button'
export type ButtonStyle = 'line' | 'website' | 'custom'

export interface Block {
  id: string
  type: BlockType
  image_url?: string
  link_url?: string
  text?: string
  button_style?: ButtonStyle
  button_text?: string
  button_url?: string
  button_value?: string
}

export interface TrackingConfig {
  cta_event_name: string
  content_name: string
  content_value: number
  currency: string
}

export interface SalePageContentV2 {
  version: 2
  blocks: Block[]
  tracking: TrackingConfig
  style?: PageStyle
}

export interface SalePage {
  id: string
  customer_id: string
  pixel_ids: string[]
  name: string
  slug: string
  template_name: string
  content: SalePageContent | SalePageContentV2
  is_published: boolean
  created_at: string
  updated_at: string
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  per_page: number
  total_pages: number
}

export interface APIResponse<T> {
  data?: T
  error?: string
  message?: string
}

export interface RealtimeEvent {
  id: string
  pixel_id: string
  pixel_name: string
  event_name: string
  source_url?: string
  forwarded_to_capi: boolean
  event_time: string
  created_at: string
}

export interface AuthTokens {
  access_token: string
  refresh_token: string
  customer: Customer
}

export interface AppNotification {
  id: string
  customer_id: string
  type: 'replay_completed' | 'replay_failed' | 'capi_auth_error' | 'system'
  title: string
  body: string
  metadata?: Record<string, unknown>
  is_read: boolean
  created_at: string
  read_at?: string
}

export interface NotificationListResult {
  notifications: AppNotification[]
  unread_count: number
}

// Billing types
export interface Purchase {
  id: string
  customer_id: string
  pack_type: string
  amount_satang: number
  currency: string
  status: string
  created_at: string
  completed_at?: string
}

export interface ReplayCredit {
  id: string
  customer_id: string
  pack_type: string
  total_replays: number
  used_replays: number
  max_events_per_replay: number
  expires_at: string
  created_at: string
}

export interface Subscription {
  id: string
  customer_id: string
  addon_type: string
  status: string
  current_period_start?: string
  current_period_end?: string
  cancel_at_period_end: boolean
}

export interface BillingOverview {
  plan: string
  purchases: Purchase[]
  credits: ReplayCredit[]
  subscriptions: Subscription[]
}

export interface CustomerQuota {
  plan: string
  max_pixels: number
  max_events_per_month: number
  events_used_this_month: number
  retention_days: number
  max_sale_pages: number
  can_replay: boolean
  remaining_replays: number
  max_events_per_replay: number
}
