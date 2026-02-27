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

export interface EventRule {
  id: string
  pixel_id: string
  page_url: string
  event_name: string
  trigger_type: string
  css_selector?: string
  xpath?: string
  element_text?: string
  conditions?: Record<string, unknown>
  parameters?: Record<string, unknown>
  fire_once: boolean
  delay_ms: number
  is_active: boolean
  created_at: string
  updated_at: string
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
  started_at?: string
  completed_at?: string
  created_at: string
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
}

// Block-based content (v2)
export type BlockType = 'image' | 'text' | 'button'
export type ButtonStyle = 'line' | 'website' | 'custom'

export interface Block {
  id: string
  type: BlockType
  image_url?: string
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

export interface AuthTokens {
  access_token: string
  refresh_token: string
  customer: Customer
}
