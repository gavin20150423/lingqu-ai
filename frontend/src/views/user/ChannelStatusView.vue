<template>
  <UserWorkspaceLayout>
    <MonitorHero
      :overall-status="overallStatus"
      :total="items.length"
      :counts="statusCounts"
      :loading="loading"
      :auto-refresh="autoRefresh"
      @refresh="manualReload"
    />

    <section class="monitor-filters" aria-label="渠道筛选">
      <div class="monitor-filter-row">
        <span class="monitor-filter-row__label">模型</span>
        <button
          v-for="option in providerOptions"
          :key="option.value"
          type="button"
          :class="{ 'monitor-filter-chip--active': providerFilter === option.value }"
          @click="providerFilter = option.value"
        >
          {{ option.label }}
        </button>
      </div>
      <div class="monitor-filter-row">
        <span class="monitor-filter-row__label">渠道</span>
        <button
          v-for="group in groupOptions"
          :key="group"
          type="button"
          :class="{ 'monitor-filter-chip--active': groupFilter === group }"
          @click="groupFilter = group"
        >
          {{ group }}
        </button>
      </div>
      <div class="monitor-filter-row monitor-filter-row--window">
        <span class="monitor-filter-row__label">统计</span>
        <button
          v-for="option in windowOptions"
          :key="option.value"
          type="button"
          :class="{ 'monitor-filter-chip--active': currentWindow === option.value }"
          @click="handleWindowChange(option.value)"
        >
          {{ option.label }}
        </button>
      </div>
    </section>

    <MonitorCardGrid
      :items="filteredItems"
      :window="currentWindow"
      :countdown-seconds="countdown"
      :loading="loading"
      :detail-cache="detailCache"
      @card-click="openDetail"
    />

    <MonitorDetailDialog
      :show="showDetail"
      :monitor-id="detailTarget?.id ?? null"
      :title="detailTitle"
      :detail-override="detailTarget ? detailCache[detailTarget.id] : null"
      @close="closeDetail"
    />
  </UserWorkspaceLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  list as listChannelMonitorViews,
  status as fetchChannelMonitorDetail,
  type Provider,
  type UserMonitorView,
  type UserMonitorDetail,
} from '@/api/channelMonitor'
import UserWorkspaceLayout from '@/components/layout/UserWorkspaceLayout.vue'
import MonitorHero, {
  type MonitorWindow,
  type OverallStatus,
} from '@/components/user/monitor/MonitorHero.vue'
import MonitorCardGrid from '@/components/user/monitor/MonitorCardGrid.vue'
import MonitorDetailDialog from '@/components/user/MonitorDetailDialog.vue'
import { DEFAULT_INTERVAL_SECONDS, STATUS_OPERATIONAL } from '@/constants/channelMonitor'
import { useAutoRefresh } from '@/composables/useAutoRefresh'
import {
  createMockChannelMonitorDetails,
  createMockChannelMonitors,
} from '@/mocks/channelMonitor'

const { t } = useI18n()
const appStore = useAppStore()
const USE_CHANNEL_STATUS_MOCK = import.meta.env.DEV

// ── State ──
const items = ref<UserMonitorView[]>([])
const loading = ref(false)
const currentWindow = ref<MonitorWindow>('7d')
const detailCache = reactive<Record<number, UserMonitorDetail>>({})
const showDetail = ref(false)
const detailTarget = ref<UserMonitorView | null>(null)
const providerFilter = ref('全部')
const groupFilter = ref('全部')

let abortController: AbortController | null = null

if (USE_CHANNEL_STATUS_MOCK) {
  Object.assign(detailCache, createMockChannelMonitorDetails())
}

const autoRefresh = useAutoRefresh({
  storageKey: 'channel-status-auto-refresh',
  intervals: [30, 60, 120] as const,
  defaultInterval: DEFAULT_INTERVAL_SECONDS,
  onRefresh: () => reload(true),
  shouldPause: () => document.hidden || loading.value,
})
const countdown = autoRefresh.countdown

// ── Computed ──
const overallStatus = computed<OverallStatus>(() => {
  if (items.value.length === 0) return 'operational'
  for (const it of items.value) {
    if (it.primary_status === 'failed' || it.primary_status === 'error') return 'degraded'
    if (it.primary_status !== STATUS_OPERATIONAL) return 'degraded'
  }
  return 'operational'
})

const statusCounts = computed(() => {
  return items.value.reduce(
    (counts, item) => {
      if (item.primary_status === 'operational') counts.operational += 1
      else if (item.primary_status === 'degraded') counts.degraded += 1
      else counts.failed += 1
      return counts
    },
    { operational: 0, degraded: 0, failed: 0 },
  )
})

const providerOptions = computed(() => {
  const options: Array<{ value: Provider; label: string }> = [
    { value: 'anthropic', label: 'Claude' },
    { value: 'openai', label: 'GPT' },
    { value: 'gemini', label: 'Gemini' },
  ]
  const providers = new Set(items.value.map(item => item.provider))
  return [
    { value: '全部', label: '全部' },
    ...options.filter(option => providers.has(option.value)),
  ]
})

const groupOptions = computed(() => {
  const preferredOrder = ['aws', 'kiro', 'max', 'plus', 'pro', '福利特价', '福利特价！！！']
  const groups = Array.from(new Set(items.value.map(item => item.group_name).filter(Boolean)))
  groups.sort((a, b) => {
    const aIndex = preferredOrder.indexOf(a)
    const bIndex = preferredOrder.indexOf(b)
    if (aIndex === -1 && bIndex === -1) return a.localeCompare(b)
    if (aIndex === -1) return 1
    if (bIndex === -1) return -1
    return aIndex - bIndex
  })
  return ['全部', ...groups]
})

const windowOptions: Array<{ value: MonitorWindow; label: string }> = [
  { value: '7d', label: '7天' },
  { value: '15d', label: '15天' },
  { value: '30d', label: '30天' },
]

const filteredItems = computed(() => {
  return items.value.filter((item) => {
    const providerMatches = providerFilter.value === '全部' || item.provider === providerFilter.value
    const groupMatches = groupFilter.value === '全部' || item.group_name === groupFilter.value
    return providerMatches && groupMatches
  })
})

const detailTitle = computed(() => {
  return detailTarget.value?.name || t('channelStatus.detailTitle')
})

// ── Loaders ──
async function reload(silent = false) {
  if (USE_CHANNEL_STATUS_MOCK) {
    if (!silent) loading.value = true
    items.value = createMockChannelMonitors()
    countdown.value = DEFAULT_INTERVAL_SECONDS
    if (!silent) loading.value = false
    return
  }

  if (abortController) abortController.abort()
  const ctrl = new AbortController()
  abortController = ctrl
  if (!silent) loading.value = true
  try {
    const res = await listChannelMonitorViews({ signal: ctrl.signal })
    if (ctrl.signal.aborted || abortController !== ctrl) return
    items.value = res.items || []
  } catch (err: unknown) {
    const e = err as { name?: string; code?: string }
    if (e?.name === 'AbortError' || e?.code === 'ERR_CANCELED') return
    appStore.showError(extractApiErrorMessage(err, t('channelStatus.loadError')))
  } finally {
    if (abortController === ctrl) {
      if (!silent) loading.value = false
      countdown.value = DEFAULT_INTERVAL_SECONDS
      abortController = null
    }
  }
}

async function manualReload() {
  await reload(false)
  // After base reload, refresh any cached detail records so non-7d availability
  // values stay in sync without forcing the user to switch tabs again.
  if (currentWindow.value !== '7d') {
    await Promise.all(items.value.map(it => loadDetail(it.id, true)))
  }
}

async function loadDetail(id: number, force = false) {
  if (!force && detailCache[id]) return
  if (USE_CHANNEL_STATUS_MOCK) {
    Object.assign(detailCache, createMockChannelMonitorDetails())
    return
  }
  try {
    detailCache[id] = await fetchChannelMonitorDetail(id)
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('channelStatus.detailLoadError')))
  }
}

async function ensureDetailsForWindow() {
  if (currentWindow.value === '7d') return
  await Promise.all(items.value.map(it => loadDetail(it.id)))
}

// ── Handlers ──
async function handleWindowChange(value: MonitorWindow) {
  currentWindow.value = value
  await ensureDetailsForWindow()
}

function openDetail(row: UserMonitorView) {
  detailTarget.value = row
  showDetail.value = true
}

function closeDetail() {
  showDetail.value = false
  detailTarget.value = null
}

watch(items, () => {
  void ensureDetailsForWindow()
})

watch(
  () => appStore.cachedPublicSettings?.channel_monitor_enabled,
  (enabled) => {
    if (enabled === false) autoRefresh.stop()
    else if (autoRefresh.enabled.value) autoRefresh.start()
  },
)

onMounted(() => {
  void reload(false)
  if (appStore.cachedPublicSettings?.channel_monitor_enabled !== false) {
    autoRefresh.setEnabled(true)
  }
})

onBeforeUnmount(() => {
  if (abortController) abortController.abort()
})
</script>

<style scoped>
.monitor-filters {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.45rem 1rem;
  margin-bottom: 0.75rem;
  padding: 0.08rem 0;
}

.monitor-filter-row {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.34rem;
}

.monitor-filter-row--window {
  margin-left: auto;
}

.monitor-filter-row__label {
  margin-right: 0.1rem;
  color: #777169;
  font-size: 0.68rem;
  font-weight: 750;
}

.monitor-filter-row button {
  min-height: 1.72rem;
  border: 1px solid #e1ddd5;
  border-radius: 999px;
  background: #fffdf9;
  color: #68625b;
  padding: 0 0.62rem;
  font-size: 0.68rem;
  font-weight: 650;
  white-space: nowrap;
}

.monitor-filter-row button:hover {
  border-color: #d1c9bf;
  color: #34312d;
}

.monitor-filter-row .monitor-filter-chip--active {
  border-color: #d27659;
  background: #d27659;
  color: #fff;
}

@media (max-width: 900px) {
  .monitor-filter-row--window {
    width: 100%;
    margin-left: 0;
  }
}

@media (max-width: 640px) {
  .monitor-filter-row {
    width: 100%;
    overflow-x: auto;
    padding-bottom: 0.18rem;
  }
}
</style>
