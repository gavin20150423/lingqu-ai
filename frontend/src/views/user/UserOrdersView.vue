<template>
  <UserWorkspaceLayout>
    <div class="lingqu-console-page lingqu-billing-page">
      <section class="lingqu-console-hero">
        <div>
          <span class="lingqu-console-eyebrow">订单记录</span>
          <h1>我的订单</h1>
          <p>充值、订阅、退款状态都在这里。</p>
        </div>
        <div class="lingqu-console-actions">
          <button type="button" class="lingqu-console-button" @click="fetchOrders" :disabled="loading">
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
            {{ t('common.refresh') }}
          </button>
          <button type="button" class="lingqu-console-button lingqu-console-button--primary" @click="router.push('/purchase')">
            <Icon name="creditCard" size="sm" />
            {{ t('payment.result.backToRecharge') }}
          </button>
        </div>
      </section>

      <section class="lingqu-console-stats">
        <article class="lingqu-console-stat">
          <small>订单数量</small>
          <strong>{{ orderSummary.total }}</strong>
        </article>
        <article class="lingqu-console-stat">
          <small>{{ t('payment.status.pending') }}</small>
          <strong>{{ orderSummary.pending }}</strong>
        </article>
        <article class="lingqu-console-stat">
          <small>{{ t('payment.status.completed') }}</small>
          <strong>{{ orderSummary.completed }}</strong>
        </article>
        <article class="lingqu-console-stat">
          <small>本页金额</small>
          <strong>¥{{ orderSummary.amount }}</strong>
        </article>
      </section>

      <!-- Filters -->
      <div class="card p-4">
        <div class="flex flex-wrap items-center gap-3">
          <Select v-model="currentFilter" :options="statusFilters" class="w-36" @change="fetchOrders" />
          <div class="flex flex-1 items-center justify-end gap-2">
            <button class="btn btn-primary" @click="router.push('/purchase')">{{ t('payment.result.backToRecharge') }}</button>
          </div>
        </div>
      </div>

      <!-- Table -->
      <OrderTable :orders="orders" :loading="loading">
        <template #actions="{ row }">
          <div class="flex items-center gap-2">
            <button v-if="row.status === 'PENDING'" @click="handleCancel(row.id)" class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-yellow-600 hover:bg-yellow-50 dark:text-yellow-400 dark:hover:bg-yellow-900/20">
              <Icon name="x" size="sm" />
              <span>{{ t('payment.orders.cancel') }}</span>
            </button>
            <button v-if="canRequestRefund(row)" @click="openRefundDialog(row)" class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-purple-600 hover:bg-purple-50 dark:text-purple-400 dark:hover:bg-purple-900/20">
              <Icon name="dollar" size="sm" />
              <span>{{ t('payment.orders.requestRefund') }}</span>
            </button>
          </div>
        </template>
      </OrderTable>

      <!-- Pagination -->
      <Pagination
        v-if="pagination.total > 0"
        :page="pagination.page"
        :total="pagination.total"
        :page-size="pagination.page_size"
        @update:page="handlePageChange"
        @update:pageSize="handlePageSizeChange"
      />
    </div>

    <!-- Cancel Confirm Dialog -->
    <BaseDialog :show="!!cancelTargetId" :title="t('payment.orders.cancel')" width="narrow" @close="cancelTargetId = null">
      <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('payment.confirmCancel') }}</p>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" @click="cancelTargetId = null">{{ t('common.cancel') }}</button>
          <button class="btn btn-danger" :disabled="actionLoading" @click="confirmCancel">{{ actionLoading ? t('common.processing') : t('payment.orders.cancel') }}</button>
        </div>
      </template>
    </BaseDialog>

    <!-- Refund Dialog -->
    <BaseDialog :show="!!refundTarget" :title="t('payment.orders.requestRefund')" @close="refundTarget = null">
      <div v-if="refundTarget" class="space-y-4">
        <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-800">
          <div class="flex justify-between text-sm">
            <span class="text-gray-500 dark:text-gray-400">{{ t('payment.orders.orderId') }}</span>
            <span class="font-mono text-gray-900 dark:text-white">#{{ refundTarget.id }}</span>
          </div>
          <div class="mt-2 flex justify-between text-sm">
            <span class="text-gray-500 dark:text-gray-400">{{ t('payment.orders.amount') }}</span>
            <span class="text-gray-900 dark:text-white">${{ refundTarget.amount.toFixed(2) }}</span>
          </div>
        </div>
        <div>
          <label class="input-label">{{ t('payment.refundReason') }}</label>
          <textarea v-model="refundReason" rows="3" class="input mt-1 w-full" :placeholder="t('payment.refundReasonPlaceholder')" />
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" @click="refundTarget = null">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary" :disabled="actionLoading || !refundReason.trim()" @click="confirmRefund">{{ actionLoading ? t('common.processing') : t('payment.orders.requestRefund') }}</button>
        </div>
      </template>
    </BaseDialog>
  </UserWorkspaceLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores'
import { paymentAPI } from '@/api/payment'
import { extractI18nErrorMessage } from '@/utils/apiError'
import type { PaymentOrder } from '@/types/payment'
import UserWorkspaceLayout from '@/components/layout/UserWorkspaceLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import OrderTable from '@/components/payment/OrderTable.vue'

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()

const loading = ref(false)
const actionLoading = ref(false)
const orders = ref<PaymentOrder[]>([])
const refundEligibleProviders = ref<Set<string>>(new Set())
const currentFilter = ref('')
const cancelTargetId = ref<number | null>(null)
const refundTarget = ref<PaymentOrder | null>(null)
const refundReason = ref('')
const pagination = reactive({ page: 1, page_size: 20, total: 0 })

const orderSummary = computed(() => ({
  total: pagination.total || orders.value.length,
  pending: orders.value.filter((order) => order.status === 'PENDING').length,
  completed: orders.value.filter((order) => order.status === 'COMPLETED').length,
  amount: orders.value.reduce((sum, order) => sum + (order.pay_amount || 0), 0).toFixed(2)
}))

const statusFilters = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'PENDING', label: t('payment.status.pending') },
  { value: 'COMPLETED', label: t('payment.status.completed') },
  { value: 'FAILED', label: t('payment.status.failed') },
  { value: 'REFUNDED', label: t('payment.status.refunded') },
])

async function fetchOrders() {
  loading.value = true
  try {
    const res = await paymentAPI.getMyOrders({
      page: pagination.page,
      page_size: pagination.page_size,
      status: currentFilter.value || undefined,
    })
    orders.value = res.data.items || []
    pagination.total = res.data.total || 0
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    loading.value = false
  }
}

function handlePageChange(page: number) { pagination.page = page; fetchOrders() }
function handlePageSizeChange(size: number) { pagination.page_size = size; pagination.page = 1; fetchOrders() }

function handleCancel(orderId: number) { cancelTargetId.value = orderId }

async function confirmCancel() {
  if (!cancelTargetId.value) return
  actionLoading.value = true
  try {
    await paymentAPI.cancelOrder(cancelTargetId.value)
    appStore.showSuccess(t('common.success'))
    cancelTargetId.value = null
    await fetchOrders()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    actionLoading.value = false
  }
}

function openRefundDialog(order: PaymentOrder) { refundTarget.value = order; refundReason.value = '' }

async function confirmRefund() {
  if (!refundTarget.value || !refundReason.value.trim()) return
  actionLoading.value = true
  try {
    await paymentAPI.requestRefund(refundTarget.value.id, { reason: refundReason.value.trim() })
    appStore.showSuccess(t('common.success'))
    refundTarget.value = null
    refundReason.value = ''
    await fetchOrders()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    actionLoading.value = false
  }
}

function canRequestRefund(order: PaymentOrder): boolean {
  if (order.status !== 'COMPLETED') return false
  if (!order.provider_instance_id) return false
  return refundEligibleProviders.value.has(order.provider_instance_id)
}

async function loadRefundEligibility() {
  try {
    const res = await paymentAPI.getRefundEligibleProviders()
    refundEligibleProviders.value = new Set(res.data.provider_instance_ids || [])
  } catch { /* ignore — default to hiding refund button */ }
}

onMounted(() => { fetchOrders(); loadRefundEligibility() })
</script>
