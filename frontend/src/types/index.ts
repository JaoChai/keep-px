export interface Customer {
  id: string
  email: string
  name: string
  api_key: string
  plan: string
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
  created_at: string
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
  pixel_id: string | null
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

export interface CustomDomain {
  id: string
  customer_id: string
  sale_page_id: string
  domain: string
  cf_hostname_id: string | null
  verification_token: string
  dns_verified: boolean
  ssl_active: boolean
  verified_at: string | null
  created_at: string
  updated_at: string
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
