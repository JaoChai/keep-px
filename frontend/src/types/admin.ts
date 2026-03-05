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
