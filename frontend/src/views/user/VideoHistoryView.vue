<template>
  <UserWorkspaceLayout>
    <div class="video-history">
      <VideoWorkspaceTabs />

      <section class="video-history__toolbar">
        <div class="video-history__heading">
          <span><Icon name="clock" size="sm" /></span>
          <div><strong>历史任务</strong><small>查看生成状态并保存已完成的视频</small></div>
        </div>
        <label class="video-history__key">
          <span>查看 Key</span>
          <select v-model="selectedKeyId" :disabled="loadingKeys || videoKeys.length === 0">
            <option value="">选择 XiaoAPI 分组 Key</option>
            <option v-for="key in videoKeys" :key="key.id" :value="String(key.id)">
              {{ key.name }} · {{ key.group?.name || '视频分组' }}
            </option>
          </select>
        </label>
        <button type="button" class="video-history__refresh" title="刷新任务" :disabled="loadingJobs" @click="loadJobs">
          <Icon name="refresh" size="sm" :class="{ 'animate-spin': loadingJobs }" />
        </button>
      </section>

      <section class="video-retention-notice" aria-label="视频保存提醒">
        <span class="video-retention-notice__icon"><Icon name="exclamationTriangle" size="md" /></span>
        <div>
          <strong>成品视频不是永久存储</strong>
          <p>当前接口没有承诺固定保存天数，上游可能回收成片文件。任务显示“已完成”后请立即下载，并保存到自己的设备或对象存储。</p>
        </div>
        <span class="video-retention-notice__tag">及时下载</span>
      </section>

      <section class="video-history__summary" aria-label="任务汇总">
        <div><span>全部任务</span><strong>{{ jobs.length }}</strong></div>
        <div><span>处理中</span><strong>{{ activeCount }}</strong></div>
        <div><span>已完成</span><strong>{{ completedCount }}</strong></div>
        <div><span>已结算费用</span><strong>{{ totalAmount }}</strong></div>
      </section>

      <section v-if="previewUrl" class="video-history__preview" aria-label="视频预览">
        <div>
          <span>正在预览</span>
          <strong>{{ previewJob?.model }} · {{ previewJob?.resolution }} · {{ previewJob?.duration }} 秒</strong>
        </div>
        <video :src="previewUrl" controls autoplay playsinline></video>
        <button type="button" title="关闭预览" @click="closePreview"><Icon name="x" size="sm" /></button>
      </section>

      <section class="video-history__list">
        <header>
          <div><strong>任务记录</strong><small>{{ filteredJobs.length }} 条结果</small></div>
          <div class="video-history__filters" role="tablist" aria-label="任务状态筛选">
            <button
              v-for="item in filters"
              :key="item.value"
              type="button"
              role="tab"
              :aria-selected="statusFilter === item.value"
              :class="{ 'video-history__filter--active': statusFilter === item.value }"
              @click="statusFilter = item.value"
            >{{ item.label }}</button>
          </div>
        </header>

        <div v-if="loadingKeys || (loadingJobs && jobs.length === 0)" class="video-history__state">
          <Icon name="refresh" size="md" class="animate-spin" /><span>正在读取任务…</span>
        </div>
        <div v-else-if="videoKeys.length === 0" class="video-history__state">
          <Icon name="key" size="lg" /><strong>还没有可用的视频 Key</strong><router-link to="/keys?create=1">创建 Key</router-link>
        </div>
        <div v-else-if="filteredJobs.length === 0" class="video-history__state">
          <Icon name="clock" size="lg" /><strong>这里还没有任务</strong><router-link to="/videos">创建视频</router-link>
        </div>

        <div v-else class="video-history__rows">
          <article v-for="job in filteredJobs" :key="job.job_id" class="video-history-row">
            <div class="video-history-row__status">
              <span :class="`video-status video-status--${job.status}`">{{ statusLabel(job.status) }}</span>
              <time>{{ formatJobTime(job.created_at) }}</time>
            </div>
            <div class="video-history-row__model"><strong>{{ job.model }}</strong><small :title="job.job_id">{{ shortJobId(job.job_id) }}</small></div>
            <div class="video-history-row__spec"><strong>{{ job.resolution }}</strong><span>{{ job.duration }} 秒 · {{ job.aspect_ratio }}</span></div>
            <div class="video-history-row__amount"><strong>{{ formatAmount(job) }}</strong><span>{{ settlementLabel(job.status) }}</span></div>
            <div class="video-history-row__actions">
              <button v-if="job.status === 'completed'" type="button" title="播放" :disabled="contentLoadingId === job.job_id" @click="playJob(job)"><Icon name="play" size="sm" /></button>
              <button v-if="job.status === 'completed'" type="button" title="下载" :disabled="contentLoadingId === job.job_id" @click="downloadJob(job)"><Icon name="download" size="sm" /></button>
              <button v-if="isActiveStatus(job.status)" type="button" title="取消任务" :disabled="cancelingJobId === job.job_id" @click="cancelJob(job)"><Icon name="x" size="sm" /></button>
            </div>
          </article>
        </div>
      </section>
    </div>
  </UserWorkspaceLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { keysAPI, videoAPI } from '@/api'
import { VideoAPIError, type VideoJob, type VideoJobStatus } from '@/api/video'
import UserWorkspaceLayout from '@/components/layout/UserWorkspaceLayout.vue'
import VideoWorkspaceTabs from '@/components/video/VideoWorkspaceTabs.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import type { ApiKey } from '@/types'

type StatusFilter = 'all' | 'active' | 'completed' | 'failed'

const route = useRoute()
const appStore = useAppStore()
const apiKeys = ref<ApiKey[]>([])
const jobs = ref<VideoJob[]>([])
const selectedKeyId = ref('')
const loadingKeys = ref(false)
const loadingJobs = ref(false)
const statusFilter = ref<StatusFilter>('all')
const contentLoadingId = ref('')
const cancelingJobId = ref('')
const previewUrl = ref('')
const previewJob = ref<VideoJob | null>(null)
let pollTimer: number | undefined
let jobsRequest = 0

const filters = [
  { value: 'all', label: '全部' },
  { value: 'active', label: '处理中' },
  { value: 'completed', label: '已完成' },
  { value: 'failed', label: '失败 / 取消' },
] as const
const videoKeys = computed(() => apiKeys.value.filter((key) => key.status === 'active' && key.group?.platform === 'xiaoapi'))
const selectedKey = computed(() => videoKeys.value.find((key) => String(key.id) === selectedKeyId.value) || null)
const filteredJobs = computed(() => jobs.value.filter((job) => {
  if (statusFilter.value === 'active') return isActiveStatus(job.status)
  if (statusFilter.value === 'completed') return job.status === 'completed'
  if (statusFilter.value === 'failed') return job.status === 'failed' || job.status === 'canceled'
  return true
}))
const activeCount = computed(() => jobs.value.filter((job) => isActiveStatus(job.status)).length)
const completedCount = computed(() => jobs.value.filter((job) => job.status === 'completed').length)
const totalAmount = computed(() => {
  const amount = jobs.value
    .filter((job) => job.status === 'completed')
    .reduce((sum, job) => sum + (Number(job.amount) || 0), 0)
  return `$${amount.toFixed(2)}`
})

function selectedIdStorageKey() { return 'lingqu:video-studio:selected-key-id' }
function isActiveStatus(status: VideoJobStatus) { return ['pending', 'running', 'settling'].includes(status) }
function statusLabel(status: VideoJobStatus) {
  return ({ pending: '排队中', running: '生成中', settling: '结算中', completed: '已完成', failed: '失败', canceled: '已取消' } as const)[status] || status
}
function settlementLabel(status: VideoJobStatus) {
  if (status === 'completed') return '已结算'
  if (status === 'failed' || status === 'canceled') return '已释放'
  return '处理中'
}
function formatJobTime(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit',
  }).format(date)
}
function formatAmount(job: VideoJob) {
  const amount = Number(job.amount)
  return `${job.currency === 'USD' ? '$' : `${job.currency} `}${Number.isFinite(amount) ? amount.toFixed(2) : job.amount}`
}
function shortJobId(id: string) { return id.length > 18 ? `${id.slice(0, 10)}…${id.slice(-5)}` : id }
function errorMessage(error: unknown) {
  if (error instanceof VideoAPIError) return `${error.message}${error.requestId ? `（请求 ID：${error.requestId}）` : ''}`
  return error instanceof Error ? error.message : '请求失败，请稍后重试'
}

async function loadKeys() {
  loadingKeys.value = true
  try {
    const response = await keysAPI.list(1, 100, { status: 'active', sort_by: 'created_at', sort_order: 'desc' })
    apiKeys.value = response.items
    const ids = new Set(videoKeys.value.map((key) => String(key.id)))
    const queryId = typeof route.query.key_id === 'string' ? route.query.key_id : ''
    const storedId = window.localStorage.getItem(selectedIdStorageKey()) || ''
    selectedKeyId.value = [queryId, storedId, String(videoKeys.value[0]?.id || '')].find((id) => ids.has(id)) || ''
  } catch (error) {
    appStore.showError(errorMessage(error))
  } finally {
    loadingKeys.value = false
  }
}
async function loadJobs() {
  const key = selectedKey.value
  const requestId = ++jobsRequest
  if (!key) { jobs.value = []; return }
  loadingJobs.value = true
  try {
    const nextJobs = await videoAPI.listJobs(key.key, 100)
    if (requestId === jobsRequest) jobs.value = nextJobs
  } catch (error) {
    if (requestId === jobsRequest) appStore.showError(errorMessage(error))
  } finally {
    if (requestId === jobsRequest) loadingJobs.value = false
  }
}
function schedulePolling() {
  clearPolling()
  pollTimer = window.setInterval(() => {
    if (jobs.value.some((job) => isActiveStatus(job.status))) void loadJobs()
  }, 5000)
}
function clearPolling() { if (pollTimer) window.clearInterval(pollTimer); pollTimer = undefined }
async function cancelJob(job: VideoJob) {
  const key = selectedKey.value
  if (!key) return
  cancelingJobId.value = job.job_id
  try {
    await videoAPI.cancelJob(key.key, job.job_id)
    appStore.showSuccess('已提交取消请求')
    await loadJobs()
  } catch (error) { appStore.showError(errorMessage(error)) }
  finally { cancelingJobId.value = '' }
}
async function playJob(job: VideoJob) {
  const key = selectedKey.value
  if (!key) return
  contentLoadingId.value = job.job_id
  try {
    const blob = await videoAPI.fetchContent(key.key, job.job_id)
    closePreview()
    previewUrl.value = URL.createObjectURL(blob)
    previewJob.value = job
  } catch (error) { appStore.showError(errorMessage(error)) }
  finally { contentLoadingId.value = '' }
}
async function downloadJob(job: VideoJob) {
  const key = selectedKey.value
  if (!key) return
  contentLoadingId.value = job.job_id
  try {
    const blob = await videoAPI.fetchContent(key.key, job.job_id, true)
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `${job.model}-${job.job_id}.mp4`
    link.click()
    window.setTimeout(() => URL.revokeObjectURL(url), 1000)
  } catch (error) { appStore.showError(errorMessage(error)) }
  finally { contentLoadingId.value = '' }
}
function closePreview() {
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
  previewUrl.value = ''
  previewJob.value = null
}

watch(selectedKeyId, () => {
  try { if (selectedKeyId.value) window.localStorage.setItem(selectedIdStorageKey(), selectedKeyId.value) }
  catch { /* History still works without persisted selection. */ }
  closePreview()
  void loadJobs().then(schedulePolling)
})
onMounted(loadKeys)
onBeforeUnmount(() => { clearPolling(); closePreview() })
</script>

<style scoped>
.video-history {
  --history-ink: #122433;
  --history-muted: #7c8f99;
  --history-line: #d5e0e5;
  --history-cyan: #24a8af;
  display: grid;
  min-width: 0;
  gap: .85rem;
  color: var(--history-ink);
  font-family: 'Microsoft YaHei UI', 'Microsoft YaHei', sans-serif;
}
.video-history__toolbar {
  display: grid;
  grid-template-columns: minmax(13rem, auto) minmax(20rem, 34rem) 2.25rem;
  align-items: end;
  gap: 1rem;
  border: 1px solid var(--history-line);
  border-radius: 10px;
  background: #fff;
  padding: .78rem 1rem;
}
.video-history__heading { display: flex; align-items: center; gap: .65rem; }
.video-history__heading > span { display: grid; width: 2.1rem; height: 2.1rem; place-items: center; border-radius: 7px; background: var(--history-ink); color: #5acbd0; }
.video-history__heading > div { display: grid; gap: .14rem; }
.video-history__heading strong { font-size: .75rem; font-weight: 850; }
.video-history__heading small { color: var(--history-muted); font-size: .55rem; }
.video-history__key { display: grid; gap: .28rem; }
.video-history__key > span { color: #566c77; font-size: .58rem; font-weight: 850; }
.video-history__key select { width: 100%; min-height: 2.25rem; border: 1px solid #c4d3d9; border-radius: 7px; background: #fff; padding: 0 .7rem; color: var(--history-ink); font-size: .67rem; font-weight: 750; outline: 0; }
.video-history__key select:focus { border-color: var(--history-cyan); box-shadow: 0 0 0 3px rgba(36, 168, 175, .13); }
.video-history__refresh { display: grid; width: 2.25rem; height: 2.25rem; place-items: center; border: 1px solid #c4d3d9; border-radius: 7px; background: #fff; color: #566c77; }
.video-history__refresh:hover { border-color: var(--history-cyan); color: var(--history-cyan); }
.video-retention-notice { display: grid; grid-template-columns: 2.2rem minmax(0, 1fr) auto; align-items: center; gap: .75rem; border: 1px solid #ead39b; border-radius: 10px; background: #fff9e9; padding: .75rem 1rem; }
.video-retention-notice__icon { display: grid; width: 2.2rem; height: 2.2rem; place-items: center; border-radius: 7px; background: #f7e4ad; color: #936613; }
.video-retention-notice > div { display: grid; gap: .16rem; }
.video-retention-notice strong { color: #694d16; font-size: .68rem; }
.video-retention-notice p { margin: 0; color: #81692f; font-size: .59rem; line-height: 1.55; }
.video-retention-notice__tag { border-radius: 5px; background: #e4a83e; padding: .25rem .4rem; color: #fff; font-size: .52rem; font-weight: 850; }
.video-history__summary { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); overflow: hidden; border: 1px solid var(--history-line); border-radius: 10px; background: #fff; }
.video-history__summary > div { display: grid; min-height: 4.2rem; align-content: center; gap: .2rem; border-right: 1px solid var(--history-line); padding: .65rem .85rem; }
.video-history__summary > div:last-child { border-right: 0; }
.video-history__summary span { color: var(--history-muted); font-size: .55rem; }
.video-history__summary strong { font-family: Consolas, monospace; font-size: 1rem; }
.video-history__preview { position: relative; display: grid; grid-template-columns: minmax(10rem, .36fr) minmax(20rem, .64fr); align-items: center; gap: 1rem; overflow: hidden; border-radius: 10px; background: #101e27; padding: .8rem; color: #fff; }
.video-history__preview > div { display: grid; gap: .25rem; padding-left: .4rem; }
.video-history__preview span { color: #7fa0ad; font-size: .55rem; }
.video-history__preview strong { font-size: .7rem; }
.video-history__preview video { width: 100%; max-height: 23rem; border-radius: 7px; background: #081219; }
.video-history__preview > button { position: absolute; top: .6rem; right: .6rem; display: grid; width: 1.8rem; height: 1.8rem; place-items: center; border: 1px solid rgba(255,255,255,.25); border-radius: 50%; background: rgba(0,0,0,.5); color: #fff; }
.video-history__list { overflow: hidden; border: 1px solid var(--history-line); border-radius: 10px; background: #fff; }
.video-history__list > header { display: flex; min-height: 3.8rem; align-items: center; justify-content: space-between; gap: 1rem; border-bottom: 1px solid var(--history-line); padding: .7rem 1rem; }
.video-history__list > header > div:first-child { display: grid; gap: .14rem; }
.video-history__list header strong { font-size: .72rem; }
.video-history__list header small { color: var(--history-muted); font-size: .53rem; }
.video-history__filters { display: flex; gap: .2rem; border-radius: 7px; background: #eef3f5; padding: .2rem; }
.video-history__filters button { min-height: 1.8rem; border: 0; border-radius: 5px; background: transparent; padding: 0 .55rem; color: #687c86; font-size: .55rem; font-weight: 800; }
.video-history__filters .video-history__filter--active { background: #fff; box-shadow: 0 1px 4px rgba(18,36,51,.12); color: var(--history-ink); }
.video-history__state { display: grid; min-height: 13rem; place-items: center; align-content: center; gap: .45rem; color: var(--history-muted); font-size: .61rem; }
.video-history__state strong { color: var(--history-ink); font-size: .7rem; }
.video-history__state a { color: #2470a8; font-size: .58rem; font-weight: 800; text-decoration: none; }
.video-history__rows { display: grid; }
.video-history-row { display: grid; grid-template-columns: 8.5rem minmax(12rem, 1.2fr) minmax(8rem, .8fr) 6rem auto; min-height: 4.4rem; align-items: center; gap: .8rem; border-bottom: 1px solid #e1e8eb; padding: .55rem 1rem; }
.video-history-row:last-child { border-bottom: 0; }
.video-history-row:hover { background: #f8fbfc; }
.video-history-row__status, .video-history-row__model, .video-history-row__spec, .video-history-row__amount { display: grid; min-width: 0; gap: .2rem; }
.video-history-row__status { justify-items: start; }
.video-history-row time, .video-history-row small, .video-history-row span { color: var(--history-muted); font-size: .52rem; }
.video-history-row strong { overflow: hidden; font-size: .63rem; text-overflow: ellipsis; white-space: nowrap; }
.video-history-row__model small { overflow: hidden; font-family: Consolas, monospace; text-overflow: ellipsis; white-space: nowrap; }
.video-history-row__spec strong { font-family: Consolas, monospace; }
.video-history-row__amount strong { color: #df655d; font-family: Consolas, monospace; }
.video-history-row__actions { display: flex; justify-content: flex-end; gap: .22rem; }
.video-history-row__actions button { display: grid; width: 1.8rem; height: 1.8rem; place-items: center; border: 1px solid #c4d3d9; border-radius: 6px; background: #fff; color: #526a76; }
.video-history-row__actions button:hover { border-color: var(--history-cyan); color: var(--history-cyan); }
.video-history-row__actions button:disabled { opacity: .5; }
.video-status { width: max-content; border-radius: 5px; padding: .16rem .3rem; font-size: .51rem; font-weight: 800; }
.video-status--pending { background: #fff5dc; color: #966e19; }
.video-status--running, .video-status--settling { background: #e3f5f5; color: #25757b; }
.video-status--completed { background: #e4f6ee; color: #27765d; }
.video-status--failed { background: #fff0ef; color: #ad423a; }
.video-status--canceled { background: #edf1f2; color: #6d7d84; }

@media (max-width: 900px) {
  .video-history__toolbar { grid-template-columns: 1fr minmax(16rem, 1fr) 2.25rem; }
  .video-history-row { grid-template-columns: 7.5rem minmax(10rem, 1fr) 7rem auto; }
  .video-history-row__spec { grid-column: 2; grid-row: 2; }
  .video-history-row__amount { grid-column: 3; grid-row: 1 / 3; }
  .video-history-row__actions { grid-column: 4; grid-row: 1 / 3; }
}
@media (max-width: 680px) {
  .video-history__toolbar { grid-template-columns: 1fr auto; gap: .6rem; }
  .video-history__heading { grid-column: 1 / -1; }
  .video-retention-notice { grid-template-columns: 2.2rem minmax(0, 1fr); }
  .video-retention-notice__tag { display: none; }
  .video-history__summary { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .video-history__summary > div:nth-child(2) { border-right: 0; }
  .video-history__summary > div:nth-child(-n+2) { border-bottom: 1px solid var(--history-line); }
  .video-history__preview { grid-template-columns: 1fr; }
  .video-history__list > header { align-items: stretch; flex-direction: column; }
  .video-history__filters { overflow-x: auto; }
  .video-history__filters button { flex: 1 0 auto; }
  .video-history-row { grid-template-columns: minmax(0, 1fr) auto; gap: .42rem .6rem; padding: .7rem .8rem; }
  .video-history-row__status { grid-column: 1 / -1; display: flex; align-items: center; justify-content: space-between; }
  .video-history-row__model { grid-column: 1; }
  .video-history-row__spec { grid-column: 1; grid-row: 3; }
  .video-history-row__amount { grid-column: 2; grid-row: 2 / 4; text-align: right; }
  .video-history-row__actions { grid-column: 1 / -1; grid-row: 4; justify-content: flex-start; }
}
</style>
