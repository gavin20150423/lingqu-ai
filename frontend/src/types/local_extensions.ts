import type { AccountPlatform, AccountType, FetchOptions } from './index'

// Local feature types retained while tracking upstream's core type catalog.
// Keeping these additive declarations separate makes future upstream merges
// less likely to drop the account-sharing, quota, store, and conversation UI.

export type ReceiptCodePaymentMethod = 'alipay' | 'wechat'

export interface ReceiptCode {
  id: number
  user_id: number
  payment_method: ReceiptCodePaymentMethod
  storage_provider: string
  url?: string | null
  content_type: string
  byte_size: number
  sha256: string
  created_at: string
  updated_at: string
}

export type WithdrawalStatus = 'PENDING' | 'SETTLED' | 'CANCELLED' | 'REJECTED'

export interface WithdrawalRequest {
  id: number
  user_id: number
  user_email: string
  amount: number
  fee_amount: number
  total_deducted: number
  balance_before: number
  balance_after: number
  payment_method: ReceiptCodePaymentMethod
  receipt_code_url?: string | null
  receipt_code_content_type: string
  receipt_code_byte_size: number
  receipt_code_sha256: string
  receipt_code_updated_at: string
  status: WithdrawalStatus
  user_cancel_reason?: string | null
  admin_note?: string | null
  rejection_reason?: string | null
  processed_by_user_id?: number | null
  processed_at?: string | null
  created_at: string
  updated_at: string
}

export interface UserMenuConfig {
  visibility: Record<string, boolean>
  order: string[]
}

export interface User {
  points_balance?: number
  load_factor_credits_balance?: number
  load_factor_credits_used_total?: number
  prefer_points_billing?: boolean
}

export interface UserAffiliateDetail {
  transfer_disabled: boolean
}

export interface PublicSettings {
  registration_email_alias_restriction_enabled: boolean
  user_menu_config: UserMenuConfig
  withdrawal_management_enabled?: boolean
  withdrawal_rate_limit_window_days?: number
  withdrawal_rate_limit_max?: number
  withdrawal_rate_limit_exempt_amount?: number
  user_account_import_limit?: number
  openai_account_levels?: OpenAIAccountLevelConfig[]
}

export interface Group {
  auto_assign_accounts_by_rate?: boolean
  auto_assign_max_rate?: number | null
}

export interface Account {
  account_level?: AccountLevel
  owner_user_id?: number | null
  max_accounts?: number
  share_mode?: AccountShareMode | string
  share_status?: AccountShareStatus | string
  share_policy_id?: number | null
  account_share_mode_listing_id?: number | null
}

export interface Proxy {
  max_accounts?: number
}

export interface UserSubscription {
  plan_id?: number | null
  daily_limit_usd?: number | null
  weekly_limit_usd?: number | null
  monthly_limit_usd?: number | null
  entitlements?: Record<string, number>
}

export interface PromoCode {
  discount_percent?: number
  applies_to_subscriptions?: boolean
  starts_at?: string | null
}

export interface CreatePromoCodeRequest {
  discount_percent?: number
  applies_to_subscriptions?: boolean
  starts_at?: number | null
}

export interface UpdatePromoCodeRequest {
  discount_percent?: number
  applies_to_subscriptions?: boolean
  starts_at?: number | null
}

export interface OpenAIAccountLevelConfig {
  key: string
  label: string
  aliases?: string[]
  enabled: boolean
  requires_proxy_login: boolean
  sort_order: number
}

export type AccountLevel = 'unknown' | (string & {})
export type AccountShareMode = 'private' | 'public'
export type AccountShareStatus = 'pending' | 'approved' | 'suspended'
export type AccountStatus = 'active' | 'inactive' | 'disabled' | 'error'

export interface AccountQuotaDimensionSummary {
  enabled_account_count: number
  exhausted_account_count: number
  limit: number
  used: number
  remaining: number
  utilization: number
}

export interface AccountUsageWindowSummary {
  window: string
  account_count: number
  known_account_count: number
  average_utilization: number
  remaining_capacity_percent: number
  estimated_support_hours?: number | null
  min_remaining_seconds?: number | null
  next_reset_at?: string | null
}

export interface AccountQuotaSummary {
  platform: AccountPlatform | 'all' | string
  type: AccountType | 'all' | string
  account_count: number
  active_account_count: number
  schedulable_account_count: number
  rate_limited_account_count: number
  codex_quota_protected_account_count: number
  error_account_count: number
  disabled_account_count: number
  quota_account_count: number
  unlimited_account_count: number
  total: AccountQuotaDimensionSummary
  daily: AccountQuotaDimensionSummary
  weekly: AccountQuotaDimensionSummary
  usage_windows?: AccountUsageWindowSummary[]
}

export interface AccountQuotaGroupSummary extends AccountQuotaSummary {
  group_id?: number | null
  group_name: string
  group_status: string
}

export interface AccountQuotaDashboard {
  generated_at: string
  summaries: AccountQuotaSummary[]
  totals: AccountQuotaSummary
  group_summaries?: AccountQuotaGroupSummary[]
}

export interface UserAccountQuotaPoolDashboard {
  generated_at: string
  mine: AccountQuotaDashboard
  platform: AccountQuotaDashboard
}

export interface OpenAIRateLimitWindow {
  used_percent: number
  limit_window_seconds: number
  reset_after_seconds: number
  reset_at: number
}

export interface OpenAIRateLimit {
  allowed: boolean
  limit_reached: boolean
  primary_window?: OpenAIRateLimitWindow | null
  secondary_window?: OpenAIRateLimitWindow | null
}

export interface OpenAIAdditionalRateLimit {
  limit_name: string
  metered_feature: string
  rate_limit?: OpenAIRateLimit | null
}

export interface OpenAIRateLimitResetCredits {
  available_count: number
}

export interface OpenAIQuotaUsage {
  user_id?: string
  account_id?: string
  email?: string
  plan_type?: string
  rate_limit?: OpenAIRateLimit | null
  additional_rate_limits?: OpenAIAdditionalRateLimit[]
  rate_limit_reset_credits?: OpenAIRateLimitResetCredits | null
  fetched_at: number
}

export interface OpenAIQuotaResetCredit {
  id?: string
  reset_type?: string
  status?: string
  granted_at?: string
  expires_at?: string
  redeem_started_at?: string
  redeemed_at?: string
}

export interface OpenAIQuotaResetResult {
  code: string
  credit?: OpenAIQuotaResetCredit | null
  windows_reset: number
}

export type ConversationStatus = 'open' | 'pending_user' | 'pending_admin' | 'resolved' | 'closed'
export type ConversationPriority = 'low' | 'normal' | 'high' | 'urgent'
export type ConversationType = 'support' | 'notice' | 'billing' | 'subscription' | 'account' | 'security'
export type ConversationKind = 'ticket' | 'system_notice'
export type ConversationSenderType = 'user' | 'admin' | 'system'
export type ConversationMessageType = 'text' | 'notice' | 'operation_log' | 'system_event'
export type ConversationContentFormat = 'plain' | 'markdown'

export interface Conversation {
  id: number
  user_id: number
  user_email?: string
  user_name?: string
  subject: string
  kind: ConversationKind
  referenced_notice_id?: number | null
  status: ConversationStatus
  priority: ConversationPriority
  type: ConversationType
  source?: string
  source_id?: string
  assigned_admin_id?: number | null
  last_message_id?: number | null
  last_message_sender_type: ConversationSenderType | ''
  last_message_excerpt: string
  last_message_at: string
  user_last_read_message_id?: number | null
  user_last_read_at?: string | null
  admin_last_read_message_id?: number | null
  admin_last_read_at?: string | null
  user_unread: boolean
  admin_unread?: boolean
  created_at: string
  updated_at: string
  messages?: ConversationMessage[]
}

export interface ConversationMessage {
  id: number
  conversation_id: number
  sender_type: ConversationSenderType
  sender_id?: number | null
  message_type: ConversationMessageType
  content_format: ConversationContentFormat
  content: string
  metadata?: Record<string, unknown>
  created_at: string
}

export interface ConversationMessageListOptions extends FetchOptions {
  beforeId?: number
  latest?: boolean
}

export interface ConversationListFilters {
  kind?: ConversationKind | ''
  status?: ConversationStatus | ''
  priority?: ConversationPriority | ''
  type?: ConversationType | ''
  search?: string
  unread_only?: boolean
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

export interface AdminConversationListFilters extends ConversationListFilters {
  user_id?: number | null
  assigned_admin_id?: number | null
}

export interface CreateConversationRequest {
  subject: string
  content: string
  priority?: ConversationPriority
  type?: ConversationType
  referenced_notice_id?: number | null
}

export interface CreateAdminConversationRequest extends CreateConversationRequest {
  user_id: number
  kind?: ConversationKind
  source?: string
  source_id?: string
  content_format?: ConversationContentFormat
}

export interface AddConversationMessageRequest {
  content: string
}
