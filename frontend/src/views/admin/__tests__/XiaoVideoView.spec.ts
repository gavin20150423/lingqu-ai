import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { Account, AdminGroup } from '@/types'

const {
  listAccounts,
  getAccount,
  createAccount,
  updateAccount,
  testAccount,
  syncUpstreamModels,
  syncUpstreamModelsPreview,
  deleteAccount,
  getGroups,
  showError,
  showSuccess,
  showInfo
} = vi.hoisted(() => ({
  listAccounts: vi.fn(),
  getAccount: vi.fn(),
  createAccount: vi.fn(),
  updateAccount: vi.fn(),
  testAccount: vi.fn(),
  syncUpstreamModels: vi.fn(),
  syncUpstreamModelsPreview: vi.fn(),
  deleteAccount: vi.fn(),
  getGroups: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  showInfo: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listAccounts,
      getById: getAccount,
      create: createAccount,
      update: updateAccount,
      testAccount,
      syncUpstreamModels,
      syncUpstreamModelsPreview,
      delete: deleteAccount
    },
    groups: {
      getByPlatform: getGroups
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess, showInfo })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

import XiaoVideoView from '../XiaoVideoView.vue'

function xiaoAccount(overrides: Partial<Account> = {}): Account {
  return {
    id: 41,
    name: 'Video upstream A',
    platform: 'xiaoapi',
    type: 'apikey',
    account_level: 'unknown',
    credentials: {
      base_url: 'https://upstream.example.com',
      model_mapping: { 'public-video': 'provider-video-v2' },
      video_pricing: [
        {
          model: 'public-video',
          resolution: '720p',
          price_per_second: 0.12,
          audio_price_per_second: 0.03,
          default_resolution: true,
          default_duration: 8
        }
      ],
      custom_option: 'preserved'
    },
    credentials_status: { has_api_key: true },
    proxy_id: null,
    concurrency: 2,
    priority: 0,
    status: 'active',
    error_message: null,
    last_used_at: null,
    expires_at: null,
    auto_pause_on_expired: true,
    created_at: '2026-08-01T00:00:00Z',
    updated_at: '2026-08-01T00:00:00Z',
    group_ids: [7],
    schedulable: true,
    rate_limited_at: null,
    rate_limit_reset_at: null,
    overload_until: null,
    temp_unschedulable_until: null,
    temp_unschedulable_reason: null,
    session_window_start: null,
    session_window_end: null,
    session_window_status: null,
    ...overrides
  }
}

const xiaoGroup = {
  id: 7,
  name: 'xiao-video-default',
  platform: 'xiaoapi',
  status: 'active'
} as AdminGroup

function listResponse(items: Account[]) {
  return {
    items,
    total: items.length,
    page: 1,
    page_size: 100,
    pages: 1
  }
}

function mountView() {
  return mount(XiaoVideoView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        Icon: true,
        RouterLink: { template: '<a><slot /></a>' }
      }
    }
  })
}

describe('XiaoVideoView', () => {
  beforeEach(() => {
    listAccounts.mockReset()
    getAccount.mockReset()
    createAccount.mockReset()
    updateAccount.mockReset()
    testAccount.mockReset()
    syncUpstreamModels.mockReset()
    syncUpstreamModelsPreview.mockReset()
    deleteAccount.mockReset()
    getGroups.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    showInfo.mockReset()

    const account = xiaoAccount()
    listAccounts.mockResolvedValue(listResponse([account]))
    getAccount.mockResolvedValue(account)
    createAccount.mockResolvedValue(xiaoAccount({ id: 42 }))
    updateAccount.mockImplementation(async (_id: number, payload: Record<string, unknown>) =>
      xiaoAccount(payload as Partial<Account>)
    )
    testAccount.mockResolvedValue({ success: true, message: 'connected' })
    syncUpstreamModels.mockResolvedValue({ models: ['12:provider-video', '7:fast-video'] })
    syncUpstreamModelsPreview.mockResolvedValue({ models: ['12:provider-video', '7:fast-video'] })
    deleteAccount.mockResolvedValue({ message: 'deleted' })
    getGroups.mockResolvedValue([xiaoGroup])
  })

  it('loads only XiaoAPI upstreams and exposes their dynamic configuration', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(listAccounts).toHaveBeenCalledWith(1, 100, { platform: 'xiaoapi' })
    expect(getGroups).toHaveBeenCalledWith('xiaoapi')
    expect((wrapper.get('[data-testid="xiao-base-url"]').element as HTMLInputElement).value)
      .toBe('https://upstream.example.com')
    expect((wrapper.get('[data-testid="xiao-protocol"]').element as HTMLSelectElement).value)
      .toBe('native')
    expect((wrapper.get('[data-testid="xiao-mapping-public-0"]').element as HTMLInputElement).value)
      .toBe('public-video')
    expect((wrapper.get('[data-testid="xiao-price-base-0"]').element as HTMLInputElement).value)
      .toBe('0.12')
  })

  it('updates mappings and prices while preserving a redacted API key and other credentials', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="xiao-base-url"]').setValue('https://replacement.example.com')
    await wrapper.get('[data-testid="xiao-mapping-upstream-0"]').setValue('provider-video-v3')
    await wrapper.get('[data-testid="xiao-price-base-0"]').setValue('0.25')
    await wrapper.get('[data-testid="xiao-save"]').trigger('submit')
    await flushPromises()

    expect(updateAccount).toHaveBeenCalled()
    const payload = updateAccount.mock.calls[0][1]
    expect(payload.credentials).toMatchObject({
      base_url: 'https://replacement.example.com',
      custom_option: 'preserved',
      model_mapping: { 'public-video': 'provider-video-v3' },
      video_pricing: [expect.objectContaining({ price_per_second: 0.25 })]
    })
    expect(payload.credentials).not.toHaveProperty('api_key')
  })

  it('creates a standalone XiaoAPI upstream from administrator-provided values', async () => {
    listAccounts.mockResolvedValue(listResponse([]))
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="xiao-name"]').setValue('Provider B')
    await wrapper.get('[data-testid="xiao-base-url"]').setValue('https://provider-b.example.com')
    await wrapper.get('[data-testid="xiao-api-key"]').setValue('secret-b')
    await wrapper.get('[data-testid="xiao-concurrency"]').setValue('4')
    await wrapper.get('[data-testid="xiao-group-7"]').setValue(true)
    await wrapper.get('[data-testid="xiao-add-mapping"]').trigger('click')
    await wrapper.get('[data-testid="xiao-mapping-public-0"]').setValue('customer-model')
    await wrapper.get('[data-testid="xiao-mapping-upstream-0"]').setValue('provider-model')
    await wrapper.get('[data-testid="xiao-price-model-0"]').setValue('customer-model')
    await wrapper.get('[data-testid="xiao-price-resolution-0"]').setValue('544p')
    await wrapper.get('[data-testid="xiao-price-base-0"]').setValue('0.08')
    await wrapper.get('[data-testid="xiao-price-duration-0"]').setValue('6')
    await wrapper.get('[data-testid="xiao-save"]').trigger('submit')
    await flushPromises()

    expect(createAccount).toHaveBeenCalledWith(expect.objectContaining({
      name: 'Provider B',
      platform: 'xiaoapi',
      type: 'apikey',
      concurrency: 4,
      group_ids: [7],
      credentials: expect.objectContaining({
        base_url: 'https://provider-b.example.com',
        api_key: 'secret-b',
        video_protocol: 'native',
        model_mapping: { 'customer-model': 'provider-model' },
        video_pricing: [expect.objectContaining({
          model: 'customer-model',
          resolution: '544p',
          price_per_second: 0.08,
          default_duration: 6
        })]
      })
    }))
  })

  it('configures AIStartLab with its OpenAI/Sora compatible endpoint', async () => {
    listAccounts.mockResolvedValue(listResponse([]))
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="xiao-name"]').setValue('AIStartLab')
    await wrapper.get('[data-testid="xiao-protocol"]').setValue('openai_sora')
    expect((wrapper.get('[data-testid="xiao-base-url"]').element as HTMLInputElement).value)
      .toBe('https://api.video.aistarslab.com/openai')
    await wrapper.get('[data-testid="xiao-api-key"]').setValue('rotated-secret')
    await wrapper.get('[data-testid="xiao-group-7"]').setValue(true)
    await wrapper.get('[data-testid="xiao-add-mapping"]').trigger('click')
    await wrapper.get('[data-testid="xiao-mapping-public-0"]').setValue('customer-video')
    await wrapper.get('[data-testid="xiao-mapping-upstream-0"]').setValue('12:provider-video')
    await wrapper.get('[data-testid="xiao-price-model-0"]').setValue('customer-video')
    await wrapper.get('[data-testid="xiao-price-resolution-0"]').setValue('720p')
    await wrapper.get('[data-testid="xiao-price-base-0"]').setValue('0.08')
    await wrapper.get('[data-testid="xiao-price-duration-0"]').setValue('5')
    await wrapper.get('[data-testid="xiao-save"]').trigger('submit')
    await flushPromises()

    expect(createAccount).toHaveBeenCalledWith(expect.objectContaining({
      credentials: expect.objectContaining({
        base_url: 'https://api.video.aistarslab.com/openai',
        video_protocol: 'openai_sora',
        model_mapping: { 'customer-video': '12:provider-video' }
      })
    }))
  })

  it('fetches and batch imports AIStartLab models before creating the account', async () => {
    listAccounts.mockResolvedValue(listResponse([]))
    syncUpstreamModelsPreview.mockResolvedValue({
      models: ['12:provider-video'],
      model_specs: [
        {
          id: '12:provider-video',
          resolution: '720p',
          upstream_cost: 500,
          cost_currency: 'CREDITS',
          cost_unit: 'request',
          default_duration: 5,
          default_resolution: true
        },
        {
          id: '12:provider-video',
          resolution: '1080p',
          upstream_cost: 1000,
          cost_currency: 'CREDITS',
          cost_unit: 'request',
          default_duration: 5
        }
      ],
      pricing_source: 'aistartlab_config'
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="xiao-protocol"]').setValue('openai_sora')
    await wrapper.get('[data-testid="xiao-api-key"]').setValue('rotated-secret')
    await wrapper.get('[data-testid="xiao-fetch-models"]').trigger('click')
    await flushPromises()

    expect(syncUpstreamModelsPreview).toHaveBeenCalledWith({
      platform: 'xiaoapi',
      type: 'apikey',
      base_url: 'https://api.video.aistarslab.com/openai',
      api_key: 'rotated-secret',
      video_protocol: 'openai_sora'
    })
    expect(wrapper.get('[data-testid="xiao-model-picker"]').isVisible()).toBe(true)

    await wrapper.get('[data-testid="xiao-import-models"]').trigger('click')
    expect((wrapper.get('[data-testid="xiao-mapping-public-0"]').element as HTMLInputElement).value)
      .toBe('12:provider-video')
    expect((wrapper.get('[data-testid="xiao-mapping-upstream-0"]').element as HTMLInputElement).value)
      .toBe('12:provider-video')
    expect((wrapper.get('[data-testid="xiao-price-model-0"]').element as HTMLInputElement).value)
      .toBe('12:provider-video')
    expect((wrapper.get('[data-testid="xiao-price-resolution-0"]').element as HTMLInputElement).value)
      .toBe('720p')
    expect((wrapper.get('[data-testid="xiao-price-base-0"]').element as HTMLInputElement).value)
      .toBe('1.3')
    expect((wrapper.get('[data-testid="xiao-price-resolution-1"]').element as HTMLInputElement).value)
      .toBe('1080p')
    expect((wrapper.get('[data-testid="xiao-price-base-1"]').element as HTMLInputElement).value)
      .toBe('2.6')
    expect(wrapper.find('[data-testid="xiao-model-picker"]').exists()).toBe(false)
    expect(showSuccess).toHaveBeenCalledWith('admin.xiaoVideo.modelsImported')
  })

  it('uses native XiaoAPI USD prices as 1:1 internal costs', async () => {
    listAccounts.mockResolvedValue(listResponse([]))
    syncUpstreamModelsPreview.mockResolvedValue({
      models: ['native-video'],
      model_specs: [{
        id: 'native-video',
        resolution: '720p',
        upstream_cost: 0.2,
        cost_currency: 'USD',
        cost_unit: 'second',
        default_duration: 4
      }],
      pricing_source: 'model_list'
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="xiao-base-url"]').setValue('https://native.example.com')
    await wrapper.get('[data-testid="xiao-api-key"]').setValue('native-secret')
    await wrapper.get('[data-testid="xiao-fetch-models"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="xiao-import-models"]').trigger('click')

    expect(syncUpstreamModelsPreview).toHaveBeenCalledWith({
      platform: 'xiaoapi',
      type: 'apikey',
      base_url: 'https://native.example.com',
      api_key: 'native-secret',
      video_protocol: 'native'
    })
    expect((wrapper.get('[data-testid="xiao-price-base-0"]').element as HTMLInputElement).value)
      .toBe('0.26')
  })

  it('imports AIStartLab per-second and fixed-total credit prices into sales rules', async () => {
    listAccounts.mockResolvedValue(listResponse([]))
    syncUpstreamModelsPreview.mockResolvedValue({
      models: ['grok-imagine-video-1.0', 'gemini-omni-flash'],
      model_specs: [
        {
          id: 'grok-imagine-video-1.0',
          resolution: '480p',
          upstream_cost: 9,
          cost_currency: 'CREDITS',
          cost_unit: 'second',
          default_duration: 1,
          default_resolution: true
        },
        {
          id: 'gemini-omni-flash',
          resolution: '720p',
          upstream_cost: 150,
          cost_currency: 'CREDITS',
          cost_unit: 'request',
          default_duration: 10,
          default_resolution: true
        }
      ],
      pricing_source: 'aistartlab_config'
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="xiao-protocol"]').setValue('openai_sora')
    await wrapper.get('[data-testid="xiao-api-key"]').setValue('rotated-secret')
    await wrapper.get('[data-testid="xiao-fetch-models"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="xiao-import-models"]').trigger('click')

    expect((wrapper.get('[data-testid="xiao-price-model-0"]').element as HTMLInputElement).value)
      .toBe('gemini-omni-flash')
    expect((wrapper.get('[data-testid="xiao-price-base-0"]').element as HTMLInputElement).value)
      .toBe('0.195')
    expect((wrapper.get('[data-testid="xiao-price-duration-0"]').element as HTMLInputElement).value)
      .toBe('10')
    expect((wrapper.get('[data-testid="xiao-price-model-1"]').element as HTMLInputElement).value)
      .toBe('grok-imagine-video-1.0')
    expect((wrapper.get('[data-testid="xiao-price-base-1"]').element as HTMLInputElement).value)
      .toBe('0.117')
    expect((wrapper.get('[data-testid="xiao-price-duration-1"]').element as HTMLInputElement).value)
      .toBe('1')
  })

  it('maps models without creating zero-price or hidden default-duration rules', async () => {
    listAccounts.mockResolvedValue(listResponse([]))
    syncUpstreamModelsPreview.mockResolvedValue({
      models: ['model-without-price'],
      model_specs: [{ id: 'model-without-price', resolution: '720p' }],
      pricing_source: 'model_list',
      pricing_note: 'aistartlab_config_unavailable'
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="xiao-protocol"]').setValue('openai_sora')
    await wrapper.get('[data-testid="xiao-api-key"]').setValue('video-secret')
    await wrapper.get('[data-testid="xiao-fetch-models"]').trigger('click')
    await flushPromises()
    expect(wrapper.get('[data-testid="xiao-pricing-note"]').exists()).toBe(true)

    await wrapper.get('[data-testid="xiao-import-models"]').trigger('click')
    expect((wrapper.get('[data-testid="xiao-mapping-upstream-0"]').element as HTMLInputElement).value)
      .toBe('model-without-price')
    expect((wrapper.get('[data-testid="xiao-price-model-0"]').element as HTMLInputElement).value).toBe('')
    expect(showInfo).toHaveBeenCalledWith('admin.xiaoVideo.importedWithoutPricing')
  })

  it('uses saved credentials when fetching models for an unchanged account', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="xiao-fetch-models"]').trigger('click')
    await flushPromises()

    expect(syncUpstreamModels).toHaveBeenCalledWith(41)
    expect(syncUpstreamModelsPreview).not.toHaveBeenCalled()
  })

  it('fills missing selling prices for an existing mapping without duplicating it', async () => {
    const account = xiaoAccount({
      credentials: {
        base_url: 'https://api.video.aistarslab.com/openai',
        video_protocol: 'openai_sora',
        model_mapping: { 'grok-public': 'grok-imagine-video-1.0' },
        video_pricing: []
      }
    })
    listAccounts.mockResolvedValue(listResponse([account]))
    getAccount.mockResolvedValue(account)
    syncUpstreamModels.mockResolvedValue({
      models: ['grok-imagine-video-1.0'],
      model_specs: [{
        id: 'grok-imagine-video-1.0',
        resolution: '480p',
        upstream_cost: 9,
        cost_currency: 'CREDITS',
        cost_unit: 'second',
        default_duration: 1,
        default_resolution: true
      }],
      pricing_source: 'aistartlab_config'
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="xiao-fetch-models"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('admin.xiaoVideo.modelPricingMissing')
    await wrapper.get('[data-testid="xiao-import-models"]').trigger('click')

    expect(wrapper.findAll('[data-testid^="xiao-mapping-upstream-"]')).toHaveLength(1)
    expect((wrapper.get('[data-testid="xiao-mapping-upstream-0"]').element as HTMLInputElement).value)
      .toBe('grok-imagine-video-1.0')
    expect((wrapper.get('[data-testid="xiao-price-model-0"]').element as HTMLInputElement).value)
      .toBe('grok-public')
    expect((wrapper.get('[data-testid="xiao-price-base-0"]').element as HTMLInputElement).value)
      .toBe('0.117')
  })

  it('tests and deletes the selected upstream through account APIs', async () => {
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(true)
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="xiao-test"]').trigger('click')
    await flushPromises()
    expect(testAccount).toHaveBeenCalledWith(41)
    expect(showSuccess).toHaveBeenCalledWith('connected')

    await wrapper.get('[data-testid="xiao-delete"]').trigger('click')
    await flushPromises()
    expect(deleteAccount).toHaveBeenCalledWith(41)

    confirm.mockRestore()
  })
})
