<template>
  <AppLayout>
    <div class="admin-dashboard space-y-5">
      <div v-if="loading" class="flex items-center justify-center py-16">
        <LoadingSpinner />
      </div>

      <template v-else-if="stats">
        <section class="overview-panel">
          <div class="overview-copy">
            <div class="inline-flex items-center gap-2 rounded-full bg-white/80 px-3 py-1 text-xs font-semibold text-slate-600 ring-1 ring-slate-200">
              <span class="h-2 w-2 rounded-full bg-emerald-500"></span>
              系统运行中
            </div>
            <h2>运营总览</h2>
            <p>快速查看 Key、账号、请求和响应速度，核心状态一屏看清。</p>
          </div>

          <div class="overview-metrics">
            <div class="metric-main">
              <span class="metric-label">今日请求</span>
              <strong>{{ formatNumber(stats.today_requests) }}</strong>
              <span class="metric-note">累计 {{ formatNumber(stats.total_requests) }}</span>
            </div>
            <div class="metric-main">
              <span class="metric-label">平均响应</span>
              <strong>{{ formatDuration(stats.average_duration_ms) }}</strong>
              <span class="metric-note">{{ stats.active_users }} 活跃用户</span>
            </div>
            <div class="metric-main">
              <span class="metric-label">当前吞吐</span>
              <strong>{{ formatTokens(stats.rpm) }}</strong>
              <span class="metric-note">{{ formatTokens(stats.tpm) }} TPM</span>
            </div>
          </div>
        </section>

        <section class="stats-grid">
          <div
            v-for="item in statCards"
            :key="item.label"
            class="stat-tile"
          >
            <div class="stat-tile__top">
              <span class="stat-tile__icon" :class="item.tone">
                <Icon :name="item.icon" size="sm" :stroke-width="2" />
              </span>
              <span class="stat-tile__label">{{ item.label }}</span>
            </div>
            <div class="stat-tile__value">{{ item.value }}</div>
            <div class="stat-tile__hint" :class="item.warning ? 'text-red-500' : ''">
              {{ item.hint }}
            </div>
          </div>
        </section>

        <section class="control-strip">
          <div class="flex flex-wrap items-center gap-3">
            <span class="control-label">{{ t('admin.dashboard.timeRange') }}</span>
            <DateRangePicker
              v-model:start-date="startDate"
              v-model:end-date="endDate"
              @change="onDateRangeChange"
            />
            <button @click="loadDashboardStats" :disabled="chartsLoading" class="btn btn-secondary h-10 px-3">
              <Icon name="refresh" size="sm" :class="{ 'animate-spin': chartsLoading }" />
              {{ t('common.refresh') }}
            </button>
          </div>
          <div class="flex items-center gap-3">
            <span class="control-label">{{ t('admin.dashboard.granularity') }}</span>
            <div class="w-28">
              <Select
                v-model="granularity"
                :options="granularityOptions"
                @change="loadChartData"
              />
            </div>
          </div>
        </section>

        <section class="grid grid-cols-1 gap-5 xl:grid-cols-2">
          <ModelDistributionChart
            :model-stats="modelStats"
            :enable-ranking-view="true"
            :ranking-items="rankingItems"
            :ranking-total-actual-cost="rankingTotalActualCost"
            :ranking-total-requests="rankingTotalRequests"
            :ranking-total-tokens="rankingTotalTokens"
            :loading="chartsLoading"
            :ranking-loading="rankingLoading"
            :ranking-error="rankingError"
            :start-date="startDate"
            :end-date="endDate"
            @ranking-click="goToUserUsage"
          />
          <TokenUsageTrend :trend-data="trendData" :loading="chartsLoading" />
        </section>

        <section class="card p-4">
          <div class="mb-4 flex items-center justify-between gap-3">
            <div>
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ t('admin.dashboard.recentUsage') }} (Top 12)
              </h3>
              <p class="mt-1 text-xs text-slate-500">按用户聚合最近使用趋势，便于发现异常消耗。</p>
            </div>
            <span class="rounded-full bg-slate-100 px-3 py-1 text-xs font-medium text-slate-500">
              {{ granularity === 'hour' ? t('admin.dashboard.hour') : t('admin.dashboard.day') }}
            </span>
          </div>
          <div class="h-64">
            <div v-if="userTrendLoading" class="flex h-full items-center justify-center">
              <LoadingSpinner size="md" />
            </div>
            <Line v-else-if="userTrendChartData" :data="userTrendChartData" :options="lineOptions" />
            <div
              v-else
              class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400"
            >
              {{ t('admin.dashboard.noDataAvailable') }}
            </div>
          </div>
        </section>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type {
  DashboardStats,
  TrendDataPoint,
  ModelStat,
  UserUsageTrendPoint,
  UserSpendingRankingItem
} from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import ModelDistributionChart from '@/components/charts/ModelDistributionChart.vue'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'

import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import { Line } from 'vue-chartjs'

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
)

type StatCardTone = 'tone-red' | 'tone-blue' | 'tone-green' | 'tone-amber' | 'tone-violet'

interface StatCard {
  label: string
  value: string
  hint: string
  icon: 'key' | 'server' | 'users' | 'cube' | 'database'
  tone: StatCardTone
  warning?: boolean
}

const { t } = useI18n()
const appStore = useAppStore()
const router = useRouter()
const stats = ref<DashboardStats | null>(null)
const loading = ref(false)
const chartsLoading = ref(false)
const userTrendLoading = ref(false)
const rankingLoading = ref(false)
const rankingError = ref(false)

const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const userTrend = ref<UserUsageTrendPoint[]>([])
const rankingItems = ref<UserSpendingRankingItem[]>([])
const rankingTotalActualCost = ref(0)
const rankingTotalRequests = ref(0)
const rankingTotalTokens = ref(0)
let chartLoadSeq = 0
let usersTrendLoadSeq = 0
let rankingLoadSeq = 0
const rankingLimit = 12

const formatLocalDate = (date: Date): string => {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

const getLast24HoursRangeDates = (): { start: string; end: string } => {
  const end = new Date()
  const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
  return {
    start: formatLocalDate(start),
    end: formatLocalDate(end)
  }
}

const granularity = ref<'day' | 'hour'>('hour')
const defaultRange = getLast24HoursRangeDates()
const startDate = ref(defaultRange.start)
const endDate = ref(defaultRange.end)

const granularityOptions = computed(() => [
  { value: 'day', label: t('admin.dashboard.day') },
  { value: 'hour', label: t('admin.dashboard.hour') }
])

const isDarkMode = computed(() => {
  return document.documentElement.classList.contains('dark')
})

const chartColors = computed(() => ({
  text: isDarkMode.value ? '#e5e7eb' : '#475569',
  grid: isDarkMode.value ? '#374151' : '#e2e8f0'
}))

const statCards = computed<StatCard[]>(() => {
  if (!stats.value) return []
  return [
    {
      label: t('admin.dashboard.apiKeys'),
      value: formatNumber(stats.value.total_api_keys),
      hint: `${stats.value.active_api_keys} ${t('common.active')}`,
      icon: 'key',
      tone: 'tone-red'
    },
    {
      label: t('admin.dashboard.accounts'),
      value: formatNumber(stats.value.total_accounts),
      hint: stats.value.error_accounts > 0
        ? `${stats.value.error_accounts} ${t('common.error')}`
        : `${stats.value.normal_accounts} ${t('common.active')}`,
      icon: 'server',
      tone: 'tone-blue',
      warning: stats.value.error_accounts > 0
    },
    {
      label: t('admin.dashboard.users'),
      value: formatNumber(stats.value.total_users),
      hint: `今日新增 ${stats.value.today_new_users}`,
      icon: 'users',
      tone: 'tone-green'
    },
    {
      label: t('admin.dashboard.todayTokens'),
      value: formatTokens(stats.value.today_tokens),
      hint: `$${formatCost(stats.value.today_actual_cost)} 实际成本`,
      icon: 'cube',
      tone: 'tone-amber'
    },
    {
      label: t('admin.dashboard.totalTokens'),
      value: formatTokens(stats.value.total_tokens),
      hint: `$${formatCost(stats.value.total_actual_cost)} 累计成本`,
      icon: 'database',
      tone: 'tone-violet'
    }
  ]
})

const lineOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    intersect: false,
    mode: 'index' as const
  },
  plugins: {
    legend: {
      position: 'top' as const,
      labels: {
        color: chartColors.value.text,
        usePointStyle: true,
        pointStyle: 'circle',
        padding: 15,
        font: {
          size: 11
        }
      }
    },
    tooltip: {
      itemSort: (a: any, b: any) => {
        const aValue = typeof a?.raw === 'number' ? a.raw : Number(a?.parsed?.y ?? 0)
        const bValue = typeof b?.raw === 'number' ? b.raw : Number(b?.parsed?.y ?? 0)
        return bValue - aValue
      },
      callbacks: {
        label: (context: any) => {
          return `${context.dataset.label}: ${formatTokens(context.raw)}`
        }
      }
    }
  },
  scales: {
    x: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: 10
        }
      }
    },
    y: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: 10
        },
        callback: (value: string | number) => formatTokens(Number(value))
      }
    }
  }
}))

const userTrendChartData = computed(() => {
  if (!userTrend.value?.length) return null

  const getDisplayName = (point: UserUsageTrendPoint): string => {
    const username = point.username?.trim()
    if (username) {
      return username
    }

    const email = point.email?.trim()
    if (email) {
      return email
    }

    return t('admin.redeem.userPrefix', { id: point.user_id })
  }

  const userGroups = new Map<number, { name: string; data: Map<string, number> }>()
  const allDates = new Set<string>()

  userTrend.value.forEach((point) => {
    allDates.add(point.date)
    const key = point.user_id
    if (!userGroups.has(key)) {
      userGroups.set(key, { name: getDisplayName(point), data: new Map() })
    }
    userGroups.get(key)!.data.set(point.date, point.tokens)
  })

  const sortedDates = Array.from(allDates).sort()
  const colors = [
    '#ef4444',
    '#0ea5e9',
    '#22c55e',
    '#f59e0b',
    '#8b5cf6',
    '#14b8a6',
    '#f97316',
    '#6366f1',
    '#84cc16',
    '#06b6d4',
    '#ec4899',
    '#64748b'
  ]

  const datasets = Array.from(userGroups.values()).map((group, idx) => ({
    label: group.name,
    data: sortedDates.map((date) => group.data.get(date) || 0),
    borderColor: colors[idx % colors.length],
    backgroundColor: `${colors[idx % colors.length]}18`,
    fill: false,
    tension: 0.32
  }))

  return {
    labels: sortedDates,
    datasets
  }
})

const formatTokens = (value: number | undefined): string => {
  if (value === undefined || value === null) return '0'
  if (value >= 1_000_000_000) {
    return `${(value / 1_000_000_000).toFixed(2)}B`
  } else if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(2)}M`
  } else if (value >= 1_000) {
    return `${(value / 1_000).toFixed(2)}K`
  }
  return value.toLocaleString()
}

const formatNumber = (value: number): string => {
  return value.toLocaleString()
}

const formatCost = (value: number): string => {
  if (value >= 1000) {
    return (value / 1000).toFixed(2) + 'K'
  } else if (value >= 1) {
    return value.toFixed(2)
  } else if (value >= 0.01) {
    return value.toFixed(3)
  }
  return value.toFixed(4)
}

const formatDuration = (ms: number): string => {
  if (ms >= 1000) {
    return `${(ms / 1000).toFixed(2)}s`
  }
  return `${Math.round(ms)}ms`
}

const goToUserUsage = (item: UserSpendingRankingItem) => {
  void router.push({
    path: '/admin/usage',
    query: {
      user_id: String(item.user_id),
      start_date: startDate.value,
      end_date: endDate.value
    }
  })
}

const onDateRangeChange = (range: {
  startDate: string
  endDate: string
  preset: string | null
}) => {
  const start = new Date(range.startDate)
  const end = new Date(range.endDate)
  const daysDiff = Math.ceil((end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24))

  if (daysDiff <= 1) {
    granularity.value = 'hour'
  } else {
    granularity.value = 'day'
  }

  loadChartData()
}

const loadDashboardSnapshot = async (includeStats: boolean) => {
  const currentSeq = ++chartLoadSeq
  if (includeStats && !stats.value) {
    loading.value = true
  }
  chartsLoading.value = true
  try {
    const response = await adminAPI.dashboard.getSnapshotV2({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      include_stats: includeStats,
      include_trend: true,
      include_model_stats: true,
      include_group_stats: false,
      include_users_trend: false
    })
    if (currentSeq !== chartLoadSeq) return
    if (includeStats && response.stats) {
      stats.value = response.stats
    }
    trendData.value = response.trend || []
    modelStats.value = response.models || []
  } catch (error) {
    if (currentSeq !== chartLoadSeq) return
    appStore.showError(t('admin.dashboard.failedToLoad'))
    console.error('Error loading dashboard snapshot:', error)
  } finally {
    if (currentSeq === chartLoadSeq) {
      loading.value = false
      chartsLoading.value = false
    }
  }
}

const loadUsersTrend = async () => {
  const currentSeq = ++usersTrendLoadSeq
  userTrendLoading.value = true
  try {
    const response = await adminAPI.dashboard.getUserUsageTrend({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      limit: 12
    })
    if (currentSeq !== usersTrendLoadSeq) return
    userTrend.value = response.trend || []
  } catch (error) {
    if (currentSeq !== usersTrendLoadSeq) return
    console.error('Error loading users trend:', error)
    userTrend.value = []
  } finally {
    if (currentSeq === usersTrendLoadSeq) {
      userTrendLoading.value = false
    }
  }
}

const loadUserSpendingRanking = async () => {
  const currentSeq = ++rankingLoadSeq
  rankingLoading.value = true
  rankingError.value = false
  try {
    const response = await adminAPI.dashboard.getUserSpendingRanking({
      start_date: startDate.value,
      end_date: endDate.value,
      limit: rankingLimit
    })
    if (currentSeq !== rankingLoadSeq) return
    rankingItems.value = response.ranking || []
    rankingTotalActualCost.value = response.total_actual_cost || 0
    rankingTotalRequests.value = response.total_requests || 0
    rankingTotalTokens.value = response.total_tokens || 0
  } catch (error) {
    if (currentSeq !== rankingLoadSeq) return
    console.error('Error loading user spending ranking:', error)
    rankingItems.value = []
    rankingTotalActualCost.value = 0
    rankingTotalRequests.value = 0
    rankingTotalTokens.value = 0
    rankingError.value = true
  } finally {
    if (currentSeq === rankingLoadSeq) {
      rankingLoading.value = false
    }
  }
}

const loadDashboardStats = async () => {
  await Promise.all([
    loadDashboardSnapshot(true),
    loadUsersTrend(),
    loadUserSpendingRanking()
  ])
}

const loadChartData = async () => {
  await Promise.all([
    loadDashboardSnapshot(false),
    loadUsersTrend(),
    loadUserSpendingRanking()
  ])
}

onMounted(() => {
  loadDashboardStats()
})
</script>

<style scoped>
.admin-dashboard {
  color: #172033;
}

.overview-panel {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(420px, 0.9fr);
  gap: 1rem;
  overflow: hidden;
  border: 1px solid rgb(226 232 240);
  border-radius: 1.25rem;
  background:
    radial-gradient(circle at 8% 12%, rgba(255, 212, 71, 0.28), transparent 28%),
    radial-gradient(circle at 92% 16%, rgba(8, 169, 214, 0.18), transparent 30%),
    linear-gradient(135deg, #fffdf8 0%, #f8fbff 52%, #fff8f6 100%);
  padding: 1.25rem;
  box-shadow: 0 14px 34px rgba(15, 23, 42, 0.06);
}

.overview-copy h2 {
  margin-top: 0.8rem;
  font-size: clamp(1.35rem, 2vw, 2rem);
  font-weight: 800;
  line-height: 1.1;
  color: #172033;
}

.overview-copy p {
  margin-top: 0.45rem;
  max-width: 34rem;
  color: #64748b;
  font-size: 0.9rem;
}

.overview-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
}

.metric-main {
  border: 1px solid rgba(226, 232, 240, 0.9);
  border-radius: 1rem;
  background: rgba(255, 255, 255, 0.82);
  padding: 0.9rem;
}

.metric-label,
.control-label {
  font-size: 0.76rem;
  font-weight: 700;
  color: #64748b;
}

.metric-main strong {
  display: block;
  margin-top: 0.35rem;
  font-size: 1.55rem;
  line-height: 1;
  color: #0f172a;
}

.metric-note {
  display: block;
  margin-top: 0.4rem;
  font-size: 0.76rem;
  color: #64748b;
}

.stat-tile {
  min-height: 7.35rem;
  border: 1px solid rgb(226 232 240);
  border-radius: 1rem;
  background: rgba(255, 255, 255, 0.96);
  padding: 0.9rem;
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.045);
  transition:
    transform 180ms ease,
    border-color 180ms ease,
    box-shadow 180ms ease;
}

.stat-tile:hover {
  transform: translateY(-2px);
  border-color: rgb(203 213 225);
  box-shadow: 0 14px 32px rgba(15, 23, 42, 0.075);
}

.stat-tile__top {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.stat-tile__icon {
  display: inline-flex;
  height: 2rem;
  width: 2rem;
  align-items: center;
  justify-content: center;
  border-radius: 0.75rem;
}

.tone-red {
  background: #fff1ef;
  color: #d92d20;
}

.tone-blue {
  background: #eefbff;
  color: #0788b5;
}

.tone-green {
  background: #edfff7;
  color: #0f9f72;
}

.tone-amber {
  background: #fff8dc;
  color: #b77900;
}

.tone-violet {
  background: #f4f1ff;
  color: #6d4ee6;
}

.stat-tile__label {
  min-width: 0;
  overflow: hidden;
  color: #64748b;
  font-size: 0.78rem;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.stat-tile__value {
  margin-top: 0.85rem;
  color: #0f172a;
  font-size: 1.55rem;
  font-weight: 800;
  line-height: 1;
}

.stat-tile__hint {
  margin-top: 0.55rem;
  color: #64748b;
  font-size: 0.78rem;
  font-weight: 500;
}

.control-strip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border: 1px solid rgb(226 232 240);
  border-radius: 1rem;
  background: rgba(255, 255, 255, 0.96);
  padding: 0.75rem 0.9rem;
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.045);
}

.stats-grid {
  display: grid;
  gap: 0.75rem;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
}

@media (max-width: 1180px) {
  .overview-panel {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 760px) {
  .overview-metrics {
    grid-template-columns: 1fr;
  }

  .control-strip {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
