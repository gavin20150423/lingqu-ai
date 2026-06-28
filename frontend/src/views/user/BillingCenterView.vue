<template>
  <UserWorkspaceLayout>
    <div class="lingqu-console-page lingqu-billing">
      <section class="lingqu-billing__actions" aria-label="账单功能">
        <router-link
          v-for="item in actionCards"
          :key="item.path"
          :to="item.path"
          class="lingqu-billing__action-card"
        >
          <span :class="item.className">
            <Icon :name="item.icon" size="lg" />
          </span>
          <div>
            <small>{{ item.kicker }}</small>
            <strong>{{ item.title }}</strong>
            <p>{{ item.description }}</p>
          </div>
          <Icon name="chevronRight" size="md" />
        </router-link>
      </section>

      <section class="lingqu-billing__overview" aria-label="账单概览">
        <article class="lingqu-billing__balance">
          <small>账户余额</small>
          <strong>${{ balanceText }}</strong>
          <p :class="{ 'lingqu-billing__warning': balanceLow }">
            {{ balanceLow ? '余额偏低，建议及时补充。' : '余额充足，可以继续稳定调用。' }}
          </p>
        </article>

        <article v-for="item in summaryCards" :key="item.label" class="lingqu-billing__summary-card">
          <span :class="item.className">
            <Icon :name="item.icon" size="md" />
          </span>
          <small>{{ item.label }}</small>
          <strong>{{ item.value }}</strong>
        </article>
      </section>

      <section class="lingqu-billing__mini-grid">
        <article class="lingqu-billing__panel">
          <div class="lingqu-billing__panel-head">
            <div>
              <span class="lingqu-console-eyebrow">订阅状态</span>
              <h2>当前可用套餐</h2>
            </div>
            <router-link to="/subscriptions">查看全部</router-link>
          </div>

          <div v-if="loadingSubscriptions" class="lingqu-billing__loading">订阅加载中...</div>
          <div v-else-if="activeSubscriptions.length === 0" class="lingqu-billing__empty">
            <Icon name="badge" size="lg" />
            <p>暂无可用订阅，可以选择一个套餐开启更稳定的调用额度。</p>
          </div>
          <div v-else class="lingqu-billing__subscription-list">
            <article v-for="subscription in activeSubscriptions.slice(0, 3)" :key="subscription.id">
              <div>
                <strong>{{ subscription.group?.name || `套餐 #${subscription.group_id}` }}</strong>
                <small>{{ formatSubscriptionTime(subscription.expires_at) }}</small>
              </div>
              <span>{{ subscription.status === 'active' ? '可用' : '需处理' }}</span>
            </article>
          </div>
        </article>

        <article class="lingqu-billing__panel">
          <div class="lingqu-billing__panel-head">
            <div>
              <span class="lingqu-console-eyebrow">最近订单</span>
              <h2>支付进度</h2>
            </div>
            <router-link to="/orders">订单列表</router-link>
          </div>

          <div v-if="loadingOrders" class="lingqu-billing__loading">订单加载中...</div>
          <div v-else-if="recentOrders.length === 0" class="lingqu-billing__empty">
            <Icon name="document" size="lg" />
            <p>还没有订单记录。</p>
          </div>
          <div v-else class="lingqu-billing__order-list">
            <article v-for="order in recentOrders" :key="order.id">
              <div>
                <strong>#{{ order.id }}</strong>
                <small>{{ formatDate(order.created_at) }}</small>
              </div>
              <span>{{ order.status }}</span>
              <b>{{ formatOrderAmount(order) }}</b>
            </article>
          </div>
        </article>
      </section>
    </div>
  </UserWorkspaceLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import UserWorkspaceLayout from '@/components/layout/UserWorkspaceLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore, useSubscriptionStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { paymentAPI } from '@/api/payment'
import type { PaymentOrder } from '@/types/payment'

const appStore = useAppStore()
const authStore = useAuthStore()
const subscriptionStore = useSubscriptionStore()

const loadingOrders = ref(false)
const recentOrders = ref<PaymentOrder[]>([])

const user = computed(() => authStore.user)
const activeSubscriptions = computed(() => subscriptionStore.activeSubscriptions)
const loadingSubscriptions = computed(() => subscriptionStore.loading)
const balanceText = computed(() => Number(user.value?.balance || 0).toFixed(2))
const balanceLowThreshold = computed(() => appStore.cachedPublicSettings?.balance_low_notify_threshold ?? 0)
const balanceLow = computed(() => {
  const threshold = balanceLowThreshold.value
  return threshold > 0 && Number(user.value?.balance || 0) <= threshold
})

const completedOrderCount = computed(() => recentOrders.value.filter(order => order.status === 'COMPLETED').length)
const pendingOrderCount = computed(() => recentOrders.value.filter(order => order.status === 'PENDING').length)

const summaryCards = computed(() => [
  {
    label: '可用订阅',
    value: activeSubscriptions.value.length,
    icon: 'badge' as const,
    className: 'lingqu-billing__summary-icon lingqu-billing__summary-icon--cyan'
  },
  {
    label: '待支付订单',
    value: pendingOrderCount.value,
    icon: 'clock' as const,
    className: 'lingqu-billing__summary-icon lingqu-billing__summary-icon--yellow'
  },
  {
    label: '完成订单',
    value: completedOrderCount.value,
    icon: 'checkCircle' as const,
    className: 'lingqu-billing__summary-icon lingqu-billing__summary-icon--mint'
  }
])

const actionCards = [
  {
    path: '/purchase',
    kicker: '补充余额 / 开通套餐',
    title: '充值/订阅',
    description: '补足调用额度。',
    icon: 'creditCard' as const,
    className: 'lingqu-billing__action-icon lingqu-billing__action-icon--pink'
  },
  {
    path: '/subscriptions',
    kicker: '查看额度和到期',
    title: '我的订阅',
    description: '额度和到期时间。',
    icon: 'badge' as const,
    className: 'lingqu-billing__action-icon lingqu-billing__action-icon--cyan'
  },
  {
    path: '/orders',
    kicker: '支付和退款记录',
    title: '我的订单',
    description: '支付状态和记录。',
    icon: 'document' as const,
    className: 'lingqu-billing__action-icon lingqu-billing__action-icon--yellow'
  },
  {
    path: '/redeem',
    kicker: '兑换码入口',
    title: '兑换',
    description: '兑换余额或套餐。',
    icon: 'gift' as const,
    className: 'lingqu-billing__action-icon lingqu-billing__action-icon--mint'
  }
]

function formatDate(value?: string): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleDateString('zh-CN', {
    month: '2-digit',
    day: '2-digit'
  })
}

function formatSubscriptionTime(value: string | null): string {
  if (!value) return '长期有效'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '时间待确认'
  const days = Math.max(0, Math.ceil((date.getTime() - Date.now()) / (1000 * 60 * 60 * 24)))
  return `${days} 天后到期`
}

function formatOrderAmount(order: PaymentOrder): string {
  const amount = Number(order.pay_amount || order.amount || 0)
  const currency = order.currency || 'USD'
  if (currency.toUpperCase() === 'CNY') return `¥${amount.toFixed(2)}`
  return `$${amount.toFixed(2)}`
}

async function loadOrders() {
  loadingOrders.value = true
  try {
    const response = await paymentAPI.getMyOrders({ page: 1, page_size: 3 })
    recentOrders.value = response.data.items || []
  } catch (error) {
    console.warn('Failed to load billing orders:', error)
  } finally {
    loadingOrders.value = false
  }
}

onMounted(() => {
  authStore.refreshUser().catch((error) => {
    console.warn('Failed to refresh billing user:', error)
  })
  appStore.fetchPublicSettings().catch((error) => {
    console.warn('Failed to load billing settings:', error)
  })
  subscriptionStore.fetchActiveSubscriptions().catch((error) => {
    console.warn('Failed to load billing subscriptions:', error)
  })
  loadOrders()
})
</script>

<style scoped>
.lingqu-billing {
  --billing-ink: #211f1c;
  --billing-line: rgba(33, 31, 28, 0.1);
  --billing-shadow: 0 10px 26px rgba(29, 42, 42, 0.07);
}

.lingqu-billing__overview {
  display: grid;
  grid-template-columns: minmax(18rem, 1.55fr) repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
}

.lingqu-billing__balance,
.lingqu-billing__summary-card,
.lingqu-billing__action-card,
.lingqu-billing__panel {
  border: 1px solid var(--billing-line);
  background: rgba(255, 255, 255, 0.84);
  box-shadow: var(--billing-shadow);
}

.lingqu-billing__balance {
  min-height: 6.65rem;
  display: grid;
  align-content: center;
  gap: 0.24rem;
  border-radius: 18px;
  background:
    radial-gradient(circle at 92% 14%, rgba(245, 215, 101, 0.18), transparent 36%),
    rgba(255, 255, 255, 0.88);
  padding: 0.95rem;
}

.lingqu-billing__balance small,
.lingqu-billing__summary-card small,
.lingqu-billing__action-card small {
  color: rgba(33, 31, 28, 0.52);
  font-size: 0.73rem;
  font-weight: 950;
}

.lingqu-billing__balance strong {
  color: var(--billing-ink);
  font-family: theme('fontFamily.display');
  font-size: clamp(1.85rem, 3.4vw, 2.65rem);
  font-weight: 950;
  line-height: 1;
}

.lingqu-billing__balance p {
  color: rgba(33, 31, 28, 0.62);
  font-size: 0.88rem;
  font-weight: 820;
}

.lingqu-billing__warning {
  color: #c35b76 !important;
}

.lingqu-billing__summary-card {
  min-height: 6.65rem;
  display: grid;
  align-content: center;
  gap: 0.34rem;
  border-radius: 18px;
  padding: 0.86rem;
  transition: transform 160ms ease, box-shadow 160ms ease;
}

.lingqu-billing__summary-card:hover,
.lingqu-billing__action-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 14px 30px rgba(29, 42, 42, 0.1);
}

.lingqu-billing__summary-card strong {
  color: var(--billing-ink);
  font-size: 1.55rem;
  font-weight: 950;
}

.lingqu-billing__summary-icon,
.lingqu-billing__action-icon {
  display: grid;
  place-items: center;
  border: 1px solid rgba(33, 31, 28, 0.1);
  color: var(--billing-ink);
}

.lingqu-billing__summary-icon {
  width: 2.35rem;
  height: 2.35rem;
  border-radius: 14px;
}

.lingqu-billing__actions {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.68rem;
}

.lingqu-billing__action-card {
  min-height: 6.05rem;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: start;
  gap: 0.75rem;
  border-radius: 18px;
  color: var(--billing-ink);
  padding: 0.78rem;
  transition: transform 160ms ease, box-shadow 160ms ease;
}

.lingqu-billing__action-icon {
  width: 2.3rem;
  height: 2.3rem;
  border-radius: 13px;
}

.lingqu-billing__action-card strong {
  display: block;
  margin-top: 0.18rem;
  color: var(--billing-ink);
  font-size: 1.08rem;
  font-weight: 950;
}

.lingqu-billing__action-card p {
  margin-top: 0.18rem;
  color: rgba(33, 31, 28, 0.58);
  font-size: 0.84rem;
  font-weight: 760;
  line-height: 1.36;
}

.lingqu-billing__summary-icon--pink,
.lingqu-billing__action-icon--pink {
  background: #fff0f4;
}

.lingqu-billing__summary-icon--cyan,
.lingqu-billing__action-icon--cyan {
  background: #edfafa;
}

.lingqu-billing__summary-icon--yellow,
.lingqu-billing__action-icon--yellow {
  background: #fff7da;
}

.lingqu-billing__summary-icon--mint,
.lingqu-billing__action-icon--mint {
  background: #edf9f3;
}

.lingqu-billing__mini-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.9rem;
}

.lingqu-billing__panel {
  min-height: 14.4rem;
  border-radius: 18px;
  padding: 0.88rem;
}

.lingqu-billing__panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 0.85rem;
}

.lingqu-billing__panel-head h2 {
  margin-top: 0.34rem;
  color: var(--billing-ink);
  font-family: theme('fontFamily.display');
  font-size: 1.22rem;
  font-weight: 950;
}

.lingqu-billing__panel-head a {
  flex: 0 0 auto;
  border: 1px solid rgba(33, 31, 28, 0.1);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.88);
  color: var(--billing-ink);
  padding: 0.35rem 0.65rem;
  font-size: 0.76rem;
  font-weight: 950;
}

.lingqu-billing__loading,
.lingqu-billing__empty {
  min-height: 7.8rem;
  display: grid;
  place-items: center;
  gap: 0.55rem;
  color: rgba(33, 31, 28, 0.54);
  text-align: center;
  font-size: 0.88rem;
  font-weight: 830;
}

.lingqu-billing__empty p {
  max-width: 23rem;
}

.lingqu-billing__subscription-list,
.lingqu-billing__order-list {
  display: grid;
  gap: 0.65rem;
}

.lingqu-billing__subscription-list article,
.lingqu-billing__order-list article {
  display: grid;
  align-items: center;
  gap: 0.6rem;
  border: 1px solid rgba(33, 31, 28, 0.08);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.76);
  padding: 0.76rem;
}

.lingqu-billing__subscription-list article {
  grid-template-columns: minmax(0, 1fr) auto;
}

.lingqu-billing__order-list article {
  grid-template-columns: minmax(0, 1fr) auto auto;
}

.lingqu-billing__subscription-list strong,
.lingqu-billing__order-list strong {
  display: block;
  overflow: hidden;
  color: var(--billing-ink);
  font-weight: 950;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.lingqu-billing__subscription-list small,
.lingqu-billing__order-list small {
  color: rgba(33, 31, 28, 0.5);
  font-size: 0.74rem;
  font-weight: 820;
}

.lingqu-billing__subscription-list span,
.lingqu-billing__order-list span {
  border-radius: 999px;
  background: #fff7da;
  color: var(--billing-ink);
  padding: 0.24rem 0.55rem;
  font-size: 0.72rem;
  font-weight: 950;
}

.lingqu-billing__order-list b {
  color: var(--billing-ink);
  font-size: 0.9rem;
  font-weight: 950;
}

@media (max-width: 1080px) {
  .lingqu-billing__overview {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .lingqu-billing__balance {
    grid-column: 1 / -1;
  }

  .lingqu-billing__actions {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 780px) {
  .lingqu-billing__mini-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 560px) {
  .lingqu-billing__overview,
  .lingqu-billing__actions {
    grid-template-columns: 1fr;
  }

  .lingqu-billing__action-card {
    min-height: 0;
  }
}
</style>
