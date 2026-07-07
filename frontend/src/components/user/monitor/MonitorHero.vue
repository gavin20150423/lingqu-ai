<template>
  <section class="lingqu-console-hero monitor-hero">
    <div class="monitor-hero__copy">
      <span class="lingqu-console-eyebrow">Channel Watch</span>
      <h1>{{ t('channelStatus.title') }}</h1>
      <p>自动观察各模型通道的可用率和延迟。你只需要看哪条线路稳定、哪条线路最快，Key 会继续走灵渠AI调度。</p>

      <div class="monitor-hero__badges" aria-label="渠道状态优势">
        <span>
          <Icon name="shield" size="sm" />
          稳定优先
        </span>
        <span>
          <Icon name="bolt" size="sm" />
          延迟可见
        </span>
        <span>
          <Icon name="server" size="sm" />
          {{ intervalSeconds }}s 巡检
        </span>
      </div>
    </div>

    <div class="monitor-hero__panel">
      <span class="monitor-hero__status" :class="overallChipClass">
        <span :class="overallDotClass"></span>
        {{ overallLabel }}
      </span>

      <div role="tablist" class="monitor-hero__tabs" aria-label="状态时间窗口">
        <button
          v-for="opt in windowOptions"
          :key="opt.value"
          type="button"
          role="tab"
          :aria-selected="window === opt.value"
          :class="{ 'monitor-hero__tab--active': window === opt.value }"
          @click="emit('update:window', opt.value)"
        >
          {{ opt.label }}
        </button>
      </div>

      <div class="monitor-hero__actions">
        <button
          type="button"
          class="monitor-hero__refresh"
          :disabled="loading"
          :title="t('common.refresh')"
          @click="emit('refresh')"
        >
          <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
        </button>

        <AutoRefreshButton
          v-if="autoRefresh"
          :enabled="autoRefresh.enabled.value"
          :interval-seconds="autoRefresh.intervalSeconds.value"
          :countdown="autoRefresh.countdown.value"
          :intervals="autoRefresh.intervals"
          @update:enabled="autoRefresh.setEnabled"
          @update:interval="autoRefresh.setInterval"
        />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import AutoRefreshButton from '@/components/common/AutoRefreshButton.vue'
export type MonitorWindow = '7d' | '15d' | '30d'
export type OverallStatus = 'operational' | 'degraded'

const props = defineProps<{
  overallStatus: OverallStatus
  intervalSeconds: number
  window: MonitorWindow
  loading: boolean
  autoRefresh?: {
    enabled: { value: boolean }
    intervalSeconds: { value: number }
    countdown: { value: number }
    intervals: readonly number[]
    setEnabled: (v: boolean) => void
    setInterval: (v: number) => void
  }
}>()

const emit = defineEmits<{
  (e: 'update:window', value: MonitorWindow): void
  (e: 'refresh'): void
}>()

const { t } = useI18n()

const windowOptions = computed<{ value: MonitorWindow; label: string }[]>(() => [
  { value: '7d', label: t('channelStatus.windowTab.7d') },
  { value: '15d', label: t('channelStatus.windowTab.15d') },
  { value: '30d', label: t('channelStatus.windowTab.30d') },
])

const overallLabel = computed(() => t(`channelStatus.overall.${props.overallStatus}`))

const overallChipClass = computed(() => {
  switch (props.overallStatus) {
    case 'operational':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
    case 'degraded':
    default:
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'
  }
})

const overallDotClass = computed(() => {
  switch (props.overallStatus) {
    case 'operational':
      return 'monitor-hero__dot monitor-hero__dot--ok'
    case 'degraded':
    default:
      return 'monitor-hero__dot monitor-hero__dot--warn'
  }
})

</script>

<style scoped>
.monitor-hero {
  margin-bottom: 0.8rem;
  border-color: rgba(33, 31, 28, 0.1);
  background:
    radial-gradient(circle at 92% 12%, rgba(69, 213, 209, 0.13), transparent 28%),
    rgba(255, 255, 255, 0.84);
  box-shadow: 0 1px 0 rgba(255, 255, 255, 0.9) inset, 0 14px 34px rgba(33, 31, 28, 0.045);
  padding: 1.05rem 1.15rem;
}

.monitor-hero__copy {
  min-width: 0;
}

.monitor-hero__badges {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
  margin-top: 0.7rem;
}

.monitor-hero__badges span,
.monitor-hero__status {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  border: 1px solid rgba(33, 31, 28, 0.12);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.76);
  box-shadow: none;
  color: rgba(33, 31, 28, 0.78);
  padding: 0.28rem 0.62rem;
  font-size: 0.74rem;
  font-weight: 850;
}

.monitor-hero__panel {
  min-width: min(100%, 21rem);
  display: grid;
  gap: 0.5rem;
  justify-items: end;
}

.monitor-hero__status {
  text-transform: uppercase;
}

.monitor-hero__dot {
  width: 0.55rem;
  height: 0.55rem;
  border-radius: 999px;
  animation: monitorPulse 1.5s ease-in-out infinite;
}

.monitor-hero__dot--ok {
  background: #2ecf9f;
}

.monitor-hero__dot--warn {
  background: #ffd447;
}

.monitor-hero__tabs {
  display: inline-flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 0.35rem;
  border: 1px solid rgba(33, 31, 28, 0.1);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.72);
  box-shadow: 0 1px 0 rgba(255, 255, 255, 0.9) inset;
  padding: 0.24rem;
}

.monitor-hero__tabs button {
  min-height: 1.85rem;
  border-radius: 12px;
  color: rgba(33, 31, 28, 0.58);
  padding: 0 0.68rem;
  font-size: 0.8rem;
  font-weight: 850;
  transition: background 150ms ease, color 150ms ease, box-shadow 150ms ease;
}

.monitor-hero__tabs button:hover,
.monitor-hero__tab--active {
  background: #fff0bd;
  color: #211f1c;
  box-shadow: inset 0 0 0 1px rgba(33, 31, 28, 0.06);
}

.monitor-hero__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 0.45rem;
}

.monitor-hero__refresh {
  width: 2.15rem;
  height: 2.15rem;
  display: grid;
  place-items: center;
  border: 1px solid rgba(33, 31, 28, 0.12);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.8);
  box-shadow: 0 1px 0 rgba(255, 255, 255, 0.9) inset;
  color: rgba(33, 31, 28, 0.76);
  transition: transform 150ms ease, box-shadow 150ms ease;
}

.monitor-hero__refresh:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 8px 18px rgba(33, 31, 28, 0.075);
}

.monitor-hero__refresh:disabled {
  opacity: 0.65;
}

.monitor-hero :deep(.relative > button) {
  min-height: 2.15rem;
  border: 1px solid rgba(33, 31, 28, 0.12);
  border-radius: 14px;
  background: #fff7d0;
  box-shadow: 0 1px 0 rgba(255, 255, 255, 0.9) inset;
  color: rgba(33, 31, 28, 0.78);
  font-weight: 850;
}

.monitor-hero :deep(.absolute) {
  border: 1px solid rgba(33, 31, 28, 0.12);
  border-radius: 16px;
  box-shadow: 0 14px 32px rgba(33, 31, 28, 0.11);
}

@media (max-width: 900px) {
  .monitor-hero {
    padding: 0.8rem;
  }

  .monitor-hero__panel {
    justify-items: start;
  }

  .monitor-hero__actions,
  .monitor-hero__tabs {
    justify-content: flex-start;
  }
}

@keyframes monitorPulse {
  0%,
  100% {
    transform: scale(1);
    opacity: 1;
  }
  50% {
    transform: scale(1.35);
    opacity: 0.65;
  }
}
</style>
