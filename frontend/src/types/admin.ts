import type { Purchase, ReplayCredit, Subscription } from '@/types'

export interface AdminCustomer {
  id: string
  email: string
  name: string
  plan: string
  is_admin: boolean
  suspended_at?: string
  stripe_customer_id?: string
  created_at: string
  updated_at: string
}

export interface AdminCustomerDetail {
  customer: AdminCustomer & { api_key: string }
  pixel_count: number
  event_count: number
  sale_page_count: number
  replay_count: number
  purchases: Purchase[]
  credits: ReplayCredit[]
  subscriptions: Subscription[]
}

export interface AdminAnalytics {
  total_customers: number
  active_customers: number
  suspended_customers: number
  total_pixels: number
  events_today: number
  events_this_month: number
  total_replays: number
  successful_replays: number
  failed_replays: number
  total_revenue_thb: number
  revenue_this_month_thb: number
  customers_by_plan: Record<string, number>
}

export interface AdminPurchase {
  id: string
  customer_id: string
  customer_email: string
  customer_name: string
  pack_type: string
  amount_satang: number
  currency: string
  status: string
  created_at: string
  completed_at?: string
}

export interface AdminSubscription {
  id: string
  customer_id: string
  customer_email: string
  customer_name: string
  addon_type: string
  status: string
  current_period_start?: string
  current_period_end?: string
  cancel_at_period_end: boolean
}

export interface AdminCreditGrant {
  id: string
  admin_id: string
  customer_id: string
  customer_email: string
  customer_name: string
  pack_type: string
  total_replays: number
  max_events_per_replay: number
  expires_at: string
  reason?: string
  credit_id?: string
  created_at: string
}

export interface RevenueChartPoint {
  date: string
  amount_satang: number
  purchase_count: number
}

export interface GrowthChartPoint {
  date: string
  new_customers: number
  total_customers: number
}

// F1: Admin Sale Pages
export interface AdminSalePage {
  id: string
  customer_id: string
  pixel_ids: string[]
  name: string
  slug: string
  template_name: string
  is_published: boolean
  created_at: string
  updated_at: string
  customer_email: string
  customer_name: string
  event_count: number
}

export interface AdminSalePageDetail {
  sale_page: {
    id: string
    customer_id: string
    pixel_ids: string[]
    name: string
    slug: string
    template_name: string
    content: unknown
    is_published: boolean
    created_at: string
    updated_at: string
  }
  customer_email: string
  customer_name: string
  linked_pixels: Array<{ id: string; name: string; fb_pixel_id: string; is_active: boolean }>
  event_count: number
}

// F2: Admin Pixels
export interface AdminPixel {
  id: string
  customer_id: string
  fb_pixel_id: string
  name: string
  is_active: boolean
  created_at: string
  updated_at: string
  customer_email: string
  customer_name: string
  event_count: number
  sale_page_count: number
}

export interface AdminPixelDetail {
  pixel: {
    id: string
    customer_id: string
    fb_pixel_id: string
    name: string
    is_active: boolean
    created_at: string
    updated_at: string
  }
  customer_email: string
  customer_name: string
  event_count: number
  linked_sale_pages: Array<{ id: string; name: string; slug: string; is_published: boolean }>
}

// F3: Admin Replay Sessions
export interface AdminReplaySession {
  id: string
  customer_id: string
  source_pixel_id: string
  target_pixel_id: string
  status: string
  total_events: number
  replayed_events: number
  failed_events: number
  created_at: string
  started_at?: string
  completed_at?: string
  cancelled_at?: string
  customer_email: string
  customer_name: string
  source_pixel_name: string
  target_pixel_name: string
}

export interface AdminReplaySessionDetail {
  session: AdminReplaySession & {
    event_types?: string[]
    date_from?: string
    date_to?: string
    error_message?: string
  }
  customer_email: string
  customer_name: string
  source_pixel: { id: string; name: string; fb_pixel_id: string; is_active: boolean }
  target_pixel: { id: string; name: string; fb_pixel_id: string; is_active: boolean }
}

// F4: Admin Events
export interface AdminEvent {
  id: string
  pixel_id: string
  event_name: string
  event_data: unknown
  user_data?: unknown
  source_url?: string
  event_time: string
  forwarded_to_capi: boolean
  capi_response_code?: number
  created_at: string
  pixel_name: string
  customer_email: string
  customer_name: string
}

export interface AdminEventStats {
  total_today: number
  total_this_hour: number
  capi_success_rate: number
  capi_failure_count: number
  top_event_types: Array<{ event_name: string; count: number }>
  timeseries: Array<{
    timestamp: string
    event_count: number
    capi_success: number
    capi_failure: number
  }>
}

// F5: Audit Log
export interface AuditLogEntry {
  id: string
  admin_id: string
  admin_email: string
  action: string
  target_type: string
  target_id: string
  target_customer_id?: string
  customer_email?: string
  details?: unknown
  created_at: string
}
