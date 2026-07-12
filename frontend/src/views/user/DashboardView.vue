<template>
  <UserWorkspaceLayout>
    <div v-if="theme === 'business'" class="business-dashboard">
      <section class="business-dashboard__metrics" aria-label="账户概览">
        <article v-for="item in businessMetrics" :key="item.label">
          <span class="business-dashboard__metric-icon" :class="item.iconClass">
            <Icon :name="item.icon" size="md" />
          </span>
          <div>
            <small>{{ item.label }}</small>
            <strong :class="{ 'business-dashboard__metric-value--accent': item.accent }">
              {{ item.value }}
            </strong>
            <span>{{ item.detail }}</span>
          </div>
        </article>
      </section>

      <div class="business-dashboard__analytics">
        <section class="business-dashboard__panel business-dashboard__trend">
          <div class="business-dashboard__panel-head">
            <h2>Token 使用趋势</h2>
            <div class="business-dashboard__range">
              <button type="button" class="is-active">近 7 天</button>
              <router-link to="/usage">详细用量</router-link>
            </div>
          </div>
          <div class="business-dashboard__chart">
            <Line
              v-if="trend.length > 0"
              :data="trendChartData"
              :options="trendChartOptions"
            />
            <div v-else class="business-dashboard__empty-chart">
              <Icon name="chart" size="lg" />
              <span>{{ dashboardLoading ? '正在加载趋势数据' : '近 7 天暂无调用数据' }}</span>
            </div>
          </div>
        </section>

        <section class="business-dashboard__panel business-dashboard__availability-panel">
          <div class="business-dashboard__panel-head">
            <h2>服务可用性</h2>
            <router-link to="/monitor">状态详情</router-link>
          </div>
          <div class="business-dashboard__availability-summary">
            <div>
              <span :class="{ 'is-healthy': serviceHealthy }">
                <i></i>
                {{ serviceStatusLabel }}
              </span>
              <strong>{{ availabilityRate }}</strong>
            </div>
            <small>{{ monitors.length }} 个监控服务</small>
          </div>
          <div class="business-dashboard__availability-track" aria-label="近期服务状态">
            <i
              v-for="(point, index) in availabilityPoints"
              :key="index"
              :class="`is-${point}`"
            ></i>
          </div>
          <div class="business-dashboard__monitor-list">
            <div v-for="monitor in monitors.slice(0, 3)" :key="monitor.id">
              <span>{{ monitor.name }}</span>
              <strong>{{ monitor.availability_7d.toFixed(1) }}%</strong>
            </div>
            <p v-if="monitors.length === 0">
              {{ dashboardLoading ? '正在读取服务状态' : '暂无服务监控数据' }}
            </p>
          </div>
        </section>
      </div>

      <div class="business-dashboard__details">
        <section class="business-dashboard__panel business-dashboard__recent">
          <div class="business-dashboard__panel-head">
            <h2>最近调用</h2>
            <router-link to="/usage">查看全部</router-link>
          </div>
          <div class="business-dashboard__table-wrap">
            <table>
              <thead>
                <tr>
                  <th>模型</th>
                  <th>Token</th>
                  <th>耗时</th>
                  <th>费用</th>
                  <th>时间</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="log in recentUsage" :key="log.id">
                  <td><strong>{{ log.model }}</strong></td>
                  <td>{{ formatCompactNumber(log.input_tokens + log.output_tokens) }}</td>
                  <td>{{ formatDuration(log.duration_ms) }}</td>
                  <td class="business-dashboard__cost">${{ formatCost(log.actual_cost) }}</td>
                  <td>{{ formatUsageTime(log.created_at) }}</td>
                </tr>
              </tbody>
            </table>
            <div v-if="recentUsage.length === 0" class="business-dashboard__table-empty">
              {{ dashboardLoading ? '正在加载调用记录' : '暂无调用记录' }}
            </div>
          </div>
        </section>

        <section class="business-dashboard__panel business-dashboard__models">
          <div class="business-dashboard__panel-head">
            <h2>模型使用分布</h2>
            <span>近 7 天</span>
          </div>
          <div v-if="models.length > 0" class="business-dashboard__model-body">
            <div class="business-dashboard__doughnut">
              <Doughnut :data="modelChartData" :options="modelChartOptions" />
              <div>
                <strong>{{ formatCompactNumber(modelTokenTotal) }}</strong>
                <span>tokens</span>
              </div>
            </div>
            <div class="business-dashboard__model-list">
              <div v-for="(model, index) in models.slice(0, 4)" :key="model.model">
                <span><i :style="{ backgroundColor: chartColors[index % chartColors.length] }"></i>{{ model.model }}</span>
                <strong>{{ formatCompactNumber(model.total_tokens) }}</strong>
                <em>${{ formatCost(model.actual_cost) }}</em>
              </div>
            </div>
          </div>
          <div v-else class="business-dashboard__empty-chart business-dashboard__empty-chart--models">
            <Icon name="chart" size="lg" />
            <span>{{ dashboardLoading ? '正在加载模型统计' : '暂无模型使用数据' }}</span>
          </div>
        </section>
      </div>
    </div>

    <div v-else class="user-start-page">
      <section class="user-start-hero">
        <div class="user-start-hero__copy">
          <span class="user-start-hero__badge">欢迎使用灵渠AI</span>
          <h1>先创建一个 Key，就可以开始调用模型。</h1>
          <p>
            不需要先理解账号池、通道、调度这些后台概念。创建 Key、复制接入地址，
            然后像调用 OpenAI 一样调用灵渠AI。
          </p>

          <div class="user-start-hero__actions">
            <router-link to="/keys?create=1" class="user-start-primary">
              <Icon name="plus" size="md" />
              创建我的 Key
            </router-link>
            <button type="button" class="user-start-secondary" @click="copyBaseUrl">
              <Icon name="copy" size="md" />
              复制接入地址
            </button>
          </div>
        </div>

        <div class="user-start-card" aria-label="接入信息">
          <div class="user-start-card__mascot" aria-hidden="true">
            <div class="user-start-card__face">
              <span></span>
              <span></span>
              <i></i>
            </div>
            <Icon name="key" size="lg" />
          </div>
          <div class="user-start-card__line">
            <small>Base URL</small>
            <code>{{ baseUrl }}</code>
          </div>
          <div class="user-start-card__line">
            <small>API Key</small>
            <code>sk-lingqu-••••••••••••</code>
          </div>
        </div>
      </section>

      <section class="user-start-steps" aria-label="快速开始步骤">
        <article v-for="step in steps" :key="step.title" class="user-start-step">
          <span>{{ step.index }}</span>
          <h2>{{ step.title }}</h2>
          <p>{{ step.desc }}</p>
          <router-link v-if="step.to" :to="step.to">{{ step.action }}</router-link>
        </article>
      </section>

      <section class="user-start-lite">
        <div class="user-start-lite__stat">
          <small>余额</small>
          <strong>${{ formatBalance(user?.balance || 0) }}</strong>
        </div>
        <div class="user-start-lite__stat">
          <small>今天请求</small>
          <strong>{{ stats?.today_requests || 0 }}</strong>
        </div>
        <div class="user-start-lite__stat">
          <small>API Key</small>
          <strong>{{ stats?.total_api_keys || 0 }}</strong>
        </div>
        <router-link to="/usage" class="user-start-lite__link">
          查看详细用量
          <Icon name="arrowRight" size="sm" />
        </router-link>
      </section>
    </div>
  </UserWorkspaceLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { Doughnut, Line } from 'vue-chartjs'
import {
  ArcElement,
  CategoryScale,
  Chart as ChartJS,
  Filler,
  Legend,
  LinearScale,
  LineElement,
  PointElement,
  Tooltip
} from 'chart.js'
import UserWorkspaceLayout from '@/components/layout/UserWorkspaceLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useUserThemeStore } from '@/stores/userTheme'
import { usageAPI, type UserDashboardStats as UserStatsType } from '@/api/usage'
import { channelMonitorUserAPI, type UserMonitorView } from '@/api/channelMonitor'
import { useClipboard } from '@/composables/useClipboard'
import type { ModelStat, TrendDataPoint, UsageLog } from '@/types'

ChartJS.register(
  ArcElement,
  CategoryScale,
  Filler,
  Legend,
  LinearScale,
  LineElement,
  PointElement,
  Tooltip
)

const authStore = useAuthStore()
const appStore = useAppStore()
const userThemeStore = useUserThemeStore()
const { copyToClipboard } = useClipboard()
const { theme } = storeToRefs(userThemeStore)

const stats = ref<UserStatsType | null>(null)
const trend = ref<TrendDataPoint[]>([])
const models = ref<ModelStat[]>([])
const recentUsage = ref<UsageLog[]>([])
const monitors = ref<UserMonitorView[]>([])
const dashboardLoading = ref(false)
const user = computed(() => authStore.user)
const baseUrl = computed(() => {
  const configured = appStore.cachedPublicSettings?.api_base_url || appStore.apiBaseUrl
  return configured || `${window.location.origin}/v1`
})

const chartColors = ['#d76442', '#3285d8', '#28a47b', '#8a67d5', '#d6a22e']
const modelTokenTotal = computed(() => models.value.reduce((sum, item) => sum + item.total_tokens, 0))
const serviceHealthy = computed(() => (
  monitors.value.length > 0
  && monitors.value.every(item => item.primary_status === 'operational')
))
const serviceStatusLabel = computed(() => {
  if (monitors.value.length === 0) return '未配置服务监控'
  return serviceHealthy.value ? '服务运行正常' : '部分服务异常'
})
const availabilityRate = computed(() => {
  if (monitors.value.length === 0) return '--'
  const average = monitors.value.reduce((sum, item) => sum + item.availability_7d, 0) / monitors.value.length
  return `${average.toFixed(1)}%`
})
const availabilityPoints = computed(() => {
  const timeline = monitors.value[0]?.timeline?.slice(-28) || []
  if (timeline.length === 0) return Array.from({ length: 28 }, () => 'neutral')
  return timeline.map(point => point.status === 'operational' ? 'healthy' : 'failed')
})
const businessMetrics = computed(() => [
  {
    label: '账户余额',
    value: `$${formatBalance(user.value?.balance || 0)}`,
    detail: '当前可用',
    icon: 'creditCard' as const,
    iconClass: 'is-green',
    accent: true
  },
  {
    label: '今日请求',
    value: formatCompactNumber(stats.value?.today_requests || 0),
    detail: `累计 ${formatCompactNumber(stats.value?.total_requests || 0)}`,
    icon: 'chart' as const,
    iconClass: 'is-blue',
    accent: false
  },
  {
    label: '今日消耗',
    value: `$${formatCost(stats.value?.today_actual_cost || 0)}`,
    detail: `累计 $${formatCost(stats.value?.total_actual_cost || 0)}`,
    icon: 'dollar' as const,
    iconClass: 'is-purple',
    accent: false
  },
  {
    label: '今日 Token',
    value: formatCompactNumber(stats.value?.today_tokens || 0),
    detail: `累计 ${formatCompactNumber(stats.value?.total_tokens || 0)}`,
    icon: 'cube' as const,
    iconClass: 'is-orange',
    accent: false
  },
  {
    label: '平均响应',
    value: formatDuration(stats.value?.average_duration_ms || 0),
    detail: '全部请求均值',
    icon: 'clock' as const,
    iconClass: 'is-violet',
    accent: false
  },
  {
    label: '实时性能',
    value: `${formatCompactNumber(stats.value?.rpm || 0)} RPM`,
    detail: `${formatCompactNumber(stats.value?.tpm || 0)} TPM`,
    icon: 'bolt' as const,
    iconClass: 'is-blue',
    accent: false
  },
  {
    label: 'API 密钥',
    value: `${stats.value?.active_api_keys || 0} 启用`,
    detail: `共 ${stats.value?.total_api_keys || 0} 个`,
    icon: 'key' as const,
    iconClass: 'is-green',
    accent: false
  },
  {
    label: '累计 Token',
    value: formatCompactNumber(stats.value?.total_tokens || 0),
    detail: `输入 ${formatCompactNumber(stats.value?.total_input_tokens || 0)} / 输出 ${formatCompactNumber(stats.value?.total_output_tokens || 0)}`,
    icon: 'database' as const,
    iconClass: 'is-indigo',
    accent: false
  }
])

const trendChartData = computed(() => ({
  labels: trend.value.map(item => formatTrendDate(item.date)),
  datasets: [{
    label: 'Token',
    data: trend.value.map(item => item.total_tokens),
    borderColor: '#d76442',
    backgroundColor: 'rgba(215, 100, 66, 0.09)',
    pointBackgroundColor: '#d76442',
    pointBorderWidth: 0,
    pointRadius: 3,
    pointHoverRadius: 5,
    borderWidth: 2,
    tension: 0.35,
    fill: true
  }]
}))

const trendChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      displayColors: false,
      callbacks: {
        label: (context: { parsed: { y: number | null } }) => `${formatCompactNumber(context.parsed.y || 0)} tokens`
      }
    }
  },
  scales: {
    x: {
      grid: { display: false },
      border: { display: false },
      ticks: { color: '#7b746e', font: { size: 10 } }
    },
    y: {
      beginAtZero: true,
      border: { display: false },
      grid: { color: '#ece5dd' },
      ticks: {
        color: '#7b746e',
        font: { size: 10 },
        callback: (value: string | number) => formatCompactNumber(Number(value))
      }
    }
  }
}

const modelChartData = computed(() => ({
  labels: models.value.map(item => item.model),
  datasets: [{
    data: models.value.map(item => item.total_tokens),
    backgroundColor: models.value.map((_, index) => chartColors[index % chartColors.length]),
    borderWidth: 0,
    hoverOffset: 2
  }]
}))

const modelChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  cutout: '72%',
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (context: { label: string, parsed: number }) => `${context.label}: ${formatCompactNumber(context.parsed)} tokens`
      }
    }
  }
}

const steps = [
  {
    index: '01',
    title: '创建 Key',
    desc: '点一下创建，默认配置就能先用。',
    action: '去创建',
    to: '/keys?create=1'
  },
  {
    index: '02',
    title: '复制接入信息',
    desc: '把 Base URL 和 Key 填进你的项目。',
    action: '',
    to: ''
  },
  {
    index: '03',
    title: '直接调用模型',
    desc: '保持 OpenAI 兼容格式，请求会自动进入灵渠AI。',
    action: '',
    to: ''
  }
] as const

function formatBalance(value: number): string {
  return Number(value || 0).toFixed(2)
}

function formatCost(value: number): string {
  return Number(value || 0).toFixed(4)
}

function formatCompactNumber(value: number): string {
  const amount = Number(value || 0)
  if (amount >= 1_000_000) return `${(amount / 1_000_000).toFixed(2)}M`
  if (amount >= 1_000) return `${(amount / 1_000).toFixed(amount >= 100_000 ? 0 : 1)}K`
  return amount.toLocaleString('zh-CN', { maximumFractionDigits: 1 })
}

function formatDuration(value: number | null | undefined): string {
  const duration = Number(value || 0)
  if (duration <= 0) return '0ms'
  if (duration >= 1000) return `${(duration / 1000).toFixed(2)}s`
  return `${Math.round(duration)}ms`
}

function formatTrendDate(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value.slice(5)
  return `${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

function formatUsageTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}

function copyBaseUrl() {
  copyToClipboard(baseUrl.value, '接入地址已复制')
}

async function loadStats() {
  dashboardLoading.value = true
  try {
    await authStore.refreshUser()
    const endDate = new Date()
    const startDate = new Date()
    startDate.setDate(endDate.getDate() - 6)
    const dateParams = {
      start_date: startDate.toISOString().slice(0, 10),
      end_date: endDate.toISOString().slice(0, 10),
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone
    }
    const [dashboardResult, snapshotResult, usageResult, monitorResult] = await Promise.allSettled([
      usageAPI.getDashboardStats(),
      usageAPI.getDashboardSnapshotV2({
        ...dateParams,
        granularity: 'day',
        include_trend: true,
        include_model_stats: true
      }),
      usageAPI.query({ page: 1, page_size: 8, sort_by: 'created_at', sort_order: 'desc' }),
      channelMonitorUserAPI.list()
    ])
    if (dashboardResult.status === 'fulfilled') {
      stats.value = dashboardResult.value
    }
    if (snapshotResult.status === 'fulfilled') {
      trend.value = snapshotResult.value.trend || []
      models.value = snapshotResult.value.models || []
    }
    if (usageResult.status === 'fulfilled') {
      recentUsage.value = usageResult.value.items || []
    }
    if (monitorResult.status === 'fulfilled') {
      monitors.value = monitorResult.value.items || []
    }
  } catch (error) {
    console.warn('Failed to load user dashboard data:', error)
  } finally {
    dashboardLoading.value = false
  }
}

onMounted(() => {
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
  loadStats()
})
</script>

<style scoped>
.user-start-page {
  display: grid;
  gap: 0.9rem;
  width: 100%;
}

.business-dashboard {
  display: grid;
  gap: 0.9rem;
  width: 100%;
  color: #2f2b28;
}

.business-dashboard__metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.75rem;
}

.business-dashboard__metrics article {
  display: grid;
  min-height: 5.65rem;
  min-width: 0;
  grid-template-columns: 2.25rem minmax(0, 1fr);
  gap: 0.7rem;
  align-items: center;
  border: 1px solid #e8e1da;
  border-radius: 8px;
  background: #fffefc;
  box-shadow: 0 5px 16px rgba(71, 57, 47, 0.055);
  padding: 0.82rem 0.9rem;
}

.business-dashboard__metric-icon {
  width: 2.25rem;
  height: 2.25rem;
  display: grid;
  place-items: center;
  border-radius: 6px;
  background: #eaf2ff;
  color: #3177d4;
}

.business-dashboard__metric-icon.is-green {
  background: #e3f4e8;
  color: #149255;
}

.business-dashboard__metric-icon.is-purple {
  background: #f2e5f7;
  color: #9a46c4;
}

.business-dashboard__metric-icon.is-orange {
  background: #f9ead7;
  color: #dc7a20;
}

.business-dashboard__metric-icon.is-violet {
  background: #f1e5fb;
  color: #944dd0;
}

.business-dashboard__metric-icon.is-indigo {
  background: #ebe9ff;
  color: #6159d9;
}

.business-dashboard__metrics article > div {
  min-width: 0;
  display: grid;
  gap: 0.05rem;
}

.business-dashboard__metrics small {
  color: #78716b;
  font-size: 0.68rem;
  font-weight: 650;
}

.business-dashboard__metrics strong {
  overflow: hidden;
  color: #25211f;
  font-size: 1.08rem;
  font-weight: 800;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.business-dashboard__metrics strong.business-dashboard__metric-value--accent {
  color: #d45d3b;
}

.business-dashboard__metrics article > div > span {
  overflow: hidden;
  color: #817a74;
  font-size: 0.62rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.business-dashboard__panel {
  min-width: 0;
  overflow: hidden;
  border: 1px solid #e8e1da;
  border-radius: 8px;
  background: #fffefc;
  box-shadow: 0 5px 16px rgba(71, 57, 47, 0.055);
}

.business-dashboard__panel-head {
  display: flex;
  min-height: 3.25rem;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.65rem 0.85rem;
}

.business-dashboard__panel-head h2 {
  color: #2c2825;
  font-size: 0.8rem;
  font-weight: 800;
}

.business-dashboard__panel-head > a,
.business-dashboard__panel-head > span {
  color: #736b65;
  font-size: 0.66rem;
  font-weight: 650;
  white-space: nowrap;
}

.business-dashboard__analytics {
  display: grid;
  grid-template-columns: minmax(0, 2.05fr) minmax(17rem, 1fr);
  gap: 0.75rem;
}

.business-dashboard__range {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.business-dashboard__range button {
  min-height: 1.7rem;
  border-radius: 4px;
  background: #f7e7e1;
  padding: 0 0.48rem;
  color: #c75b3d;
  font-size: 0.64rem;
  font-weight: 750;
}

.business-dashboard__range a {
  color: #736b65;
  font-size: 0.64rem;
}

.business-dashboard__chart {
  height: 16.2rem;
  padding: 0.1rem 0.7rem 0.7rem;
}

.business-dashboard__empty-chart {
  height: 100%;
  display: grid;
  place-items: center;
  align-content: center;
  gap: 0.45rem;
  color: #9a9189;
  font-size: 0.72rem;
}

.business-dashboard__availability-summary {
  display: grid;
  gap: 0.35rem;
  padding: 0.15rem 0.85rem 0.75rem;
}

.business-dashboard__availability-summary > div {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.business-dashboard__availability-summary span {
  display: inline-flex;
  align-items: center;
  gap: 0.38rem;
  color: #b34e3e;
  font-size: 0.72rem;
  font-weight: 750;
}

.business-dashboard__availability-summary span.is-healthy {
  color: #23825e;
}

.business-dashboard__availability-summary span i {
  width: 0.42rem;
  height: 0.42rem;
  border-radius: 50%;
  background: currentColor;
}

.business-dashboard__availability-summary strong {
  border-radius: 4px;
  background: #f4e8e4;
  padding: 0.18rem 0.42rem;
  color: #b9503d;
  font-size: 0.7rem;
}

.business-dashboard__availability-summary small {
  color: #8b837c;
  font-size: 0.62rem;
}

.business-dashboard__availability-track {
  display: flex;
  gap: 0.18rem;
  padding: 0 0.85rem 0.85rem;
}

.business-dashboard__availability-track i {
  height: 0.22rem;
  flex: 1 1 0;
  border-radius: 2px;
  background: #d9d3cd;
}

.business-dashboard__availability-track i.is-healthy {
  background: #5a967b;
}

.business-dashboard__availability-track i.is-failed {
  background: #d05846;
}

.business-dashboard__monitor-list {
  display: grid;
  min-height: 9.2rem;
  align-content: start;
  border-top: 1px solid #eee8e2;
  padding: 0.65rem 0.85rem;
}

.business-dashboard__monitor-list > div {
  display: flex;
  min-height: 2.35rem;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid #f0ebe6;
  color: #625b55;
  font-size: 0.68rem;
}

.business-dashboard__monitor-list > div:last-child {
  border-bottom: 0;
}

.business-dashboard__monitor-list strong {
  color: #23825e;
  font-size: 0.68rem;
}

.business-dashboard__monitor-list p {
  margin: auto;
  color: #9a9189;
  font-size: 0.7rem;
}

.business-dashboard__details {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.08fr);
  gap: 0.75rem;
}

.business-dashboard__table-wrap {
  position: relative;
  min-height: 13.25rem;
  overflow-x: auto;
  padding: 0 0.85rem 0.75rem;
}

.business-dashboard__table-wrap table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.66rem;
}

.business-dashboard__table-wrap th,
.business-dashboard__table-wrap td {
  height: 2rem;
  border-bottom: 1px solid #eee8e2;
  color: #625b55;
  text-align: right;
  white-space: nowrap;
}

.business-dashboard__table-wrap th {
  color: #8a817a;
  font-weight: 650;
}

.business-dashboard__table-wrap th:first-child,
.business-dashboard__table-wrap td:first-child {
  max-width: 11rem;
  overflow: hidden;
  text-align: left;
  text-overflow: ellipsis;
}

.business-dashboard__table-wrap td strong {
  color: #302b28;
  font-weight: 700;
}

.business-dashboard__table-wrap td.business-dashboard__cost {
  color: #169257;
  font-weight: 750;
}

.business-dashboard__table-empty {
  position: absolute;
  inset: 2rem 0.85rem 0.75rem;
  display: grid;
  place-items: center;
  color: #9a9189;
  font-size: 0.7rem;
}

.business-dashboard__model-body {
  min-height: 13.25rem;
  display: grid;
  grid-template-columns: 7.5rem minmax(0, 1fr);
  gap: 0.9rem;
  align-items: center;
  padding: 0 0.85rem 0.75rem;
}

.business-dashboard__doughnut {
  position: relative;
  width: 7.5rem;
  height: 7.5rem;
}

.business-dashboard__doughnut > div {
  position: absolute;
  inset: 0;
  display: grid;
  place-content: center;
  text-align: center;
  pointer-events: none;
}

.business-dashboard__doughnut strong {
  color: #2f2a27;
  font-size: 0.9rem;
  font-weight: 800;
}

.business-dashboard__doughnut span {
  color: #8a817a;
  font-size: 0.58rem;
}

.business-dashboard__model-list {
  min-width: 0;
}

.business-dashboard__model-list > div {
  display: grid;
  min-height: 2rem;
  grid-template-columns: minmax(0, 1fr) 4.5rem 4rem;
  align-items: center;
  gap: 0.5rem;
  border-bottom: 1px solid #eee8e2;
  font-size: 0.65rem;
}

.business-dashboard__model-list span {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 0.38rem;
  overflow: hidden;
  color: #4c4540;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.business-dashboard__model-list span i {
  width: 0.42rem;
  height: 0.42rem;
  flex: 0 0 auto;
  border-radius: 50%;
}

.business-dashboard__model-list strong {
  color: #5d554f;
  font-weight: 650;
  text-align: right;
}

.business-dashboard__model-list em {
  color: #169257;
  font-style: normal;
  font-weight: 750;
  text-align: right;
}

.business-dashboard__empty-chart--models {
  min-height: 13.25rem;
}

@media (max-width: 1120px) {
  .business-dashboard__metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 900px) {
  .business-dashboard__analytics,
  .business-dashboard__details {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 560px) {
  .business-dashboard__metrics {
    gap: 0.55rem;
  }

  .business-dashboard__metrics article {
    min-height: 5.15rem;
    grid-template-columns: 2rem minmax(0, 1fr);
    gap: 0.55rem;
    padding: 0.7rem;
  }

  .business-dashboard__metric-icon {
    width: 2rem;
    height: 2rem;
  }

  .business-dashboard__metrics strong {
    font-size: 0.96rem;
  }

  .business-dashboard__chart {
    height: 13rem;
  }

  .business-dashboard__model-body {
    grid-template-columns: 1fr;
  }

  .business-dashboard__doughnut {
    margin: 0 auto;
  }

  .business-dashboard__model-list > div {
    grid-template-columns: minmax(0, 1fr) 3.6rem 3.8rem;
  }
}

.user-start-hero {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(17rem, 0.52fr);
  gap: clamp(0.75rem, 1.8vw, 1.2rem);
  align-items: center;
  overflow: hidden;
  border: 1px solid rgba(33, 31, 28, 0.11);
  border-radius: 18px;
  background:
    radial-gradient(circle at 90% 14%, rgba(72, 185, 200, 0.1), transparent 28%),
    linear-gradient(135deg, rgba(255, 255, 255, 0.95), rgba(255, 252, 243, 0.92));
  box-shadow: 0 12px 30px rgba(29, 42, 42, 0.07);
  padding: clamp(0.78rem, 1.8vw, 1rem);
  animation: userStartRise 520ms cubic-bezier(0.2, 0.78, 0.24, 1) both;
}

.user-start-hero::before {
  display: none;
}

.user-start-hero::after {
  display: none;
}

.user-start-hero > * {
  position: relative;
  z-index: 1;
}

:global(.dark) .user-start-hero {
  background: linear-gradient(135deg, rgba(24, 24, 32, 0.9), rgba(34, 42, 48, 0.82));
}

.user-start-hero__badge {
  display: inline-flex;
  width: fit-content;
  border: 1px solid rgba(33, 31, 28, 0.1);
  border-radius: 999px;
  background: #fff8df;
  box-shadow: none;
  padding: 0.24rem 0.58rem;
  color: #211f1c;
  font-size: 0.68rem;
  font-weight: 950;
  animation: userStartPop 460ms ease 160ms both;
}

.user-start-hero h1 {
  margin-top: 0.52rem;
  max-width: 22ch;
  font-family: theme('fontFamily.display');
  font-size: clamp(1.58rem, 2.75vw, 2.18rem);
  font-weight: 950;
  letter-spacing: 0;
  line-height: 1.02;
  color: #242321;
  text-shadow: none;
  animation: userStartSlide 560ms cubic-bezier(0.2, 0.78, 0.24, 1) 90ms both;
}

.user-start-hero p {
  margin-top: 0.48rem;
  max-width: 31rem;
  color: rgba(33, 31, 28, 0.58);
  font-size: 0.86rem;
  font-weight: 750;
  line-height: 1.48;
  animation: userStartSlide 560ms cubic-bezier(0.2, 0.78, 0.24, 1) 180ms both;
}

:global(.dark) .user-start-hero p {
  color: rgba(255, 250, 240, 0.66);
}

.user-start-hero__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.55rem;
  margin-top: 0.72rem;
  animation: userStartSlide 560ms cubic-bezier(0.2, 0.78, 0.24, 1) 260ms both;
}

.user-start-primary,
.user-start-secondary,
.user-start-lite__link {
  display: inline-flex;
  min-height: 2.35rem;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
  border: 1px solid rgba(33, 31, 28, 0.12);
  border-radius: 12px;
  box-shadow: 0 8px 18px rgba(29, 42, 42, 0.07);
  padding: 0 0.78rem;
  color: #211f1c;
  font-weight: 950;
  transition: transform 150ms ease, box-shadow 150ms ease, filter 150ms ease;
}

.user-start-primary:hover,
.user-start-secondary:hover,
.user-start-lite__link:hover {
  transform: translateY(-2px);
  box-shadow: 0 14px 28px rgba(29, 42, 42, 0.1);
  filter: none;
}

.user-start-primary:active,
.user-start-secondary:active,
.user-start-lite__link:active {
  transform: translateY(1px);
  box-shadow: 0 6px 12px rgba(33, 31, 28, 0.07);
}

.user-start-primary {
  background: linear-gradient(135deg, #f8e08a, #f4b4bd);
}

.user-start-secondary,
.user-start-lite__link {
  background: #fff;
}

.user-start-card {
  position: relative;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 0.55rem 0.65rem;
  align-items: center;
  overflow: hidden;
  border: 1px solid rgba(33, 31, 28, 0.1);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.8);
  box-shadow: 0 10px 24px rgba(29, 42, 42, 0.06);
  padding: 0.72rem;
  transform-origin: center;
  animation: userStartFloatIn 680ms cubic-bezier(0.2, 0.78, 0.24, 1) 230ms both;
  transition: transform 180ms ease, box-shadow 180ms ease;
}

.user-start-card::before {
  display: none;
}

.user-start-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 14px 28px rgba(29, 42, 42, 0.09);
}

.user-start-card > * {
  position: relative;
  z-index: 1;
}

.user-start-card__mascot {
  display: grid;
  grid-row: span 2;
  justify-items: center;
  gap: 0.12rem;
  color: #4eaad0;
  animation: none;
}

.user-start-card__face {
  position: relative;
  display: flex;
  width: 3.25rem;
  height: 3rem;
  align-items: center;
  justify-content: center;
  gap: 0.46rem;
  border: 1px solid rgba(33, 31, 28, 0.12);
  border-radius: 45% 45% 42% 42%;
  background: #fff6dc;
}

.user-start-card__face::before {
  content: '';
  position: absolute;
  top: -0.48rem;
  width: 2.55rem;
  height: 0.92rem;
  border: 1px solid rgba(33, 31, 28, 0.12);
  border-radius: 999px 999px 10px 10px;
  background: #f4b4bd;
}

.user-start-card__face span {
  width: 0.28rem;
  height: 0.44rem;
  border-radius: 999px;
  background: #211f1c;
  animation: userStartBlink 4.2s infinite;
}

.user-start-card__face i {
  position: absolute;
  bottom: 0.58rem;
  width: 0.78rem;
  height: 0.36rem;
  border-bottom: 2px solid #211f1c;
  border-radius: 999px;
}

.user-start-card__line {
  display: grid;
  gap: 0.2rem;
  min-width: 0;
  border: 1px solid rgba(33, 31, 28, 0.08);
  border-radius: 13px;
  background: #fffdf7;
  padding: 0.52rem 0.62rem;
  transition: transform 160ms ease, border-color 160ms ease, background-color 160ms ease;
}

.user-start-card__line:hover {
  transform: none;
  border-color: rgba(74, 198, 160, 0.35);
  background: #ffffff;
}

.user-start-card__line small {
  color: rgba(33, 31, 28, 0.48);
  font-size: 0.68rem;
  font-weight: 950;
}

.user-start-card__line code {
  min-width: 0;
  overflow: hidden;
  color: #211f1c;
  font-family: theme('fontFamily.mono');
  font-size: 0.78rem;
  font-weight: 900;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-start-steps {
  position: relative;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
}

.user-start-steps::before {
  display: none;
}

.user-start-step,
.user-start-lite {
  border: 1px solid rgba(33, 31, 28, 0.11);
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.84);
  box-shadow: 0 10px 24px rgba(29, 42, 42, 0.06);
  animation: userStartRise 520ms cubic-bezier(0.2, 0.78, 0.24, 1) 420ms both;
}

.user-start-step {
  position: relative;
  padding: 0.85rem;
  transition: transform 170ms ease, box-shadow 170ms ease, background-color 170ms ease;
  animation: userStartRise 520ms cubic-bezier(0.2, 0.78, 0.24, 1) both;
}

.user-start-step:nth-child(1) {
  animation-delay: 260ms;
}

.user-start-step:nth-child(2) {
  animation-delay: 340ms;
}

.user-start-step:nth-child(3) {
  animation-delay: 420ms;
}

.user-start-step:hover {
  transform: translateY(-2px);
  background: rgba(255, 255, 255, 0.9);
  box-shadow: 0 14px 28px rgba(29, 42, 42, 0.09);
}

.user-start-step span {
  display: inline-flex;
  border-radius: 999px;
  background: #fff8df;
  padding: 0.2rem 0.48rem;
  color: #211f1c;
  font-size: 0.72rem;
  font-weight: 950;
  transition: transform 160ms ease, background-color 160ms ease;
}

.user-start-step:hover span {
  transform: none;
  background: #eaf8f7;
}

.user-start-step h2 {
  margin-top: 0.58rem;
  font-family: theme('fontFamily.display');
  font-size: 1.28rem;
  font-weight: 950;
  letter-spacing: 0;
}

.user-start-step p {
  margin-top: 0.32rem;
  color: rgba(33, 31, 28, 0.62);
  font-size: 0.9rem;
  font-weight: 750;
  line-height: 1.45;
}

.user-start-step a {
  display: inline-flex;
  margin-top: 0.48rem;
  color: #2f8aa2;
  font-weight: 950;
  transition: transform 140ms ease;
}

.user-start-step a:hover {
  transform: translateX(2px);
}

.user-start-lite {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr)) auto;
  gap: 0.8rem;
  align-items: center;
  padding: 1rem;
}

.user-start-lite__stat {
  display: grid;
  gap: 0.15rem;
  min-width: 0;
  border-radius: 16px;
  padding: 0.65rem;
  transition: background-color 160ms ease, transform 160ms ease;
}

.user-start-lite__stat:hover {
  background: rgba(255, 247, 208, 0.45);
  transform: translateY(-1px);
}

.user-start-lite__stat small {
  color: rgba(33, 31, 28, 0.48);
  font-size: 0.78rem;
  font-weight: 950;
}

.user-start-lite__stat strong {
  font-size: 1.55rem;
  font-weight: 950;
}

.user-start-lite__link {
  min-height: 2.7rem;
  border-width: 2px;
}

@media (max-width: 900px) {
  .user-start-hero,
  .user-start-steps,
  .user-start-lite {
    grid-template-columns: 1fr;
  }

  .user-start-card {
    max-width: 28rem;
  }

  .user-start-steps::before {
    display: none;
  }
}

@media (max-width: 640px) {
  .user-start-hero__actions > *,
  .user-start-lite__link {
    width: 100%;
  }
}

@keyframes userStartRise {
  from {
    opacity: 0;
    transform: translateY(18px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes userStartSlide {
  from {
    opacity: 0;
    transform: translateX(-18px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

@keyframes userStartPop {
  from {
    opacity: 0;
    transform: scale(0.9) rotate(-3deg);
  }
  to {
    opacity: 1;
    transform: scale(1) rotate(0);
  }
}

@keyframes userStartFloatIn {
  from {
    opacity: 0;
    transform: translateY(20px) rotate(4deg);
  }
  to {
    opacity: 1;
    transform: translateY(0) rotate(0);
  }
}

@keyframes userStartFloat {
  0%, 100% {
    translate: 0 0;
  }
  50% {
    translate: 0 -4px;
  }
}

@keyframes userStartMascot {
  0%, 100% {
    transform: rotate(0deg);
  }
  50% {
    transform: rotate(1deg) translateY(-2px);
  }
}

@keyframes userStartBlink {
  0%, 92%, 100% {
    transform: scaleY(1);
  }
  95% {
    transform: scaleY(0.12);
  }
}

@keyframes userStartShine {
  0%, 58% {
    transform: translateX(-120%);
  }
  76%, 100% {
    transform: translateX(120%);
  }
}

@keyframes userStartPulse {
  0%, 100% {
    opacity: 0.65;
    transform: scale(0.96);
  }
  50% {
    opacity: 1;
    transform: scale(1.05);
  }
}

@keyframes userStartSpin {
  to {
    transform: rotate(360deg);
  }
}

@media (prefers-reduced-motion: reduce) {
  .user-start-page *,
  .user-start-page *::before,
  .user-start-page *::after {
    animation: none !important;
    transition: none !important;
  }
}
</style>
