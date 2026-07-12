<template>
  <button
    type="button"
    class="monitor-card"
    :class="statusClass"
    @click="emit('click')"
  >
    <div class="monitor-card__header">
      <span class="monitor-card__provider" :class="providerTintClass">
        <ProviderIcon :provider="item.provider" :size="18" />
      </span>
      <div class="monitor-card__title-wrap">
        <div class="monitor-card__title">{{ item.name }}</div>
        <div class="monitor-card__breadcrumb">
          {{ providerLabel(item.provider) }}
          <span>›</span>
          {{ item.group_name || '默认组' }}
          <span>›</span>
          {{ item.primary_model }}
        </div>
      </div>
      <span class="monitor-card__status" :class="statusClass">
        <i></i>{{ statusLabel(item.primary_status) }}
      </span>
    </div>

    <div class="monitor-card__primary">
      <div>
        <span class="monitor-card__eyebrow">可用率 · {{ windowLabel }}</span>
        <strong class="monitor-card__availability" :style="availabilityStyle">
          {{ availabilityText }}<small v-if="availabilityValue !== null">%</small>
        </strong>
      </div>
      <dl class="monitor-card__latencies">
        <div>
          <dt>延迟</dt>
          <dd>{{ latencyText }}</dd>
        </div>
        <div>
          <dt>首字</dt>
          <dd>{{ pingText }}</dd>
        </div>
      </dl>
    </div>

    <div class="monitor-card__metrics">
      <div>
        <span>模型数</span>
        <strong>{{ modelCount }}</strong>
      </div>
      <div>
        <span>响应延迟</span>
        <strong>{{ latencyText }}</strong>
      </div>
      <div>
        <span>端点 PING</span>
        <strong>{{ pingText }}</strong>
      </div>
    </div>

    <MonitorTimeline
      :buckets="item.timeline"
      :countdown-seconds="countdownSeconds"
    />
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { UserMonitorView } from '@/api/channelMonitor'
import { hslForPct, useChannelMonitorFormat } from '@/composables/useChannelMonitorFormat'
import ProviderIcon from './ProviderIcon.vue'
import MonitorTimeline from './MonitorTimeline.vue'

const props = defineProps<{
  item: UserMonitorView
  window: '7d' | '15d' | '30d'
  availabilityValue: number | null
  countdownSeconds: number
}>()

const emit = defineEmits<{
  (e: 'click'): void
}>()

const { statusLabel, providerLabel, formatLatency } = useChannelMonitorFormat()

const providerTintClass = computed(() => `monitor-card__provider--${props.item.provider}`)
const statusClass = computed(() => `monitor-card--${props.item.primary_status}`)
const windowLabel = computed(() => ({ '7d': '7天', '15d': '15天', '30d': '30天' })[props.window])
const modelCount = computed(() => 1 + (props.item.extra_models?.length ?? 0))
const latencyText = computed(() => {
  const value = formatLatency(props.item.primary_latency_ms)
  return value === '-' ? value : `${value}ms`
})
const pingText = computed(() => {
  const value = formatLatency(props.item.primary_ping_latency_ms)
  return value === '-' ? value : `${value}ms`
})
const availabilityText = computed(() => {
  if (props.availabilityValue === null || Number.isNaN(props.availabilityValue)) return '-'
  return props.availabilityValue.toFixed(1)
})
const availabilityStyle = computed(() => {
  if (props.item.primary_status === 'failed' || props.item.primary_status === 'error') {
    return { color: '#bd4039' }
  }
  if (props.item.primary_status === 'degraded') {
    return { color: '#b37913' }
  }
  const color = hslForPct(props.availabilityValue)
  return color ? { color } : undefined
})
</script>

<style scoped>
.monitor-card {
  position: relative;
  display: flex;
  min-height: 14.2rem;
  width: 100%;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid #e2ded7;
  border-top: 3px solid #669173;
  border-radius: 7px;
  background: #fff;
  padding: 0.78rem;
  text-align: left;
  box-shadow: 0 5px 14px rgba(58, 50, 41, 0.06);
  transition: border-color 150ms ease, box-shadow 150ms ease, transform 150ms ease;
}

.monitor-card:hover {
  box-shadow: 0 9px 22px rgba(58, 50, 41, 0.1);
  transform: translateY(-1px);
}

.monitor-card--degraded {
  border-top-color: #bf8218;
}

.monitor-card--failed,
.monitor-card--error {
  border-top-color: #c64c43;
}

.monitor-card__header {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  gap: 0.56rem;
}

.monitor-card__provider {
  display: grid;
  width: 1.75rem;
  height: 1.75rem;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 6px;
  background: #f5f3ee;
  color: #4b4945;
}

.monitor-card__provider--openai {
  color: #31755a;
}

.monitor-card__provider--anthropic {
  color: #a85e39;
}

.monitor-card__provider--gemini {
  color: #47789a;
}

.monitor-card__title-wrap {
  min-width: 0;
  flex: 1;
}

.monitor-card__title {
  overflow: hidden;
  color: #292723;
  font-size: 0.78rem;
  font-weight: 780;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.monitor-card__breadcrumb {
  overflow: hidden;
  margin-top: 0.14rem;
  color: #817b73;
  font-size: 0.6rem;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.monitor-card__breadcrumb span {
  margin: 0 0.18rem;
  color: #b0aaa1;
}

.monitor-card__status {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 0.25rem;
  border-radius: 999px;
  background: #eef5ef;
  color: #477a55;
  padding: 0.18rem 0.42rem;
  font-size: 0.6rem;
  font-weight: 750;
}

.monitor-card__status i {
  width: 0.34rem;
  height: 0.34rem;
  border-radius: 50%;
  background: currentColor;
}

.monitor-card__status.monitor-card--degraded {
  background: #fcf6e5;
  color: #ad7817;
}

.monitor-card__status.monitor-card--failed,
.monitor-card__status.monitor-card--error {
  background: #fbecea;
  color: #bb443d;
}

.monitor-card__primary {
  display: flex;
  align-items: end;
  justify-content: space-between;
  gap: 0.75rem;
  margin-top: 0.78rem;
}

.monitor-card__eyebrow {
  display: block;
  color: #817b73;
  font-size: 0.6rem;
}

.monitor-card__availability {
  display: block;
  margin-top: 0.08rem;
  color: #3f8055;
  font-size: 1.42rem;
  font-weight: 800;
  line-height: 1;
  font-variant-numeric: tabular-nums;
}

.monitor-card__availability small {
  margin-left: 0.04rem;
  font-size: 0.72rem;
}

.monitor-card__latencies {
  display: grid;
  gap: 0.2rem;
  margin: 0;
}

.monitor-card__latencies div {
  display: flex;
  justify-content: flex-end;
  gap: 0.4rem;
}

.monitor-card__latencies dt {
  color: #817b73;
  font-size: 0.6rem;
}

.monitor-card__latencies dd {
  min-width: 3.6rem;
  margin: 0;
  color: #4a4640;
  font-size: 0.64rem;
  font-weight: 700;
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.monitor-card__metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  margin-top: 0.72rem;
  border-radius: 6px;
  background: #f8f6f2;
  padding: 0.45rem 0;
}

.monitor-card__metrics div {
  min-width: 0;
  border-left: 1px solid #e7e2da;
  text-align: center;
}

.monitor-card__metrics div:first-child {
  border-left: 0;
}

.monitor-card__metrics span,
.monitor-card__metrics strong {
  display: block;
  overflow: hidden;
  padding: 0 0.2rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.monitor-card__metrics span {
  color: #827c74;
  font-size: 0.55rem;
}

.monitor-card__metrics strong {
  margin-top: 0.08rem;
  color: #34312d;
  font-size: 0.68rem;
  font-weight: 750;
  font-variant-numeric: tabular-nums;
}
</style>
