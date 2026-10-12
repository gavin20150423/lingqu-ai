/**
 * 渠道定价「保存即清空」回归测试。
 *
 * 线上事故：渠道「国模」(id=3) 在页面上点一次保存，7 条定价被静默删光。
 * 链路是 formToAPI() 的 `if (!section.enabled) continue` —— 平台分区被取消勾选后，
 * 该分区的定价被静默丢弃，产出的空数组又命中后端「全量替换」语义执行了 DELETE。
 *
 * 修正后的约定：
 *   1. 平台分区被取消勾选但仍持有定价时，界面上必须给出警告；
 *   2. 本次提交会清空已有定价/规则时，必须先弹确认框，用户确认后才带 clear_* 标记下发；
 *   3. 渠道本来就没有数据时不下发空数组（省略字段 = 后端保持原样）。
 */
import { defineComponent, h } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Channel, ChannelModelPricing } from '@/api/admin/channels'
import { resetPlatformCatalog } from '@/constants/platformCatalog'
import ChannelsView from '@/views/admin/ChannelsView.vue'

const { listChannels, getAllGroups, updateChannel, createChannel, getAccountById, showSuccess, showError } =
  vi.hoisted(() => ({
    listChannels: vi.fn(),
    getAllGroups: vi.fn(),
    updateChannel: vi.fn(),
    createChannel: vi.fn(),
    getAccountById: vi.fn(),
    showSuccess: vi.fn(),
    showError: vi.fn()
  }))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    channels: {
      list: listChannels,
      update: updateChannel,
      create: createChannel,
      syncPricingModels: vi.fn(),
      remove: vi.fn()
    },
    groups: { getAll: getAllGroups },
    accounts: { getById: getAccountById },
    settings: { getWebSearchEmulationConfig: vi.fn().mockResolvedValue({ enabled: false }) }
  }
}))

vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess, showError }) }))

vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),
  // 返回 key 本身，便于断言「渲染了哪个文案」
  useI18n: () => ({ t: (key: string) => key })
}))

// DataTable 真组件依赖列渲染函数；这里退化成「每行渲染 actions 插槽」，
// 让测试能点到行内的编辑按钮。
const DataTableStub = defineComponent({
  name: 'DataTable',
  props: {
    columns: { type: Array, default: () => [] },
    data: { type: Array, default: () => [] }
  },
  setup(props, { slots }) {
    return () => h('div', { class: 'data-table-stub' },
      (props.data as Channel[]).map(row =>
        h('div', { class: 'data-table-row', key: row.id },
          slots['cell-actions']?.({ row, value: row.id }))
      )
    )
  }
})

const ConfirmDialogStub = defineComponent({
  name: 'ConfirmDialog',
  props: {
    show: { type: Boolean, default: false },
    title: { type: String, default: '' },
    message: { type: String, default: '' },
    confirmText: { type: String, default: '' },
    cancelText: { type: String, default: '' },
    danger: { type: Boolean, default: false }
  },
  emits: ['confirm', 'cancel'],
  setup(props, { emit }) {
    return () => props.show
      ? h('div', { class: 'confirm-stub', 'data-message': props.message }, [
        h('button', { class: 'confirm-ok', onClick: () => emit('confirm') }, 'confirm'),
        h('button', { class: 'confirm-cancel', onClick: () => emit('cancel') }, 'cancel')
      ])
      : null
  }
})

function pricingFixture(models: string[]): ChannelModelPricing {
  return {
    platform: 'openai',
    models,
    billing_mode: 'token',
    input_price: 0.000003,
    output_price: 0.000015,
    cache_write_price: 0.00000375,
    cache_read_price: 0.0000003,
    image_input_price: null,
    image_output_price: null,
    per_request_price: null,
    intervals: [],
    time_pricing: null
  }
}

function channelFixture(overrides: Partial<Channel> = {}): Channel {
  return {
    id: 3,
    name: '国模',
    description: 'GLM / DeepSeek / Kimi / MiniMax 渠道',
    status: 'active',
    billing_model_source: 'channel_mapped',
    restrict_models: true,
    features_config: {},
    group_ids: [142],
    model_pricing: [pricingFixture(['glm-5.3']), pricingFixture(['kimi-k3'])],
    model_mapping: {},
    apply_pricing_to_account_stats: false,
    account_stats_pricing_rules: [],
    created_at: '2026-09-16T01:14:35Z',
    updated_at: '2026-10-12T03:19:51Z',
    ...overrides
  }
}

function mountView() {
  return mount(ChannelsView, {
    global: {
      stubs: {
        AppLayout: defineComponent({ template: '<main><slot /></main>' }),
        TablePageLayout: defineComponent({
          template: '<section><slot name="filters" /><slot name="table" /><slot name="pagination" /></section>'
        }),
        BaseDialog: defineComponent({
          props: ['show'],
          template: '<section v-if="show"><slot /><slot name="footer" /></section>'
        }),
        DataTable: DataTableStub,
        ConfirmDialog: ConfirmDialogStub,
        PricingEntryCard: defineComponent({
          props: ['entry'],
          template: '<div class="pricing-entry">{{ entry.models.join(",") }}</div>'
        }),
        Pagination: true,
        EmptyState: true,
        Select: true,
        Icon: true,
        PlatformIcon: true,
        Toggle: true
      }
    }
  })
}

async function openEditDialog(wrapper: VueWrapper) {
  await flushPromises()
  const editButton = wrapper.findAll('button').find(button => button.text() === 'common.edit')
  expect(editButton, '未能找到行内编辑按钮').toBeDefined()
  await editButton!.trigger('click')
  await flushPromises()
}

/** 平台勾选区里的复选框（按平台名定位 label）。 */
function platformCheckbox(wrapper: VueWrapper, platform: string) {
  const label = wrapper.findAll('label')
    .find(item => item.text() === `admin.groups.platforms.${platform}`)
  expect(label, `未找到平台 ${platform} 的勾选项`).toBeDefined()
  return label!.find('input[type="checkbox"]')
}

/** 提交按钮在 BaseDialog 的 footer 里，靠 form 属性关联表单；jsdom 下点 click 不会触发 submit，直接派发 submit 事件。 */
async function submitForm(wrapper: VueWrapper) {
  const form = wrapper.find('form#channel-form')
  expect(form.exists(), '未找到渠道表单').toBe(true)
  await form.trigger('submit')
  await flushPromises()
}

/** 切换到某平台的 tab，定价卡片才可见（v-show="section.enabled && activeTab === section.platform"）。 */
async function openPlatformTab(wrapper: VueWrapper, platform: string) {
  const tab = wrapper.findAll('button.channel-tab')
    .find(button => button.text() === `admin.groups.platforms.${platform}`)
  expect(tab, `未找到平台 ${platform} 的 tab`).toBeDefined()
  await tab!.trigger('click')
  await flushPromises()
}

function visiblePricingEntries(wrapper: VueWrapper) {
  return wrapper.findAll('.pricing-entry').filter(entry => entry.isVisible())
}

function updateCalls() {
  return updateChannel.mock.calls as Array<[number, Record<string, unknown>]>
}

function lastUpdateBody() {
  const calls = updateCalls()
  expect(calls.length, '未调用更新接口').toBeGreaterThan(0)
  return calls[calls.length - 1][1]
}

describe('ChannelsView 渠道定价保存守卫', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    resetPlatformCatalog()
    getAccountById.mockResolvedValue({ id: 1, name: 'acc' })
    getAllGroups.mockResolvedValue([
      {
        id: 142,
        name: '国模订阅',
        platform: 'composite',
        subscription_type: 'subscription',
        status: 'active'
      }
    ])
    updateChannel.mockResolvedValue(undefined)
    createChannel.mockResolvedValue(undefined)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('取消勾选平台后保存：弹出确认框，不直接提交', async () => {
    const channel = channelFixture()
    listChannels.mockResolvedValue({ items: [channel], total: 1 })

    const wrapper = mountView()
    await openEditDialog(wrapper)

    // 取消勾选 openai（该渠道的定价全部挂在这个平台上）
    await platformCheckbox(wrapper, 'openai').setValue(false)
    await flushPromises()

    // 界面上应提示「这些平台的定价会被丢弃」
    expect(wrapper.text()).toContain('admin.channels.form.droppedPricingWarning')
    // 定价卡片随之隐藏（v-show，仍在 DOM 但不可见）
    expect(visiblePricingEntries(wrapper)).toHaveLength(0)

    await submitForm(wrapper)

    // 只是弹确认框，没有发请求
    expect(wrapper.find('.confirm-stub').exists()).toBe(true)
    expect(updateChannel).not.toHaveBeenCalled()
    expect(wrapper.find('.confirm-stub').attributes('data-message'))
      .toContain('admin.channels.form.clearPricingMessage')
  })

  it('确认清空后才提交，并显式带上 clear_model_pricing', async () => {
    const channel = channelFixture()
    listChannels.mockResolvedValue({ items: [channel], total: 1 })

    const wrapper = mountView()
    await openEditDialog(wrapper)
    await platformCheckbox(wrapper, 'openai').setValue(false)
    await flushPromises()
    await submitForm(wrapper)

    await wrapper.find('.confirm-ok').trigger('click')
    await flushPromises()

    const body = lastUpdateBody()
    expect(body.model_pricing).toEqual([])
    expect(body.clear_model_pricing).toBe(true)
    // 确认清空后弹窗关闭、请求成功
    expect(wrapper.find('.confirm-stub').exists()).toBe(false)
    expect(showSuccess).toHaveBeenCalled()
  })

  it('取消确认：完全不提交', async () => {
    const channel = channelFixture()
    listChannels.mockResolvedValue({ items: [channel], total: 1 })

    const wrapper = mountView()
    await openEditDialog(wrapper)
    await platformCheckbox(wrapper, 'openai').setValue(false)
    await flushPromises()
    await submitForm(wrapper)

    await wrapper.find('.confirm-cancel').trigger('click')
    await flushPromises()

    expect(updateChannel).not.toHaveBeenCalled()
    expect(wrapper.find('.confirm-stub').exists()).toBe(false)
    // 表单仍然开着，用户可继续修改
    expect(wrapper.find('form#channel-form').exists()).toBe(true)
  })

  it('平台保持勾选：正常提交全部定价，且不带任何 clear 标记', async () => {
    const channel = channelFixture()
    listChannels.mockResolvedValue({ items: [channel], total: 1 })

    const wrapper = mountView()
    await openEditDialog(wrapper)
    // 定价卡片只在对应平台 tab 下可见
    await openPlatformTab(wrapper, 'openai')
    expect(visiblePricingEntries(wrapper)).toHaveLength(2)

    await submitForm(wrapper)

    expect(wrapper.find('.confirm-stub').exists()).toBe(false)

    const body = lastUpdateBody()
    const pricing = body.model_pricing as ChannelModelPricing[]
    expect(pricing).toHaveLength(2)
    expect(pricing.map(p => p.models[0]).sort()).toEqual(['glm-5.3', 'kimi-k3'])
    // 关键：省略 clear_* 键，而不是传 false
    expect('clear_model_pricing' in body).toBe(false)
    expect('clear_account_stats_pricing_rules' in body).toBe(false)
    // 未触发「已取消勾选」警告
    expect(wrapper.text()).not.toContain('admin.channels.form.droppedPricingWarning')
  })

  it('渠道本来就没有定价：省略 model_pricing 字段，不下发空数组', async () => {
    const channel = channelFixture({ model_pricing: [] })
    listChannels.mockResolvedValue({ items: [channel], total: 1 })

    const wrapper = mountView()
    await openEditDialog(wrapper)
    await submitForm(wrapper)

    expect(wrapper.find('.confirm-stub').exists()).toBe(false)

    const body = lastUpdateBody()
    // 关键：字段被省略（undefined），后端据此「保持原样」，不会误删
    expect('model_pricing' in body).toBe(false)
    expect(body.model_pricing).toBeUndefined()
    expect('account_stats_pricing_rules' in body).toBe(false)
  })

  it('渠道持有平台清单外的定价：提示装不下，并在提交前确认', async () => {
    const channel = channelFixture({
      model_pricing: [
        pricingFixture(['glm-5.3']),
        { ...pricingFixture(['legacy-model']), platform: 'platform-removed-from-catalog' }
      ]
    })
    listChannels.mockResolvedValue({ items: [channel], total: 1 })

    const wrapper = mountView()
    await openEditDialog(wrapper)

    // 界面提示有装不下的条目
    expect(wrapper.text()).toContain('admin.channels.form.unloadablePricingWarning')

    await submitForm(wrapper)

    // 不能静默丢弃：先弹确认
    expect(wrapper.find('.confirm-stub').exists()).toBe(true)
    expect(wrapper.find('.confirm-stub').attributes('data-message'))
      .toContain('admin.channels.form.unloadablePricingMessage')
    expect(updateChannel).not.toHaveBeenCalled()

    await wrapper.find('.confirm-ok').trigger('click')
    await flushPromises()

    // 确认后按「全量替换」下发剩下那一条；后端非空数组即整体替换，
    // 装不下的条目自然被移除，无需再带 clear_* 标记。
    const body = lastUpdateBody()
    expect(body.model_pricing).toHaveLength(1)
    expect('clear_model_pricing' in body).toBe(false)
  })

  it('新建渠道时仍下发 model_pricing 数组（无全量替换风险）', async () => {
    const channel = channelFixture()
    listChannels.mockResolvedValue({ items: [channel], total: 1 })

    const wrapper = mountView()
    await flushPromises()
    const createButton = wrapper.findAll('button').find(button => button.text() === 'admin.channels.createChannel')
    expect(createButton).toBeDefined()
    await createButton!.trigger('click')
    await flushPromises()

    await wrapper.find('form#channel-form input[type="text"]').setValue('新渠道')
    await flushPromises()

    await submitForm(wrapper)

    expect(showError).not.toHaveBeenCalled()
    expect(wrapper.find('.confirm-stub').exists()).toBe(false)
    expect(createChannel).toHaveBeenCalledTimes(1)
    const body = createChannel.mock.calls[0][0] as Record<string, unknown>
    expect(Array.isArray(body.model_pricing)).toBe(true)
    expect('clear_model_pricing' in body).toBe(false)
  })
})
