<template>
  <BaseDialog :show="show" :title="t('admin.users.resetCardsTitle')" width="narrow" @close="$emit('close')">
    <form v-if="user" id="reset-cards-form" @submit.prevent="handleSubmit" class="space-y-5">
      <div class="flex items-center gap-3 rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
        <div class="flex h-10 w-10 items-center justify-center rounded-full bg-primary-100"><span class="text-lg font-medium text-primary-700">{{ user.email.charAt(0).toUpperCase() }}</span></div>
        <div class="flex-1"><p class="font-medium text-gray-900 dark:text-gray-100">{{ user.email }}</p><p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.users.resetCardsCurrent', { count: currentBalance }) }}</p></div>
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.resetCardsAmount') }}</label>
        <input v-model.number="amount" type="number" min="0" step="1" class="input" />
      </div>
      <div class="space-y-2">
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input v-model="mode" type="radio" value="add" />
          <span>{{ t('admin.users.resetCardsModeAdd') }}</span>
        </label>
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input v-model="mode" type="radio" value="set" />
          <span>{{ t('admin.users.resetCardsModeSet') }}</span>
        </label>
      </div>
      <div class="rounded-xl border border-blue-200 bg-blue-50 p-4 text-sm text-gray-700 dark:border-blue-800 dark:bg-blue-950 dark:text-gray-300">
        {{ t('admin.users.resetCardsHint') }}
      </div>
      <p v-if="formError" class="text-sm text-red-600 dark:text-red-400">{{ formError }}</p>
    </form>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" @click="$emit('close')" class="btn btn-secondary">{{ t('common.cancel') }}</button>
        <button type="submit" form="reset-cards-form" :disabled="submitting || !canSubmit" class="btn btn-primary">
          {{ submitting ? t('common.saving') : t('common.confirm') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import usersAPI from '@/api/admin/users'
import type { AdminUser } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'

const props = defineProps<{ show: boolean; user: AdminUser | null }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'success'): void }>()

const { t } = useI18n()
const appStore = useAppStore()

const amount = ref(1)
const mode = ref<'add' | 'set'>('add')
const currentBalance = ref(0)
const submitting = ref(false)
const formError = ref('')

const canSubmit = computed(() => {
  const value = Number(amount.value)
  if (!Number.isFinite(value)) return false
  return mode.value === 'add' ? value > 0 : value >= 0
})

watch(
  () => [props.show, props.user?.id] as const,
  async ([visible]) => {
    if (!visible) return
    amount.value = 1
    mode.value = 'add'
    formError.value = ''
    currentBalance.value = 0
    if (!props.user) return
    try {
      const result = await usersAPI.getSubscriptionResetCards(props.user.id)
      currentBalance.value = result.balance
    } catch {
      // 读不到当前余额不阻塞发放，提交后以服务端返回为准
    }
  },
  { immediate: true }
)

async function handleSubmit() {
  if (!props.user || !canSubmit.value || submitting.value) return
  submitting.value = true
  formError.value = ''
  try {
    const value = Math.trunc(Number(amount.value))
    const payload = mode.value === 'add' ? { delta: value } : { set: value }
    const result = await usersAPI.grantSubscriptionResetCards(props.user.id, payload)
    currentBalance.value = result.balance
    appStore.showSuccess(t('admin.users.resetCardsSuccess', { count: result.balance }))
    emit('success')
    emit('close')
  } catch (error: any) {
    const message = error?.response?.data?.message || t('admin.users.resetCardsFailed')
    formError.value = message
    appStore.showError(message)
  } finally {
    submitting.value = false
  }
}
</script>
