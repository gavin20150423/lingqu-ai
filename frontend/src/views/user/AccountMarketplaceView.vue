<template>
  <UserWorkspaceLayout>
    <main class="community-page community-page--wide marketplace">
      <header class="community-head">
        <div><p class="community-eyebrow">ACCOUNT MODE</p><h1>账号广场</h1><p>账号模式 Key 只调度当前激活账号；每个 Key 最多预约 5 个账号。</p></div>
        <div class="community-actions"><button class="community-btn community-btn--quiet" @click="showGuide = true">使用说明</button><button class="community-btn community-btn--quiet" @click="openConsumption">我的消费</button><button class="community-icon-btn" title="刷新" :disabled="loading" @click="load"><Icon name="refresh" size="sm" /></button><router-link to="/accounts" class="community-btn"><Icon name="plus" size="sm" />新增账号</router-link><button class="community-btn community-btn--accent" @click="showAssistant = true"><Icon name="lightbulb" size="sm" />选号助手</button></div>
      </header>

      <div class="community-notice"><Icon name="infoCircle" size="sm" /><span>加入席位后不会获得号主凭据。下一次 API 请求会按预约顺序尝试激活可用账号。</span></div>

      <nav class="marketplace-provider-tabs" aria-label="账号模式平台"><button :class="{ active: provider === 'openai' }" @click="provider = 'openai'; load()"><strong>OpenAI</strong><span>OpenAI 账号模式</span></button><button :class="{ active: provider === 'anthropic' }" @click="provider = 'anthropic'; load()"><strong>Anthropic</strong><span>Anthropic 账号模式</span></button></nav>

      <section class="marketplace-stats"><div><span>当前结果</span><strong>{{ items.length }}</strong></div><div><span>可用席位</span><strong>{{ availableSeats }}</strong></div><div><span>已用席位</span><strong>{{ usedSeats }}</strong></div><div><span>账号模式 Key</span><strong>{{ assignedKeyCount }}</strong></div></section>

      <section class="marketplace-workbench">
        <div class="marketplace-main-filters">
          <label class="community-search"><Icon name="search" size="sm" /><input v-model="search" placeholder="搜索账号、号主或模型" @keyup.enter="load" /></label>
          <div class="community-segmented" aria-label="账号广场分类"><button v-for="entry in modes" :key="entry.key" :class="{ active: mode === entry.key }" @click="selectMode(entry.key)">{{ entry.label }}<small v-if="entry.hint">{{ entry.hint }}</small></button></div>
        </div>
        <div class="marketplace-filter-head"><div><strong>排序与筛选</strong><span>{{ hasFilters ? '已应用筛选条件' : '默认展示全部可见账号' }}</span></div><div><button class="community-btn community-btn--quiet" :disabled="!hasFilters" @click="resetFilters">重置</button><button class="community-btn" @click="load">应用</button></div></div>
        <div class="marketplace-filter-grid">
          <label>状态<select v-model="filters.status" class="community-select"><option value="">默认状态</option><option value="published">可加入</option><option value="sold_out">已满员</option><option value="paused">已暂停</option></select></label>
          <label>账号等级<select v-model="filters.tier" class="community-select"><option value="">全部等级</option><option v-for="tier in ['free','plus','pro','team','k12']" :key="tier" :value="tier">{{ tier.toUpperCase() }}</option></select></label>
          <label>账号席位<select v-model="filters.seats" class="community-select"><option value="">全部席位</option><option value="available">有空位</option><option value="full">已满员</option></select></label>
          <label>标签<input v-model="filters.tag" class="community-input" placeholder="全部标签" /></label>
          <label>可用模型<input v-model="filters.model" class="community-input" placeholder="全部模型" /></label>
        </div>
        <div class="marketplace-sort"><span>排序</span><button v-for="sort in sorts" :key="sort.key" :class="{ active: sortKey === sort.key }" @click="toggleSort(sort.key)">{{ sort.label }}<Icon v-if="sortKey === sort.key && sort.key" :name="sortDirection === 'desc' ? 'arrowDown' : 'arrowUp'" size="xs" /></button></div>
      </section>

      <section v-if="mode==='owner'&&ownerSummary" class="owner-summary"><div><span>上架账号</span><strong>{{ownerSummary.published_listings}}</strong></div><div><span>活跃使用者</span><strong>{{ownerSummary.active_members}}</strong></div><div><span>累计流水</span><strong>¥{{money(ownerSummary.gross_revenue)}}</strong></div><div><span>平台抽成</span><strong>¥{{money(ownerSummary.platform_fees)}}</strong></div><div><span>号主净收益</span><strong>¥{{money(ownerSummary.net_revenue)}}</strong></div></section>

      <div v-if="mode === 'history'" class="community-table-wrap"><table class="community-table"><thead><tr><th>预约顺序</th><th>共享账号</th><th>Key</th><th>状态</th><th>加入时间</th><th>最近使用</th><th>操作</th></tr></thead><tbody><tr v-if="memberships.length === 0"><td colspan="7"><div class="community-empty compact">暂无历史使用记录</div></td></tr><tr v-for="membership in memberships" v-else :key="membership.id"><td>#{{ membership.reservation_order }}</td><td>#{{ membership.listing_id }}</td><td>{{ membership.api_key_id || '-' }}</td><td>{{ membership.status }}</td><td>{{ formatTime(membership.joined_at) }}</td><td>{{ formatTime(membership.last_used_at) }}</td><td><button v-if="membership.activated_at" class="community-btn community-btn--quiet" @click="openReview(membership)">评价</button></td></tr></tbody></table></div>
      <div v-else-if="loading" class="community-empty">正在加载账号广场...</div>
      <div v-else-if="visibleItems.length === 0" class="community-empty"><strong>没有符合条件的共享账号</strong><span>调整筛选条件，或上传自己的 OAuth 账号。</span></div>

      <section v-else class="marketplace-list">
        <article v-for="item in visibleItems" :key="item.id" class="marketplace-card" :class="{'marketplace-card--owner':item.owner_user_id===Number(auth.user?.id||0)}">
          <header><div class="marketplace-title"><div class="community-meta"><span class="community-provider" :data-provider="item.provider">{{ providerName(item.provider) }}</span><span class="community-chip">{{ item.account_tier || 'OAuth' }}</span><span v-for="tag in (item.tags || []).slice(0, 3)" :key="tag" class="community-chip">{{ tag }}</span></div><h2>{{ item.title }}</h2><p>号主：{{ item.owner_name }} <button class="community-link-btn" @click="showOwnerAccounts(item)">其他账号</button></p></div><div class="marketplace-rating"><span>评分</span><strong>{{ item.rating_count ? `${item.score.toFixed(1)}/10` : '未评分' }}</strong><small v-if="item.rating_count">{{ item.rating_count }} 人</small></div></header>
          <div class="marketplace-health"><div><span>账号状态</span><strong>{{ item.health_status || '正常可用' }}</strong><small>{{ statusText(effectiveStatus(item)) }}</small></div><div><span>席位</span><strong>{{ item.seats_used }} / {{ item.seat_limit }}</strong><small>{{ remainingSeats(item) ? `剩余 ${remainingSeats(item)}` : '已满员，可预约' }}</small></div><div><span>实时容量</span><strong>{{ item.live_concurrency || 0 }} / {{ item.account_concurrency || item.seat_limit * item.per_user_concurrency }}</strong><small>并发占用</small></div><div class="marketplace-quota"><span>5 小时可用量</span><strong>{{ 100 - (item.usage_5h_percent || 0) }}%</strong><progress :value="item.usage_5h_percent || 0" max="100" /></div><div class="marketplace-quota"><span>7 天可用量</span><strong>{{ 100 - (item.usage_7d_percent || 0) }}%</strong><progress :value="item.usage_7d_percent || 0" max="100" /></div></div>
          <div class="marketplace-terms"><div><span>倍率</span><strong>{{ item.usage_multiplier }}x</strong></div><div><span>最低余额</span><strong>¥{{ money(item.minimum_balance || 0) }}</strong></div><div><span>账号并发</span><strong>{{ item.account_concurrency || item.seat_limit * item.per_user_concurrency }}</strong></div><div><span>单用户并发</span><strong>{{ item.per_user_concurrency }}</strong></div><div><span>小时费</span><strong>¥{{ money(item.hourly_price) }}</strong></div><div><span>免小时费低消</span><strong>{{ item.hourly_minimum_spend ? `¥${money(item.hourly_minimum_spend)}/小时` : '未开启' }}</strong></div></div>
          <div class="marketplace-models"><span v-for="model in item.supported_models.slice(0, 6)" :key="model">{{ model }}</span><small v-if="item.supported_models.length > 6">+{{ item.supported_models.length - 6 }}</small></div>
          <footer v-if="item.owner_user_id===Number(auth.user?.id||0)" class="owner-actions"><div><strong>号主管理</strong><small>当前状态：{{statusText(item.status)}} · 抽成 {{item.commission_rate}}%</small></div><label>自用 Key<select v-model.number="selectedKeys[item.id]" class="community-select"><option :value="0">选择{{ providerName(item.provider) }}账号模式 Key</option><option v-for="key in accountModeKeys" :key="key.id" :value="key.id">{{ key.name }}{{ key.account_mode_platform ? '' : '（首次使用将设为账号模式）' }}</option></select></label><button class="community-btn community-btn--accent" :disabled="!selectedKeys[item.id]||item.status!=='published'" @click="join(item)">加入自用</button><router-link to="/accounts" class="community-btn community-btn--quiet">编辑共享条款</router-link><button class="community-btn" :class="{'community-btn--danger':item.status==='published'}" @click="toggleListingStatus(item)">{{item.status==='published'?'暂停上架':'重新上架'}}</button></footer><footer v-else><label>账号模式 Key<select v-model.number="selectedKeys[item.id]" class="community-select"><option :value="0">选择{{ providerName(item.provider) }}账号模式 Key</option><option v-for="key in accountModeKeys" :key="key.id" :value="key.id">{{ key.name }}{{ key.account_mode_platform ? '' : '（首次使用将设为账号模式）' }}</option></select></label><label>空闲退出<div class="community-number-suffix"><input v-model.number="idleMinutes[item.id]" type="number" min="1" max="1440" /><span>分钟</span></div></label><button class="community-btn" :disabled="!selectedKeys[item.id]" @click="join(item)">{{ remainingSeats(item) < 1 ? '预约使用' : '加入使用' }}</button></footer>
        </article>
      </section>

      <div v-if="showGuide" class="community-modal-backdrop" @click.self="showGuide=false"><section class="community-modal community-modal--large"><header><div><p class="community-eyebrow">HOW IT WORKS</p><h2>账号模式使用说明</h2></div><button class="community-icon-btn" title="关闭" @click="showGuide=false"><Icon name="x" size="sm" /></button></header><div class="community-guide guide-details"><div class="community-notice"><Icon name="infoCircle" size="sm" /><strong>激活后按分钟预扣小时费，最长 1 小时窗口做最终核销。</strong></div><ol><li><strong>加入或预约</strong><span>绑定账号模式 Key；账号满员时进入预约队列，等待期间不收小时费。</span></li><li><strong>激活使用</strong><span>预约不会靠页面等待自动激活，下一次 API 请求才按顺序尝试账号并占用席位。</span></li><li><strong>窗口核销</strong><span>小时费按实际激活分钟预扣，低消按实际时长折算；单个窗口最长 1 小时，达标后退回该窗口小时费。</span></li></ol><div class="guide-fees"><div><strong>请求费用</strong><span>模型实际费用乘账号倍率。</span></div><div><strong>小时费</strong><span>激活期间逐分钟预扣，不会一次扣满一小时。</span></div><div><strong>免小时费低消</strong><span>窗口内请求消费达标后退回对应小时费。</span></div><div><strong>空闲退出</strong><span>连续空闲到期释放席位，并停止预扣。</span></div></div><div class="guide-example"><strong>退款示例</strong><p>小时费 ¥0.60/小时、低消 ¥0.30/小时，激活 5 分钟先预扣 ¥0.05；低消要求为 ¥0.025。请求消费达到 ¥0.03 时退回 ¥0.05，只有 ¥0.01 时不退。</p><p>跨分钟请求按真实执行区间计入核销。自用自己的上架账号不收小时费，也不产生号主收益。</p></div></div></section></div>

      <div v-if="showConsumption" class="community-modal-backdrop" @click.self="showConsumption=false"><section class="community-modal community-modal--large"><header><div><h2>我的消费</h2><p>选择使用过的账号，查看请求消费、小时预扣和低消退款。</p></div><button class="community-icon-btn" title="关闭" @click="showConsumption=false"><Icon name="x" size="sm" /></button></header><label>选择使用过的账号<select v-model.number="selectedConsumptionMembership" class="community-select" @change="loadConsumptionSummary"><option :value="0">请选择账号</option><option v-for="entry in consumptionAccounts" :key="entry.membership_id" :value="entry.membership_id">{{entry.title}} · {{entry.status}}</option></select><small>包含正在使用、预约中和历史使用记录。</small></label><div class="community-segmented" aria-label="消费统计范围"><button :class="{active:consumptionScope==='session'}" @click="setConsumptionScope('session')">本次使用</button><button :class="{active:consumptionScope==='today'}" @click="setConsumptionScope('today')">今天</button><button :class="{active:consumptionScope==='7d'}" @click="setConsumptionScope('7d')">近 7 天</button></div><div v-if="consumptionSummary" class="consumption-summary"><div><span>请求消费</span><strong>¥{{money(consumptionSummary.request_spend)}}</strong></div><div><span>小时预扣</span><strong>¥{{money(consumptionSummary.hourly_precharged)}}</strong></div><div><span>低消退款</span><strong>-¥{{money(consumptionSummary.hourly_refunded)}}</strong></div><div><span>合计</span><strong>¥{{money(consumptionSummary.total)}}</strong></div></div><div v-else class="community-empty compact">{{consumptionLoading?'正在加载消费统计...':'请选择使用过的账号'}}</div></section></div>

      <div v-if="showAssistant" class="community-modal-backdrop" @click.self="showAssistant=false"><section class="community-modal community-modal--large"><header><div><h2>账号模式选号助手</h2><p>{{providerName(provider)}} · 按预计每小时额度升序推荐</p></div><button class="community-icon-btn" title="关闭" @click="showAssistant=false"><Icon name="x" size="sm" /></button></header><div class="assistant-presets"><span>测算预设</span><button v-for="preset in assistantPresets" :key="preset.key" class="community-btn community-btn--quiet" @click="applyPreset(preset.key)">{{preset.label}}</button></div><form class="community-form" @submit.prevent="runAssistant"><div class="community-form-grid"><label>账号模式 Key<select v-model.number="assistant.api_key_id" class="community-select" required><option :value="0">选择{{providerName(provider)}}账号模式 Key</option><option v-for="key in accountModeKeys" :key="key.id" :value="key.id">{{key.name}}</option></select></label><label>模型<select v-model="assistant.model" class="community-select" required><option v-for="model in assistantModels" :key="model" :value="model">{{model}}</option></select></label><label>请求次数<input v-model.number="assistant.request_count" type="number" min="1" max="100000" class="community-input" /></label><label>使用时长（小时）<input v-model.number="assistant.hours" type="number" min="0.01" max="720" step="0.25" class="community-input" /></label><label>单次输入 Token<input v-model.number="assistant.input_tokens" type="number" min="0" class="community-input" /></label><label>单次输出 Token<input v-model.number="assistant.output_tokens" type="number" min="0" class="community-input" /></label><label>单次 Cache 写入<input v-model.number="assistant.cache_creation_tokens" type="number" min="0" class="community-input" /></label><label>单次 Cache 读取<input v-model.number="assistant.cache_read_tokens" type="number" min="0" class="community-input" /></label></div><footer><button type="button" class="community-btn community-btn--quiet" @click="showAssistant=false">取消</button><button class="community-btn" :disabled="assistantLoading||!assistant.api_key_id">{{assistantLoading?'测算中...':'测算并推荐'}}</button></footer></form><div v-if="assistantResults.length" class="assistant-results"><article v-for="(result,index) in assistantResults" :key="result.listing.id"><b>#{{index+1}}</b><div><strong>{{result.listing.title}}</strong><span>{{result.listing.usage_multiplier}}x · 剩余 {{result.remaining_seats}} 席</span></div><div><small>请求 ¥{{money(result.request_spend)}} · 小时费 ¥{{money(result.hourly_fee)}}<template v-if="result.hourly_fee_waived">（低消达标已免）</template></small><strong>¥{{money(result.estimated_per_hour)}} / 小时</strong></div></article></div></section></div>

      <div v-if="reviewMembership" class="community-modal-backdrop" @click.self="reviewMembership=null"><section class="community-modal"><header><div><h2>评价共享账号</h2><p>仅实际激活使用过的账号可以评价。</p></div><button class="community-icon-btn" title="关闭" @click="reviewMembership=null"><Icon name="x" size="sm" /></button></header><form class="community-form" @submit.prevent="submitReview"><label>评分（1-10）<input v-model.number="reviewForm.score" type="number" min="1" max="10" step="0.5" class="community-input" required /></label><label>使用感受<textarea v-model.trim="reviewForm.content" class="community-textarea" maxlength="500" placeholder="可选" /></label><footer><button type="button" class="community-btn community-btn--quiet" @click="reviewMembership=null">取消</button><button class="community-btn" :disabled="reviewSaving">{{reviewSaving?'提交中...':'提交评价'}}</button></footer></form></section></div>
    </main>
  </UserWorkspaceLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import UserWorkspaceLayout from '@/components/layout/UserWorkspaceLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { communityAPI, type CommunityAccountModeKey, type CommunityConsumptionAccount, type CommunityConsumptionSummary, type CommunityListing, type CommunityMembership, type CommunityOwnerSummary, type CommunitySelectionResult } from '@/api/community'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const app = useAppStore()
const auth = useAuthStore()
const items = ref<CommunityListing[]>([])
const memberships = ref<CommunityMembership[]>([])
const accountModeKeys = ref<CommunityAccountModeKey[]>([])
const loading = ref(true)
const provider = ref<'openai' | 'anthropic'>('openai')
const search = ref('')
const mode = ref('all')
const sortKey = ref('')
const sortDirection = ref<'asc' | 'desc'>('desc')
const showGuide = ref(false)
const showAssistant = ref(false)
const showConsumption = ref(false)
const reviewMembership = ref<CommunityMembership|null>(null)
const reviewSaving = ref(false)
const consumptionLoading = ref(false)
const assistantLoading = ref(false)
const consumptionAccounts = ref<CommunityConsumptionAccount[]>([])
const consumptionSummary = ref<CommunityConsumptionSummary|null>(null)
const selectedConsumptionMembership = ref(0)
const consumptionScope = ref<'session'|'today'|'7d'>('session')
const assistantResults = ref<CommunitySelectionResult[]>([])
const ownerSummary = ref<CommunityOwnerSummary|null>(null)
const selectedKeys = reactive<Record<number, number>>({})
const idleMinutes = reactive<Record<number, number>>({})
const filters = reactive({ status: '', tier: '', seats: '', tag: '', model: '' })
const assistant = reactive({ api_key_id: 0, model: 'gpt-5.6-sol', request_count: 500, hours: 2, input_tokens: 3000, output_tokens: 1000, cache_creation_tokens: 0, cache_read_tokens: 500 })
const reviewForm = reactive({ score: 10, content: '' })
const assistantPresets = [{key:'light',label:'轻量'},{key:'balanced',label:'均衡'},{key:'heavy',label:'重度'},{key:'recent',label:'近3天均值'}]
const modes = [{ key: 'owner', label: '我的账号', hint: '号主管理' }, { key: 'active', label: '使用 / 预约' }, { key: 'history', label: '历史使用' }, { key: 'all', label: '全部' }]
const sorts = [{ key: '', label: '默认' }, { key: 'account_concurrency', label: '账号并发' }, { key: 'per_user_concurrency', label: '单人并发' }, { key: 'minimum_balance', label: '最低余额' }, { key: 'hourly_price', label: '小时费' }, { key: 'hourly_minimum_spend', label: '免小时低消' }, { key: 'usage_multiplier', label: '倍率' }, { key: 'remaining_seats', label: '剩余席位' }, { key: 'score', label: '评分' }, { key: 'updated_at', label: '更新时间' }]

const availableSeats = computed(() => items.value.reduce((sum, item) => sum + remainingSeats(item), 0))
const usedSeats = computed(() => items.value.reduce((sum, item) => sum + item.seats_used, 0))
const assignedKeyCount = computed(() => accountModeKeys.value.filter(key => key.account_mode_platform === provider.value).length)
const assistantModels = computed(() => {
  const listed = items.value.flatMap(item => item.supported_models)
  const defaults = provider.value === 'openai'
    ? ['gpt-5.6-sol', 'gpt-5.6-terra', 'gpt-5.5', 'gpt-5.4', 'gpt-5.4-mini']
    : ['claude-opus-4-6', 'claude-sonnet-4-6', 'claude-haiku-4-5']
  return [...new Set([...listed, ...defaults])].sort()
})
const hasFilters = computed(() => Boolean(search.value || Object.values(filters).some(Boolean) || sortKey.value))
const visibleItems = computed(() => {
  let result = items.value.filter(item => {
    if (mode.value === 'owner' && item.owner_user_id !== Number(auth.user?.id || 0)) return false
    if (mode.value === 'active' && !memberships.value.some(m => m.listing_id === item.id && ['active', 'reserved'].includes(m.status))) return false
    if (filters.status && effectiveStatus(item) !== filters.status) return false
    if (filters.tier && item.account_tier?.toLowerCase() !== filters.tier) return false
    if (filters.seats === 'available' && remainingSeats(item) < 1) return false
    if (filters.seats === 'full' && remainingSeats(item) > 0) return false
    if (filters.tag && !(item.tags || []).some(tag => tag.toLowerCase().includes(filters.tag.toLowerCase()))) return false
    if (filters.model && !item.supported_models.some(model => model.toLowerCase().includes(filters.model.toLowerCase()))) return false
    return true
  })
  if (!sortKey.value) return result
  const factor = sortDirection.value === 'desc' ? -1 : 1
  return [...result].sort((a, b) => (sortValue(a, sortKey.value) - sortValue(b, sortKey.value)) * factor)
})

function sortValue(item: CommunityListing, key: string) { if (key === 'remaining_seats') return remainingSeats(item); if (key === 'updated_at') return new Date(item.updated_at).getTime(); return Number((item as unknown as Record<string, unknown>)[key] || 0) }
function remainingSeats(item: CommunityListing) { return Math.max(0, item.seat_limit - item.seats_used) }
function effectiveStatus(item: CommunityListing) { return item.status === 'published' && remainingSeats(item) === 0 ? 'sold_out' : item.status }
function providerName(value: string) { return value === 'openai' ? 'OpenAI' : 'Anthropic' }
function money(value: number) { return Number(value || 0).toFixed(2) }
function statusText(status: string) { return ({ published: '已上架', sold_out: '已满员', paused: '已暂停' } as Record<string, string>)[status] || status }
function formatTime(value?: string) { return value ? new Date(value).toLocaleString('zh-CN') : '-' }
function toggleSort(key: string) { if (sortKey.value === key && key) sortDirection.value = sortDirection.value === 'desc' ? 'asc' : 'desc'; else { sortKey.value = key; sortDirection.value = 'desc' } }
function resetFilters() { search.value = ''; Object.assign(filters, { status: '', tier: '', seats: '', tag: '', model: '' }); sortKey.value = '' }
function selectMode(next:string) { mode.value = next; void load() }
function showOwnerAccounts(item:CommunityListing) { mode.value = 'all'; search.value = item.owner_name; void load() }
function openReview(membership:CommunityMembership) { reviewMembership.value = membership; Object.assign(reviewForm,{score:10,content:''}) }
async function submitReview() { if (!reviewMembership.value) return; reviewSaving.value = true; try { await communityAPI.reviewMembership(reviewMembership.value.id,{...reviewForm}); app.showSuccess('评价已提交'); reviewMembership.value = null; await load() } catch { app.showError('评价失败，仅实际激活且非自用的账号可以评价') } finally { reviewSaving.value = false } }
async function applyPreset(key:string) { if(key==='recent'){if(!assistant.api_key_id){app.showError('请先选择账号模式 Key');return}try{const average=await communityAPI.recentUsageAverage(assistant.api_key_id);if(!average.request_count){app.showError('该 Key 近 3 天暂无使用记录');return}Object.assign(assistant,{request_count:Math.max(1,average.request_count),input_tokens:Math.round(average.input_tokens),output_tokens:Math.round(average.output_tokens),cache_creation_tokens:Math.round(average.cache_creation_tokens),cache_read_tokens:Math.round(average.cache_read_tokens)});app.showSuccess('已带入该 Key 近 3 天真实请求均值')}catch{app.showError('近 3 天均值读取失败')}return} const values = key === 'light' ? {request_count:100,hours:1,input_tokens:1000,output_tokens:400,cache_creation_tokens:0,cache_read_tokens:100} : key === 'heavy' ? {request_count:1500,hours:4,input_tokens:8000,output_tokens:3000,cache_creation_tokens:1000,cache_read_tokens:3000} : {request_count:500,hours:2,input_tokens:3000,output_tokens:1000,cache_creation_tokens:0,cache_read_tokens:500}; Object.assign(assistant, values) }
async function openConsumption() { showConsumption.value = true; consumptionSummary.value = null; consumptionLoading.value = true; try { consumptionAccounts.value = await communityAPI.consumptionAccounts(); selectedConsumptionMembership.value ||= consumptionAccounts.value[0]?.membership_id || 0; await loadConsumptionSummary() } catch { app.showError('消费账号加载失败') } finally { consumptionLoading.value = false } }
async function loadConsumptionSummary() { if (!selectedConsumptionMembership.value) { consumptionSummary.value = null; return } consumptionLoading.value = true; try { consumptionSummary.value = await communityAPI.consumptionSummary(selectedConsumptionMembership.value, consumptionScope.value) } catch { app.showError('消费统计加载失败') } finally { consumptionLoading.value = false } }
function setConsumptionScope(scope:'session'|'today'|'7d') { consumptionScope.value = scope; void loadConsumptionSummary() }
async function runAssistant() { assistantLoading.value = true; assistantResults.value = []; try { assistantResults.value = await communityAPI.recommendListings({provider:provider.value,...assistant}); if(!assistantResults.value.length) app.showError('没有支持该模型的可推荐账号') } catch { app.showError('测算失败，请确认账号模式 Key、模型与 Token 参数') } finally { assistantLoading.value = false } }

async function load() { loading.value = true; try { items.value = mode.value==='owner' ? await communityAPI.ownerListings(provider.value) : await communityAPI.marketplace({ search: search.value, provider: provider.value }); items.value.forEach(item => { idleMinutes[item.id] ||= item.idle_timeout_minutes || 10; selectedKeys[item.id] ||= 0 }); const requests = [communityAPI.memberships(), communityAPI.accountModeKeys(provider.value), ...(mode.value==='owner'?[communityAPI.ownerSummary()]:[])]; const settled = await Promise.allSettled(requests); if (settled[0].status === 'fulfilled') memberships.value = settled[0].value as CommunityMembership[]; if (settled[1].status === 'fulfilled') accountModeKeys.value = settled[1].value as CommunityAccountModeKey[]; if(settled[2]?.status==='fulfilled') ownerSummary.value=settled[2].value as CommunityOwnerSummary } catch { app.showError('账号广场加载失败') } finally { loading.value = false } }
async function join(item: CommunityListing) { const keyID = selectedKeys[item.id]; if (!keyID) { app.showError('请先选择账号模式 Key'); return } try { const key = accountModeKeys.value.find(entry => entry.id === keyID); if (key && !key.account_mode_platform) await communityAPI.setAccountModeKey(keyID, provider.value); await communityAPI.join(item.id, keyID, idleMinutes[item.id] || 10); app.showSuccess('已加入预约队列，将在下一次 API 请求时尝试激活'); await load() } catch { app.showError('加入失败，请确认 Key 类型和余额要求') } }
async function toggleListingStatus(item:CommunityListing){const next=item.status==='published'?'paused':'published';try{await communityAPI.setListingStatus(item.id,next);app.showSuccess(next==='published'?'账号已重新上架':'账号已暂停上架');await load()}catch{app.showError('状态更新失败，请确认账号仍处于审核通过状态')}}
onMounted(load)
</script>

<style src="@/styles/community.css"></style>
<style scoped>
.owner-summary{display:grid;grid-template-columns:repeat(5,minmax(0,1fr));border:1px solid var(--line);border-radius:var(--community-radius-lg);background:var(--community-surface);overflow:hidden}.owner-summary>div{display:grid;gap:4px;padding:12px;border-right:1px solid var(--line)}.owner-summary>div:last-child{border-right:0;background:var(--community-accent-soft)}.owner-summary span,.owner-actions small{color:var(--muted);font-size:.68rem}.marketplace-card--owner{grid-column:1/-1}.owner-actions{grid-template-columns:minmax(130px,.7fr) minmax(220px,1fr) auto auto auto!important}.owner-actions>div{display:grid;gap:3px;min-width:0}.owner-actions>div strong,.owner-actions .community-btn{white-space:nowrap}.consumption-summary{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));border:1px solid var(--line);border-radius:var(--community-radius-lg);background:var(--community-surface);overflow:hidden}.consumption-summary>div{display:grid;gap:5px;padding:14px;border-right:1px solid var(--line)}.consumption-summary>div:last-child{border-right:0;background:var(--community-accent-soft)}.consumption-summary span,.assistant-results span,.assistant-results small{color:var(--muted);font-size:.7rem}.consumption-summary strong{font-size:1rem}.assistant-presets{display:flex;align-items:center;flex-wrap:wrap;gap:7px}.assistant-presets>span{margin-right:auto;font-size:.74rem;font-weight:900}.assistant-results{display:grid;gap:7px;border-top:1px solid var(--line);padding-top:14px}.assistant-results article{display:grid;grid-template-columns:30px minmax(0,1fr) minmax(190px,.65fr);align-items:center;gap:10px;border:1px solid var(--line);border-radius:var(--community-radius);padding:10px}.assistant-results article>b{display:grid;width:28px;height:28px;place-items:center;border-radius:var(--community-radius);background:var(--yellow)}.assistant-results article>div{display:grid;gap:3px}.assistant-results article>div:last-child{text-align:right}@media(max-width:700px){.owner-summary{grid-template-columns:repeat(2,minmax(0,1fr))}.owner-actions{grid-template-columns:1fr!important}.owner-actions>div strong,.owner-actions .community-btn{white-space:normal}.consumption-summary{grid-template-columns:repeat(2,minmax(0,1fr))}.consumption-summary>div:nth-child(2){border-right:0}.assistant-results article{grid-template-columns:30px minmax(0,1fr)}.assistant-results article>div:last-child{grid-column:1/-1;text-align:left}}
</style>
<style scoped>
.guide-details{display:grid;gap:14px}.guide-details ol{display:grid;gap:9px;margin:0;padding:0;list-style:none}.guide-details li{display:grid;grid-template-columns:minmax(120px,.35fr) 1fr;gap:10px;border:1px solid var(--line);padding:10px}.guide-details li span,.guide-fees span,.guide-example p{color:var(--muted);font-size:.75rem}.guide-fees{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:8px}.guide-fees>div,.guide-example{display:grid;gap:4px;border:1px solid var(--line);padding:10px}@media(max-width:700px){.guide-details li,.guide-fees{grid-template-columns:1fr}}
</style>
