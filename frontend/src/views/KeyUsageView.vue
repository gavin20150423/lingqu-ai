<template>
  <div class="key-usage-anime-shell relative flex min-h-screen flex-col overflow-hidden">
    <div class="pointer-events-none absolute inset-0 key-usage-anime-shell__grid"></div>
    <div class="pointer-events-none absolute inset-x-0 top-0 h-1 bg-comic-ink/10"></div>

    <!-- Header -->
    <header class="relative z-20 border-b-2 border-comic-ink/80 bg-[#fffdf5]/[0.96] px-4 py-4 backdrop-blur-sm sm:px-6 dark:border-dark-700/70 dark:bg-dark-950/[0.92]">
      <nav class="mx-auto flex max-w-7xl items-center justify-between gap-4">
        <router-link to="/home" class="flex items-center gap-3">
          <div class="h-11 w-11 overflow-hidden rounded-[16px] border-[3px] border-comic-ink bg-white shadow-[4px_4px_0_rgba(33,31,28,0.9)]">
            <img :src="siteLogo || '/brand/lingqu-ai-logo.svg'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <div class="min-w-0">
            <span class="block truncate text-lg font-black text-comic-ink dark:text-white">{{ siteName }}</span>
            <span class="block text-[11px] font-black uppercase tracking-[0.2em] text-comic-ink/45 dark:text-white/45">KEY 观测舱</span>
          </div>
        </router-link>
        <div class="flex items-center gap-3">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex items-center gap-2 rounded-[14px] border-2 border-transparent px-3 py-2 text-sm font-medium text-comic-ink transition-colors hover:border-comic-ink/70 hover:bg-yellow-100/80 dark:text-dark-200 dark:hover:border-dark-600 dark:hover:bg-dark-800"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
            <span class="hidden sm:inline">{{ t('home.docs') }}</span>
          </a>
        </div>
      </nav>
    </header>

    <!-- Main Content -->
    <main class="relative z-10 flex-1 px-4 py-6 sm:px-6 lg:px-8">
      <div class="mx-auto flex max-w-7xl flex-col gap-6">
        <section class="grid gap-6 lg:grid-cols-[1.15fr_0.85fr]">
          <div class="key-usage-panel key-usage-panel--hero">
            <div class="key-usage-kicker">灵渠AI · Key 观测舱</div>
            <h1 class="key-usage-title">
              一把 Key，看见全部用量脉络。
            </h1>
            <p class="key-usage-lead">
              输入任意 Key，立刻展开额度、周期、模型分布与今日消耗。这里是灵渠AI的透明仪表盘，不再是默认后台模板。
            </p>

            <form class="mt-6 key-usage-hero-console" @submit.prevent="queryKey">
              <div class="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
                <div>
                  <p class="key-usage-console-label">{{ t('keyUsage.subtitle') }}</p>
                  <h2 class="key-usage-console-title">把 Key 丢进来，立刻出结果。</h2>
                </div>
                <p class="key-usage-console-note">
                  {{ t('keyUsage.privacyNote') }}
                </p>
              </div>

              <div class="mt-4 query-input-frame" :class="{ 'query-input-frame--active': apiKey || keyVisible }">
                <div class="query-input-icon" aria-hidden="true">
                  <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <rect x="3" y="11" width="18" height="10" rx="2" />
                    <path d="M7 11V7a5 5 0 0 1 10 0v4" />
                  </svg>
                </div>
                <span v-if="!apiKey" class="query-placeholder" aria-hidden="true">
                  <span class="query-placeholder__text">输入你的万能 Key 密令</span>
                </span>
                <input
                  v-model="apiKey"
                  :type="keyVisible ? 'text' : 'password'"
                  :placeholder="t('keyUsage.placeholder')"
                  class="query-input"
                  autocomplete="off"
                  spellcheck="false"
                  @keydown.enter.prevent="queryKey"
                />
                <button
                  type="button"
                  @click="keyVisible = !keyVisible"
                  class="query-visibility-btn"
                  :aria-label="keyVisible ? '隐藏密钥' : '显示密钥'"
                >
                  <svg v-if="!keyVisible" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" />
                    <line x1="1" y1="1" x2="23" y2="23" />
                  </svg>
                  <svg v-else class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                    <circle cx="12" cy="12" r="3" />
                  </svg>
                </button>
              </div>

              <div class="mt-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div class="key-usage-signal-strip" aria-hidden="true">
                  <span></span>
                  <span></span>
                  <span></span>
                  <strong>Multi-model route ready</strong>
                </div>
                <button
                  type="submit"
                  :disabled="isQuerying"
                  class="query-submit-btn"
                >
                  <svg v-if="isQuerying" class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
                    <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" opacity="0.25" />
                    <path d="M12 2a10 10 0 0 1 10 10" stroke="currentColor" stroke-width="3" stroke-linecap="round" />
                  </svg>
                  <svg v-else class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <circle cx="11" cy="11" r="8" />
                    <line x1="21" y1="21" x2="16.65" y2="16.65" />
                  </svg>
                  <span>{{ isQuerying ? t('keyUsage.querying') : t('keyUsage.query') }}</span>
                </button>
              </div>

              <div v-if="showDatePicker" class="mt-5 key-usage-filter-bar">
                <div class="flex flex-wrap items-center gap-2">
                  <span class="key-usage-filter-bar__label">{{ t('keyUsage.dateRange') }}</span>
                  <button
                    v-for="range in dateRanges"
                    :key="range.key"
                    type="button"
                    @click="setDateRange(range.key)"
                    class="key-usage-range-btn"
                    :class="currentRange === range.key && 'key-usage-range-btn--active'"
                  >
                    {{ range.label }}
                  </button>
                </div>
                <div v-if="currentRange === 'custom'" class="mt-3 flex flex-wrap items-center gap-2">
                  <input
                    v-model="customStartDate"
                    type="date"
                    class="key-usage-date-input"
                  />
                  <span class="text-xs font-semibold text-comic-ink/45 dark:text-white/45">-</span>
                  <input
                    v-model="customEndDate"
                    type="date"
                    class="key-usage-date-input"
                  />
                  <button
                    type="button"
                    @click="queryKey"
                    class="key-usage-apply-btn"
                  >
                    {{ t('keyUsage.apply') }}
                  </button>
                </div>
              </div>
            </form>

            <div class="mt-6 grid gap-3 sm:grid-cols-3">
              <div class="key-usage-stat">
                <span class="key-usage-stat__label">单 Key</span>
                <strong class="key-usage-stat__value">全链路</strong>
              </div>
              <div class="key-usage-stat">
                <span class="key-usage-stat__label">模型</span>
                <strong class="key-usage-stat__value">一屏追踪</strong>
              </div>
              <div class="key-usage-stat">
                <span class="key-usage-stat__label">用量</span>
                <strong class="key-usage-stat__value">实时可见</strong>
              </div>
            </div>
          </div>

          <div class="key-usage-panel key-usage-panel--art">
            <div class="key-usage-art__ribbon">流量路由图</div>
            <div class="key-usage-art__frame">
              <img
                src="/illustrations/anime-gateway-hero.svg"
                alt="动漫风格 AI 模型网关插画"
                class="key-usage-art__image"
              />
            </div>
            <div class="mt-4 grid gap-3 sm:grid-cols-2">
              <div class="key-usage-mini-card">
                <span class="key-usage-mini-card__label">接入</span>
                <strong class="key-usage-mini-card__value">一个 Key</strong>
              </div>
              <div class="key-usage-mini-card">
                <span class="key-usage-mini-card__label">风格</span>
                <strong class="key-usage-mini-card__value">动漫卡通</strong>
              </div>
            </div>
          </div>
        </section>

      </div>

      <!-- Results Container -->
      <div v-if="showResults" class="mx-auto mt-8 max-w-7xl">
        <!-- Loading Skeleton -->
        <div v-if="showLoading" class="space-y-6">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div class="key-usage-result-card p-8">
              <div class="skeleton h-5 w-24 mb-6"></div>
              <div class="flex justify-center"><div class="skeleton w-44 h-44 rounded-full"></div></div>
            </div>
            <div class="key-usage-result-card p-8">
              <div class="skeleton h-5 w-24 mb-6"></div>
              <div class="flex justify-center"><div class="skeleton w-44 h-44 rounded-full"></div></div>
            </div>
          </div>
          <div class="key-usage-result-card p-8">
            <div class="skeleton h-5 w-32 mb-6"></div>
            <div class="space-y-4">
              <div class="skeleton h-4 w-full"></div>
              <div class="skeleton h-4 w-3/4"></div>
              <div class="skeleton h-4 w-5/6"></div>
              <div class="skeleton h-4 w-2/3"></div>
            </div>
          </div>
        </div>

        <!-- Result Content -->
        <div v-else-if="resultData" class="space-y-6">
          <!-- Status Badge -->
          <div v-if="statusInfo" class="fade-up flex items-center justify-center mb-2">
            <div class="key-usage-status-badge">
              <span
                class="w-2.5 h-2.5 rounded-full pulse-dot"
                :class="statusInfo.isActive ? 'bg-emerald-500' : 'bg-rose-500'"
              ></span>
              <span class="text-sm font-black text-comic-ink dark:text-white">{{ statusInfo.label }}</span>
              <span class="text-xs text-comic-ink/35 dark:text-white/35">|</span>
              <span class="text-xs font-bold text-comic-ink/58 dark:text-white/58">{{ statusInfo.statusText }}</span>
            </div>
          </div>

          <!-- Ring Cards Grid -->
          <div v-if="ringItems.length > 0" :class="ringGridClass">
            <div
              v-for="(ring, i) in ringItems"
              :key="i"
              class="fade-up key-usage-result-card p-8 transition-transform duration-300 hover:-translate-y-1"
              :class="`fade-up-delay-${Math.min(i + 1, 4)}`"
            >
              <div class="flex items-center justify-between mb-6">
                <h3 class="text-sm font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400">
                  {{ ring.title }}
                </h3>
                <!-- Clock icon -->
                <svg v-if="ring.iconType === 'clock'" class="w-5 h-5 text-gray-400 dark:text-dark-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
                </svg>
                <!-- Calendar icon -->
                <svg v-else-if="ring.iconType === 'calendar'" class="w-5 h-5 text-gray-400 dark:text-dark-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/>
                </svg>
                <!-- Dollar icon -->
                <svg v-else class="w-5 h-5 text-gray-400 dark:text-dark-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <line x1="12" y1="1" x2="12" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/>
                </svg>
              </div>
              <div class="flex justify-center">
                <div class="relative">
                  <svg class="w-44 h-44" viewBox="0 0 160 160">
                    <circle cx="80" cy="80" r="68" fill="none" :stroke="ringTrackColor" stroke-width="10"/>
                    <circle
                      class="progress-ring"
                      cx="80" cy="80" r="68" fill="none"
                      :stroke="`url(#ring-grad-${i})`"
                      stroke-width="10" stroke-linecap="round"
                      :stroke-dasharray="CIRCUMFERENCE.toFixed(2)"
                      :stroke-dashoffset="getRingOffset(ring)"
                    />
                    <defs>
                      <linearGradient :id="`ring-grad-${i}`" x1="0%" y1="0%" x2="100%" y2="100%">
                        <stop offset="0%" :stop-color="RING_GRADIENTS[i % 4].from"/>
                        <stop offset="100%" :stop-color="RING_GRADIENTS[i % 4].to"/>
                      </linearGradient>
                    </defs>
                  </svg>
                  <div class="absolute inset-0 flex flex-col items-center justify-center">
                    <template v-if="ring.isBalance">
                      <span class="text-2xl font-bold tabular-nums" :style="{ color: RING_GRADIENTS[i % 4].from }">
                        {{ ring.amount }}
                      </span>
                    </template>
                    <template v-else>
                      <span class="text-3xl font-bold tabular-nums text-gray-900 dark:text-white">
                        {{ displayPcts[i] ?? 0 }}%
                      </span>
                      <span class="text-xs text-gray-500 dark:text-dark-400 mt-0.5">{{ t('keyUsage.used') }}</span>
                      <span
                        class="text-sm font-semibold mt-1 tabular-nums"
                        :style="{ color: RING_GRADIENTS[i % 4].from }"
                      >{{ ring.amount }}</span>
                      <p v-if="ring.resetAt && formatResetTime(ring.resetAt)" class="text-xs text-gray-400 dark:text-gray-500 mt-0.5 tabular-nums">
                        ⟳ {{ formatResetTime(ring.resetAt) }}
                      </p>
                    </template>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Detail Card -->
          <div
            v-if="detailRows.length > 0"
            class="fade-up fade-up-delay-3 key-usage-result-card overflow-hidden"
          >
            <div class="px-8 py-5 border-b border-gray-200 dark:border-dark-700">
              <h3 class="text-sm font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('keyUsage.detailInfo') }}</h3>
            </div>
            <div class="divide-y divide-gray-100 dark:divide-dark-800">
              <div
                v-for="(row, i) in detailRows"
                :key="i"
                class="px-8 py-4 flex items-center justify-between"
              >
                <div class="flex items-center gap-3">
                  <div class="w-8 h-8 rounded-lg flex items-center justify-center" :class="row.iconBg">
                    <svg
                      class="w-4 h-4"
                      :class="row.iconColor"
                      viewBox="0 0 24 24" fill="none" stroke="currentColor"
                      stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
                      v-html="row.iconSvg"
                    ></svg>
                  </div>
                  <span class="text-sm text-gray-700 dark:text-dark-200">{{ row.label }}</span>
                </div>
                <span class="text-sm font-semibold tabular-nums" :class="row.valueClass || 'text-gray-900 dark:text-white'">
                  {{ row.value }}
                </span>
              </div>
            </div>
          </div>

          <!-- Usage Stats Card -->
          <div
            v-if="usageStatCells.length > 0"
            class="fade-up fade-up-delay-3 key-usage-result-card overflow-hidden"
          >
            <div class="px-8 py-5 border-b border-gray-200 dark:border-dark-700">
              <h3 class="text-sm font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('keyUsage.tokenStats') }}</h3>
            </div>
            <div class="grid grid-cols-2 md:grid-cols-4 gap-px bg-gray-100 dark:bg-dark-800">
              <div
                v-for="(cell, i) in usageStatCells"
                :key="i"
                class="bg-white px-6 py-4 dark:bg-dark-900"
              >
                <div class="text-xs text-gray-500 dark:text-dark-400 mb-1">{{ cell.label }}</div>
                <div class="text-sm font-semibold tabular-nums text-gray-900 dark:text-white">{{ cell.value }}</div>
              </div>
            </div>
          </div>

          <!-- Daily Usage Table -->
          <div
            v-if="showDailyUsage"
            class="fade-up fade-up-delay-4 key-usage-result-card overflow-hidden"
          >
            <div class="flex flex-col gap-3 px-8 py-5 border-b border-gray-200 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
              <h3 class="text-sm font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('keyUsage.dailyDetail') }}</h3>
              <div class="inline-flex rounded-lg border border-gray-200 bg-white p-0.5 dark:border-dark-700 dark:bg-dark-950">
                <button
                  v-for="option in dailyUsageOptions"
                  :key="option.value"
                  @click="setDailyUsageDays(option.value)"
                  class="min-w-12 rounded-md px-3 py-1.5 text-xs font-medium transition-colors"
                  :class="dailyUsageDays === option.value
                    ? 'bg-primary-500 text-white'
                    : 'text-gray-600 hover:bg-gray-100 dark:text-dark-300 dark:hover:bg-dark-800'"
                >
                  {{ option.label }}
                </button>
              </div>
            </div>
            <div v-if="dailyUsageRows.length > 0" class="overflow-x-auto">
              <table class="w-full">
                <thead>
                  <tr class="border-b border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-950">
                    <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('keyUsage.date') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('keyUsage.requests') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('keyUsage.inputTokens') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('keyUsage.outputTokens') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('keyUsage.cacheReadTokens') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('keyUsage.cacheWriteTokens') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('keyUsage.cost') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="row in dailyUsageRows"
                    :key="row.date"
                    class="border-b border-gray-100 last:border-b-0 dark:border-dark-800"
                  >
                    <td class="px-4 py-3 text-sm font-medium whitespace-nowrap text-gray-900 dark:text-white">{{ row.date }}</td>
                    <td class="px-4 py-3 text-sm tabular-nums text-right text-gray-700 dark:text-dark-200">{{ fmtNum(row.requests) }}</td>
                    <td class="px-4 py-3 text-sm tabular-nums text-right text-gray-700 dark:text-dark-200">{{ fmtNum(row.input_tokens) }}</td>
                    <td class="px-4 py-3 text-sm tabular-nums text-right text-gray-700 dark:text-dark-200">{{ fmtNum(row.output_tokens) }}</td>
                    <td class="px-4 py-3 text-sm tabular-nums text-right text-gray-700 dark:text-dark-200">{{ fmtNum(row.cache_read_tokens) }}</td>
                    <td class="px-4 py-3 text-sm tabular-nums text-right text-gray-700 dark:text-dark-200">{{ fmtNum(row.cache_write_tokens) }}</td>
                    <td class="px-4 py-3 text-sm tabular-nums text-right font-medium text-gray-900 dark:text-white">{{ usd(row.actual_cost != null ? row.actual_cost : row.cost) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div v-else class="px-8 py-8 text-center text-sm text-gray-500 dark:text-dark-400">
              {{ t('keyUsage.noDailyUsage') }}
            </div>
          </div>

          <!-- Model Stats Table -->
          <div
            v-if="modelStats.length > 0"
            class="fade-up fade-up-delay-4 key-usage-result-card overflow-hidden"
          >
            <div class="px-8 py-5 border-b border-gray-200 dark:border-dark-700">
              <h3 class="text-sm font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('keyUsage.modelStats') }}</h3>
            </div>
            <div class="overflow-x-auto">
              <table class="w-full">
                <thead>
                  <tr class="border-b border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-950">
                    <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('keyUsage.model') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('keyUsage.requests') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('keyUsage.inputTokens') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('keyUsage.outputTokens') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('keyUsage.cacheCreationTokens') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('keyUsage.cacheReadTokens') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('keyUsage.totalTokens') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('keyUsage.cost') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="(m, i) in modelStats"
                    :key="i"
                    class="border-b border-gray-100 last:border-b-0 dark:border-dark-800"
                  >
                    <td class="px-4 py-3 text-sm font-medium whitespace-nowrap text-gray-900 dark:text-white">{{ m.model || '-' }}</td>
                    <td class="px-4 py-3 text-sm tabular-nums text-right text-gray-700 dark:text-dark-200">{{ fmtNum(m.requests) }}</td>
                    <td class="px-4 py-3 text-sm tabular-nums text-right text-gray-700 dark:text-dark-200">{{ fmtNum(m.input_tokens) }}</td>
                    <td class="px-4 py-3 text-sm tabular-nums text-right text-gray-700 dark:text-dark-200">{{ fmtNum(m.output_tokens) }}</td>
                    <td class="px-4 py-3 text-sm tabular-nums text-right text-gray-700 dark:text-dark-200">{{ fmtNum(m.cache_creation_tokens) }}</td>
                    <td class="px-4 py-3 text-sm tabular-nums text-right text-gray-700 dark:text-dark-200">{{ fmtNum(m.cache_read_tokens) }}</td>
                    <td class="px-4 py-3 text-sm tabular-nums text-right text-gray-700 dark:text-dark-200">{{ fmtNum(m.total_tokens) }}</td>
                    <td class="px-4 py-3 text-sm tabular-nums text-right font-medium text-gray-900 dark:text-white">{{ usd(m.actual_cost != null ? m.actual_cost : m.cost) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
    </main>

    <!-- Footer (same pattern as HomeView) -->
    <footer class="relative z-10 border-t-2 border-comic-ink/15 px-6 py-8 dark:border-dark-800/50">
      <div class="mx-auto flex max-w-6xl flex-col items-center justify-center gap-4 text-center sm:flex-row sm:text-left">
        <p class="text-sm font-semibold text-comic-ink/50 dark:text-dark-400">
          &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
        </p>
        <div class="flex items-center gap-4">
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="text-sm font-semibold text-comic-ink/50 transition-colors hover:text-comic-ink dark:text-dark-400 dark:hover:text-white"
          >{{ t('home.docs') }}</a>
          <a
            :href="githubUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="text-sm font-semibold text-comic-ink/50 transition-colors hover:text-comic-ink dark:text-dark-400 dark:hover:text-white"
          >GitHub</a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { buildGatewayUrl } from '@/api/client'
import { resolveBrandLogo, resolveBrandName } from '@/constants/brand'
import { formatDateLocalInput } from '@/utils/format'
import { sanitizeUrl } from '@/utils/url'

const { t, locale } = useI18n()
const appStore = useAppStore()

// ==================== Site Settings (same as HomeView) ====================

const siteName = computed(() => resolveBrandName(appStore.cachedPublicSettings?.site_name || appStore.siteName))
const siteLogo = computed(() =>
  sanitizeUrl(resolveBrandLogo(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo), {
    allowRelative: true,
    allowDataUrl: true
  })
)
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const githubUrl = 'https://github.com/Wei-Shaw/sub2api'

const currentYear = computed(() => new Date().getFullYear())

// ==================== Key Query State ====================

const apiKey = ref('')
const keyVisible = ref(false)
const isQuerying = ref(false)
const showResults = ref(false)
const showLoading = ref(false)
const showDatePicker = ref(false)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const resultData = ref<any>(null)
const now = ref(new Date())
let resetTimer: ReturnType<typeof setInterval> | null = null

// ==================== Date Range State ====================

type DateRangeKey = 'today' | '7d' | '30d' | 'custom'
const currentRange = ref<DateRangeKey>('today')
const customStartDate = ref('')
const customEndDate = ref('')
const dailyUsageDays = ref<7 | 30 | 90>(30)

const dateRanges = computed(() => [
  { key: 'today' as const, label: t('keyUsage.dateRangeToday') },
  { key: '7d' as const, label: t('keyUsage.dateRange7d') },
  { key: '30d' as const, label: t('keyUsage.dateRange30d') },
  { key: 'custom' as const, label: t('keyUsage.dateRangeCustom') },
])

const dailyUsageOptions = computed(() => [
  { value: 7 as const, label: t('keyUsage.dateRange7d') },
  { value: 30 as const, label: t('keyUsage.dateRange30d') },
  { value: 90 as const, label: t('keyUsage.dateRange90d') },
])

function setDateRange(key: DateRangeKey) {
  currentRange.value = key
  if (key !== 'custom') {
    queryKey()
  }
}

function getDateParams(): string {
  const now = new Date()
  const params = new URLSearchParams()

  if (currentRange.value === 'custom') {
    if (customStartDate.value && customEndDate.value) {
      params.set('start_date', customStartDate.value)
      params.set('end_date', customEndDate.value)
    }
  } else {
    const end = formatDateLocalInput(now)
    let start: string
    switch (currentRange.value) {
      case 'today': start = end; break
      case '7d': start = formatDateLocalInput(new Date(now.getTime() - 7 * 86400000)); break
      case '30d': start = formatDateLocalInput(new Date(now.getTime() - 30 * 86400000)); break
      default: start = formatDateLocalInput(new Date(now.getTime() - 30 * 86400000))
    }
    params.set('start_date', start)
    params.set('end_date', end)
  }
  params.set('days', String(dailyUsageDays.value))
  params.set('timezone', getBrowserTimezone())
  return params.toString()
}

function setDailyUsageDays(days: 7 | 30 | 90) {
  if (dailyUsageDays.value === days) return
  dailyUsageDays.value = days
  if (resultData.value && apiKey.value.trim()) {
    queryKey()
  }
}

// ==================== Ring Animation ====================

const CIRCUMFERENCE = 2 * Math.PI * 68
const RING_GRADIENTS = [
  { from: '#14b8a6', to: '#5eead4' },
  { from: '#6366F1', to: '#A5B4FC' },
  { from: '#10B981', to: '#6EE7B7' },
  { from: '#F59E0B', to: '#FCD34D' },
]

const ringAnimated = ref(false)
const displayPcts = ref<number[]>([])

const ringTrackColor = computed(() => '#F0F0EE')

interface RingItem {
  title: string
  pct: number
  amount: string
  isBalance?: boolean
  iconType: 'clock' | 'calendar' | 'dollar'
  resetAt?: string | null
}

function getRingOffset(ring: RingItem): number {
  if (!ringAnimated.value) return CIRCUMFERENCE
  if (ring.isBalance) return 0
  return CIRCUMFERENCE - (Math.min(ring.pct, 100) / 100) * CIRCUMFERENCE
}

function triggerRingAnimation(items: RingItem[]) {
  ringAnimated.value = false
  displayPcts.value = items.map(() => 0)

  nextTick(() => {
    requestAnimationFrame(() => {
      setTimeout(() => {
        ringAnimated.value = true

        // Animate percentage numbers
        const duration = 1000
        const startTime = performance.now()
        const targets = items.map(item => item.isBalance ? 0 : item.pct)

        function tick() {
          const elapsed = performance.now() - startTime
          const p = Math.min(elapsed / duration, 1)
          const ease = 1 - Math.pow(1 - p, 3)
          displayPcts.value = targets.map(target => Math.round(ease * target))
          if (p < 1) requestAnimationFrame(tick)
        }
        requestAnimationFrame(tick)
      }, 50)
    })
  })
}

// ==================== Computed Data ====================

const statusInfo = computed(() => {
  const data = resultData.value
  if (!data) return null

  if (data.mode === 'quota_limited') {
    const isValid = data.isValid !== false
    const statusMap: Record<string, string> = {
      active: 'Active',
      quota_exhausted: 'Quota Exhausted',
      expired: 'Expired',
    }
    return {
      label: t('keyUsage.quotaMode'),
      statusText: statusMap[data.status] || data.status || 'Unknown',
      isActive: isValid && data.status === 'active',
    }
  }

  return {
    label: data.planName || t('keyUsage.walletBalance'),
    statusText: 'Active',
    isActive: true,
  }
})

const ringItems = computed<RingItem[]>(() => {
  const data = resultData.value
  if (!data) return []

  const items: RingItem[] = []

  if (data.mode === 'quota_limited') {
    if (data.quota) {
      const pct = data.quota.limit > 0 ? Math.min(Math.round((data.quota.used / data.quota.limit) * 100), 100) : 0
      items.push({ title: t('keyUsage.totalQuota'), pct, amount: `${usd(data.quota.used)} / ${usd(data.quota.limit)}`, iconType: 'dollar' })
    }
    if (data.rate_limits) {
      const windowLabels: Record<string, string> = { '5h': t('keyUsage.limit5h'), '1d': t('keyUsage.limitDaily'), '7d': t('keyUsage.limit7d') }
      const windowIcons: Record<string, 'clock' | 'calendar'> = { '5h': 'clock', '1d': 'calendar', '7d': 'calendar' }
      for (const rl of data.rate_limits) {
        const pct = rl.limit > 0 ? Math.min(Math.round((rl.used / rl.limit) * 100), 100) : 0
        items.push({
          title: windowLabels[rl.window] || rl.window,
          pct,
          amount: `${usd(rl.used)} / ${usd(rl.limit)}`,
          iconType: windowIcons[rl.window] || 'clock',
          resetAt: rl.reset_at,
        })
      }
    }
  } else {
    if (data.subscription) {
      const sub = data.subscription
      const limits = [
        { label: t('keyUsage.limitDaily'), usage: sub.daily_usage_usd, limit: sub.daily_limit_usd },
        { label: t('keyUsage.limitWeekly'), usage: sub.weekly_usage_usd, limit: sub.weekly_limit_usd },
        { label: t('keyUsage.limitMonthly'), usage: sub.monthly_usage_usd, limit: sub.monthly_limit_usd },
      ]
      for (const l of limits) {
        if (l.limit != null && l.limit > 0) {
          const pct = Math.min(Math.round((l.usage / l.limit) * 100), 100)
          items.push({ title: l.label, pct, amount: `${usd(l.usage)} / ${usd(l.limit)}`, iconType: 'calendar' })
        }
      }
    }
    if (!data.subscription && data.balance != null) {
      items.push({ title: t('keyUsage.walletBalance'), pct: 0, amount: usd(data.balance), isBalance: true, iconType: 'dollar' })
    }
  }

  return items
})

const ringGridClass = computed(() => {
  const len = ringItems.value.length
  if (len === 1) return 'grid grid-cols-1 max-w-md mx-auto gap-6'
  if (len === 2) return 'grid grid-cols-1 md:grid-cols-2 gap-6'
  return 'grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6'
})

interface DetailRow {
  iconBg: string
  iconColor: string
  iconSvg: string
  label: string
  value: string
  valueClass: string
}

function getUsageColor(pct: number): string {
  if (pct > 90) return 'text-rose-500'
  if (pct > 70) return 'text-amber-500'
  return 'text-emerald-500'
}

const detailRows = computed<DetailRow[]>(() => {
  const data = resultData.value
  if (!data) return []

  const rows: DetailRow[] = []
  const ICON_SHIELD = '<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>'
  const ICON_CALENDAR = '<rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/>'
  const ICON_DOLLAR = '<line x1="12" y1="1" x2="12" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/>'
  const ICON_CHECK = '<polyline points="20 6 9 17 4 12"/>'

  if (data.mode === 'quota_limited') {
    if (data.quota) {
      const remainColor = data.quota.remaining <= 0 ? 'text-rose-500'
        : data.quota.remaining < data.quota.limit * 0.1 ? 'text-amber-500'
        : 'text-emerald-500'
      rows.push({
        iconBg: 'bg-emerald-500/10', iconColor: 'text-emerald-500', iconSvg: ICON_SHIELD,
        label: t('keyUsage.remainingQuota'), value: usd(data.quota.remaining), valueClass: remainColor,
      })
    }
    if (data.expires_at) {
      const daysLeft = data.days_until_expiry
      let expiryStr = formatDate(data.expires_at)
      if (daysLeft != null) {
        expiryStr += daysLeft > 0 ? ` ${t('keyUsage.daysLeft', { days: daysLeft })}` : daysLeft === 0 ? ` ${t('keyUsage.todayExpires')}` : ''
      }
      rows.push({
        iconBg: 'bg-amber-500/10', iconColor: 'text-amber-500', iconSvg: ICON_CALENDAR,
        label: t('keyUsage.expiresAt'), value: expiryStr, valueClass: '',
      })
    }
    if (data.rate_limits) {
      const windowMap: Record<string, string> = { '5h': '5H', '1d': locale.value === 'zh' ? '日' : 'D', '7d': '7D' }
      for (const rl of data.rate_limits) {
        const pct = rl.limit > 0 ? (rl.used / rl.limit) * 100 : 0
        let valueStr = `${usd(rl.used)} / ${usd(rl.limit)}`
        const resetStr = formatResetTime(rl.reset_at)
        if (resetStr) {
          valueStr += ` (⟳ ${resetStr})`
        }
        rows.push({
          iconBg: 'bg-primary-500/10', iconColor: 'text-primary-500', iconSvg: ICON_DOLLAR,
          label: `${t('keyUsage.usedQuota')} (${windowMap[rl.window] || rl.window})`,
          value: valueStr,
          valueClass: getUsageColor(pct),
        })
      }
    }
  } else {
    rows.push({
      iconBg: 'bg-emerald-500/10', iconColor: 'text-emerald-500', iconSvg: ICON_CHECK,
      label: t('keyUsage.subscriptionType'), value: data.planName || t('keyUsage.walletBalance'), valueClass: '',
    })

    if (data.subscription) {
      const sub = data.subscription
      if (sub.daily_limit_usd > 0) {
        const pct = (sub.daily_usage_usd / sub.daily_limit_usd) * 100
        rows.push({
          iconBg: 'bg-primary-500/10', iconColor: 'text-primary-500', iconSvg: ICON_DOLLAR,
          label: `${t('keyUsage.usedQuota')} (${locale.value === 'zh' ? '日' : 'D'})`, value: `${usd(sub.daily_usage_usd)} / ${usd(sub.daily_limit_usd)}`, valueClass: getUsageColor(pct),
        })
      }
      if (sub.weekly_limit_usd > 0) {
        const pct = (sub.weekly_usage_usd / sub.weekly_limit_usd) * 100
        rows.push({
          iconBg: 'bg-indigo-500/10', iconColor: 'text-indigo-500', iconSvg: ICON_DOLLAR,
          label: `${t('keyUsage.usedQuota')} (${locale.value === 'zh' ? '周' : 'W'})`, value: `${usd(sub.weekly_usage_usd)} / ${usd(sub.weekly_limit_usd)}`, valueClass: getUsageColor(pct),
        })
      }
      if (sub.monthly_limit_usd > 0) {
        const pct = (sub.monthly_usage_usd / sub.monthly_limit_usd) * 100
        rows.push({
          iconBg: 'bg-emerald-500/10', iconColor: 'text-emerald-500', iconSvg: ICON_DOLLAR,
          label: `${t('keyUsage.usedQuota')} (${locale.value === 'zh' ? '月' : 'M'})`, value: `${usd(sub.monthly_usage_usd)} / ${usd(sub.monthly_limit_usd)}`, valueClass: getUsageColor(pct),
        })
      }
      if (sub.expires_at) {
        rows.push({
          iconBg: 'bg-amber-500/10', iconColor: 'text-amber-500', iconSvg: ICON_CALENDAR,
          label: t('keyUsage.subscriptionExpires'), value: formatDate(sub.expires_at), valueClass: '',
        })
      }
    }

    const remainColor = data.remaining != null
      ? (data.remaining <= 0 ? 'text-rose-500' : data.remaining < 10 ? 'text-amber-500' : 'text-emerald-500')
      : ''
    rows.push({
      iconBg: 'bg-emerald-500/10', iconColor: 'text-emerald-500', iconSvg: ICON_SHIELD,
      label: t('keyUsage.remainingQuota'), value: data.remaining != null ? usd(data.remaining) : '-', valueClass: remainColor,
    })
  }

  return rows
})

interface StatCell {
  label: string
  value: string
}

const usageStatCells = computed<StatCell[]>(() => {
  const usage = resultData.value?.usage
  if (!usage) return []

  const today = usage.today || {}
  const total = usage.total || {}

  return [
    { label: t('keyUsage.todayRequests'), value: fmtNum(today.requests) },
    { label: t('keyUsage.todayInputTokens'), value: fmtNum(today.input_tokens) },
    { label: t('keyUsage.todayOutputTokens'), value: fmtNum(today.output_tokens) },
    { label: t('keyUsage.todayTokens'), value: fmtNum(today.total_tokens) },
    { label: t('keyUsage.todayCacheCreation'), value: fmtNum(today.cache_creation_tokens) },
    { label: t('keyUsage.todayCacheRead'), value: fmtNum(today.cache_read_tokens) },
    { label: t('keyUsage.todayCost'), value: usd(today.actual_cost) },
    { label: t('keyUsage.rpmTpm'), value: `${usage.rpm || 0} / ${usage.tpm || 0}` },
    { label: t('keyUsage.totalRequests'), value: fmtNum(total.requests) },
    { label: t('keyUsage.totalInputTokens'), value: fmtNum(total.input_tokens) },
    { label: t('keyUsage.totalOutputTokens'), value: fmtNum(total.output_tokens) },
    { label: t('keyUsage.totalTokensLabel'), value: fmtNum(total.total_tokens) },
    { label: t('keyUsage.totalCacheCreation'), value: fmtNum(total.cache_creation_tokens) },
    { label: t('keyUsage.totalCacheRead'), value: fmtNum(total.cache_read_tokens) },
    { label: t('keyUsage.totalCost'), value: usd(total.actual_cost) },
    { label: t('keyUsage.avgDuration'), value: usage.average_duration_ms ? `${Math.round(usage.average_duration_ms)} ms` : '-' },
  ]
})

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const modelStats = computed<any[]>(() => resultData.value?.model_stats || [])

interface DailyUsageRow {
  date: string
  requests: number
  input_tokens: number
  output_tokens: number
  cache_read_tokens: number
  cache_write_tokens: number
  cost: number
  actual_cost?: number
}

const dailyUsageRows = computed<DailyUsageRow[]>(() => {
  const rows = resultData.value?.daily_usage
  return Array.isArray(rows) ? rows : []
})

const showDailyUsage = computed(() => Boolean(resultData.value && Array.isArray(resultData.value.daily_usage)))

// ==================== Utility Functions ====================

function usd(value: number | null | undefined): string {
  if (value == null || value < 0) return '-'
  return '$' + Number(value).toFixed(2)
}

function fmtNum(val: number | null | undefined): string {
  if (val == null) return '-'
  return val.toLocaleString()
}

function formatDate(iso: string | null | undefined): string {
  if (!iso) return '-'
  const d = new Date(iso)
  const loc = locale.value === 'zh' ? 'zh-CN' : 'en-US'
  return d.toLocaleDateString(loc, { year: 'numeric', month: 'long', day: 'numeric' })
}

function getBrowserTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
  } catch {
    return 'UTC'
  }
}

// ==================== API Query ====================

async function fetchUsage(key: string) {
  const dateParams = getDateParams()
  const url = buildGatewayUrl('/v1/usage') + (dateParams ? '?' + dateParams : '')
  const res = await fetch(url, {
    headers: { 'Authorization': 'Bearer ' + key },
  })
  if (!res.ok) {
    const body = await res.json().catch(() => null)
    const msg = body?.error?.message || body?.message || `${t('keyUsage.queryFailed')} (${res.status})`
    throw new Error(msg)
  }
  return await res.json()
}

async function queryKey() {
  if (isQuerying.value) return
  const key = apiKey.value.trim()
  if (!key) {
    appStore.showInfo(t('keyUsage.enterApiKey'))
    return
  }

  isQuerying.value = true
  showResults.value = true
  showLoading.value = true
  resultData.value = null

  try {
    const data = await fetchUsage(key)
    resultData.value = data
    showLoading.value = false
    showDatePicker.value = true

    // Trigger ring animations after DOM update
    nextTick(() => {
      triggerRingAnimation(ringItems.value)
    })

    appStore.showSuccess(t('keyUsage.querySuccess'))
  } catch (err) {
    showResults.value = false
    showLoading.value = false
    appStore.showError((err as Error).message || t('keyUsage.queryFailedRetry'))
  } finally {
    isQuerying.value = false
  }
}

// ==================== Lifecycle ====================

function formatResetTime(resetAt: string | null | undefined): string {
  if (!resetAt) return ''
  const diff = new Date(resetAt).getTime() - now.value.getTime()
  if (diff <= 0) return t('keyUsage.resetNow')
  const days = Math.floor(diff / 86400000)
  const hours = Math.floor((diff % 86400000) / 3600000)
  const mins = Math.floor((diff % 3600000) / 60000)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${mins}m`
  return `${mins}m`
}

onMounted(() => {
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
  resetTimer = setInterval(() => { now.value = new Date() }, 60000)
})

onUnmounted(() => {
  if (resetTimer) clearInterval(resetTimer)
})
</script>

<style scoped>
.key-usage-anime-shell {
  background:
    radial-gradient(circle at 12% 10%, rgba(255, 122, 174, 0.18), transparent 26%),
    radial-gradient(circle at 88% 14%, rgba(78, 233, 255, 0.18), transparent 28%),
    radial-gradient(circle at 50% 0%, rgba(255, 212, 71, 0.22), transparent 24%),
    linear-gradient(135deg, #fff9ed 0%, #eefbff 48%, #fff1f8 100%);
}

:global(.dark) .key-usage-anime-shell {
  background:
    radial-gradient(circle at 12% 10%, rgba(255, 122, 174, 0.12), transparent 26%),
    radial-gradient(circle at 88% 14%, rgba(78, 233, 255, 0.12), transparent 28%),
    radial-gradient(circle at 50% 0%, rgba(255, 212, 71, 0.12), transparent 24%),
    linear-gradient(135deg, #12110f 0%, #171a28 52%, #25181e 100%);
}

.key-usage-anime-shell__grid {
  background-image:
    linear-gradient(rgba(33, 31, 28, 0.045) 1px, transparent 1px),
    linear-gradient(90deg, rgba(33, 31, 28, 0.045) 1px, transparent 1px);
  background-size: 24px 24px;
  mask-image: linear-gradient(to bottom, #000 0%, transparent 88%);
}

.key-usage-panel {
  border: 4px solid #211f1c;
  border-radius: 30px;
  background: rgba(255, 255, 255, 0.82);
  box-shadow: 12px 12px 0 rgba(33, 31, 28, 0.92);
  backdrop-filter: blur(18px);
}

:global(.dark) .key-usage-panel {
  background: rgba(24, 24, 32, 0.82);
}

.key-usage-panel--hero,
.key-usage-panel--query {
  padding: clamp(1.25rem, 3vw, 1.75rem);
}

.key-usage-panel--art {
  padding: clamp(1.15rem, 3vw, 1.5rem);
}

.key-usage-kicker,
.key-usage-section-label {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  border: 2px solid #211f1c;
  border-radius: 999px;
  background: #ffd447;
  padding: 0.35rem 0.7rem;
  color: #211f1c;
  font-size: 0.72rem;
  font-weight: 900;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  box-shadow: 3px 3px 0 rgba(33, 31, 28, 0.9);
}

.key-usage-title,
.key-usage-section-title {
  margin-top: 1rem;
  font-family: theme('fontFamily.display');
  font-weight: 950;
  line-height: 0.95;
  letter-spacing: 0;
  color: #ff4f7b;
  text-shadow: 4px 4px 0 #211f1c;
}

.key-usage-title {
  max-width: 12ch;
  font-size: clamp(2.6rem, 6vw, 5rem);
}

.key-usage-section-title {
  font-size: clamp(1.8rem, 4vw, 3rem);
}

.key-usage-lead {
  margin-top: 1rem;
  max-width: 42rem;
  color: rgba(33, 31, 28, 0.68);
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.85;
}

:global(.dark) .key-usage-lead {
  color: rgba(255, 250, 240, 0.68);
}

.key-usage-stat,
.key-usage-mini-card {
  border: 2px solid #211f1c;
  border-radius: 20px;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.88), rgba(255, 249, 220, 0.92));
  box-shadow: 5px 5px 0 rgba(33, 31, 28, 0.9);
}

:global(.dark) .key-usage-stat,
:global(.dark) .key-usage-mini-card {
  background: linear-gradient(180deg, rgba(30, 30, 40, 0.96), rgba(24, 24, 32, 0.96));
}

.key-usage-stat {
  padding: 0.95rem 1rem;
}

.key-usage-mini-card {
  min-height: 4rem;
  padding: 0.8rem 0.95rem;
}

.key-usage-stat__label,
.key-usage-mini-card__label {
  display: block;
  color: rgba(33, 31, 28, 0.46);
  font-size: 0.72rem;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

:global(.dark) .key-usage-stat__label,
:global(.dark) .key-usage-mini-card__label {
  color: rgba(255, 250, 240, 0.46);
}

.key-usage-stat__value,
.key-usage-mini-card__value {
  display: block;
  margin-top: 0.45rem;
  color: #211f1c;
  font-size: 1.1rem;
  font-weight: 900;
}

:global(.dark) .key-usage-stat__value,
:global(.dark) .key-usage-mini-card__value {
  color: #fffaf0;
}

.key-usage-art__ribbon {
  display: inline-flex;
  margin-bottom: 0.9rem;
  border: 2px solid #211f1c;
  border-radius: 999px;
  background: #4ee9ff;
  padding: 0.35rem 0.72rem;
  color: #211f1c;
  font-size: 0.72rem;
  font-weight: 900;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  box-shadow: 3px 3px 0 rgba(33, 31, 28, 0.88);
}

.key-usage-art__frame {
  overflow: hidden;
  border: 3px solid #211f1c;
  border-radius: 24px;
  background: #fff;
  box-shadow: 7px 7px 0 rgba(33, 31, 28, 0.86);
}

.key-usage-art__image {
  display: block;
  width: 100%;
  aspect-ratio: 16 / 11;
  object-fit: cover;
}

.key-usage-hero-console {
  border: 3px solid #211f1c;
  border-radius: 26px;
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.9), rgba(255, 244, 209, 0.82)),
    radial-gradient(circle at top right, rgba(78, 233, 255, 0.2), transparent 34%);
  padding: clamp(1rem, 2.2vw, 1.25rem);
  box-shadow: 7px 7px 0 rgba(33, 31, 28, 0.9);
}

:global(.dark) .key-usage-hero-console {
  background:
    linear-gradient(135deg, rgba(30, 30, 40, 0.94), rgba(38, 33, 44, 0.86)),
    radial-gradient(circle at top right, rgba(78, 233, 255, 0.14), transparent 34%);
}

.key-usage-console-label {
  color: rgba(33, 31, 28, 0.52);
  font-size: 0.68rem;
  font-weight: 900;
  letter-spacing: 0.13em;
  text-transform: uppercase;
}

.key-usage-console-title {
  margin-top: 0.28rem;
  color: #211f1c;
  font-size: clamp(1.08rem, 2.2vw, 1.55rem);
  font-weight: 950;
  line-height: 1.08;
  letter-spacing: 0;
}

:global(.dark) .key-usage-console-label {
  color: rgba(255, 250, 240, 0.52);
}

:global(.dark) .key-usage-console-title {
  color: #fffaf0;
}

.key-usage-console-note {
  max-width: 15rem;
  color: rgba(33, 31, 28, 0.48);
  font-size: 0.78rem;
  font-weight: 800;
  line-height: 1.6;
}

:global(.dark) .key-usage-console-note {
  color: rgba(255, 250, 240, 0.48);
}

.query-input-frame {
  position: relative;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  border: 4px solid #211f1c;
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.86);
  padding: 0.75rem 0.85rem;
  box-shadow: 8px 8px 0 rgba(33, 31, 28, 0.92);
  transition:
    transform 180ms ease,
    box-shadow 180ms ease,
    border-color 180ms ease;
}

:global(.dark) .query-input-frame {
  background: rgba(27, 27, 35, 0.9);
}

.query-input-frame--active {
  transform: translateY(-1px);
  box-shadow: 10px 10px 0 rgba(33, 31, 28, 0.95);
}

.query-input-icon,
.query-visibility-btn {
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 2.75rem;
  height: 2.75rem;
  border: 2px solid #211f1c;
  border-radius: 16px;
  background: #ffd447;
  color: #211f1c;
  box-shadow: 3px 3px 0 rgba(33, 31, 28, 0.9);
}

.query-visibility-btn {
  background: #4ee9ff;
}

.query-input {
  position: relative;
  z-index: 2;
  min-width: 0;
  flex: 1;
  border: 0;
  background: transparent;
  color: #211f1c;
  font-size: 1rem;
  font-weight: 700;
  outline: none;
}

:global(.dark) .query-input {
  color: #fffaf0;
}

.query-placeholder {
  pointer-events: none;
  position: absolute;
  left: 4.4rem;
  right: 4.2rem;
  top: 50%;
  z-index: 1;
  display: flex;
  transform: translateY(-50%);
  overflow: hidden;
  color: rgba(33, 31, 28, 0.44);
  font-size: 0.92rem;
  font-weight: 800;
  white-space: nowrap;
}

:global(.dark) .query-placeholder {
  color: rgba(255, 250, 240, 0.44);
}

.query-placeholder__text {
  display: inline-block;
  overflow: hidden;
  max-width: 0;
  white-space: nowrap;
  animation: keyUsageTypewriter 4.8s steps(14, end) infinite;
}

.query-placeholder__text::after {
  content: '';
  display: inline-block;
  width: 2px;
  height: 1em;
  margin-left: 0.16rem;
  background: #ff4f7b;
  vertical-align: -0.12em;
  animation: keyUsageCaret 0.8s steps(2, end) infinite;
}

.query-submit-btn,
.key-usage-apply-btn,
.key-usage-range-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
  border: 2px solid #211f1c;
  border-radius: 16px;
  padding: 0.72rem 1rem;
  font-size: 0.9rem;
  font-weight: 900;
  transition:
    transform 160ms ease,
    box-shadow 160ms ease,
    background-color 160ms ease;
}

.query-submit-btn:hover,
.key-usage-apply-btn:hover,
.key-usage-range-btn:hover {
  transform: translateY(-1px);
}

.query-submit-btn {
  background: #ff4f7b;
  color: #fffaf0;
  box-shadow: 4px 4px 0 rgba(33, 31, 28, 0.9);
}

.query-submit-btn:disabled {
  cursor: not-allowed;
  opacity: 0.6;
  transform: none;
}

.key-usage-signal-strip {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 0.45rem;
  color: rgba(33, 31, 28, 0.54);
  font-size: 0.68rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.key-usage-signal-strip span {
  width: 0.55rem;
  height: 0.55rem;
  border: 2px solid #211f1c;
  border-radius: 999px;
  background: #4ee9ff;
  box-shadow: 2px 2px 0 rgba(33, 31, 28, 0.82);
  animation: keyUsageSignalPulse 1.6s ease-in-out infinite;
}

.key-usage-signal-strip span:nth-child(2) {
  background: #ffd447;
  animation-delay: 0.18s;
}

.key-usage-signal-strip span:nth-child(3) {
  background: #ff7aae;
  animation-delay: 0.36s;
}

.key-usage-signal-strip strong {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:global(.dark) .key-usage-signal-strip {
  color: rgba(255, 250, 240, 0.54);
}

.key-usage-filter-bar {
  border: 3px solid #211f1c;
  border-radius: 22px;
  background: rgba(255, 255, 255, 0.7);
  padding: 1rem;
  box-shadow: 6px 6px 0 rgba(33, 31, 28, 0.9);
}

:global(.dark) .key-usage-filter-bar {
  background: rgba(24, 24, 32, 0.72);
}

.key-usage-status-badge,
.key-usage-result-card {
  border: 3px solid #211f1c;
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.84);
  box-shadow: 7px 7px 0 rgba(33, 31, 28, 0.9);
  backdrop-filter: blur(18px);
}

:global(.dark) .key-usage-status-badge,
:global(.dark) .key-usage-result-card {
  background: rgba(24, 24, 32, 0.84);
}

.key-usage-status-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.7rem 1rem;
}

.key-usage-filter-bar__label {
  display: inline-flex;
  align-items: center;
  border: 2px solid #211f1c;
  border-radius: 999px;
  background: #ffd447;
  padding: 0.3rem 0.7rem;
  color: #211f1c;
  font-size: 0.72rem;
  font-weight: 900;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  box-shadow: 3px 3px 0 rgba(33, 31, 28, 0.85);
}

.key-usage-range-btn {
  background: #fffaf0;
  color: #211f1c;
  box-shadow: 3px 3px 0 rgba(33, 31, 28, 0.85);
}

:global(.dark) .key-usage-range-btn {
  background: rgba(30, 30, 40, 0.96);
  color: #fffaf0;
}

.key-usage-range-btn--active {
  background: #4ee9ff;
}

.key-usage-date-input {
  border: 2px solid #211f1c;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.95);
  padding: 0.6rem 0.8rem;
  color: #211f1c;
  font-size: 0.85rem;
  font-weight: 700;
  box-shadow: 3px 3px 0 rgba(33, 31, 28, 0.85);
}

:global(.dark) .key-usage-date-input {
  background: rgba(30, 30, 40, 0.96);
  color: #fffaf0;
}

/* Ring animation */
.progress-ring {
  transition: stroke-dashoffset 1.2s cubic-bezier(0.4, 0, 0.2, 1);
  transform: rotate(-90deg);
  transform-origin: 50% 50%;
}

/* Skeleton loading */
@keyframes shimmer-kv {
  0%   { background-position: -200% 0; }
  100% { background-position: 200% 0; }
}
.skeleton {
  background: linear-gradient(90deg, #e5e7eb 25%, #f3f4f6 50%, #e5e7eb 75%);
  background-size: 200% 100%;
  animation: shimmer-kv 1.8s ease-in-out infinite;
  border-radius: 8px;
}
:global(.dark) .skeleton {
  background: linear-gradient(90deg, #334155 25%, #1e293b 50%, #334155 75%);
  background-size: 200% 100%;
}

/* Fade up animation */
@keyframes fade-up-kv {
  from { opacity: 0; transform: translateY(16px); }
  to { opacity: 1; transform: translateY(0); }
}
.fade-up {
  animation: fade-up-kv 0.5s cubic-bezier(0.4, 0, 0.2, 1) forwards;
}
.fade-up-delay-1 { animation-delay: 0.1s; opacity: 0; }
.fade-up-delay-2 { animation-delay: 0.2s; opacity: 0; }
.fade-up-delay-3 { animation-delay: 0.3s; opacity: 0; }
.fade-up-delay-4 { animation-delay: 0.4s; opacity: 0; }

/* Pulse dot */
@keyframes pulse-dot-kv {
  0%, 100% { opacity: 1; box-shadow: 0 0 0 0 currentColor; }
  50% { opacity: 0.6; box-shadow: 0 0 8px 2px currentColor; }
}
.pulse-dot {
  animation: pulse-dot-kv 2s ease-in-out infinite;
}

/* Tabular nums */
.tabular-nums {
  font-variant-numeric: tabular-nums;
  letter-spacing: 0;
}

@keyframes keyUsageTypewriter {
  0%,
  16% {
    max-width: 0;
  }
  52%,
  80% {
    max-width: 22ch;
  }
  100% {
    max-width: 0;
  }
}

@keyframes keyUsageCaret {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.18;
  }
}

@keyframes keyUsageSignalPulse {
  0%,
  100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-2px);
  }
}

@media (max-width: 640px) {
  .key-usage-panel {
    border-width: 3px;
    border-radius: 24px;
    box-shadow: 7px 7px 0 rgba(33, 31, 28, 0.92);
  }

  .key-usage-panel--hero,
  .key-usage-panel--art {
    padding: 1rem;
  }

  .key-usage-title {
    max-width: 12ch;
    font-size: clamp(2.1rem, 13vw, 2.95rem);
    line-height: 0.98;
    text-shadow: 3px 3px 0 #211f1c;
  }

  .key-usage-lead {
    font-size: 0.92rem;
    line-height: 1.72;
  }

  .key-usage-hero-console {
    border-width: 2px;
    border-radius: 22px;
    padding: 0.9rem;
    box-shadow: 5px 5px 0 rgba(33, 31, 28, 0.9);
  }

  .query-input-frame {
    gap: 0.55rem;
    border-width: 3px;
    border-radius: 20px;
    padding: 0.58rem;
    box-shadow: 5px 5px 0 rgba(33, 31, 28, 0.92);
  }

  .query-input-icon,
  .query-visibility-btn {
    width: 2.35rem;
    height: 2.35rem;
    border-radius: 14px;
  }

  .query-placeholder {
    left: 3.65rem;
    right: 3.65rem;
    font-size: 0.82rem;
  }

  .query-submit-btn {
    width: 100%;
  }

  .key-usage-signal-strip {
    width: 100%;
    justify-content: center;
  }

  .key-usage-mini-card,
  .key-usage-stat {
    border-radius: 18px;
    box-shadow: 4px 4px 0 rgba(33, 31, 28, 0.88);
  }
}
</style>
