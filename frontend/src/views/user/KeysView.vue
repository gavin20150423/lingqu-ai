<template>
  <UserWorkspaceLayout>
    <div class="lingqu-keys">
      <section class="lingqu-keys__hero">
        <div class="lingqu-keys__copy">
          <span>我的 Key</span>
          <h1>创建一个 Key，马上接入 AI</h1>
          <p>复制 Key 和 Base URL 即可使用，额度与限制可以之后再设置。</p>
        </div>

        <div class="lingqu-keys__hero-tools">
          <button @click="showCreateModal = true" class="lingqu-keys__primary lingqu-keys__primary--hero" data-tour="keys-create-btn">
            <Icon name="plus" size="md" />
            创建 Key
          </button>

          <router-link to="/usage" class="lingqu-keys__usage-entry">
            <Icon name="chart" size="md" />
            使用记录
          </router-link>

          <button
            type="button"
            class="lingqu-keys__endpoint"
            :title="baseUrlCopied ? 'Base URL 已复制' : '复制 Base URL'"
            @click="copyBaseUrl"
          >
            <span>
              <Icon name="terminal" size="sm" />
              Base URL
            </span>
            <code>{{ apiBaseUrl }}</code>
            <Icon :name="baseUrlCopied ? 'check' : 'copy'" size="sm" />
          </button>
        </div>
      </section>

      <section class="lingqu-keys__stats" aria-label="Key 概览">
        <article>
          <Icon name="key" size="md" />
          <small>全部 Key</small>
          <strong>{{ keySummary.total }}</strong>
        </article>
        <article>
          <Icon name="checkCircle" size="md" />
          <small>可用 Key</small>
          <strong>{{ keySummary.active }}</strong>
        </article>
        <article>
          <Icon name="dollar" size="md" />
          <small>今日消耗</small>
          <strong>${{ keySummary.todayCost }}</strong>
        </article>
        <article>
          <Icon name="chart" size="md" />
          <small>累计消耗</small>
          <strong>${{ keySummary.totalCost }}</strong>
        </article>
      </section>

      <section class="lingqu-keys__toolbar">
        <SearchInput
          v-model="filterSearch"
          :placeholder="t('keys.searchPlaceholder')"
          class="lingqu-keys__search"
          @search="onFilterChange"
        />
        <Select
          :model-value="filterGroupId"
          class="lingqu-keys__select"
          :options="groupFilterOptions"
          @update:model-value="onGroupFilterChange"
        />
        <Select
          :model-value="filterStatus"
          class="lingqu-keys__select"
          :options="statusFilterOptions"
          @update:model-value="onStatusFilterChange"
        />
        <EndpointPopover
          v-if="publicSettings?.api_base_url || (publicSettings?.custom_endpoints?.length ?? 0) > 0"
          :api-base-url="publicSettings?.api_base_url || ''"
          :custom-endpoints="publicSettings?.custom_endpoints || []"
        />
        <button @click="loadApiKeys" :disabled="loading" class="lingqu-keys__secondary lingqu-keys__refresh">
          <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          刷新
        </button>
      </section>

      <section class="lingqu-key-list" aria-label="我的 API Keys">
        <template v-if="loading">
          <article v-for="item in 4" :key="item" class="lingqu-key-card lingqu-key-card--loading">
            <div></div>
            <div></div>
            <div></div>
          </article>
        </template>

        <article v-else-if="apiKeys.length === 0" class="lingqu-keys__empty">
          <div class="lingqu-keys__empty-icon">
            <Icon name="key" size="xl" />
          </div>
          <h2>{{ t('keys.noKeysYet') }}</h2>
          <p>{{ t('keys.createFirstKey') }}</p>
          <button type="button" class="lingqu-keys__primary" @click="showCreateModal = true">
            <Icon name="plus" size="md" />
            {{ t('keys.createKey') }}
          </button>
        </article>

        <article v-for="key in apiKeys" v-else :key="key.id" class="lingqu-key-card">
          <div class="lingqu-key-card__top">
            <div>
              <div class="lingqu-key-card__title-row">
                <h2>{{ key.name }}</h2>
                <span :class="['lingqu-key-card__status', `lingqu-key-card__status--${key.status}`]">
                  {{ t('keys.status.' + key.status) }}
                </span>
              </div>
              <div class="lingqu-key-card__meta">
                <button
                  :ref="(el) => setGroupButtonRef(key.id, el)"
                  type="button"
                  class="lingqu-key-card__group group/dropdown"
                  :title="t('keys.clickToChangeGroup')"
                  @click="openGroupSelector(key)"
                >
                  <GroupBadge
                    v-if="key.group"
                    :name="key.group.name"
                    :platform="key.group.platform"
                    :subscription-type="key.group.subscription_type"
                    :rate-multiplier="key.group.rate_multiplier"
                    :user-rate-multiplier="userGroupRates[key.group.id]"
                    :peak-rate-enabled="key.group.peak_rate_enabled"
                    :peak-start="key.group.peak_start"
                    :peak-end="key.group.peak_end"
                    :peak-rate-multiplier="key.group.peak_rate_multiplier"
                  />
                  <span v-else>{{ t('keys.noGroup') }}</span>
                  <Icon name="chevronDown" size="xs" />
                </button>
                <span>{{ t('keys.created') }} {{ formatDateTime(key.created_at) }}</span>
              </div>
            </div>
            <div class="lingqu-key-card__tools">
              <button
                type="button"
                class="lingqu-key-card__visibility"
                :title="revealedKeyIds.has(key.id) ? t('keys.hideFullKey') : t('keys.showFullKey')"
                :aria-label="revealedKeyIds.has(key.id) ? t('keys.hideFullKey') : t('keys.showFullKey')"
                :aria-pressed="revealedKeyIds.has(key.id)"
                @click="toggleKeyVisibility(key.id)"
              >
                <Icon :name="revealedKeyIds.has(key.id) ? 'eyeOff' : 'eye'" size="sm" />
              </button>
              <button
                type="button"
                class="lingqu-key-card__copy"
                :title="copiedKeyId === key.id ? t('keys.copied') : t('keys.copyToClipboard')"
                @click="copyToClipboard(key.key, key.id)"
              >
                <Icon :name="copiedKeyId === key.id ? 'check' : 'copy'" size="sm" />
              </button>
            </div>
          </div>

          <div class="lingqu-key-card__secret">
            <small>API Key</small>
            <code>{{ revealedKeyIds.has(key.id) ? key.key : maskApiKey(key.key) }}</code>
          </div>

          <div class="lingqu-key-card__metrics">
            <div>
              <small>{{ t('keys.today') }}</small>
              <strong>${{ (usageStats[key.id]?.today_actual_cost ?? 0).toFixed(4) }}</strong>
            </div>
            <div>
              <small>{{ t('keys.total') }}</small>
              <strong>${{ (usageStats[key.id]?.total_actual_cost ?? 0).toFixed(4) }}</strong>
            </div>
            <div>
              <small>{{ t('keys.quota') }}</small>
              <strong>{{ formatQuota(key) }}</strong>
            </div>
            <div>
              <small>{{ t('keys.lastUsedAt') }}</small>
              <strong>{{ key.last_used_at ? formatDateTime(key.last_used_at) : '-' }}</strong>
            </div>
          </div>

          <div class="lingqu-key-card__limits">
            <span>
              <Icon name="calendar" size="sm" />
              {{ key.expires_at ? formatDateTime(key.expires_at) : t('keys.noExpiration') }}
            </span>
            <span>
              <Icon name="bolt" size="sm" />
              {{ formatRateLimit(key) }}
            </span>
            <span>
              <Icon name="users" size="sm" />
              {{ t('keys.currentConcurrency') }}: {{ key.current_concurrency ?? 0 }}
            </span>
            <span v-if="key.last_used_ip">
              <Icon name="globe" size="sm" />
              {{ t('keys.lastUsedIP') }}: {{ key.last_used_ip }}
            </span>
          </div>

          <div class="lingqu-key-card__actions">
            <router-link :to="{ path: '/usage', query: { api_key_id: key.id } }">
              <Icon name="chart" size="sm" />
              用量明细
            </router-link>
            <button type="button" @click="openUseKeyModal(key)">
              <Icon name="terminal" size="sm" />
              {{ t('keys.useKey') }}
            </button>
            <button
              v-if="!publicSettings?.hide_ccs_import_button"
              type="button"
              @click="importToCcswitch(key)"
            >
              <Icon name="upload" size="sm" />
              {{ t('keys.importToCcSwitch') }}
            </button>
            <button type="button" @click="toggleKeyStatus(key)">
              <Icon :name="key.status === 'active' ? 'ban' : 'checkCircle'" size="sm" />
              {{ key.status === 'active' ? t('keys.disable') : t('keys.enable') }}
            </button>
            <button type="button" @click="editKey(key)">
              <Icon name="edit" size="sm" />
              {{ t('common.edit') }}
            </button>
            <button type="button" class="lingqu-key-card__danger" @click="confirmDelete(key)">
              <Icon name="trash" size="sm" />
              {{ t('common.delete') }}
            </button>
          </div>
        </article>
      </section>

      <Pagination
        v-if="pagination.total > 0"
        class="lingqu-keys__pagination"
        :page="pagination.page"
        :total="pagination.total"
        :page-size="pagination.page_size"
        @update:page="handlePageChange"
        @update:pageSize="handlePageSizeChange"
      />
    </div>

    <!-- Create/Edit Modal -->
    <BaseDialog
      :show="showCreateModal || showEditModal"
      :title="showEditModal ? t('keys.editKey') : t('keys.createKey')"
      width="normal"
      @close="closeModals"
    >
      <form id="key-form" @submit.prevent="handleSubmit" class="space-y-5">
        <div>
          <label class="input-label">{{ t('keys.nameLabel') }}</label>
          <input
            v-model="formData.name"
            type="text"
            required
            class="input"
            :placeholder="t('keys.namePlaceholder')"
            data-tour="key-form-name"
          />
        </div>

        <div>
          <label class="input-label">{{ t('keys.groupLabel') }}</label>
          <Select
            v-model="formData.group_id"
            :options="groupOptions"
            :placeholder="t('keys.selectGroup')"
            :searchable="true"
            :search-placeholder="t('keys.searchGroup')"
            data-tour="key-form-group"
          >
            <template #selected="{ option }">
              <GroupBadge
                v-if="option"
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :rate-multiplier="(option as unknown as GroupOption).rate"
                :user-rate-multiplier="(option as unknown as GroupOption).userRate"
              />
              <span v-else class="text-gray-400">{{ t('keys.selectGroup') }}</span>
            </template>
            <template #option="{ option, selected }">
              <GroupOptionItem
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :rate-multiplier="(option as unknown as GroupOption).rate"
                :user-rate-multiplier="(option as unknown as GroupOption).userRate"
                :description="(option as unknown as GroupOption).description"
                :selected="selected"
              />
            </template>
          </Select>
        </div>

        <div
          v-if="!showEditModal"
          class="rounded-2xl border border-dashed border-gray-200 bg-white/70 p-4 dark:border-dark-600 dark:bg-dark-700/40"
        >
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div class="min-w-0">
              <p class="text-sm font-semibold text-gray-900 dark:text-white">默认配置已经可以直接使用</p>
              <p class="mt-1 text-xs leading-relaxed text-gray-500 dark:text-gray-400">
                创建后马上复制 Key 接入；自定义密钥、IP、额度和有效期可以之后再设置。
              </p>
            </div>
            <button
              type="button"
              class="btn btn-secondary shrink-0 px-3 py-2 text-sm"
              :aria-expanded="showAdvancedCreateOptions"
              @click="showAdvancedCreateOptions = !showAdvancedCreateOptions"
            >
              <Icon :name="showAdvancedCreateOptions ? 'chevronUp' : 'chevronDown'" size="sm" class="mr-1" />
              {{ showAdvancedCreateOptions ? '收起高级设置' : '高级设置' }}
            </button>
          </div>
        </div>

        <!-- Custom Key Section (only for create) -->
        <div v-if="!showEditModal && showAdvancedCreateOptions" class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.customKeyLabel') }}</label>
            <button
              type="button"
              @click="formData.use_custom_key = !formData.use_custom_key"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.use_custom_key ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  formData.use_custom_key ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>
          <div v-if="formData.use_custom_key">
            <input
              v-model="formData.custom_key"
              type="text"
              class="input font-mono"
              :placeholder="t('keys.customKeyPlaceholder')"
              :class="{ 'border-red-500 dark:border-red-500': customKeyError }"
            />
            <p v-if="customKeyError" class="mt-1 text-sm text-red-500">{{ customKeyError }}</p>
            <p v-else class="input-hint">{{ t('keys.customKeyHint') }}</p>
          </div>
        </div>

        <div v-if="showEditModal">
          <label class="input-label">{{ t('keys.statusLabel') }}</label>
          <Select
            v-model="formData.status"
            :options="statusOptions"
            :placeholder="t('keys.selectStatus')"
          />
        </div>

        <!-- IP Restriction Section -->
        <div v-if="showEditModal || showAdvancedCreateOptions" class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.ipRestriction') }}</label>
            <button
              type="button"
              @click="formData.enable_ip_restriction = !formData.enable_ip_restriction"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.enable_ip_restriction ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  formData.enable_ip_restriction ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>

          <div v-if="formData.enable_ip_restriction" class="space-y-4 pt-2">
            <div>
              <label class="input-label">{{ t('keys.ipWhitelist') }}</label>
              <textarea
                v-model="formData.ip_whitelist"
                rows="3"
                class="input font-mono text-sm"
                :placeholder="t('keys.ipWhitelistPlaceholder')"
              />
              <p class="input-hint">{{ t('keys.ipWhitelistHint') }}</p>
            </div>

            <div>
              <label class="input-label">{{ t('keys.ipBlacklist') }}</label>
              <textarea
                v-model="formData.ip_blacklist"
                rows="3"
                class="input font-mono text-sm"
                :placeholder="t('keys.ipBlacklistPlaceholder')"
              />
              <p class="input-hint">{{ t('keys.ipBlacklistHint') }}</p>
            </div>
          </div>
        </div>

        <!-- Quota Limit Section -->
        <div v-if="showEditModal || showAdvancedCreateOptions" class="space-y-3">
          <label class="input-label">{{ t('keys.quotaLimit') }}</label>
          <!-- Switch commented out - always show input, 0 = unlimited
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.quotaLimit') }}</label>
            <button
              type="button"
              @click="formData.enable_quota = !formData.enable_quota"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.enable_quota ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  formData.enable_quota ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>
          -->

          <div class="space-y-4">
            <div>
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">$</span>
                <input
                  v-model.number="formData.quota"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-7"
                  :placeholder="t('keys.quotaAmountPlaceholder')"
                />
              </div>
              <p class="input-hint">{{ t('keys.quotaAmountHint') }}</p>
            </div>

            <!-- Quota used display (only in edit mode) -->
            <div v-if="showEditModal && selectedKey && selectedKey.quota > 0">
              <label class="input-label">{{ t('keys.quotaUsed') }}</label>
              <div class="flex items-center gap-2">
                <div class="flex-1 rounded-lg bg-gray-100 px-3 py-2 dark:bg-dark-700">
                  <span class="font-medium text-gray-900 dark:text-white">
                    ${{ selectedKey.quota_used?.toFixed(4) || '0.0000' }}
                  </span>
                  <span class="mx-2 text-gray-400">/</span>
                  <span class="text-gray-500 dark:text-gray-400">
                    ${{ selectedKey.quota?.toFixed(2) || '0.00' }}
                  </span>
                </div>
                <button
                  type="button"
                  @click="confirmResetQuota"
                  class="btn btn-secondary text-sm"
                  :title="t('keys.resetQuotaUsed')"
                >
                  {{ t('keys.reset') }}
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Rate Limit Section -->
        <div v-if="showEditModal || showAdvancedCreateOptions" class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.rateLimitSection') }}</label>
            <button
              type="button"
              @click="formData.enable_rate_limit = !formData.enable_rate_limit"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.enable_rate_limit ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  formData.enable_rate_limit ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>

          <div v-if="formData.enable_rate_limit" class="space-y-4 pt-2">
            <p class="input-hint -mt-2">{{ t('keys.rateLimitHint') }}</p>
            <!-- 5-Hour Limit -->
            <div>
              <label class="input-label">{{ t('keys.rateLimit5h') }}</label>
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">$</span>
                <input
                  v-model.number="formData.rate_limit_5h"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-7"
                  :placeholder="'0'"
                />
              </div>
              <!-- Usage info (edit mode only) -->
              <div v-if="showEditModal && selectedKey && selectedKey.rate_limit_5h > 0" class="mt-2">
                <div class="flex items-center gap-2">
                  <div class="flex-1 rounded-lg bg-gray-100 px-3 py-2 dark:bg-dark-700 text-sm">
                    <span :class="[
                      'font-medium',
                      selectedKey.usage_5h >= selectedKey.rate_limit_5h ? 'text-red-500' :
                      selectedKey.usage_5h >= selectedKey.rate_limit_5h * 0.8 ? 'text-yellow-500' :
                      'text-gray-900 dark:text-white'
                    ]">
                      ${{ selectedKey.usage_5h?.toFixed(4) || '0.0000' }}
                    </span>
                    <span class="mx-2 text-gray-400">/</span>
                    <span class="text-gray-500 dark:text-gray-400">
                      ${{ selectedKey.rate_limit_5h?.toFixed(2) || '0.00' }}
                    </span>
                  </div>
                </div>
                <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      selectedKey.usage_5h >= selectedKey.rate_limit_5h ? 'bg-red-500' :
                      selectedKey.usage_5h >= selectedKey.rate_limit_5h * 0.8 ? 'bg-yellow-500' :
                      'bg-green-500'
                    ]"
                    :style="{ width: Math.min((selectedKey.usage_5h / selectedKey.rate_limit_5h) * 100, 100) + '%' }"
                  />
                </div>
              </div>
            </div>

            <!-- Daily Limit -->
            <div>
              <label class="input-label">{{ t('keys.rateLimit1d') }}</label>
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">$</span>
                <input
                  v-model.number="formData.rate_limit_1d"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-7"
                  :placeholder="'0'"
                />
              </div>
              <!-- Usage info (edit mode only) -->
              <div v-if="showEditModal && selectedKey && selectedKey.rate_limit_1d > 0" class="mt-2">
                <div class="flex items-center gap-2">
                  <div class="flex-1 rounded-lg bg-gray-100 px-3 py-2 dark:bg-dark-700 text-sm">
                    <span :class="[
                      'font-medium',
                      selectedKey.usage_1d >= selectedKey.rate_limit_1d ? 'text-red-500' :
                      selectedKey.usage_1d >= selectedKey.rate_limit_1d * 0.8 ? 'text-yellow-500' :
                      'text-gray-900 dark:text-white'
                    ]">
                      ${{ selectedKey.usage_1d?.toFixed(4) || '0.0000' }}
                    </span>
                    <span class="mx-2 text-gray-400">/</span>
                    <span class="text-gray-500 dark:text-gray-400">
                      ${{ selectedKey.rate_limit_1d?.toFixed(2) || '0.00' }}
                    </span>
                  </div>
                </div>
                <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      selectedKey.usage_1d >= selectedKey.rate_limit_1d ? 'bg-red-500' :
                      selectedKey.usage_1d >= selectedKey.rate_limit_1d * 0.8 ? 'bg-yellow-500' :
                      'bg-green-500'
                    ]"
                    :style="{ width: Math.min((selectedKey.usage_1d / selectedKey.rate_limit_1d) * 100, 100) + '%' }"
                  />
                </div>
              </div>
            </div>

            <!-- 7-Day Limit -->
            <div>
              <label class="input-label">{{ t('keys.rateLimit7d') }}</label>
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">$</span>
                <input
                  v-model.number="formData.rate_limit_7d"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-7"
                  :placeholder="'0'"
                />
              </div>
              <!-- Usage info (edit mode only) -->
              <div v-if="showEditModal && selectedKey && selectedKey.rate_limit_7d > 0" class="mt-2">
                <div class="flex items-center gap-2">
                  <div class="flex-1 rounded-lg bg-gray-100 px-3 py-2 dark:bg-dark-700 text-sm">
                    <span :class="[
                      'font-medium',
                      selectedKey.usage_7d >= selectedKey.rate_limit_7d ? 'text-red-500' :
                      selectedKey.usage_7d >= selectedKey.rate_limit_7d * 0.8 ? 'text-yellow-500' :
                      'text-gray-900 dark:text-white'
                    ]">
                      ${{ selectedKey.usage_7d?.toFixed(4) || '0.0000' }}
                    </span>
                    <span class="mx-2 text-gray-400">/</span>
                    <span class="text-gray-500 dark:text-gray-400">
                      ${{ selectedKey.rate_limit_7d?.toFixed(2) || '0.00' }}
                    </span>
                  </div>
                </div>
                <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      selectedKey.usage_7d >= selectedKey.rate_limit_7d ? 'bg-red-500' :
                      selectedKey.usage_7d >= selectedKey.rate_limit_7d * 0.8 ? 'bg-yellow-500' :
                      'bg-green-500'
                    ]"
                    :style="{ width: Math.min((selectedKey.usage_7d / selectedKey.rate_limit_7d) * 100, 100) + '%' }"
                  />
                </div>
              </div>
            </div>

            <!-- Reset Rate Limit button (edit mode only) -->
            <div v-if="showEditModal && selectedKey && (selectedKey.rate_limit_5h > 0 || selectedKey.rate_limit_1d > 0 || selectedKey.rate_limit_7d > 0)">
              <button
                type="button"
                @click="confirmResetRateLimit"
                class="btn btn-secondary text-sm"
              >
                {{ t('keys.resetRateLimitUsage') }}
              </button>
            </div>
          </div>
        </div>

        <!-- Expiration Section -->
        <div v-if="showEditModal || showAdvancedCreateOptions" class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.expiration') }}</label>
            <button
              type="button"
              @click="formData.enable_expiration = !formData.enable_expiration"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.enable_expiration ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  formData.enable_expiration ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>

          <div v-if="formData.enable_expiration" class="space-y-4 pt-2">
            <!-- Quick select buttons (for both create and edit mode) -->
            <div class="flex flex-wrap gap-2">
              <button
                v-for="days in ['7', '30', '90']"
                :key="days"
                type="button"
                @click="setExpirationDays(parseInt(days))"
                :class="[
                  'rounded-lg px-3 py-1.5 text-sm transition-colors',
                  formData.expiration_preset === days
                    ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-400 dark:hover:bg-dark-600'
                ]"
              >
                {{ showEditModal ? t('keys.extendDays', { days }) : t('keys.expiresInDays', { days }) }}
              </button>
              <button
                type="button"
                @click="formData.expiration_preset = 'custom'"
                :class="[
                  'rounded-lg px-3 py-1.5 text-sm transition-colors',
                  formData.expiration_preset === 'custom'
                    ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-400 dark:hover:bg-dark-600'
                ]"
              >
                {{ t('keys.customDate') }}
              </button>
            </div>

            <!-- Date picker (always show for precise adjustment) -->
            <div>
              <label class="input-label">{{ t('keys.expirationDate') }}</label>
              <input
                v-model="formData.expiration_date"
                type="datetime-local"
                class="input"
              />
              <p class="input-hint">{{ t('keys.expirationDateHint') }}</p>
            </div>

            <!-- Current expiration display (only in edit mode) -->
            <div v-if="showEditModal && selectedKey?.expires_at" class="text-sm">
              <span class="text-gray-500 dark:text-gray-400">{{ t('keys.currentExpiration') }}: </span>
              <span class="font-medium text-gray-900 dark:text-white">
                {{ formatDateTime(selectedKey.expires_at) }}
              </span>
            </div>
          </div>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button @click="closeModals" type="button" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button
            form="key-form"
            type="submit"
            :disabled="submitting"
            class="btn btn-primary"
            data-tour="key-form-submit"
          >
            <svg
              v-if="submitting"
              class="-ml-1 mr-2 h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            {{
              submitting
                ? t('keys.saving')
                : showEditModal
                  ? t('common.update')
                  : t('common.create')
            }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Delete Confirmation Dialog -->
    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('keys.deleteKey')"
      :message="t('keys.deleteConfirmMessage', { name: selectedKey?.name })"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="handleDelete"
      @cancel="showDeleteDialog = false"
    />

    <!-- Reset Quota Confirmation Dialog -->
    <ConfirmDialog
      :show="showResetQuotaDialog"
      :title="t('keys.resetQuotaTitle')"
      :message="t('keys.resetQuotaConfirmMessage', { name: selectedKey?.name, used: selectedKey?.quota_used?.toFixed(4) })"
      :confirm-text="t('keys.reset')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="resetQuotaUsed"
      @cancel="showResetQuotaDialog = false"
    />

    <!-- Reset Rate Limit Confirmation Dialog -->
    <ConfirmDialog
      :show="showResetRateLimitDialog"
      :title="t('keys.resetRateLimitTitle')"
      :message="t('keys.resetRateLimitConfirmMessage', { name: selectedKey?.name })"
      :confirm-text="t('keys.reset')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="resetRateLimitUsage"
      @cancel="showResetRateLimitDialog = false"
    />

    <!-- Use Key Modal -->
    <UseKeyModal
      :show="showUseKeyModal"
      :api-key="selectedKey?.key || ''"
      :base-url="publicSettings?.api_base_url || ''"
      :platform="selectedKey?.group?.platform || null"
      :allow-messages-dispatch="selectedKey?.group?.allow_messages_dispatch || false"
      @close="closeUseKeyModal"
    />

    <!-- CCS Client Selection Dialog for Antigravity -->
    <BaseDialog
      :show="showCcsClientSelect"
      :title="t('keys.ccsClientSelect.title')"
      width="narrow"
      @close="closeCcsClientSelect"
    >
      <div class="space-y-4">
        <p class="text-sm text-gray-600 dark:text-gray-400">
          {{ t('keys.ccsClientSelect.description') }}
	        </p>
	        <div class="grid grid-cols-2 gap-3">
	          <button
	            @click="handleCcsClientSelect('claude')"
	            class="flex flex-col items-center gap-2 p-4 rounded-xl border-2 border-gray-200 dark:border-dark-600 hover:border-primary-500 dark:hover:border-primary-500 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-all"
	          >
	            <Icon name="terminal" size="xl" class="text-gray-600 dark:text-gray-400" />
	            <span class="font-medium text-gray-900 dark:text-white">{{
	              t('keys.ccsClientSelect.claudeCode')
	            }}</span>
	            <span class="text-xs text-gray-500 dark:text-gray-400">{{
	              t('keys.ccsClientSelect.claudeCodeDesc')
	            }}</span>
	          </button>
	          <button
	            @click="handleCcsClientSelect('gemini')"
	            class="flex flex-col items-center gap-2 p-4 rounded-xl border-2 border-gray-200 dark:border-dark-600 hover:border-primary-500 dark:hover:border-primary-500 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-all"
	          >
	            <Icon name="sparkles" size="xl" class="text-gray-600 dark:text-gray-400" />
	            <span class="font-medium text-gray-900 dark:text-white">{{
	              t('keys.ccsClientSelect.geminiCli')
	            }}</span>
	            <span class="text-xs text-gray-500 dark:text-gray-400">{{
	              t('keys.ccsClientSelect.geminiCliDesc')
	            }}</span>
	          </button>
	        </div>
	      </div>
      <template #footer>
        <div class="flex justify-end">
          <button @click="closeCcsClientSelect" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Group Selector Dropdown (Teleported to body to avoid overflow clipping) -->
    <Teleport to="body">
      <div
        v-if="groupSelectorKeyId !== null && dropdownPosition"
        ref="dropdownRef"
        class="animate-in fade-in slide-in-from-top-2 fixed z-[100000020] w-max max-w-[calc(100vw-16px)] overflow-hidden rounded-xl bg-white shadow-lg ring-1 ring-black/5 duration-200 sm:min-w-[380px] dark:bg-dark-800 dark:ring-white/10"
        style="pointer-events: auto !important;"
        :style="{
          top: dropdownPosition.top !== undefined ? dropdownPosition.top + 'px' : undefined,
          bottom: dropdownPosition.bottom !== undefined ? dropdownPosition.bottom + 'px' : undefined,
          left: dropdownPosition.left + 'px'
        }"
      >
        <!-- Search box -->
        <div class="border-b border-gray-100 p-2 dark:border-dark-700">
          <div class="relative">
            <svg class="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <input
              v-model="groupSearchQuery"
              type="text"
              class="w-full rounded-lg border border-gray-200 bg-gray-50 py-1.5 pl-8 pr-3 text-sm text-gray-900 placeholder-gray-400 outline-none focus:border-primary-300 focus:ring-1 focus:ring-primary-300 dark:border-dark-600 dark:bg-dark-700 dark:text-white dark:placeholder-gray-500 dark:focus:border-primary-600 dark:focus:ring-primary-600"
              :placeholder="t('keys.searchGroup')"
              @click.stop
            />
          </div>
        </div>
        <!-- Group list -->
        <div class="max-h-80 overflow-y-auto p-1.5">
          <button
            v-for="option in filteredGroupOptions"
            :key="option.value ?? 'null'"
            @click="changeGroup(selectedKeyForGroup!, option.value)"
            :class="[
              'flex w-full items-center justify-between rounded-lg px-3 py-2.5 text-sm transition-colors',
              'border-b border-gray-100 last:border-0 dark:border-dark-700',
              selectedKeyForGroup?.group_id === option.value ||
              (!selectedKeyForGroup?.group_id && option.value === null)
                ? 'bg-primary-50 dark:bg-primary-900/20'
                : 'hover:bg-gray-100 dark:hover:bg-dark-700'
            ]"
            :title="option.description || undefined"
          >
            <GroupOptionItem
              :name="option.label"
              :platform="option.platform"
              :subscription-type="option.subscriptionType"
              :rate-multiplier="option.rate"
              :user-rate-multiplier="option.userRate"
              :description="option.description"
              :selected="
                selectedKeyForGroup?.group_id === option.value ||
                (!selectedKeyForGroup?.group_id && option.value === null)
              "
            />
          </button>
          <!-- Empty state when search has no results -->
          <div v-if="filteredGroupOptions.length === 0" class="py-4 text-center text-sm text-gray-400 dark:text-gray-500">
            {{ t('keys.noGroupFound') }}
          </div>
        </div>
      </div>
    </Teleport>
  </UserWorkspaceLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, type ComponentPublicInstance } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useOnboardingStore } from '@/stores/onboarding'
import { useClipboard } from '@/composables/useClipboard'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
import { keysAPI, authAPI, usageAPI, userGroupsAPI } from '@/api'
import UserWorkspaceLayout from '@/components/layout/UserWorkspaceLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Select from '@/components/common/Select.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Icon from '@/components/icons/Icon.vue'
import UseKeyModal from '@/components/keys/UseKeyModal.vue'
import EndpointPopover from '@/components/keys/EndpointPopover.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import GroupOptionItem from '@/components/common/GroupOptionItem.vue'
import type { ApiKey, Group, PublicSettings, SubscriptionType, GroupPlatform } from '@/types'
import type { BatchApiKeyUsageStats } from '@/api/usage'
import { formatDateTime } from '@/utils/format'
import { maskApiKey } from '@/utils/maskApiKey'
import {
  buildCcSwitchImportDeeplink,
  type CcSwitchClientType
} from '@/utils/ccswitchImport'

// Helper to format date for datetime-local input
const formatDateTimeLocal = (isoDate: string): string => {
  const date = new Date(isoDate)
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

interface GroupOption {
  value: number
  label: string
  description: string | null
  rate: number
  userRate: number | null
  subscriptionType: SubscriptionType
  platform: GroupPlatform
}

const appStore = useAppStore()
const onboardingStore = useOnboardingStore()
const { copyToClipboard: clipboardCopy } = useClipboard()

const apiKeys = ref<ApiKey[]>([])
const groups = ref<Group[]>([])
const loading = ref(false)
const submitting = ref(false)
const usageStats = ref<Record<string, BatchApiKeyUsageStats>>({})
const userGroupRates = ref<Record<number, number>>({})

const pagination = ref({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})
// Filter state
const filterSearch = ref('')
const filterStatus = ref('')
const filterGroupId = ref<string | number>('')

const showCreateModal = ref(false)
const showEditModal = ref(false)
const showDeleteDialog = ref(false)
const showResetQuotaDialog = ref(false)
const showResetRateLimitDialog = ref(false)
const showUseKeyModal = ref(false)
const showCcsClientSelect = ref(false)
const showAdvancedCreateOptions = ref(false)
const pendingCcsRow = ref<ApiKey | null>(null)
const selectedKey = ref<ApiKey | null>(null)
const copiedKeyId = ref<number | null>(null)
const revealedKeyIds = ref<Set<number>>(new Set())
const baseUrlCopied = ref(false)
const groupSelectorKeyId = ref<number | null>(null)
const publicSettings = ref<PublicSettings | null>(null)
const dropdownRef = ref<HTMLElement | null>(null)
const dropdownPosition = ref<{ top?: number; bottom?: number; left: number } | null>(null)
const groupButtonRefs = ref<Map<number, HTMLElement>>(new Map())
let abortController: AbortController | null = null

const apiBaseUrl = computed(() => {
  const configured = publicSettings.value?.api_base_url?.trim()
  return configured || `${window.location.origin}/v1`
})

const keySummary = computed(() => {
  const stats = Object.values(usageStats.value)
  return {
    total: pagination.value.total || apiKeys.value.length,
    active: apiKeys.value.filter((key) => key.status === 'active').length,
    todayCost: stats.reduce((sum, item) => sum + (item.today_actual_cost ?? 0), 0).toFixed(4),
    totalCost: stats.reduce((sum, item) => sum + (item.total_actual_cost ?? 0), 0).toFixed(4)
  }
})

function openCreateModalFromQuery(): void {
  if (route.query.create === '1' || route.query.create === 'true') {
    showCreateModal.value = true
    const nextQuery = { ...route.query }
    delete nextQuery.create
    router.replace({ path: route.path, query: nextQuery }).catch(() => undefined)
  }
}

// Get the currently selected key for group change
const selectedKeyForGroup = computed(() => {
  if (groupSelectorKeyId.value === null) return null
  return apiKeys.value.find((k) => k.id === groupSelectorKeyId.value) || null
})

watch(showCreateModal, (open) => {
  if (open) {
    showAdvancedCreateOptions.value = false
  }
})

const setGroupButtonRef = (keyId: number, el: Element | ComponentPublicInstance | null) => {
  if (el instanceof HTMLElement) {
    groupButtonRefs.value.set(keyId, el)
  } else {
    groupButtonRefs.value.delete(keyId)
  }
}

const formData = ref({
  name: '',
  group_id: null as number | null,
  status: 'active' as 'active' | 'inactive',
  use_custom_key: false,
  custom_key: '',
  enable_ip_restriction: false,
  ip_whitelist: '',
  ip_blacklist: '',
  // Quota settings (empty = unlimited)
  enable_quota: false,
  quota: null as number | null,
  // Rate limit settings
  enable_rate_limit: false,
  rate_limit_5h: null as number | null,
  rate_limit_1d: null as number | null,
  rate_limit_7d: null as number | null,
  enable_expiration: false,
  expiration_preset: '30' as '7' | '30' | '90' | 'custom',
  expiration_date: ''
})

// 自定义Key验证
const customKeyError = computed(() => {
  if (!formData.value.use_custom_key || !formData.value.custom_key) {
    return ''
  }
  const key = formData.value.custom_key
  if (key.length < 16) {
    return t('keys.customKeyTooShort')
  }
  // 检查字符：只允许字母、数字、下划线、连字符
  if (!/^[a-zA-Z0-9_-]+$/.test(key)) {
    return t('keys.customKeyInvalidChars')
  }
  return ''
})

const statusOptions = computed(() => [
  { value: 'active', label: t('common.active') },
  { value: 'inactive', label: t('common.inactive') }
])

// Filter dropdown options
const groupFilterOptions = computed(() => [
  { value: '', label: t('keys.allGroups') },
  { value: 0, label: t('keys.noGroup') },
  ...groups.value.map((g) => ({ value: g.id, label: g.name }))
])

const statusFilterOptions = computed(() => [
  { value: '', label: t('keys.allStatus') },
  { value: 'active', label: t('keys.status.active') },
  { value: 'inactive', label: t('keys.status.inactive') },
  { value: 'quota_exhausted', label: t('keys.status.quota_exhausted') },
  { value: 'expired', label: t('keys.status.expired') }
])

const onFilterChange = () => {
  pagination.value.page = 1
  loadApiKeys()
}

const onGroupFilterChange = (value: string | number | boolean | null) => {
  filterGroupId.value = value as string | number
  onFilterChange()
}

const onStatusFilterChange = (value: string | number | boolean | null) => {
  filterStatus.value = value as string
  onFilterChange()
}

// Convert groups to Select options format with rate multiplier and subscription type
const groupOptions = computed(() =>
  groups.value.map((group) => ({
    value: group.id,
    label: group.name,
    description: group.description,
    rate: group.rate_multiplier,
    userRate: userGroupRates.value[group.id] ?? null,
    subscriptionType: group.subscription_type,
    platform: group.platform
  }))
)

// Group dropdown search
const groupSearchQuery = ref('')
const filteredGroupOptions = computed(() => {
  const query = groupSearchQuery.value.trim().toLowerCase()
  if (!query) return groupOptions.value
  return groupOptions.value.filter((opt) => {
    return opt.label.toLowerCase().includes(query) ||
      (opt.description && opt.description.toLowerCase().includes(query))
  })
})

const copyToClipboard = async (text: string, keyId: number) => {
  const success = await clipboardCopy(text, t('keys.copied'))
  if (success) {
    copiedKeyId.value = keyId
    setTimeout(() => {
      copiedKeyId.value = null
    }, 800)
  }
}

const toggleKeyVisibility = (keyId: number) => {
  const nextRevealedKeyIds = new Set(revealedKeyIds.value)
  if (nextRevealedKeyIds.has(keyId)) {
    nextRevealedKeyIds.delete(keyId)
  } else {
    nextRevealedKeyIds.add(keyId)
  }
  revealedKeyIds.value = nextRevealedKeyIds
}

const copyBaseUrl = async () => {
  const success = await clipboardCopy(apiBaseUrl.value, 'Base URL 已复制')
  if (success) {
    baseUrlCopied.value = true
    setTimeout(() => {
      baseUrlCopied.value = false
    }, 900)
  }
}

const isAbortError = (error: unknown) => {
  if (!error || typeof error !== 'object') return false
  const { name, code } = error as { name?: string; code?: string }
  return name === 'AbortError' || code === 'ERR_CANCELED'
}

const loadApiKeys = async () => {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  const { signal } = controller
  loading.value = true
  try {
    // Build filters
    const filters: {
      search?: string
      status?: string
      group_id?: number | string
      sort_by?: string
      sort_order?: 'asc' | 'desc'
    } = {}
    if (filterSearch.value) filters.search = filterSearch.value
    if (filterStatus.value) filters.status = filterStatus.value
    if (filterGroupId.value !== '') filters.group_id = filterGroupId.value
    filters.sort_by = 'created_at'
    filters.sort_order = 'desc'

    const response = await keysAPI.list(pagination.value.page, pagination.value.page_size, filters, {
      signal
    })
    if (signal.aborted) return
    apiKeys.value = response.items
    pagination.value.total = response.total
    pagination.value.pages = response.pages

    // Load usage stats for all API keys in the list
    if (response.items.length > 0) {
      const keyIds = response.items.map((k) => k.id)
      try {
        const usageResponse = await usageAPI.getDashboardApiKeysUsage(keyIds, { signal })
        if (signal.aborted) return
        usageStats.value = usageResponse.stats
      } catch (e) {
        if (!isAbortError(e)) {
          console.error('Failed to load usage stats:', e)
        }
      }
    }
  } catch (error) {
    if (isAbortError(error)) {
      return
    }
    appStore.showError(t('keys.failedToLoad'))
  } finally {
    if (abortController === controller) {
      loading.value = false
    }
  }
}

const loadGroups = async () => {
  try {
    groups.value = await userGroupsAPI.getAvailable()
  } catch (error) {
    console.error('Failed to load groups:', error)
  }
}

const loadUserGroupRates = async () => {
  try {
    userGroupRates.value = await userGroupsAPI.getUserGroupRates()
  } catch (error) {
    console.error('Failed to load user group rates:', error)
  }
}

const loadPublicSettings = async () => {
  try {
    publicSettings.value = await authAPI.getPublicSettings()
  } catch (error) {
    console.error('Failed to load public settings:', error)
  }
}

const openUseKeyModal = (key: ApiKey) => {
  selectedKey.value = key
  showUseKeyModal.value = true
}

const closeUseKeyModal = () => {
  showUseKeyModal.value = false
  selectedKey.value = null
}

const handlePageChange = (page: number) => {
  pagination.value.page = page
  loadApiKeys()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.value.page_size = pageSize
  pagination.value.page = 1
  loadApiKeys()
}

const editKey = (key: ApiKey) => {
  selectedKey.value = key
  const hasIPRestriction = (key.ip_whitelist?.length > 0) || (key.ip_blacklist?.length > 0)
  const hasExpiration = !!key.expires_at
  formData.value = {
    name: key.name,
    group_id: key.group_id,
    status: key.status === 'quota_exhausted' || key.status === 'expired' ? 'inactive' : key.status,
    use_custom_key: false,
    custom_key: '',
    enable_ip_restriction: hasIPRestriction,
    ip_whitelist: (key.ip_whitelist || []).join('\n'),
    ip_blacklist: (key.ip_blacklist || []).join('\n'),
    enable_quota: key.quota > 0,
    quota: key.quota > 0 ? key.quota : null,
    enable_rate_limit: (key.rate_limit_5h > 0) || (key.rate_limit_1d > 0) || (key.rate_limit_7d > 0),
    rate_limit_5h: key.rate_limit_5h || null,
    rate_limit_1d: key.rate_limit_1d || null,
    rate_limit_7d: key.rate_limit_7d || null,
    enable_expiration: hasExpiration,
    expiration_preset: 'custom',
    expiration_date: key.expires_at ? formatDateTimeLocal(key.expires_at) : ''
  }
  showEditModal.value = true
}

const toggleKeyStatus = async (key: ApiKey) => {
  const newStatus = key.status === 'active' ? 'inactive' : 'active'
  try {
    await keysAPI.toggleStatus(key.id, newStatus)
    appStore.showSuccess(
      newStatus === 'active' ? t('keys.keyEnabledSuccess') : t('keys.keyDisabledSuccess')
    )
    loadApiKeys()
  } catch (error) {
    appStore.showError(t('keys.failedToUpdateStatus'))
  }
}

const openGroupSelector = (key: ApiKey) => {
  if (groupSelectorKeyId.value === key.id) {
    groupSelectorKeyId.value = null
    dropdownPosition.value = null
  } else {
    const buttonEl = groupButtonRefs.value.get(key.id)
    if (buttonEl) {
      const rect = buttonEl.getBoundingClientRect()
      const dropdownEstHeight = 400 // estimated max dropdown height
      const dropdownEstWidth = Math.min(380, window.innerWidth - 16)
      const spaceBelow = window.innerHeight - rect.bottom
      const spaceAbove = rect.top
      // 夹取 left，避免窄屏下浮层超出视口右缘
      const left = Math.max(8, Math.min(rect.left, window.innerWidth - dropdownEstWidth - 8))

      if (spaceBelow < dropdownEstHeight && spaceAbove > spaceBelow) {
        // Not enough space below, pop upward
        dropdownPosition.value = {
          bottom: window.innerHeight - rect.top + 4,
          left
        }
      } else {
        // Default: pop downward
        dropdownPosition.value = {
          top: rect.bottom + 4,
          left
        }
      }
    }
    groupSelectorKeyId.value = key.id
    groupSearchQuery.value = ''
  }
}

const changeGroup = async (key: ApiKey, newGroupId: number | null) => {
  groupSelectorKeyId.value = null
  dropdownPosition.value = null
  if (key.group_id === newGroupId) return

  try {
    await keysAPI.update(key.id, { group_id: newGroupId })
    appStore.showSuccess(t('keys.groupChangedSuccess'))
    loadApiKeys()
  } catch (error) {
    appStore.showError(t('keys.failedToChangeGroup'))
  }
}

const closeGroupSelector = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  // Check if click is inside the dropdown or the trigger button
  if (!target.closest('.group\\/dropdown') && !dropdownRef.value?.contains(target)) {
    groupSelectorKeyId.value = null
    dropdownPosition.value = null
  }
}

const confirmDelete = (key: ApiKey) => {
  selectedKey.value = key
  showDeleteDialog.value = true
}

const handleSubmit = async () => {
  // Validate group_id is required
  if (formData.value.group_id === null) {
    appStore.showError(t('keys.groupRequired'))
    return
  }

  // Validate custom key if enabled
  if (!showEditModal.value && formData.value.use_custom_key) {
    if (!formData.value.custom_key) {
      appStore.showError(t('keys.customKeyRequired'))
      return
    }
    if (customKeyError.value) {
      appStore.showError(customKeyError.value)
      return
    }
  }

  // Parse IP lists only if IP restriction is enabled
  const parseIPList = (text: string): string[] =>
    text.split('\n').map(ip => ip.trim()).filter(ip => ip.length > 0)
  const ipWhitelist = formData.value.enable_ip_restriction ? parseIPList(formData.value.ip_whitelist) : []
  const ipBlacklist = formData.value.enable_ip_restriction ? parseIPList(formData.value.ip_blacklist) : []

  // Calculate quota value (null/empty/0 = unlimited, stored as 0)
  const quota = formData.value.quota && formData.value.quota > 0 ? formData.value.quota : 0

  // Calculate expiration
  let expiresInDays: number | undefined
  let expiresAt: string | null | undefined
  if (formData.value.enable_expiration && formData.value.expiration_date) {
    if (!showEditModal.value) {
      // Create mode: calculate days from date
      const expDate = new Date(formData.value.expiration_date)
      const now = new Date()
      const diffDays = Math.ceil((expDate.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
      expiresInDays = diffDays > 0 ? diffDays : 1
    } else {
      // Edit mode: use custom date directly
      expiresAt = new Date(formData.value.expiration_date).toISOString()
    }
  } else if (showEditModal.value) {
    // Edit mode: if expiration disabled or date cleared, send empty string to clear
    expiresAt = ''
  }

  // Calculate rate limit values (send 0 when toggle is off)
  const rateLimitData = formData.value.enable_rate_limit ? {
    rate_limit_5h: formData.value.rate_limit_5h && formData.value.rate_limit_5h > 0 ? formData.value.rate_limit_5h : 0,
    rate_limit_1d: formData.value.rate_limit_1d && formData.value.rate_limit_1d > 0 ? formData.value.rate_limit_1d : 0,
    rate_limit_7d: formData.value.rate_limit_7d && formData.value.rate_limit_7d > 0 ? formData.value.rate_limit_7d : 0,
  } : { rate_limit_5h: 0, rate_limit_1d: 0, rate_limit_7d: 0 }

  submitting.value = true
  try {
    if (showEditModal.value && selectedKey.value) {
      await keysAPI.update(selectedKey.value.id, {
        name: formData.value.name,
        group_id: formData.value.group_id,
        status: formData.value.status,
        ip_whitelist: ipWhitelist,
        ip_blacklist: ipBlacklist,
        quota: quota,
        expires_at: expiresAt,
        rate_limit_5h: rateLimitData.rate_limit_5h,
        rate_limit_1d: rateLimitData.rate_limit_1d,
        rate_limit_7d: rateLimitData.rate_limit_7d,
      })
      appStore.showSuccess(t('keys.keyUpdatedSuccess'))
    } else {
      const customKey = formData.value.use_custom_key ? formData.value.custom_key : undefined
      await keysAPI.create(
        formData.value.name,
        formData.value.group_id,
        customKey,
        ipWhitelist,
        ipBlacklist,
        quota,
        expiresInDays,
        rateLimitData
      )
      appStore.showSuccess(t('keys.keyCreatedSuccess'))
      // Only advance tour if active, on submit step, and creation succeeded
      if (onboardingStore.isCurrentStep('[data-tour="key-form-submit"]')) {
        onboardingStore.nextStep(500)
      }
    }
    closeModals()
    loadApiKeys()
  } catch (error: any) {
    const errorMsg = error.response?.data?.detail || t('keys.failedToSave')
    appStore.showError(errorMsg)
    // Don't advance tour on error
  } finally {
    submitting.value = false
  }
}

/**
 * 处理删除 API Key 的操作
 * 优化：错误处理改进，优先显示后端返回的具体错误消息（如权限不足等），
 * 若后端未返回消息则显示默认的国际化文本
 */
const handleDelete = async () => {
  if (!selectedKey.value) return

  try {
    await keysAPI.delete(selectedKey.value.id)
    appStore.showSuccess(t('keys.keyDeletedSuccess'))
    showDeleteDialog.value = false
    loadApiKeys()
  } catch (error: any) {
    // 优先使用后端返回的错误消息，提供更具体的错误信息给用户
    const errorMsg = error?.message || t('keys.failedToDelete')
    appStore.showError(errorMsg)
  }
}

const closeModals = () => {
  showCreateModal.value = false
  showEditModal.value = false
  showAdvancedCreateOptions.value = false
  selectedKey.value = null
  formData.value = {
    name: '',
    group_id: null,
    status: 'active',
    use_custom_key: false,
    custom_key: '',
    enable_ip_restriction: false,
    ip_whitelist: '',
    ip_blacklist: '',
    enable_quota: false,
    quota: null,
    enable_rate_limit: false,
    rate_limit_5h: null,
    rate_limit_1d: null,
    rate_limit_7d: null,
    enable_expiration: false,
    expiration_preset: '30',
    expiration_date: ''
  }
}

// Show reset quota confirmation dialog
const confirmResetQuota = () => {
  showResetQuotaDialog.value = true
}

// Set expiration date based on quick select days
const setExpirationDays = (days: number) => {
  formData.value.expiration_preset = days.toString() as '7' | '30' | '90'
  const expDate = new Date()
  expDate.setDate(expDate.getDate() + days)
  formData.value.expiration_date = formatDateTimeLocal(expDate.toISOString())
}

// Reset quota used for an API key
const resetQuotaUsed = async () => {
  if (!selectedKey.value) return
  showResetQuotaDialog.value = false
  try {
    await keysAPI.update(selectedKey.value.id, { reset_quota: true })
    appStore.showSuccess(t('keys.quotaResetSuccess'))
    // Update local state
    if (selectedKey.value) {
      selectedKey.value.quota_used = 0
    }
  } catch (error: any) {
    const errorMsg = error.response?.data?.detail || t('keys.failedToResetQuota')
    appStore.showError(errorMsg)
  }
}

// Show reset rate limit confirmation dialog (from edit modal)
const confirmResetRateLimit = () => {
  showResetRateLimitDialog.value = true
}

// Reset rate limit usage for an API key
const resetRateLimitUsage = async () => {
  if (!selectedKey.value) return
  showResetRateLimitDialog.value = false
  try {
    await keysAPI.update(selectedKey.value.id, { reset_rate_limit_usage: true })
    appStore.showSuccess(t('keys.rateLimitResetSuccess'))
    // Refresh key data
    await loadApiKeys()
    // Update the editing key with fresh data
    const refreshedKey = apiKeys.value.find(k => k.id === selectedKey.value!.id)
    if (refreshedKey) {
      selectedKey.value = refreshedKey
    }
  } catch (error: any) {
    const errorMsg = error.response?.data?.detail || t('keys.failedToResetRateLimit')
    appStore.showError(errorMsg)
  }
}

const importToCcswitch = (row: ApiKey) => {
  const platform = row.group?.platform || 'anthropic'

  // For antigravity platform, show client selection dialog
  if (platform === 'antigravity') {
    pendingCcsRow.value = row
    showCcsClientSelect.value = true
    return
  }

  // For other platforms, execute directly
  executeCcsImport(row, platform === 'gemini' ? 'gemini' : 'claude')
}

const executeCcsImport = (row: ApiKey, clientType: CcSwitchClientType) => {
  const baseUrl = publicSettings.value?.api_base_url || window.location.origin
  const platform = row.group?.platform || 'anthropic'

  const usageScript = `({
    request: {
      url: "{{baseUrl}}/v1/usage",
      method: "GET",
      headers: { "Authorization": "Bearer {{apiKey}}" }
    },
    extractor: function(response) {
      const remaining = response?.remaining ?? response?.quota?.remaining ?? response?.balance;
      const unit = response?.unit ?? response?.quota?.unit ?? "USD";
      return {
        isValid: response?.is_active ?? response?.isValid ?? true,
        remaining,
        unit
      };
    }
  })`
  const providerName = (publicSettings.value?.site_name || 'sub2api').trim() || 'sub2api'
  const deeplink = buildCcSwitchImportDeeplink({
    baseUrl,
    platform,
    clientType,
    providerName,
    apiKey: row.key,
    usageScript
  })

  try {
    window.open(deeplink, '_self')

    // Check if the protocol handler worked by detecting if we're still focused
    setTimeout(() => {
      if (document.hasFocus()) {
        // Still focused means the protocol handler likely failed
        appStore.showError(t('keys.ccSwitchNotInstalled'))
      }
    }, 100)
  } catch (error) {
    appStore.showError(t('keys.ccSwitchNotInstalled'))
  }
}

const handleCcsClientSelect = (clientType: CcSwitchClientType) => {
  if (pendingCcsRow.value) {
    executeCcsImport(pendingCcsRow.value, clientType)
  }
  showCcsClientSelect.value = false
  pendingCcsRow.value = null
}

const closeCcsClientSelect = () => {
  showCcsClientSelect.value = false
  pendingCcsRow.value = null
}

function formatQuota(key: ApiKey): string {
  if (!key.quota || key.quota <= 0) return t('dashboard.platformQuota.noLimit')
  return `$${(key.quota_used || 0).toFixed(2)} / $${key.quota.toFixed(2)}`
}

function formatRateLimit(key: ApiKey): string {
  const limits = [
    key.rate_limit_5h > 0 ? `5h $${key.rate_limit_5h.toFixed(2)}` : '',
    key.rate_limit_1d > 0 ? `1d $${key.rate_limit_1d.toFixed(2)}` : '',
    key.rate_limit_7d > 0 ? `7d $${key.rate_limit_7d.toFixed(2)}` : ''
  ].filter(Boolean)
  return limits.length > 0 ? limits.join(' / ') : t('dashboard.platformQuota.noLimit')
}

onMounted(() => {
  loadApiKeys()
  loadGroups()
  loadUserGroupRates()
  loadPublicSettings()
  openCreateModalFromQuery()
  document.addEventListener('click', closeGroupSelector)
})

watch(
  () => route.query.create,
  () => {
    openCreateModalFromQuery()
  }
)

onUnmounted(() => {
  document.removeEventListener('click', closeGroupSelector)
})
</script>

<style scoped>
.lingqu-keys {
  display: grid;
  gap: 0.72rem;
}

.lingqu-keys__hero,
.lingqu-keys__stats article,
.lingqu-keys__toolbar,
.lingqu-key-card,
.lingqu-keys__empty {
  position: relative;
  overflow: hidden;
  border: 1px solid rgba(33, 31, 28, 0.1);
  background: rgba(255, 255, 255, 0.86);
  box-shadow: 0 10px 26px rgba(29, 42, 42, 0.06);
}

.lingqu-keys__hero {
  min-height: 4.6rem;
  display: grid;
  grid-template-columns: minmax(17rem, 0.9fr) minmax(24rem, 1fr);
  align-items: center;
  gap: clamp(0.75rem, 2vw, 1.2rem);
  border-radius: 20px;
  background:
    linear-gradient(90deg, rgba(255, 248, 223, 0.9), rgba(255, 255, 255, 0.78) 46%, rgba(231, 250, 255, 0.78)),
    rgba(255, 255, 255, 0.9);
  padding: 0.72rem 0.85rem;
  animation: keyPageRise 520ms ease both;
}

.lingqu-keys__hero::before,
.lingqu-key-card::before,
.lingqu-keys__empty::before {
  display: none;
}

.lingqu-keys__hero > *,
.lingqu-key-card > *,
.lingqu-keys__empty > * {
  position: relative;
  z-index: 1;
}

.lingqu-keys__copy > span {
  display: inline-flex;
  width: fit-content;
  border: 1px solid rgba(33, 31, 28, 0.12);
  border-radius: 999px;
  background: #fff8df;
  padding: 0.14rem 0.48rem;
  font-size: 0.62rem;
  font-weight: 950;
}

.lingqu-keys__copy h1 {
  max-width: none;
  margin-top: 0.22rem;
  font-family: theme('fontFamily.display');
  font-size: clamp(1.08rem, 1.8vw, 1.42rem);
  font-weight: 950;
  letter-spacing: 0;
  line-height: 1.08;
  color: #211f1c;
}

.lingqu-keys__copy p {
  max-width: 36rem;
  margin-top: 0.1rem;
  color: rgba(33, 31, 28, 0.58);
  font-size: 0.73rem;
  font-weight: 800;
  line-height: 1.35;
}

.lingqu-keys__hero-tools {
  min-width: 0;
  display: grid;
  grid-template-columns: minmax(8rem, 12rem) minmax(7rem, 10rem) minmax(12rem, 1fr);
  align-items: center;
  gap: 0.46rem;
}

.lingqu-keys__primary,
.lingqu-keys__secondary,
.lingqu-keys__endpoint,
.lingqu-keys__usage-entry,
.lingqu-key-card__actions a,
.lingqu-key-card__actions button,
.lingqu-key-card__visibility,
.lingqu-key-card__copy {
  display: inline-flex;
  min-height: 2.12rem;
  align-items: center;
  justify-content: center;
  gap: 0.32rem;
  border: 1px solid rgba(33, 31, 28, 0.13);
  border-radius: 12px;
  color: #211f1c;
  font-size: 0.78rem;
  font-weight: 950;
  box-shadow: none;
  transition: transform 150ms ease, box-shadow 150ms ease, filter 150ms ease;
}

.lingqu-keys__primary {
  background: linear-gradient(135deg, #f8df86, #f4b4bd);
  padding: 0 0.78rem;
}

.lingqu-keys__primary--hero {
  min-height: 2.28rem;
  border-radius: 13px;
  font-size: 0.82rem;
}

.lingqu-keys__secondary,
.lingqu-keys__usage-entry,
.lingqu-key-card__actions a,
.lingqu-key-card__actions button,
.lingqu-key-card__visibility,
.lingqu-key-card__copy {
  background: rgba(255, 255, 255, 0.72);
  padding: 0 0.66rem;
}

.lingqu-keys__usage-entry,
.lingqu-key-card__actions a {
  text-decoration: none;
}

.lingqu-keys__endpoint {
  min-width: 0;
  max-width: none;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  border-color: rgba(38, 51, 49, 0.92);
  border-radius: 13px;
  background: #263331;
  color: #fffdf5;
  padding: 0 0.62rem;
}

.lingqu-keys__endpoint span {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  color: rgba(255, 253, 245, 0.72);
  font-size: 0.66rem;
}

.lingqu-keys__endpoint code {
  overflow: hidden;
  color: #f8e08a;
  font-size: 0.66rem;
  font-weight: 950;
  text-align: left;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.lingqu-keys__primary:hover,
.lingqu-keys__secondary:hover:not(:disabled),
.lingqu-keys__endpoint:hover,
.lingqu-keys__usage-entry:hover,
.lingqu-key-card__actions a:hover,
.lingqu-key-card__actions button:hover,
.lingqu-key-card__visibility:hover,
.lingqu-key-card__copy:hover {
  transform: translateY(-2px);
  box-shadow: 0 12px 24px rgba(29, 42, 42, 0.1);
  filter: none;
}

.lingqu-keys__stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.48rem;
}

.lingqu-keys__stats article {
  min-height: 3.15rem;
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 0.45rem;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.64);
  padding: 0.48rem 0.62rem;
  transition: transform 160ms ease, box-shadow 160ms ease;
}

.lingqu-keys__stats article:hover {
  transform: translateY(-2px);
  box-shadow: 0 12px 24px rgba(29, 42, 42, 0.08);
}

.lingqu-keys__stats svg {
  color: #08a9d6;
  width: 1.05rem;
  height: 1.05rem;
}

.lingqu-keys__stats small,
.lingqu-key-card small {
  color: rgba(33, 31, 28, 0.54);
  font-size: 0.68rem;
  font-weight: 950;
}

.lingqu-keys__stats strong {
  overflow-wrap: anywhere;
  font-size: 0.92rem;
  font-weight: 950;
  color: #211f1c;
  text-align: right;
}

.lingqu-keys__toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.46rem;
  border-radius: 14px;
  padding: 0.48rem;
}

.lingqu-keys__refresh {
  margin-left: auto;
}

.lingqu-keys__search {
  min-width: min(100%, 18rem);
  flex: 1 1 18rem;
}

.lingqu-keys__select {
  width: 10.5rem;
}

.lingqu-key-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.8rem;
}

.lingqu-key-card,
.lingqu-keys__empty {
  border-radius: 18px;
  background:
    radial-gradient(circle at 100% 0%, rgba(72, 185, 200, 0.08), transparent 34%),
    rgba(255, 255, 255, 0.86);
  padding: 0.86rem;
  animation: keyPageRise 460ms ease both;
}

.lingqu-key-card {
  display: grid;
  gap: 0.85rem;
}

.lingqu-key-card__top {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
}

.lingqu-key-card__title-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
}

.lingqu-key-card h2,
.lingqu-keys__empty h2 {
  font-family: theme('fontFamily.display');
  font-size: 1.28rem;
  font-weight: 950;
  line-height: 1.1;
  color: #211f1c;
}

.lingqu-key-card__status {
  border: 1px solid rgba(33, 31, 28, 0.12);
  border-radius: 999px;
  background: #fff;
  padding: 0.16rem 0.52rem;
  font-size: 0.7rem;
  font-weight: 950;
}

.lingqu-key-card__status--active {
  background: #edf9f3;
}

.lingqu-key-card__status--inactive {
  background: #e8e8e8;
}

.lingqu-key-card__status--quota_exhausted,
.lingqu-key-card__status--expired {
  background: #ffd7d7;
}

.lingqu-key-card__meta,
.lingqu-key-card__limits {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
  margin-top: 0.42rem;
  color: rgba(33, 31, 28, 0.56);
  font-size: 0.78rem;
  font-weight: 850;
}

.lingqu-key-card__group,
.lingqu-key-card__limits span {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  border: 1px solid rgba(33, 31, 28, 0.12);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.74);
  padding: 0.3rem 0.55rem;
  color: #211f1c;
  font-weight: 900;
}

.lingqu-key-card__tools {
  width: fit-content;
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 0.38rem;
}

.lingqu-key-card__visibility,
.lingqu-key-card__copy {
  width: 2.34rem;
  min-height: 2.34rem;
  flex: 0 0 auto;
  padding: 0;
}

.lingqu-key-card__secret {
  display: grid;
  gap: 0.3rem;
  border: 1px solid rgba(38, 51, 49, 0.92);
  border-radius: 14px;
  background: #263331;
  padding: 0.68rem;
  box-shadow: inset 0 0 0 2px rgba(255, 255, 255, 0.05);
}

.lingqu-key-card__secret small {
  color: rgba(255, 253, 245, 0.54);
}

.lingqu-key-card__secret code {
  overflow-wrap: anywhere;
  color: #f8e08a;
  font-size: 0.78rem;
  font-weight: 900;
}

.lingqu-key-card__metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.42rem;
}

.lingqu-key-card__metrics > div {
  display: grid;
  gap: 0.18rem;
  border: 1px solid rgba(33, 31, 28, 0.1);
  border-radius: 13px;
  background: rgba(255, 255, 255, 0.58);
  padding: 0.52rem;
}

.lingqu-key-card__metrics strong {
  overflow-wrap: anywhere;
  color: #211f1c;
  font-size: 0.92rem;
  font-weight: 950;
}

.lingqu-key-card__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.42rem;
}

.lingqu-key-card__actions a,
.lingqu-key-card__actions button {
  min-height: 2.14rem;
  border-radius: 13px;
  font-size: 0.74rem;
  padding: 0 0.58rem;
}

.lingqu-key-card__actions .lingqu-key-card__danger {
  background: #ffe0e4;
}

.lingqu-keys__empty {
  min-height: 18rem;
  grid-column: 1 / -1;
  display: grid;
  place-items: center;
  text-align: center;
  padding: 2rem;
}

.lingqu-keys__empty p {
  max-width: 28rem;
  color: rgba(33, 31, 28, 0.62);
  font-weight: 800;
  line-height: 1.7;
}

.lingqu-keys__empty-icon {
  width: 4.2rem;
  height: 4.2rem;
  display: grid;
  place-items: center;
  border: 1px solid rgba(33, 31, 28, 0.12);
  border-radius: 18px;
  background: #fff7d0;
  box-shadow: 0 10px 24px rgba(29, 42, 42, 0.07);
}

.lingqu-key-card--loading {
  min-height: 16rem;
}

.lingqu-key-card--loading div {
  height: 2rem;
  border-radius: 999px;
  background: rgba(33, 31, 28, 0.08);
  animation: keyPulse 1.2s ease-in-out infinite;
}

.lingqu-key-card--loading div:nth-child(2) {
  width: 72%;
}

.lingqu-key-card--loading div:nth-child(3) {
  width: 52%;
}

@media (max-width: 1080px) {
  .lingqu-keys__hero,
  .lingqu-key-list {
    grid-template-columns: 1fr;
  }

  .lingqu-keys__stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .lingqu-keys__hero-tools {
    grid-template-columns: minmax(8rem, 1fr) minmax(7rem, 0.8fr) minmax(12rem, 1fr);
  }
}

@media (max-width: 680px) {
  .lingqu-keys__hero {
    grid-template-columns: 1fr;
    min-height: auto;
    padding: 0.7rem;
  }

  .lingqu-keys__stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .lingqu-key-card__metrics {
    grid-template-columns: 1fr;
  }

  .lingqu-keys__select {
    width: 100%;
  }

  .lingqu-key-card__top {
    flex-direction: column;
  }

  .lingqu-keys__copy h1 {
    max-width: none;
    font-size: 1.55rem;
  }

  .lingqu-keys__copy p {
    max-width: none;
    font-size: 0.78rem;
  }

  .lingqu-keys__hero-tools {
    grid-template-columns: 1fr;
  }

  .lingqu-keys__primary--hero,
  .lingqu-keys__usage-entry,
  .lingqu-keys__endpoint {
    width: 100%;
    max-width: none;
  }

  .lingqu-keys__refresh {
    width: 100%;
    margin-left: 0;
  }

  .lingqu-keys__primary,
  .lingqu-keys__secondary,
  .lingqu-keys__usage-entry {
    min-height: 2rem;
    padding: 0 0.62rem;
  }

  .lingqu-keys__toolbar {
    padding: 0.5rem;
  }
}

@media (max-width: 380px) {
  .lingqu-keys__stats article {
    min-height: 4.8rem;
    padding: 0.55rem;
  }

  .lingqu-keys__stats strong {
    font-size: 0.95rem;
  }
}

@keyframes keyPageRise {
  from {
    opacity: 0;
    transform: translateY(12px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes keyPulse {
  0%,
  100% {
    opacity: 0.48;
  }
  50% {
    opacity: 0.9;
  }
}
</style>
