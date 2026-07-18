import { apiClient } from './client'
import type { CreateOrderResult } from '@/types/payment'

export interface CommunityAccount {
  id: number
  owner_user_id: number
  name: string
  provider: 'openai' | 'anthropic'
  status: string
  review_status?: 'pending' | 'approved' | 'rejected'
  review_note?: string
  share_mode: 'private' | 'public'
  account_tier?: string
  capacity: number
  concurrency: number
  schedulable?: boolean
  today_requests?: number
  today_tokens?: number
  usage_5h_percent?: number
  usage_7d_percent?: number
  priority?: number
  group_name?: string
  tags: string[]
  supported_models: string[]
  expires_at?: string
  last_used_at?: string
  notes: string
  created_at: string
  proxy_id?: number
}

export interface CommunityAccountImportItem { name?:string; credential:unknown }
export interface CommunityAccountExportItem { name:string; provider:'openai'|'anthropic'; account_tier:string; share_mode:'private'|'public'; concurrency:number; tags:string[]; supported_models:string[]; provider_options:Record<string,unknown>; notes:string; proxy_id?:number; credential:unknown }

export interface CommunityListing {
  id: number
  account_id: number
  owner_user_id: number
  owner_name: string
  title: string
  description: string
  provider: string
  account_tier?: string
  tags?: string[]
  supported_models: string[]
  seat_limit: number
  seats_used: number
  live_concurrency?: number
  account_concurrency?: number
  per_user_concurrency: number
  minimum_balance?: number
  hourly_price: number
  hourly_minimum_spend?: number
  usage_multiplier: number
  usage_5h_percent?: number
  usage_7d_percent?: number
  usage_updated_at?: string
  idle_timeout_minutes: number
  commission_rate: number
  status: string
  health_status?: string
  score: number
  rating_count?: number
  updated_at: string
}

export interface CommunityMembership {
  id: number
  listing_id: number
  api_key_id?: number
  reservation_order: number
  status: string
  idle_timeout_minutes: number
  joined_at: string
  activated_at?: string
  last_used_at?: string
  last_request_at?: string
  idle_expires_at?: string
}
export interface CommunityAccountModeKey { id:number; name:string; status:string; account_mode_platform?:'openai'|'anthropic' }
export interface CommunityProxy { id:number; name:string; ip_type:'ipv4'|'ipv6'; protocol:'http'|'https'|'socks5'|'socks5h'; host:string; port:number; username:string; has_password:boolean; account_count:number; created_at:string }
export interface CommunityConsumptionAccount { listing_id:number; membership_id:number; title:string; provider:string; status:string; activated_at?:string; last_request_at?:string }
export interface CommunityConsumptionSummary { scope:'session'|'today'|'7d'; request_spend:number; hourly_precharged:number; hourly_refunded:number; total:number }
export interface CommunitySelectionResult { listing:CommunityListing; base_request_cost:number; request_spend:number; hourly_fee:number; hourly_fee_waived:boolean; required_spend:number; estimated_total:number; estimated_per_hour:number; remaining_seats:number }
export interface CommunityRecentUsageAverage { request_count:number; input_tokens:number; output_tokens:number; cache_creation_tokens:number; cache_read_tokens:number }
export interface CommunityOwnerSummary { published_listings:number; active_members:number; gross_revenue:number; platform_fees:number; net_revenue:number }
export interface TicketMessage { id:number; author_user_id:number; author_role:'user'|'admin'|'system'; content:string; created_at:string }
export interface SupportTicket { id:number; user_id:number; subject:string; category:string; priority:string; status:string; user_unread:number; admin_unread:number; created_at:string; updated_at:string; messages?:TicketMessage[] }
export interface PayoutMethod { method:'alipay'|'wechat'; qr_code_data:string; display_name:string; updated_at:string }
export interface Withdrawal { id:number; user_id:number; user_email?:string; payout_method:string; payout_snapshot:string; amount:number; fee:number; status:string; user_note:string; admin_note:string; payment_reference:string; created_at:string; reviewed_at?:string; paid_at?:string }
export interface StoreProduct { id:number; category:string; name:string; description:string; price:number; points_price?:number|null; fulfillment_type:string; fulfillment_value:number; status:string; stock:number; sort_order:number }
export interface StoreOrder { id:number; order_no:string; user_id:number; product_id:number; product_name:string; quantity:number; unit_price:number; total_amount:number; payment_source:string; status:string; delivery:string[]; created_at:string }
export interface StoreWallet { balance:number; points:number }
export interface AccountOAuthSession { auth_url:string; session_id:string }

const getData = async <T>(promise: Promise<{ data: T }>) => (await promise).data
export const communityAPI = {
  accounts: () => getData<CommunityAccount[]>(apiClient.get('/community/accounts')),
  createAccount: (data:Record<string,unknown>) => getData<CommunityAccount>(apiClient.post('/community/accounts',data)),
  importAccounts: (data:Record<string,unknown>) => getData<{count:number;accounts:CommunityAccount[]}>(apiClient.post('/community/accounts/import',data)),
  exportAccounts: (ids:number[]=[]) => getData<CommunityAccountExportItem[]>(apiClient.post('/community/accounts/export',{ids})),
  batchUpdateAccounts: (data:Record<string,unknown>) => getData<{updated:number}>(apiClient.patch('/community/accounts/batch',data)),
  accountOAuthURL: (data:{provider:string;proxy_id?:number;redirect_uri?:string}) => getData<AccountOAuthSession>(apiClient.post('/community/accounts/oauth/auth-url',data)),
  exchangeAccountOAuth: (data:Record<string,unknown>) => getData<CommunityAccount>(apiClient.post('/community/accounts/oauth/exchange',data)),
  proxies: () => getData<CommunityProxy[]>(apiClient.get('/community/proxies')),
  saveProxy: (data:Record<string,unknown>&{id?:number}) => getData<CommunityProxy>(data.id ? apiClient.put(`/community/proxies/${data.id}`,data) : apiClient.post('/community/proxies',data)),
  deleteProxy: (id:number) => apiClient.delete(`/community/proxies/${id}`),
  deleteAccount: (id:number) => apiClient.delete(`/community/accounts/${id}`),
  createListing: (data:Record<string,unknown>) => getData<CommunityListing>(apiClient.post('/community/listings',data)),
  marketplace: (params?:Record<string,string|number>) => getData<CommunityListing[]>(apiClient.get('/community/marketplace',{params})),
  ownerListings: (provider:'openai'|'anthropic') => getData<CommunityListing[]>(apiClient.get('/community/owner/listings',{params:{provider}})),
  ownerSummary: () => getData<CommunityOwnerSummary>(apiClient.get('/community/owner/summary')),
  setListingStatus: (id:number,status:'published'|'paused') => apiClient.put(`/community/listings/${id}/status`,{status}),
  memberships: () => getData<CommunityMembership[]>(apiClient.get('/community/memberships')),
  consumptionAccounts: () => getData<CommunityConsumptionAccount[]>(apiClient.get('/community/consumption/accounts')),
  consumptionSummary: (membershipId:number,scope:'session'|'today'|'7d') => getData<CommunityConsumptionSummary>(apiClient.get(`/community/consumption/${membershipId}`,{params:{scope}})),
  recommendListings: (data:Record<string,unknown>) => getData<CommunitySelectionResult[]>(apiClient.post('/community/selection-assistant',data)),
  recentUsageAverage: (apiKeyId:number) => getData<CommunityRecentUsageAverage>(apiClient.get(`/community/selection-assistant/recent/${apiKeyId}`)),
  accountModeKeys: (provider:'openai'|'anthropic') => getData<CommunityAccountModeKey[]>(apiClient.get('/community/account-mode-keys',{params:{provider}})),
  setAccountModeKey: (id:number,provider:'openai'|'anthropic') => apiClient.put(`/community/account-mode-keys/${id}`,{provider}),
  join: (id:number,api_key_id=0,idle_timeout_minutes=10) => apiClient.post(`/community/marketplace/${id}/join`,{api_key_id,idle_timeout_minutes}),
  leave: (id:number) => apiClient.delete(`/community/marketplace/${id}/membership`),
  reviewMembership: (id:number,data:{score:number;content:string}) => apiClient.post(`/community/memberships/${id}/review`,data),
  tickets: () => getData<SupportTicket[]>(apiClient.get('/community/tickets')),
  ticket: (id:number) => getData<SupportTicket>(apiClient.get(`/community/tickets/${id}`)),
  createTicket: (data:Record<string,string>) => getData<SupportTicket>(apiClient.post('/community/tickets',data)),
  replyTicket: (id:number,content:string) => apiClient.post(`/community/tickets/${id}/messages`,{content}),
  markTicketRead: (id:number) => apiClient.post(`/community/tickets/${id}/read`),
  closeTicket: (id:number) => apiClient.post(`/community/tickets/${id}/close`),
  payoutMethods: () => getData<PayoutMethod[]>(apiClient.get('/community/payout-methods')),
  savePayoutMethod: (data:Record<string,string>) => getData<PayoutMethod>(apiClient.put('/community/payout-methods',data)),
  deletePayoutMethod: (method:'alipay'|'wechat') => apiClient.delete(`/community/payout-methods/${method}`),
  withdrawals: () => getData<Withdrawal[]>(apiClient.get('/community/withdrawals')),
  createWithdrawal: (data:{method:string;amount:number;note:string}) => getData<Withdrawal>(apiClient.post('/community/withdrawals',data)),
  cancelWithdrawal: (id:number) => apiClient.post(`/community/withdrawals/${id}/cancel`),
  products: () => getData<StoreProduct[]>(apiClient.get('/community/store/products')),
  storeWallet: () => getData<StoreWallet>(apiClient.get('/community/store/wallet')),
  buy: (id:number,quantity=1,payment_source='balance') => getData<StoreOrder>(apiClient.post(`/community/store/products/${id}/buy`,{quantity,payment_source})),
  createPlatformStoreOrder: (id:number,data:{quantity:number;payment_type:string;return_url:string}) => getData<{store_order_id:number;payment:CreateOrderResult}>(apiClient.post(`/community/store/products/${id}/platform-order`,data)),
  storeOrders: () => getData<StoreOrder[]>(apiClient.get('/community/store/orders')),
  admin: {
    accounts: () => getData<CommunityAccount[]>(apiClient.get('/admin/community/accounts')),
    reviewAccount: (id:number,decision:'approved'|'rejected',note='') => apiClient.put(`/admin/community/accounts/${id}/review`,{decision,note}),
    tickets: () => getData<SupportTicket[]>(apiClient.get('/admin/community/tickets')),
    ticket: (id:number) => getData<SupportTicket>(apiClient.get(`/admin/community/tickets/${id}`)),
    reply: (id:number,content:string) => apiClient.post(`/admin/community/tickets/${id}/messages`,{content}),
    ticketStatus: (id:number,status:string) => apiClient.put(`/admin/community/tickets/${id}/status`,{status}),
    withdrawals: () => getData<Withdrawal[]>(apiClient.get('/admin/community/withdrawals')),
    reviewWithdrawal: (id:number,data:Record<string,string>) => apiClient.put(`/admin/community/withdrawals/${id}`,data),
    products: () => getData<StoreProduct[]>(apiClient.get('/admin/community/products')),
    saveProduct: (product:Partial<StoreProduct>) => getData<StoreProduct>(product.id ? apiClient.put(`/admin/community/products/${product.id}`,product) : apiClient.post('/admin/community/products',product)),
    addInventory: (id:number,values:string[]) => apiClient.post(`/admin/community/products/${id}/inventory`,{values}),
    orders: () => getData<StoreOrder[]>(apiClient.get('/admin/community/store-orders')),
    settings: () => getData<{commission_percent:number}>(apiClient.get('/admin/community/settings')),
    saveSettings: (commission_percent:number) => apiClient.put('/admin/community/settings',{commission_percent})
  }
}
