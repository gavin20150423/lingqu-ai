import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import type { ApiKey } from '@/types'
import KeysView from '../KeysView.vue'

const {
  listKeys,
  getPublicSettings,
  getDashboardApiKeysUsage,
  getAvailableGroups,
  getUserGroupRates,
  showError,
  showSuccess,
  copyToClipboard,
  isCurrentStep,
  nextStep,
  route,
  routerReplace,
} = vi.hoisted(() => ({
  listKeys: vi.fn(),
  getPublicSettings: vi.fn(),
  getDashboardApiKeysUsage: vi.fn(),
  getAvailableGroups: vi.fn(),
  getUserGroupRates: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  copyToClipboard: vi.fn(),
  isCurrentStep: vi.fn(),
  nextStep: vi.fn(),
  route: {
    path: '/keys',
    query: {} as Record<string, string>,
  },
  routerReplace: vi.fn(),
}))

const messages: Record<string, string> = {
  'dashboard.platformQuota.noLimit': 'Unlimited',
  'keys.allGroups': 'All Groups',
  'keys.allStatus': 'All Status',
  'keys.created': 'Created',
  'keys.currentConcurrency': 'Current Concurrency',
  'keys.lastUsedAt': 'Last Used',
  'keys.lastUsedIP': 'Last Used IP',
  'keys.noExpiration': 'No expiration',
  'keys.noGroup': 'No group',
  'keys.searchPlaceholder': 'Search name or key...',
  'keys.status.active': 'Active',
  'keys.status.expired': 'Expired',
  'keys.status.inactive': 'Inactive',
  'keys.status.quota_exhausted': 'Quota exhausted',
  'keys.today': 'Today',
  'keys.total': 'Total',
  'keys.quota': 'Quota',
}

vi.mock('vue-router', () => ({
  useRoute: () => route,
  useRouter: () => ({
    replace: routerReplace,
  }),
}))

vi.mock('@/api', () => ({
  keysAPI: {
    list: listKeys,
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    toggleStatus: vi.fn(),
  },
  authAPI: {
    getPublicSettings,
  },
  usageAPI: {
    getDashboardApiKeysUsage,
  },
  userGroupsAPI: {
    getAvailable: getAvailableGroups,
    getUserGroupRates,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
  }),
}))

vi.mock('@/stores/onboarding', () => ({
  useOnboardingStore: () => ({
    isCurrentStep,
    nextStep,
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

const createApiKey = (overrides: Partial<ApiKey> = {}): ApiKey => ({
  id: 1,
  user_id: 1,
  key: 'sk-test-key',
  name: 'test-key',
  group_id: null,
  status: 'active',
  ip_whitelist: [],
  ip_blacklist: [],
  last_used_at: null,
  last_used_ip: null,
  quota: 0,
  quota_used: 0,
  expires_at: null,
  created_at: '2026-06-27T00:00:00Z',
  updated_at: '2026-06-27T00:00:00Z',
  current_concurrency: 3,
  rate_limit_5h: 0,
  rate_limit_1d: 0,
  rate_limit_7d: 0,
  usage_5h: 0,
  usage_1d: 0,
  usage_7d: 0,
  window_5h_start: null,
  window_1d_start: null,
  window_7d_start: null,
  reset_5h_at: null,
  reset_1d_at: null,
  reset_7d_at: null,
  ...overrides,
})

const UserWorkspaceLayoutStub = {
  template: '<div><slot /></div>',
}

const RouterLinkStub = {
  props: ['to'],
  template: '<a><slot /></a>',
}

const SelectStub = {
  name: 'Select',
  props: ['modelValue', 'options'],
  emits: ['update:modelValue'],
  template: '<select :value="modelValue"></select>',
}

const SearchInputStub = {
  name: 'SearchInput',
  props: ['modelValue'],
  emits: ['update:modelValue', 'search'],
  template: '<input :value="modelValue" />',
}

const PaginationStub = {
  name: 'Pagination',
  props: ['page', 'total', 'pageSize'],
  emits: ['update:page', 'update:pageSize'],
  template: '<button data-test="page-size-50" @click="$emit(\'update:pageSize\', 50)">50</button>',
}

const GroupBadgeStub = {
  name: 'GroupBadge',
  props: [
    'name',
    'platform',
    'subscriptionType',
    'rateMultiplier',
    'userRateMultiplier',
    'peakRateEnabled',
    'peakStart',
    'peakEnd',
    'peakRateMultiplier',
  ],
  template: '<span data-test="group-badge">{{ name }}</span>',
}

const IconStub = {
  props: ['name'],
  template: '<span>{{ name }}</span>',
}

const mountView = async () => {
  const wrapper = mount(KeysView, {
    global: {
      stubs: {
        UserWorkspaceLayout: UserWorkspaceLayoutStub,
        RouterLink: RouterLinkStub,
        Pagination: PaginationStub,
        BaseDialog: true,
        ConfirmDialog: true,
        Select: SelectStub,
        SearchInput: SearchInputStub,
        Icon: IconStub,
        UseKeyModal: true,
        EndpointPopover: true,
        GroupBadge: GroupBadgeStub,
        GroupOptionItem: true,
        Teleport: true,
      },
    },
  })
  await flushPromises()
  await nextTick()
  return wrapper
}

describe('user KeysView cards', () => {
  beforeEach(() => {
    localStorage.clear()
    delete window.__APP_CONFIG__
    route.query = {}

    listKeys.mockReset()
    getPublicSettings.mockReset()
    getDashboardApiKeysUsage.mockReset()
    getAvailableGroups.mockReset()
    getUserGroupRates.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    copyToClipboard.mockReset()
    isCurrentStep.mockReset()
    nextStep.mockReset()
    routerReplace.mockReset()

    listKeys.mockResolvedValue({
      items: [createApiKey()],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    getPublicSettings.mockResolvedValue({})
    getDashboardApiKeysUsage.mockResolvedValue({ stats: {} })
    getAvailableGroups.mockResolvedValue([])
    getUserGroupRates.mockResolvedValue({})
    copyToClipboard.mockResolvedValue(true)
    isCurrentStep.mockReturnValue(false)
    routerReplace.mockResolvedValue(undefined)
  })

  it('renders current concurrency and the last-used IP on each card', async () => {
    listKeys.mockResolvedValueOnce({
      items: [createApiKey({ current_concurrency: 3, last_used_ip: '203.0.113.10' })],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })

    const wrapper = await mountView()
    const limits = wrapper.get('.lingqu-key-card__limits').text()

    expect(limits).toContain('Current Concurrency: 3')
    expect(limits).toContain('Last Used IP: 203.0.113.10')
  })

  it('passes group peak-rate and user-rate details to GroupBadge', async () => {
    listKeys.mockResolvedValueOnce({
      items: [
        createApiKey({
          group_id: 42,
          group: {
            id: 42,
            name: 'OpenAI Peak',
            platform: 'openai',
            subscription_type: 'standard',
            rate_multiplier: 1.5,
            peak_rate_enabled: true,
            peak_start: '09:00',
            peak_end: '18:00',
            peak_rate_multiplier: 2,
          } as NonNullable<ApiKey['group']>,
        }),
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    getUserGroupRates.mockResolvedValueOnce({ 42: 1.25 })

    const wrapper = await mountView()
    const badge = wrapper.getComponent({ name: 'GroupBadge' })

    expect(badge.props()).toMatchObject({
      name: 'OpenAI Peak',
      platform: 'openai',
      subscriptionType: 'standard',
      rateMultiplier: 1.5,
      userRateMultiplier: 1.25,
      peakRateEnabled: true,
      peakStart: '09:00',
      peakEnd: '18:00',
      peakRateMultiplier: 2,
    })
  })

  it('keeps page size, filters, and the newest-first API sort together', async () => {
    getAvailableGroups.mockResolvedValueOnce([{ id: 42, name: 'OpenAI' }])
    const wrapper = await mountView()

    await wrapper.get('[data-test="page-size-50"]').trigger('click')
    await flushPromises()

    const search = wrapper.getComponent({ name: 'SearchInput' })
    await search.vm.$emit('update:modelValue', 'target')
    await nextTick()
    await search.vm.$emit('search')
    await flushPromises()

    const selects = wrapper.findAllComponents({ name: 'Select' })
    await selects[0].vm.$emit('update:modelValue', 42)
    await flushPromises()
    await selects[1].vm.$emit('update:modelValue', 'active')
    await flushPromises()

    expect(listKeys).toHaveBeenLastCalledWith(
      1,
      50,
      {
        search: 'target',
        status: 'active',
        group_id: 42,
        sort_by: 'created_at',
        sort_order: 'desc',
      },
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    )
  })
})
