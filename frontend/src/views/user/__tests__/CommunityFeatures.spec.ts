import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import AccountMarketplaceView from '@/views/user/AccountMarketplaceView.vue'
import CardStoreView from '@/views/user/CardStoreView.vue'
import CommunityAccountsView from '@/views/user/CommunityAccountsView.vue'
import SupportTicketsView from '@/views/user/SupportTicketsView.vue'
import ProfileWalletPanel from '@/components/user/profile/ProfileWalletPanel.vue'
import AdminCommunityView from '@/views/admin/AdminCommunityView.vue'

const mocks = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn(),
  refreshUser: vi.fn(),
  accounts: vi.fn(),
  proxies: vi.fn(),
  marketplace: vi.fn(),
  ownerListings: vi.fn(),
  ownerSummary: vi.fn(),
  memberships: vi.fn(),
  accountModeKeys: vi.fn(),
  consumptionAccounts: vi.fn(),
  consumptionSummary: vi.fn(),
  tickets: vi.fn(),
  ticket: vi.fn(),
  replyTicket: vi.fn(),
  products: vi.fn(),
  storeOrders: vi.fn(),
  storeWallet: vi.fn(),
  buy: vi.fn(),
  payoutMethods: vi.fn(),
  withdrawals: vi.fn(),
  adminSettings: vi.fn(),
  adminAccounts: vi.fn(),
  adminWithdrawals: vi.fn(),
  adminTickets: vi.fn(),
  adminProducts: vi.fn(),
  adminOrders: vi.fn(),
  paymentConfig: vi.fn(),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: mocks.showError, showSuccess: mocks.showSuccess }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: { id: 1, email: 'owner@example.com', balance: 50 },
    refreshUser: mocks.refreshUser,
  }),
}))

vi.mock('@/api/payment', () => ({
  paymentAPI: { getConfig: mocks.paymentConfig },
}))

vi.mock('@/api/community', async () => {
  const actual = await vi.importActual<typeof import('@/api/community')>('@/api/community')
  return {
    ...actual,
    communityAPI: {
      accounts: mocks.accounts,
      proxies: mocks.proxies,
      marketplace: mocks.marketplace,
      ownerListings: mocks.ownerListings,
      ownerSummary: mocks.ownerSummary,
      memberships: mocks.memberships,
      accountModeKeys: mocks.accountModeKeys,
      consumptionAccounts: mocks.consumptionAccounts,
      consumptionSummary: mocks.consumptionSummary,
      tickets: mocks.tickets,
      ticket: mocks.ticket,
      replyTicket: mocks.replyTicket,
      markTicketRead: vi.fn(),
      closeTicket: vi.fn(),
      createTicket: vi.fn(),
      products: mocks.products,
      storeOrders: mocks.storeOrders,
      storeWallet: mocks.storeWallet,
      buy: mocks.buy,
      createPlatformStoreOrder: vi.fn(),
      payoutMethods: mocks.payoutMethods,
      withdrawals: mocks.withdrawals,
      savePayoutMethod: vi.fn(),
      deletePayoutMethod: vi.fn(),
      createWithdrawal: vi.fn(),
      cancelWithdrawal: vi.fn(),
      importAccounts: vi.fn(),
      exportAccounts: vi.fn(),
      batchUpdateAccounts: vi.fn(),
      accountOAuthURL: vi.fn(),
      exchangeAccountOAuth: vi.fn(),
      saveProxy: vi.fn(),
      deleteProxy: vi.fn(),
      deleteAccount: vi.fn(),
      createListing: vi.fn(),
      setListingStatus: vi.fn(),
      setAccountModeKey: vi.fn(),
      join: vi.fn(),
      leave: vi.fn(),
      reviewMembership: vi.fn(),
      recommendListings: vi.fn(),
      recentUsageAverage: vi.fn(),
      admin: {
        settings: mocks.adminSettings,
        accounts: mocks.adminAccounts,
        withdrawals: mocks.adminWithdrawals,
        tickets: mocks.adminTickets,
        products: mocks.adminProducts,
        orders: mocks.adminOrders,
        reviewAccount: vi.fn(),
        ticket: vi.fn(),
        reply: vi.fn(),
        ticketStatus: vi.fn(),
        reviewWithdrawal: vi.fn(),
        saveProduct: vi.fn(),
        addInventory: vi.fn(),
        saveSettings: vi.fn(),
      },
    },
  }
})

const layoutStubs = {
  UserWorkspaceLayout: { template: '<div><slot /></div>' },
  AppLayout: { template: '<div><slot /></div>' },
  RouterLink: { props: ['to'], template: '<a><slot /></a>' },
  Icon: { props: ['name'], template: '<i :data-icon="name" />' },
  Teleport: true,
}

const account = {
  id: 11,
  owner_user_id: 1,
  name: 'OpenAI Plus Pool',
  provider: 'openai',
  status: 'active',
  review_status: 'approved',
  share_mode: 'public',
  account_tier: 'plus',
  capacity: 3,
  concurrency: 3,
  schedulable: true,
  today_requests: 18,
  today_tokens: 3200,
  usage_5h_percent: 20,
  usage_7d_percent: 40,
  priority: 50,
  group_name: 'OpenAI',
  tags: ['stable'],
  supported_models: ['gpt-5.4'],
  notes: 'mock account',
  created_at: '2026-07-19T00:00:00Z',
}

const listing = {
  id: 21,
  account_id: 11,
  owner_user_id: 2,
  owner_name: 'pool-owner',
  title: 'OpenAI Plus Shared',
  description: 'Stable OAuth account',
  provider: 'openai',
  account_tier: 'plus',
  tags: ['stable'],
  supported_models: ['gpt-5.4'],
  seat_limit: 3,
  seats_used: 1,
  per_user_concurrency: 1,
  minimum_balance: 1,
  hourly_price: 0.6,
  hourly_minimum_spend: 0.3,
  usage_multiplier: 1.2,
  idle_timeout_minutes: 10,
  commission_rate: 10,
  status: 'published',
  health_status: 'healthy',
  score: 9.5,
  rating_count: 3,
  updated_at: '2026-07-19T00:00:00Z',
}

describe('community feature pages with mocked data', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.stubGlobal('confirm', vi.fn(() => true))
    mocks.refreshUser.mockResolvedValue(undefined)
    mocks.accounts.mockResolvedValue([account])
    mocks.proxies.mockResolvedValue([])
    mocks.marketplace.mockResolvedValue([listing])
    mocks.ownerListings.mockResolvedValue([listing])
    mocks.ownerSummary.mockResolvedValue({ published_listings: 1, active_members: 1, gross_revenue: 2, platform_fees: 0.2, net_revenue: 1.8 })
    mocks.memberships.mockResolvedValue([])
    mocks.accountModeKeys.mockResolvedValue([{ id: 8, name: 'Account Mode Key', status: 'active', account_mode_platform: 'openai' }])
    mocks.consumptionAccounts.mockResolvedValue([{ listing_id: 21, membership_id: 31, title: listing.title, provider: 'openai', status: 'active' }])
    mocks.consumptionSummary.mockResolvedValue({ scope: 'session', request_spend: 1.2, hourly_precharged: 0.1, hourly_refunded: 0.1, total: 1.2 })
    mocks.paymentConfig.mockResolvedValue({ data: { payment_enabled: true, enabled_payment_types: ['alipay'] } })
    mocks.payoutMethods.mockResolvedValue([{ method: 'alipay', qr_code_data: 'data:image/png;base64,AA==', display_name: '', updated_at: '2026-07-19T00:00:00Z' }])
    mocks.withdrawals.mockResolvedValue([{ id: 41, user_id: 1, payout_method: 'alipay', payout_snapshot: '', amount: 10, fee: 0.1, status: 'approved', user_note: '', admin_note: 'approved', payment_reference: '', created_at: '2026-07-19T00:00:00Z' }])
  })

  it('renders an approved OAuth pool account with usage and model data', async () => {
    const wrapper = mount(CommunityAccountsView, { global: { stubs: layoutStubs } })
    await flushPromises()

    expect(wrapper.text()).toContain('OpenAI Plus Pool')
    expect(wrapper.text()).toContain('OpenAI')
    expect(wrapper.text()).toContain('18')
    expect(mocks.accounts).toHaveBeenCalledOnce()
  })

  it('renders marketplace pricing, seats, rating, and account-mode key controls', async () => {
    const wrapper = mount(AccountMarketplaceView, { global: { stubs: layoutStubs } })
    await flushPromises()

    expect(wrapper.text()).toContain('OpenAI Plus Shared')
    expect(wrapper.text()).toContain('pool-owner')
    expect(wrapper.text()).toContain('1.2x')
    expect(wrapper.text()).toContain('9.5')
    expect(wrapper.text()).toContain('Account Mode Key')
  })

  it('renders a ticket thread with user, admin, and system messages and submits a reply', async () => {
    const detail = {
      id: 51,
      user_id: 1,
      subject: 'Offline withdrawal update',
      category: 'billing',
      priority: 'normal',
      status: 'waiting_user',
      user_unread: 1,
      admin_unread: 0,
      created_at: '2026-07-19T00:00:00Z',
      updated_at: '2026-07-19T00:10:00Z',
      messages: [
        { id: 1, author_user_id: 1, author_role: 'user', content: 'When will it be paid?', created_at: '2026-07-19T00:00:00Z' },
        { id: 2, author_user_id: 2, author_role: 'admin', content: 'Approved for offline payment.', created_at: '2026-07-19T00:05:00Z' },
        { id: 3, author_user_id: 0, author_role: 'system', content: 'Status changed to approved.', created_at: '2026-07-19T00:10:00Z' },
      ],
    }
    mocks.tickets.mockResolvedValue([detail])
    mocks.ticket.mockResolvedValue(detail)
    mocks.replyTicket.mockResolvedValue({})

    const wrapper = mount(SupportTicketsView, { global: { stubs: layoutStubs } })
    await flushPromises()
    await wrapper.find('.ticket-list button').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Approved for offline payment.')
    expect(wrapper.text()).toContain('Status changed to approved.')
    await wrapper.get('.ticket-reply textarea').setValue('Thanks')
    await wrapper.get('.ticket-reply').trigger('submit')
    await flushPromises()
    expect(mocks.replyTicket).toHaveBeenCalledWith(51, 'Thanks')
  })

  it('renders card and entitlement products plus all enabled payment choices', async () => {
    mocks.products.mockResolvedValue([
      { id: 61, category: '兑换码', name: '10 USD Code', description: 'Auto delivery', price: 10, points_price: 100, fulfillment_type: 'redeem_code', fulfillment_value: 0, status: 'active', stock: 4, sort_order: 1 },
      { id: 62, category: '负载额度', name: 'Concurrency +2', description: 'Entitlement', price: 20, points_price: null, fulfillment_type: 'entitlement', fulfillment_value: 2, status: 'active', stock: 0, sort_order: 2 },
    ])
    mocks.storeOrders.mockResolvedValue([])
    mocks.storeWallet.mockResolvedValue({ balance: 50, points: 500 })

    const wrapper = mount(CardStoreView, { global: { stubs: layoutStubs } })
    await flushPromises()
    expect(wrapper.text()).toContain('10 USD Code')
    expect(wrapper.text()).toContain('Concurrency +2')
    expect(wrapper.text()).toContain('+2')
    await wrapper.find('.store-product .community-btn').trigger('click')
    expect(wrapper.text()).toContain('余额支付')
    expect(wrapper.text()).toContain('积分支付')
    expect(wrapper.text()).toContain('支付宝')
  })

  it('renders saved payout QR state and an approved offline withdrawal', async () => {
    const wrapper = mount(ProfileWalletPanel, { global: { stubs: layoutStubs } })
    await flushPromises()

    expect(wrapper.find('.wallet-qr img').exists()).toBe(true)
    expect(wrapper.text()).toContain('待打款')
    expect(wrapper.text()).toContain('approved')
    expect(wrapper.text()).toContain('10.00')
  })

  it('renders all six admin community management sections from mocked APIs', async () => {
    mocks.adminSettings.mockResolvedValue({ commission_percent: 10 })
    mocks.adminAccounts.mockResolvedValue([account])
    mocks.adminWithdrawals.mockResolvedValue([])
    mocks.adminTickets.mockResolvedValue([])
    mocks.adminProducts.mockResolvedValue([])
    mocks.adminOrders.mockResolvedValue([])

    const wrapper = mount(AdminCommunityView, { global: { stubs: layoutStubs } })
    await flushPromises()
    for (const label of ['账号审核', '提现审核', '工单', '发卡商品', '商城订单', '抽成设置']) {
      expect(wrapper.text()).toContain(label)
    }
    expect(wrapper.text()).toContain('OpenAI Plus Pool')
    expect(mocks.adminSettings).toHaveBeenCalledOnce()
  })
})
