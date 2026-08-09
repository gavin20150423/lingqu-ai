<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl">
      <div class="grid min-h-[calc(100vh-10rem)] gap-6 lg:grid-cols-[18rem_minmax(0,1fr)] lg:gap-8">
        <aside class="min-w-0 border-b border-gray-200 pb-6 dark:border-dark-700 lg:border-b-0 lg:border-r lg:pb-0 lg:pr-6">
          <div class="mb-4 flex items-center justify-between gap-3">
            <div>
              <h2 class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ t('admin.xiaoVideo.upstreams') }}
              </h2>
              <span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.xiaoVideo.upstreamCount', { count: accounts.length }) }}
              </span>
            </div>
            <button
              type="button"
              class="btn btn-primary flex h-9 w-9 items-center justify-center p-0"
              :title="t('admin.xiaoVideo.addUpstream')"
              data-testid="xiao-add-upstream"
              @click="beginCreate()"
            >
              <Icon name="plus" size="sm" />
            </button>
          </div>

          <div v-if="loading" class="flex justify-center py-10" data-testid="xiao-list-loading">
            <span class="h-6 w-6 animate-spin rounded-full border-2 border-gray-200 border-t-primary-500" />
          </div>

          <div v-else class="space-y-2">
            <button
              v-if="isCreating"
              type="button"
              class="xiao-upstream-item xiao-upstream-item-active"
              data-testid="xiao-new-upstream-item"
            >
              <span class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-300">
                <Icon name="plus" size="sm" />
              </span>
              <span class="min-w-0 flex-1 text-left">
                <span class="block truncate text-sm font-medium">{{ t('admin.xiaoVideo.newUpstream') }}</span>
                <span class="mt-0.5 block text-xs text-gray-500 dark:text-gray-400">XiaoAPI</span>
              </span>
            </button>

            <button
              v-for="account in accounts"
              :key="account.id"
              type="button"
              class="xiao-upstream-item"
              :class="{ 'xiao-upstream-item-active': selectedId === account.id }"
              :data-testid="`xiao-upstream-${account.id}`"
              @click="selectAccount(account.id)"
            >
              <span
                class="h-2.5 w-2.5 flex-shrink-0 rounded-full"
                :class="account.status === 'active' ? 'bg-emerald-500' : account.status === 'error' ? 'bg-red-500' : 'bg-gray-300 dark:bg-dark-500'"
              />
              <span class="min-w-0 flex-1 text-left">
                <span class="block truncate text-sm font-medium text-gray-900 dark:text-white">{{ account.name }}</span>
                <span class="mt-0.5 block truncate text-xs text-gray-500 dark:text-gray-400">
                  {{ accountStatusLabel(account) }} · {{ t('admin.xiaoVideo.concurrencyValue', { count: account.concurrency }) }}
                </span>
              </span>
              <Icon name="chevronRight" size="xs" class="flex-shrink-0 text-gray-400" />
            </button>

            <div v-if="accounts.length === 0 && !isCreating" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.xiaoVideo.empty') }}
            </div>
          </div>
        </aside>

        <main class="min-w-0">
          <div v-if="detailsLoading" class="flex items-center justify-center py-24" data-testid="xiao-details-loading">
            <span class="h-8 w-8 animate-spin rounded-full border-2 border-gray-200 border-t-primary-500" />
          </div>

          <form v-else class="space-y-8" novalidate @submit.prevent="save">
            <section class="border-b border-gray-200 pb-8 dark:border-dark-700">
              <div class="mb-5 flex flex-wrap items-center justify-between gap-3">
                <div class="flex items-center gap-3">
                  <span class="flex h-10 w-10 items-center justify-center rounded-lg bg-sky-50 text-sky-700 dark:bg-sky-900/30 dark:text-sky-300">
                    <Icon name="server" size="md" />
                  </span>
                  <div>
                    <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                      {{ t('admin.xiaoVideo.upstreamSettings') }}
                    </h2>
                    <span v-if="selectedAccount" class="mt-0.5 block font-mono text-xs text-gray-400">#{{ selectedAccount.id }}</span>
                  </div>
                </div>
                <span
                  v-if="selectedAccount"
                  class="inline-flex items-center rounded-full px-2.5 py-1 text-xs font-medium"
                  :class="selectedAccount.status === 'active'
                    ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/25 dark:text-emerald-300'
                    : selectedAccount.status === 'error'
                      ? 'bg-red-50 text-red-700 dark:bg-red-900/25 dark:text-red-300'
                      : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'"
                >
                  {{ accountStatusLabel(selectedAccount) }}
                </span>
              </div>

              <div class="grid gap-5 md:grid-cols-2">
                <label class="min-w-0">
                  <span class="input-label">{{ t('admin.xiaoVideo.name') }}</span>
                  <input
                    v-model="form.name"
                    type="text"
                    class="input"
                    autocomplete="off"
                    data-testid="xiao-name"
                  />
                </label>

                <label class="min-w-0">
                  <span class="input-label">{{ t('admin.xiaoVideo.protocol') }}</span>
                  <select
                    v-model="form.protocol"
                    class="input"
                    data-testid="xiao-protocol"
                    @change="applyProtocolDefaults"
                  >
                    <option value="native">{{ t('admin.xiaoVideo.protocolNative') }}</option>
                    <option value="openai_sora">{{ t('admin.xiaoVideo.protocolOpenAISora') }}</option>
                  </select>
                </label>

                <label class="min-w-0">
                  <span class="input-label">{{ t('admin.xiaoVideo.baseUrl') }}</span>
                  <input
                    v-model="form.baseUrl"
                    type="url"
                    class="input font-mono"
                    autocomplete="url"
                    :placeholder="form.protocol === 'openai_sora' ? AISTARTLAB_BASE_URL : 'https://video-upstream.example.com'"
                    data-testid="xiao-base-url"
                  />
                </label>

                <label class="min-w-0">
                  <span class="input-label">{{ t('admin.xiaoVideo.apiKey') }}</span>
                  <input
                    v-model="form.apiKey"
                    type="password"
                    class="input font-mono"
                    autocomplete="new-password"
                    :placeholder="hasExistingApiKey ? t('admin.xiaoVideo.apiKeyUnchanged') : ''"
                    data-testid="xiao-api-key"
                  />
                </label>

                <div class="grid grid-cols-2 gap-3">
                  <label class="min-w-0">
                    <span class="input-label">{{ t('admin.xiaoVideo.status') }}</span>
                    <select v-model="form.status" class="input" data-testid="xiao-status">
                      <option value="active">{{ t('common.active') }}</option>
                      <option value="inactive">{{ t('common.inactive') }}</option>
                      <option v-if="form.status === 'error'" value="error">{{ t('admin.xiaoVideo.errorStatus') }}</option>
                    </select>
                  </label>
                  <label class="min-w-0">
                    <span class="input-label">{{ t('admin.xiaoVideo.concurrency') }}</span>
                    <input
                      v-model.number="form.concurrency"
                      type="number"
                      min="1"
                      max="10000"
                      step="1"
                      class="input"
                      data-testid="xiao-concurrency"
                    />
                  </label>
                </div>

                <label class="min-w-0 md:col-span-2">
                  <span class="input-label">{{ t('admin.xiaoVideo.notes') }}</span>
                  <textarea
                    v-model="form.notes"
                    rows="2"
                    class="input resize-y"
                    data-testid="xiao-notes"
                  />
                </label>
              </div>

              <div class="mt-6 border-t border-gray-100 pt-5 dark:border-dark-700">
                <div class="mb-3 flex items-center justify-between gap-3">
                  <span class="input-label mb-0">{{ t('admin.xiaoVideo.groups') }}</span>
                  <router-link to="/admin/groups" class="text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400">
                    {{ t('admin.xiaoVideo.manageGroups') }}
                  </router-link>
                </div>
                <div v-if="groups.length > 0" class="grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
                  <label
                    v-for="group in groups"
                    :key="group.id"
                    class="flex min-w-0 cursor-pointer items-center gap-3 rounded-lg border border-gray-200 px-3 py-2.5 hover:border-primary-300 dark:border-dark-600 dark:hover:border-primary-700"
                  >
                    <input
                      type="checkbox"
                      class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                      :checked="form.groupIds.includes(group.id)"
                      :data-testid="`xiao-group-${group.id}`"
                      @change="toggleGroup(group.id, $event)"
                    />
                    <span class="min-w-0 flex-1 truncate text-sm text-gray-700 dark:text-gray-200">{{ group.name }}</span>
                    <span v-if="group.status !== 'active'" class="text-xs text-gray-400">{{ t('common.inactive') }}</span>
                  </label>
                </div>
                <div v-else class="rounded-lg border border-dashed border-gray-300 px-4 py-5 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400">
                  {{ t('admin.xiaoVideo.noGroups') }}
                </div>
              </div>
            </section>

            <section class="min-w-0 pb-2">
              <div class="mb-5 flex flex-wrap items-center justify-between gap-3">
                <div class="flex items-center gap-3">
                  <span class="flex h-10 w-10 items-center justify-center rounded-lg bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300">
                    <Icon name="dollar" size="md" />
                  </span>
                  <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                    {{ t('admin.xiaoVideo.modelsAndPricing') }}
                  </h2>
                </div>
                <button
                  type="button"
                  class="btn btn-secondary inline-flex items-center gap-2"
                  :disabled="fetchingModels || saving || deleting"
                  data-testid="xiao-fetch-models"
                  @click="fetchUpstreamModels"
                >
                  <Icon name="refresh" size="sm" :class="{ 'animate-spin': fetchingModels }" />
                  {{ fetchingModels ? t('admin.xiaoVideo.fetchingModels') : t('admin.xiaoVideo.fetchModels') }}
                </button>
              </div>

              <div
                v-if="modelPickerOpen"
                class="mb-5 border-y border-gray-200 bg-gray-50/60 py-4 dark:border-dark-600 dark:bg-dark-800/30"
                data-testid="xiao-model-picker"
              >
                <div class="flex flex-wrap items-start justify-between gap-3 px-1">
                  <div>
                    <p class="text-sm font-medium text-gray-900 dark:text-white">
                      {{ t('admin.xiaoVideo.fetchedModels', { count: upstreamModels.length }) }}
                    </p>
                    <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                      {{ t('admin.xiaoVideo.fetchedModelsHint') }}
                    </p>
                    <p v-if="pricingNoteLabel" class="mt-2 text-xs font-medium text-amber-700 dark:text-amber-300" data-testid="xiao-pricing-note">
                      {{ pricingNoteLabel }}
                    </p>
                  </div>
                  <div class="flex items-center gap-2">
                    <span class="inline-flex items-center rounded-full bg-white px-2.5 py-1 text-xs font-medium text-gray-600 ring-1 ring-inset ring-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:ring-dark-600">
                      {{ pricingSourceLabel }}
                    </span>
                    <button
                      type="button"
                      class="flex h-8 w-8 items-center justify-center text-gray-400 hover:text-gray-700 dark:hover:text-gray-200"
                      :title="t('common.close')"
                      data-testid="xiao-close-model-picker"
                      @click="modelPickerOpen = false"
                    >
                      <Icon name="x" size="sm" />
                    </button>
                  </div>
                </div>

                <div class="mt-4 grid items-end gap-3 px-1 md:grid-cols-[minmax(14rem,1fr)_10rem_auto]">
                  <label class="relative min-w-[14rem] flex-1">
                    <Icon name="search" size="sm" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                    <input
                      v-model="modelSearch"
                      type="search"
                      class="input pl-9"
                      :placeholder="t('admin.xiaoVideo.searchModels')"
                      data-testid="xiao-model-search"
                    />
                  </label>
                  <label class="min-w-0">
                    <span class="input-label">{{ t('admin.xiaoVideo.markupMultiplier') }}</span>
                    <div class="relative">
                      <input
                        v-model.number="markupMultiplier"
                        type="number"
                        min="0"
                        max="100"
                        step="0.05"
                        class="input pr-8 tabular-nums"
                        data-testid="xiao-markup-multiplier"
                      />
                      <span class="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-sm text-gray-400">×</span>
                    </div>
                  </label>
                  <div class="flex flex-wrap items-center gap-2">
                    <button type="button" class="btn btn-secondary text-sm" @click="selectVisibleModels">
                      {{ t('admin.xiaoVideo.selectVisible') }}
                    </button>
                    <button type="button" class="btn btn-secondary text-sm" @click="selectedUpstreamModels = []">
                      {{ t('admin.xiaoVideo.clearSelection') }}
                    </button>
                  </div>
                </div>

                <p class="mt-2 px-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ hasConvertiblePricing ? t('admin.xiaoVideo.markupMultiplierHint') : t('admin.xiaoVideo.noConvertiblePricing') }}
                </p>

                <div class="mt-3 max-h-80 overflow-y-auto border-y border-gray-200 dark:border-dark-600">
                  <label
                    v-for="model in filteredUpstreamModels"
                    :key="model"
                    class="flex min-h-14 flex-wrap items-center gap-3 border-b border-gray-100 px-2 py-2.5 last:border-b-0 hover:bg-white dark:border-dark-700 dark:hover:bg-dark-700/50 sm:flex-nowrap"
                    :class="{ 'cursor-pointer': canImportUpstreamModel(model), 'opacity-60': !canImportUpstreamModel(model) }"
                  >
                    <input
                      type="checkbox"
                      class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                      :checked="selectedUpstreamModels.includes(model)"
                      :disabled="!canImportUpstreamModel(model)"
                      :data-testid="`xiao-upstream-model-${model}`"
                      @change="toggleUpstreamModel(model, $event)"
                    />
                    <span class="min-w-0 flex-1 break-all font-mono text-xs font-medium text-gray-700 dark:text-gray-200 sm:w-44 sm:flex-none">{{ model }}</span>
                    <span class="grid w-full grid-cols-2 gap-3 pl-7 sm:min-w-0 sm:flex-1 sm:pl-0">
                      <span class="min-w-0">
                        <span class="block text-[11px] text-gray-400">{{ t('admin.xiaoVideo.upstreamCost') }}</span>
                        <span class="mt-0.5 block break-words text-xs font-medium leading-5 tabular-nums text-gray-700 dark:text-gray-200" :title="modelCostSummary(model)">
                          {{ modelCostSummary(model) }}
                        </span>
                      </span>
                      <span class="min-w-0">
                        <span class="block text-[11px] text-gray-400">{{ t('admin.xiaoVideo.suggestedPrice') }}</span>
                        <span class="mt-0.5 block break-words text-xs font-semibold leading-5 tabular-nums text-amber-700 dark:text-amber-300" :title="modelSuggestedSummary(model)">
                          {{ modelSuggestedSummary(model) }}
                        </span>
                      </span>
                    </span>
                    <span v-if="configuredUpstreamModels.has(model) && canImportUpstreamModel(model)" class="flex-shrink-0 text-xs text-amber-600 dark:text-amber-300">
                      {{ t('admin.xiaoVideo.modelPricingMissing') }}
                    </span>
                    <span v-else-if="configuredUpstreamModels.has(model)" class="flex-shrink-0 text-xs text-emerald-600 dark:text-emerald-400">
                      {{ t('admin.xiaoVideo.modelConfigured') }}
                    </span>
                  </label>
                  <div v-if="filteredUpstreamModels.length === 0" class="px-3 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
                    {{ t('admin.xiaoVideo.noMatchingModels') }}
                  </div>
                </div>

                <div class="mt-4 flex flex-wrap items-center justify-between gap-3 px-1">
                  <span class="text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.xiaoVideo.selectedModels', { count: selectedUpstreamModels.length }) }}
                  </span>
                  <button
                    type="button"
                    class="btn btn-primary inline-flex items-center gap-2"
                    :disabled="selectedUpstreamModels.length === 0"
                    data-testid="xiao-import-models"
                    @click="importSelectedModels"
                  >
                    <Icon name="plus" size="sm" />
                    {{ t('admin.xiaoVideo.importModels', { count: selectedUpstreamModels.length }) }}
                  </button>
                </div>
              </div>

              <XiaoVideoConfigEditor v-model:pricing="pricing" v-model:mappings="mappings" />
            </section>

            <div class="-mx-4 flex flex-wrap items-center justify-between gap-3 border-t border-gray-200 px-4 py-4 dark:border-dark-700 sm:mx-0 sm:px-0">
              <div class="flex items-center gap-2">
                <button
                  v-if="selectedAccount"
                  type="button"
                  class="btn btn-secondary inline-flex items-center gap-2"
                  :disabled="testing || saving || deleting"
                  data-testid="xiao-test"
                  @click="testConnection"
                >
                  <Icon name="beaker" size="sm" />
                  {{ testing ? t('admin.xiaoVideo.testing') : t('admin.xiaoVideo.testConnection') }}
                </button>
                <button
                  v-if="selectedAccount"
                  type="button"
                  class="btn btn-secondary inline-flex items-center gap-2 text-red-600 dark:text-red-400"
                  :disabled="deleting || saving || testing"
                  data-testid="xiao-delete"
                  @click="removeAccount"
                >
                  <Icon name="trash" size="sm" />
                  {{ t('common.delete') }}
                </button>
              </div>
              <button
                type="submit"
                class="btn btn-primary inline-flex min-w-28 items-center justify-center gap-2"
                :disabled="saving || deleting || detailsLoading"
                data-testid="xiao-save"
              >
                <Icon name="check" size="sm" />
                {{ saving ? t('common.saving') : t('common.save') }}
              </button>
            </div>
          </form>
        </main>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import XiaoVideoConfigEditor from '@/components/account/XiaoVideoConfigEditor.vue'
import {
  createXiaoVideoPricingRule,
  normalizeXiaoVideoPricing,
  readXiaoVideoPricing,
  validateXiaoVideoPricing,
  type XiaoVideoModelMapping,
  type XiaoVideoPricingRule
} from '@/components/account/xiaoVideoPricing'
import { isValidWildcardPattern } from '@/composables/useModelWhitelist'
import { useAppStore } from '@/stores/app'
import type { Account, AdminGroup } from '@/types'
import type { UpstreamModelSpec } from '@/api/admin/accounts'

type EditableStatus = 'active' | 'inactive' | 'error'
type VideoProtocol = 'native' | 'openai_sora'

const AISTARTLAB_BASE_URL = 'https://api.video.aistarslab.com/openai'

interface XiaoVideoForm {
  name: string
  notes: string
  protocol: VideoProtocol
  baseUrl: string
  apiKey: string
  concurrency: number
  status: EditableStatus
  groupIds: number[]
}

const { t } = useI18n()
const appStore = useAppStore()

const accounts = ref<Account[]>([])
const groups = ref<AdminGroup[]>([])
const selectedId = ref<number | null>(null)
const selectedAccount = ref<Account | null>(null)
const mappings = ref<XiaoVideoModelMapping[]>([])
const pricing = ref<XiaoVideoPricingRule[]>([createXiaoVideoPricingRule()])
const hasExistingApiKey = ref(false)
const loading = ref(true)
const detailsLoading = ref(false)
const saving = ref(false)
const testing = ref(false)
const deleting = ref(false)
const fetchingModels = ref(false)
const modelPickerOpen = ref(false)
const upstreamModels = ref<string[]>([])
const upstreamModelSpecs = ref<UpstreamModelSpec[]>([])
const upstreamPricingSource = ref('none')
const upstreamPricingNote = ref('')
const selectedUpstreamModels = ref<string[]>([])
const modelSearch = ref('')
const markupMultiplier = ref(1.3)
const baseline = ref('')
let selectionRequest = 0

const form = reactive<XiaoVideoForm>({
  name: '',
  notes: '',
  protocol: 'native',
  baseUrl: '',
  apiKey: '',
  concurrency: 1,
  status: 'active',
  groupIds: []
})

const isCreating = computed(() => selectedId.value === null)
const isDirty = computed(() => baseline.value !== formSnapshot())
const configuredUpstreamModels = computed(() => new Set(
  mappings.value.map((mapping) => mapping.to.trim()).filter(Boolean)
))
const filteredUpstreamModels = computed(() => {
  const query = modelSearch.value.trim().toLowerCase()
  if (!query) return upstreamModels.value
  return upstreamModels.value.filter((model) => model.toLowerCase().includes(query))
})
const specsByModel = computed(() => {
  const grouped = new Map<string, UpstreamModelSpec[]>()
  for (const spec of upstreamModelSpecs.value) {
    const id = spec.id.trim()
    if (!id) continue
    grouped.set(id, [...(grouped.get(id) ?? []), spec])
  }
  return grouped
})
const hasConvertiblePricing = computed(() =>
  upstreamModelSpecs.value.some((spec) => internalCostPerSecond(spec) !== null)
)
const pricingSourceLabel = computed(() => {
  if (upstreamPricingSource.value === 'aistartlab_config') return t('admin.xiaoVideo.pricingSourceAIStartLab')
  if (upstreamModelSpecs.value.some((spec) => spec.upstream_cost !== undefined)) {
    return t('admin.xiaoVideo.pricingSourceModelList')
  }
  return t('admin.xiaoVideo.pricingSourceNone')
})
const pricingNoteLabel = computed(() => {
  if (upstreamPricingNote.value === 'aistartlab_config_unavailable') return t('admin.xiaoVideo.pricingNoteAIStartLabUnavailable')
  if (upstreamPricingNote.value === 'incomplete_pricing') return t('admin.xiaoVideo.pricingNoteIncomplete')
  return ''
})

function formSnapshot(): string {
  return JSON.stringify({
    form: {
      ...form,
      groupIds: [...form.groupIds].sort((a, b) => a - b)
    },
    mappings: mappings.value,
    pricing: pricing.value
  })
}

function setBaseline() {
  baseline.value = formSnapshot()
}

function errorMessage(error: unknown, fallback: string): string {
  if (error && typeof error === 'object' && 'message' in error && typeof error.message === 'string') {
    return error.message
  }
  return fallback
}

function editableStatus(account: Account): EditableStatus {
  if (account.status === 'active' || account.status === 'inactive' || account.status === 'error') {
    return account.status
  }
  return 'inactive'
}

function accountStatusLabel(account: Account): string {
  if (account.status === 'active') return t('common.active')
  if (account.status === 'error') return t('admin.xiaoVideo.errorStatus')
  return t('common.inactive')
}

function readMappings(value: unknown): XiaoVideoModelMapping[] {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return []
  return Object.entries(value as Record<string, unknown>).flatMap(([from, rawTo]) => {
    if (typeof rawTo !== 'string') return []
    return [{ from, to: rawTo }]
  })
}

function resetForm() {
  Object.assign(form, {
    name: '',
    notes: '',
    protocol: 'native' as VideoProtocol,
    baseUrl: '',
    apiKey: '',
    concurrency: 1,
    status: 'active' as EditableStatus,
    groupIds: [] as number[]
  })
  mappings.value = []
  pricing.value = [createXiaoVideoPricingRule()]
  hasExistingApiKey.value = false
  resetUpstreamModelPicker()
  setBaseline()
}

function syncForm(account: Account) {
  const credentials = (account.credentials ?? {}) as Record<string, unknown>
  Object.assign(form, {
    name: account.name,
    notes: account.notes ?? '',
    protocol: credentials.video_protocol === 'openai_sora' ? 'openai_sora' : 'native',
    baseUrl: typeof credentials.base_url === 'string' ? credentials.base_url : '',
    apiKey: '',
    concurrency: account.concurrency || 1,
    status: editableStatus(account),
    groupIds: [...(account.group_ids ?? [])]
  })
  mappings.value = readMappings(credentials.model_mapping)
  pricing.value = readXiaoVideoPricing(credentials.video_pricing)
  hasExistingApiKey.value = account.credentials_status?.has_api_key ?? typeof credentials.api_key === 'string'
  resetUpstreamModelPicker()
  setBaseline()
}

function confirmDiscard(): boolean {
  return !isDirty.value || window.confirm(t('admin.xiaoVideo.discardChanges'))
}

function beginCreate(skipConfirm = false) {
  if (!skipConfirm && !confirmDiscard()) return
  selectionRequest += 1
  selectedId.value = null
  selectedAccount.value = null
  detailsLoading.value = false
  resetForm()
}

async function selectAccount(id: number, skipConfirm = false, force = false) {
  if (!force && selectedId.value === id && selectedAccount.value) return
  if (!skipConfirm && !confirmDiscard()) return

  const previousId = selectedId.value
  const requestId = ++selectionRequest
  selectedId.value = id
  detailsLoading.value = true
  try {
    const account = await adminAPI.accounts.getById(id)
    if (requestId !== selectionRequest) return
    if (account.platform !== 'xiaoapi') {
      throw new Error(t('admin.xiaoVideo.invalidPlatform'))
    }
    selectedAccount.value = account
    syncForm(account)
  } catch (error) {
    if (requestId === selectionRequest) {
      selectedId.value = previousId
      appStore.showError(errorMessage(error, t('admin.xiaoVideo.loadFailed')))
    }
  } finally {
    if (requestId === selectionRequest) detailsLoading.value = false
  }
}

async function loadAllAccounts(): Promise<Account[]> {
  const first = await adminAPI.accounts.list(1, 100, { platform: 'xiaoapi' })
  if (first.pages <= 1) return first.items
  const remaining = await Promise.all(
    Array.from({ length: first.pages - 1 }, (_, index) =>
      adminAPI.accounts.list(index + 2, 100, { platform: 'xiaoapi' })
    )
  )
  return [first, ...remaining].flatMap((page) => page.items)
}

async function reloadAccounts(preferredId?: number) {
  const allAccounts = await loadAllAccounts()
  accounts.value = allAccounts
  const nextId = preferredId ?? selectedId.value
  if (nextId && allAccounts.some((account) => account.id === nextId)) {
    await selectAccount(nextId, true, true)
  } else if (allAccounts.length > 0) {
    await selectAccount(allAccounts[0].id, true, true)
  } else {
    beginCreate(true)
  }
}

function toggleGroup(groupId: number, event: Event) {
  const checked = (event.target as HTMLInputElement).checked
  if (checked && !form.groupIds.includes(groupId)) {
    form.groupIds = [...form.groupIds, groupId]
  } else if (!checked) {
    form.groupIds = form.groupIds.filter((id) => id !== groupId)
  }
}

function applyProtocolDefaults() {
  if (form.protocol === 'openai_sora' && !form.baseUrl.trim()) {
    form.baseUrl = AISTARTLAB_BASE_URL
  }
}

function resetUpstreamModelPicker() {
  fetchingModels.value = false
  modelPickerOpen.value = false
  upstreamModels.value = []
  upstreamModelSpecs.value = []
  upstreamPricingSource.value = 'none'
  upstreamPricingNote.value = ''
  selectedUpstreamModels.value = []
  modelSearch.value = ''
}

function savedConnectionMatchesForm(): boolean {
  if (!selectedAccount.value) return false
  const credentials = (selectedAccount.value.credentials ?? {}) as Record<string, unknown>
  const savedBaseURL = typeof credentials.base_url === 'string' ? credentials.base_url.trim() : ''
  const savedProtocol = credentials.video_protocol === 'openai_sora' ? 'openai_sora' : 'native'
  return savedBaseURL === form.baseUrl.trim() && savedProtocol === form.protocol
}

async function fetchUpstreamModels() {
  if (fetchingModels.value) return
  if (!validBaseUrl(form.baseUrl.trim())) {
    appStore.showError(t('admin.xiaoVideo.validation.baseUrlInvalid'))
    return
  }

  const apiKey = form.apiKey.trim()
  const canUseSavedCredentials = Boolean(selectedAccount.value && !apiKey && savedConnectionMatchesForm())
  if (!apiKey && !canUseSavedCredentials) {
    appStore.showError(t('admin.xiaoVideo.fetchModelsApiKeyRequired'))
    return
  }

  fetchingModels.value = true
  try {
    const result = canUseSavedCredentials && selectedAccount.value
      ? await adminAPI.accounts.syncUpstreamModels(selectedAccount.value.id)
      : await adminAPI.accounts.syncUpstreamModelsPreview({
          platform: 'xiaoapi',
          type: 'apikey',
          base_url: form.baseUrl.trim(),
          api_key: apiKey,
          video_protocol: form.protocol
        })
    const models = [...new Set(result.models.map((model) => model.trim()).filter(Boolean))]
      .sort((left, right) => left.localeCompare(right))
    if (models.length === 0) {
      appStore.showInfo(t('admin.xiaoVideo.noUpstreamModels'))
      return
    }
    upstreamModels.value = models
    upstreamModelSpecs.value = (result.model_specs ?? []).filter((spec) =>
      typeof spec.id === 'string' && models.includes(spec.id.trim())
    )
    upstreamPricingSource.value = result.pricing_source ?? 'none'
    upstreamPricingNote.value = result.pricing_note ?? ''
    selectedUpstreamModels.value = models.filter(canImportUpstreamModel)
    modelSearch.value = ''
    modelPickerOpen.value = true
  } catch (error) {
    appStore.showError(errorMessage(error, t('admin.xiaoVideo.fetchModelsFailed')))
  } finally {
    fetchingModels.value = false
  }
}

function toggleUpstreamModel(model: string, event: Event) {
  const checked = (event.target as HTMLInputElement).checked
  if (checked && !selectedUpstreamModels.value.includes(model)) {
    selectedUpstreamModels.value = [...selectedUpstreamModels.value, model]
  } else if (!checked) {
    selectedUpstreamModels.value = selectedUpstreamModels.value.filter((item) => item !== model)
  }
}

function selectVisibleModels() {
  const selected = new Set(selectedUpstreamModels.value)
  filteredUpstreamModels.value.forEach((model) => {
    if (canImportUpstreamModel(model)) selected.add(model)
  })
  selectedUpstreamModels.value = [...selected]
}

function nextPublicModelName(upstreamModel: string, usedNames: Set<string>): string {
  let candidate = upstreamModel
  let suffix = 2
  while (usedNames.has(candidate)) {
    candidate = `${upstreamModel}-${suffix}`
    suffix += 1
  }
  usedNames.add(candidate)
  return candidate
}

function internalCostPerSecond(spec: UpstreamModelSpec): number | null {
  const rawCost = Number(spec.upstream_cost)
  if (!Number.isFinite(rawCost) || rawCost < 0) return null

  const currency = spec.cost_currency?.trim().toUpperCase()
  const currencyFactor = currency === 'USD'
    ? 1
    : currency === 'CREDITS' && upstreamPricingSource.value === 'aistartlab_config' ? 0.01 : null
  if (currencyFactor === null) return null

  const unit = spec.cost_unit?.trim().toLowerCase()
  if (unit === 'second') return rawCost * currencyFactor
  const duration = Number(spec.default_duration)
  if (unit === 'request' && Number.isFinite(duration) && duration > 0) {
    return rawCost * currencyFactor / duration
  }
  return null
}

function formatPrice(value: number): string {
  return value.toFixed(6).replace(/\.?0+$/, '')
}

function formatSpecCost(spec: UpstreamModelSpec): string {
  if (!Number.isFinite(Number(spec.upstream_cost))) return ''
  const resolution = spec.resolution?.trim()
  const currency = spec.cost_currency?.trim().toUpperCase()
  const unit = spec.cost_unit?.trim().toLowerCase()
  const amount = formatPrice(Number(spec.upstream_cost))
  const value = currency === 'USD'
    ? `$${amount}`
    : currency === 'CREDITS'
      ? `${amount} ${t('admin.xiaoVideo.credits')}`
      : `${amount} ${currency || t('admin.xiaoVideo.unknownCurrency')}`
  const suffix = unit === 'second'
    ? t('admin.xiaoVideo.perSecondSuffix')
    : unit === 'request'
      ? t('admin.xiaoVideo.perRequestSuffix')
      : unit ? `/${unit}` : ''
  const internalCost = internalCostPerSecond(spec)
  const converted = internalCost === null
    ? ''
    : t('admin.xiaoVideo.internalCostEquivalent', { price: formatPrice(internalCost) })
  return [resolution, `${value}${suffix}`, converted].filter(Boolean).join(' · ')
}

function effectiveModelSpecs(model: string): UpstreamModelSpec[] {
  const specs = specsByModel.value.get(model) ?? []
  const detailed = specs.filter((spec) =>
    Boolean(spec.resolution?.trim()) || Number.isFinite(Number(spec.upstream_cost))
  )
  return detailed.length > 0 ? detailed : specs
}

function modelCostSummary(model: string): string {
  const summaries = effectiveModelSpecs(model).map(formatSpecCost).filter(Boolean)
  return summaries.length > 0 ? summaries.join(' / ') : t('admin.xiaoVideo.costUnavailable')
}

function suggestedPrice(spec: UpstreamModelSpec): number | null {
  const internalCost = internalCostPerSecond(spec)
  if (internalCost === null) return null
  const multiplier = Number(markupMultiplier.value)
  if (!Number.isFinite(multiplier) || multiplier < 0) return null
  return Number((internalCost * multiplier).toFixed(6))
}

function modelSuggestedSummary(model: string): string {
  const summaries = effectiveModelSpecs(model).flatMap((spec) => {
    const price = suggestedPrice(spec)
    if (price === null) return []
    return [[spec.resolution?.trim(), `$${formatPrice(price)}${t('admin.xiaoVideo.perSecondSuffix')}`]
      .filter(Boolean).join(' · ')]
  })
  return summaries.length > 0 ? summaries.join(' / ') : t('admin.xiaoVideo.manualPricingRequired')
}

function pricingSpecsForImport(model: string): UpstreamModelSpec[] {
  const specs = effectiveModelSpecs(model)
  const byResolution = new Map<string, UpstreamModelSpec>()
  for (const spec of specs) {
    const resolution = spec.resolution?.trim() || '720p'
    const current = byResolution.get(resolution)
    if (!current || (internalCostPerSecond(spec) !== null && internalCostPerSecond(current) === null)) {
      byResolution.set(resolution, spec)
    }
  }
  return [...byResolution.values()].filter((spec) => {
    const duration = Number(spec.default_duration)
    return suggestedPrice(spec) !== null && Number.isInteger(duration) && duration > 0
  })
}

function mappedPublicModels(upstreamModel: string): string[] {
  return mappings.value.flatMap((mapping) => {
    const publicModel = mapping.from.trim()
    return publicModel && mapping.to.trim() === upstreamModel ? [publicModel] : []
  })
}

function canImportUpstreamModel(upstreamModel: string): boolean {
  const publicModels = mappedPublicModels(upstreamModel)
  if (publicModels.length === 0) return true
  const specs = pricingSpecsForImport(upstreamModel)
  if (specs.length === 0) return false
  return publicModels.some((publicModel) => specs.some((spec) => {
    const resolution = spec.resolution?.trim() || '720p'
    return !pricing.value.some((rule) => rule.model.trim() === publicModel && rule.resolution.trim() === resolution)
  }))
}

function appendSuggestedPricing(publicModel: string, specs: UpstreamModelSpec[]): number {
  const explicitDefaultIndex = specs.findIndex((spec) => spec.default_resolution === true)
  let hasDefault = pricing.value.some((rule) => rule.model.trim() === publicModel && rule.default_resolution)
  let added = 0

  for (const [index, spec] of specs.entries()) {
    const resolution = spec.resolution?.trim() || '720p'
    const existing = pricing.value.find((rule) =>
      rule.model.trim() === publicModel && rule.resolution.trim() === resolution
    )
    const preferredDefault = explicitDefaultIndex >= 0 ? index === explicitDefaultIndex : index === 0
    if (existing) {
      if (!hasDefault && preferredDefault) {
        existing.default_resolution = true
        hasDefault = true
      }
      continue
    }

    const price = suggestedPrice(spec)
    if (price === null) continue
    let rule = pricing.value.find((item) => !item.model.trim())
    if (!rule) {
      rule = createXiaoVideoPricingRule()
      pricing.value.push(rule)
    }
    rule.model = publicModel
    rule.resolution = resolution
    rule.price_per_second = price
    rule.default_duration = Number(spec.default_duration)
    rule.default_resolution = !hasDefault && preferredDefault
    if (rule.default_resolution) hasDefault = true
    added += 1
  }
  return added
}

function importSelectedModels() {
  const selected = selectedUpstreamModels.value.filter(canImportUpstreamModel)
  if (selected.length === 0) {
    appStore.showInfo(t('admin.xiaoVideo.noModelsToImport'))
    return
  }

  const usedPublicNames = new Set(mappings.value.map((mapping) => mapping.from.trim()).filter(Boolean))
  let added = 0
  let withoutPricing = 0
  for (const upstreamModel of selected) {
    let publicModels = mappedPublicModels(upstreamModel)
    if (publicModels.length === 0) {
      const publicModel = nextPublicModelName(upstreamModel, usedPublicNames)
      mappings.value.push({ from: publicModel, to: upstreamModel })
      publicModels = [publicModel]
    }
    added += 1
    const specs = pricingSpecsForImport(upstreamModel)
    if (specs.length === 0) {
      withoutPricing += 1
      continue
    }
    publicModels.forEach((publicModel) => appendSuggestedPricing(publicModel, specs))
  }

  selectedUpstreamModels.value = []
  modelPickerOpen.value = false
  appStore.showSuccess(t('admin.xiaoVideo.modelsImported', { count: added }))
  if (withoutPricing > 0) {
    appStore.showInfo(t('admin.xiaoVideo.importedWithoutPricing', { count: withoutPricing }))
  }
}

function validBaseUrl(value: string): boolean {
  try {
    const url = new URL(value)
    return (url.protocol === 'http:' || url.protocol === 'https:') && !url.search && !url.hash
  } catch {
    return false
  }
}

function validateMappings(): string | null {
  const seen = new Set<string>()
  for (const mapping of mappings.value) {
    const from = mapping.from.trim()
    const to = mapping.to.trim()
    if (!from || !to) return 'required'
    if (!isValidWildcardPattern(from)) return 'wildcard'
    if (to.includes('*')) return 'targetWildcard'
    if (seen.has(from)) return 'duplicate'
    seen.add(from)
  }
  return null
}

function buildMappings(): Record<string, string> | null {
  const value: Record<string, string> = {}
  for (const mapping of mappings.value) {
    value[mapping.from.trim()] = mapping.to.trim()
  }
  return Object.keys(value).length > 0 ? value : null
}

function buildVideoCapabilityMap(): Record<string, Record<string, unknown>> {
  const result: Record<string, Record<string, unknown>> = {}
  for (const spec of upstreamModelSpecs.value) {
    const model = spec.id.trim()
    if (!model || !spec.durations?.length && !spec.aspect_ratios?.length && !spec.max_references) continue
    const current = result[model] ?? {}
    const durations = new Set<number>(Array.isArray(current.durations) ? current.durations as number[] : [])
    for (const duration of spec.durations ?? []) {
      if (Number.isInteger(duration) && duration > 0) durations.add(duration)
    }
    const aspectRatios = new Set<string>(Array.isArray(current.aspect_ratios) ? current.aspect_ratios as string[] : [])
    for (const ratio of spec.aspect_ratios ?? []) {
      if (typeof ratio === 'string' && ratio.trim()) aspectRatios.add(ratio.trim())
    }
    const maxReferences = {
      ...(current.max_references as Record<string, number> | undefined),
      ...(spec.max_references ?? {})
    }
    result[model] = {
      ...current,
      ...(durations.size ? { durations: [...durations].sort((a, b) => a - b) } : {}),
      ...(aspectRatios.size ? { aspect_ratios: [...aspectRatios] } : {}),
      ...(spec.default_aspect_ratio ? { default_aspect_ratio: spec.default_aspect_ratio } : {}),
      ...(spec.supports_audio ? { supports_audio: true } : {}),
      ...(spec.supports_guidances ? { supports_guidances: true } : {}),
      ...(spec.supports_start_frame ? { supports_start_frame: true } : {}),
      ...(spec.requires_start_frame ? { requires_start_frame: true } : {}),
      ...(spec.supports_end_frame ? { supports_end_frame: true } : {}),
      ...(Object.values(maxReferences).some((value) => Number(value) > 0) ? { max_references: maxReferences } : {})
    }
  }
  return result
}

function validateForm(): boolean {
  if (!form.name.trim()) {
    appStore.showError(t('admin.xiaoVideo.validation.nameRequired'))
    return false
  }
  if (!validBaseUrl(form.baseUrl.trim())) {
    appStore.showError(t('admin.xiaoVideo.validation.baseUrlInvalid'))
    return false
  }
  if (!Number.isInteger(form.concurrency) || form.concurrency < 1 || form.concurrency > 10000) {
    appStore.showError(t('admin.xiaoVideo.validation.concurrencyInvalid'))
    return false
  }
  if (!form.apiKey.trim() && !hasExistingApiKey.value) {
    appStore.showError(t('admin.xiaoVideo.validation.apiKeyRequired'))
    return false
  }
  const mappingError = validateMappings()
  if (mappingError) {
    appStore.showError(t(`admin.xiaoVideo.validation.mapping.${mappingError}`))
    return false
  }
  const pricingError = validateXiaoVideoPricing(pricing.value)
  if (pricingError) {
    appStore.showError(t(`admin.accounts.xiaoapi.validation.${pricingError}`))
    return false
  }
  return true
}

function buildCredentials(): Record<string, unknown> {
  const currentCredentials = (selectedAccount.value?.credentials ?? {}) as Record<string, unknown>
  const credentials: Record<string, unknown> = {
    ...currentCredentials,
    base_url: form.baseUrl.trim(),
    video_protocol: form.protocol,
    video_pricing: normalizeXiaoVideoPricing(pricing.value)
  }
  const videoCapabilities = buildVideoCapabilityMap()
  if (Object.keys(videoCapabilities).length > 0) credentials.video_capabilities = videoCapabilities
  if (form.apiKey.trim()) credentials.api_key = form.apiKey.trim()
  const mapping = buildMappings()
  if (mapping) credentials.model_mapping = mapping
  else delete credentials.model_mapping
  return credentials
}

async function save() {
  if (!validateForm()) return
  saving.value = true
  try {
    const credentials = buildCredentials()
    let saved: Account
    if (selectedAccount.value) {
      saved = await adminAPI.accounts.update(selectedAccount.value.id, {
        name: form.name.trim(),
        notes: form.notes.trim() || null,
        type: 'apikey',
        credentials,
        concurrency: form.concurrency,
        status: form.status,
        group_ids: [...form.groupIds],
        upstream_billing_probe_enabled: false,
        upstream_billing_rate_sync_enabled: false
      })
      appStore.showSuccess(t('admin.xiaoVideo.updated'))
    } else {
      saved = await adminAPI.accounts.create({
        name: form.name.trim(),
        notes: form.notes.trim() || null,
        platform: 'xiaoapi',
        type: 'apikey',
        credentials,
        concurrency: form.concurrency,
        group_ids: [...form.groupIds],
        upstream_billing_probe_enabled: false
      })
      if (form.status !== 'active') {
        saved = await adminAPI.accounts.update(saved.id, { status: form.status })
      }
      appStore.showSuccess(t('admin.xiaoVideo.created'))
    }
    form.apiKey = ''
    setBaseline()
    await reloadAccounts(saved.id)
  } catch (error) {
    appStore.showError(errorMessage(error, t('admin.xiaoVideo.saveFailed')))
  } finally {
    saving.value = false
  }
}

async function testConnection() {
  if (!selectedAccount.value) return
  testing.value = true
  try {
    const result = await adminAPI.accounts.testAccount(selectedAccount.value.id)
    if (result.success) appStore.showSuccess(result.message || t('admin.xiaoVideo.testSucceeded'))
    else appStore.showError(result.message || t('admin.xiaoVideo.testFailed'))
  } catch (error) {
    appStore.showError(errorMessage(error, t('admin.xiaoVideo.testFailed')))
  } finally {
    testing.value = false
  }
}

async function removeAccount() {
  if (!selectedAccount.value) return
  if (!window.confirm(t('admin.xiaoVideo.deleteConfirm', { name: selectedAccount.value.name }))) return
  deleting.value = true
  try {
    await adminAPI.accounts.delete(selectedAccount.value.id)
    appStore.showSuccess(t('admin.xiaoVideo.deleted'))
    selectedId.value = null
    selectedAccount.value = null
    resetForm()
    await reloadAccounts()
  } catch (error) {
    appStore.showError(errorMessage(error, t('admin.xiaoVideo.deleteFailed')))
  } finally {
    deleting.value = false
  }
}

setBaseline()

onMounted(async () => {
  loading.value = true
  try {
    const [, xiaoGroups] = await Promise.all([
      reloadAccounts(),
      adminAPI.groups.getByPlatform('xiaoapi')
    ])
    groups.value = xiaoGroups
  } catch (error) {
    appStore.showError(errorMessage(error, t('admin.xiaoVideo.loadFailed')))
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.xiao-upstream-item {
  display: flex;
  width: 100%;
  min-height: 3.5rem;
  align-items: center;
  gap: 0.75rem;
  border: 1px solid transparent;
  border-radius: 0.5rem;
  padding: 0.625rem 0.75rem;
  color: rgb(55 65 81);
  transition: border-color 0.15s ease, background-color 0.15s ease;
}

.xiao-upstream-item:hover {
  border-color: rgb(226 232 240);
  background: rgb(255 255 255 / 0.72);
}

.xiao-upstream-item-active {
  border-color: rgb(253 186 116 / 0.7);
  background: rgb(255 255 255);
  box-shadow: 0 6px 18px rgb(15 23 42 / 0.05);
}

:global(.dark) .xiao-upstream-item {
  color: rgb(209 213 219);
}

:global(.dark) .xiao-upstream-item:hover {
  border-color: rgb(55 65 81);
  background: rgb(31 41 55 / 0.6);
}

:global(.dark) .xiao-upstream-item-active {
  border-color: rgb(180 83 9 / 0.6);
  background: rgb(31 41 55);
}
</style>
