<template>
  <button
    type="button"
    class="monitor-card group"
    @click="emit('click')"
  >
    <!-- Header: icon + name/model + status chip -->
    <div class="monitor-card__header">
      <span
        class="monitor-card__provider"
        :class="[providerGradient(item.provider), providerTintClass]"
      >
        <ProviderIcon :provider="item.provider" :size="20" />
      </span>
      <div class="monitor-card__title-wrap">
        <div class="monitor-card__title">
          {{ item.name }}
        </div>
        <div class="monitor-card__meta">
          <span
            class="inline-flex items-center rounded-md px-1.5 py-0.5 text-[10px] font-medium flex-shrink-0"
            :class="providerBadgeClass(item.provider)"
          >
            {{ providerLabel(item.provider) }}
          </span>
          <span class="monitor-card__model">
            {{ item.primary_model }}
          </span>
          <span
            v-if="item.group_name"
            class="inline-flex items-center rounded-md px-1.5 py-0.5 text-[10px] font-medium bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300 flex-shrink-0"
          >
            {{ item.group_name }}
          </span>
        </div>
      </div>
      <span
        class="monitor-card__status"
        :class="statusBadgeClass(item.primary_status)"
      >
        {{ statusLabel(item.primary_status) }}
      </span>
    </div>

    <!-- Metrics -->
    <MonitorMetricPair
      primary-icon="bolt"
      :primary-label="t('monitorCommon.dialogLatency')"
      :primary-value="formatLatency(item.primary_latency_ms)"
      primary-unit="ms"
      secondary-icon="globe"
      :secondary-label="t('monitorCommon.endpointPing')"
      :secondary-value="formatLatency(item.primary_ping_latency_ms)"
      secondary-unit="ms"
    />

    <!-- Divider -->
    <div class="monitor-card__divider"></div>

    <!-- Availability row -->
    <MonitorAvailabilityRow
      :window-label="availabilityLabel"
      :value="availabilityValue"
      :samples-label="extraModelsCountLabel"
    />

    <!-- Timeline -->
    <MonitorTimeline
      :buckets="item.timeline"
      :countdown-seconds="countdownSeconds"
    />
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UserMonitorView } from '@/api/channelMonitor'
import {
  useChannelMonitorFormat,
  providerGradient,
} from '@/composables/useChannelMonitorFormat'
import ProviderIcon from './ProviderIcon.vue'
import MonitorMetricPair from './MonitorMetricPair.vue'
import MonitorAvailabilityRow from './MonitorAvailabilityRow.vue'
import MonitorTimeline from './MonitorTimeline.vue'

const PROVIDER_TINT: Record<string, string> = {
  openai: 'text-emerald-600 dark:text-emerald-300',
  anthropic: 'text-orange-600 dark:text-orange-300',
  gemini: 'text-sky-600 dark:text-sky-300',
}

const props = defineProps<{
  item: UserMonitorView
  window: '7d' | '15d' | '30d'
  availabilityValue: number | null
  countdownSeconds: number
}>()

const emit = defineEmits<{
  (e: 'click'): void
}>()

const { t } = useI18n()
const {
  statusLabel,
  statusBadgeClass,
  providerLabel,
  providerBadgeClass,
  formatLatency,
} = useChannelMonitorFormat()

const providerTintClass = computed(() =>
  PROVIDER_TINT[props.item.provider] ?? 'text-gray-500 dark:text-gray-300'
)

const availabilityLabel = computed(() => {
  const win = t(`channelStatus.windowTab.${props.window}`)
  return `${t('monitorCommon.availabilityPrefix')} · ${win}`
})

const extraModelsCountLabel = computed(() => {
  const count = props.item.extra_models?.length ?? 0
  if (count === 0) return undefined
  return t('monitorCommon.extraModelsCount', { n: count })
})
</script>

<style scoped>
.monitor-card {
  display: flex;
  min-height: 17.5rem;
  width: 100%;
  flex-direction: column;
  border: 1px solid rgba(33, 31, 28, 0.1);
  border-radius: 1.05rem;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.88), rgba(255, 255, 255, 0.76));
  padding: 1.2rem;
  text-align: left;
  box-shadow: 0 1px 0 rgba(255, 255, 255, 0.88) inset, 0 10px 26px rgba(33, 31, 28, 0.045);
  backdrop-filter: blur(18px);
  transition: border-color 160ms ease, box-shadow 160ms ease, transform 160ms ease;
}

.monitor-card:hover {
  border-color: rgba(33, 31, 28, 0.16);
  box-shadow: 0 1px 0 rgba(255, 255, 255, 0.9) inset, 0 14px 32px rgba(33, 31, 28, 0.065);
  transform: translateY(-1px);
}

.monitor-card__header {
  display: flex;
  align-items: flex-start;
  gap: 0.72rem;
}

.monitor-card__provider {
  display: grid;
  width: 2.32rem;
  height: 2.32rem;
  flex: 0 0 auto;
  place-items: center;
  border: 1px solid rgba(33, 31, 28, 0.07);
  border-radius: 0.8rem;
}

.monitor-card__title-wrap {
  min-width: 0;
  flex: 1 1 auto;
}

.monitor-card__title {
  overflow: hidden;
  color: #1f2937;
  font-size: 0.98rem;
  font-weight: 760;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.monitor-card__meta {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.38rem;
  margin-top: 0.28rem;
}

.monitor-card__model {
  min-width: 0;
  overflow: hidden;
  color: rgba(75, 85, 99, 0.76);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 0.75rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.monitor-card__status {
  flex: 0 0 auto;
  border-radius: 999px;
  padding: 0.24rem 0.58rem;
  font-size: 0.72rem;
  font-weight: 750;
}

.monitor-card__divider {
  margin-top: 1rem;
  border-top: 1px solid rgba(148, 163, 184, 0.14);
}

:global(.dark) .monitor-card {
  border-color: rgb(51 65 85 / 0.7);
  background: rgb(30 41 59 / 0.62);
  box-shadow: 0 12px 30px rgba(0, 0, 0, 0.18);
}

:global(.dark) .monitor-card:hover {
  border-color: rgb(148 163 184 / 0.35);
  box-shadow: 0 16px 36px rgba(0, 0, 0, 0.24);
}

:global(.dark) .monitor-card__provider {
  border-color: rgb(255 255 255 / 0.08);
}

:global(.dark) .monitor-card__title {
  color: rgb(243 244 246);
}

:global(.dark) .monitor-card__model {
  color: rgb(156 163 175);
}

:global(.dark) .monitor-card__divider {
  border-color: rgb(51 65 85 / 0.55);
}
</style>
