<template>
  <div class="monitor-timeline">
    <div class="monitor-timeline__meta">
      <span>近 1 小时</span>
      <span class="monitor-timeline__idle-key"><i></i>灰色为空闲</span>
    </div>

    <div
      v-if="maintenance"
      class="monitor-timeline__maintenance"
    >
      {{ t('monitorCommon.maintenancePaused') }}
    </div>
    <div v-else class="monitor-timeline__bars">
      <div
        v-for="(bar, idx) in displayBars"
        :key="idx"
        class="monitor-timeline__bar"
        :class="bar.colorClass"
        :style="{ height: bar.heightPct + '%' }"
        :title="bar.title"
        @mouseenter="showTooltip(bar, idx)"
        @mouseleave="hideTooltip"
      ></div>
    </div>

    <div
      v-if="hoveredBar"
      class="monitor-timeline__tooltip"
      :style="{ left: tooltipLeft }"
      role="tooltip"
    >
      {{ hoveredBar.title }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { MonitorTimelinePoint } from '@/api/channelMonitor'
import { useChannelMonitorFormat } from '@/composables/useChannelMonitorFormat'

const props = withDefaults(defineProps<{
  buckets?: MonitorTimelinePoint[]
  countdownSeconds: number
  length?: number
  maintenance?: boolean
}>(), {
  buckets: () => [],
  length: 60,
  maintenance: false,
})

const { t } = useI18n()
const { statusLabel, formatLatency, formatRelativeTime } = useChannelMonitorFormat()
const hoveredBar = ref<Bar | null>(null)
const hoveredIndex = ref(0)

interface Bar {
  colorClass: string
  heightPct: number
  title: string
}

// Height and colour encode health. Missing traffic slots are rendered as idle.
const STATUS_HEIGHT: Record<string, number> = {
  operational: 100,
  degraded: 65,
  failed: 35,
  error: 35,
  empty: 30,
}

const STATUS_COLOR: Record<string, string> = {
  operational: 'monitor-timeline__bar--ok',
  degraded: 'monitor-timeline__bar--warn',
  failed: 'monitor-timeline__bar--bad',
  error: 'monitor-timeline__bar--bad',
  empty: 'bg-gray-300 dark:bg-dark-600',
}

const displayBars = computed<Bar[]>(() => {
  // Real points come newest-first; convert to oldest-first so the rightmost
  // bar represents "now". Pad the left with empty placeholders to keep the
  // bar count stable at `length`.
  const real = [...(props.buckets ?? [])]
    .slice(0, props.length)
    .reverse()

  const padCount = Math.max(0, props.length - real.length)
  const bars: Bar[] = []

  for (let i = 0; i < padCount; i += 1) {
    bars.push({
      colorClass: STATUS_COLOR.empty,
      heightPct: STATUS_HEIGHT.empty,
      title: '空闲',
    })
  }

  for (const point of real) {
    const status = point.status as keyof typeof STATUS_HEIGHT
    const colorClass = STATUS_COLOR[status] ?? STATUS_COLOR.empty
    const heightPct = STATUS_HEIGHT[status] ?? STATUS_HEIGHT.empty
    const latency = formatLatency(point.latency_ms)
    const relative = formatRelativeTime(point.checked_at)
    const label = statusLabel(point.status)
    bars.push({
      colorClass,
      heightPct,
      title: `${relative} · ${label} · ${latency}ms`,
    })
  }

  return bars
})

const tooltipLeft = computed(() => {
  const total = displayBars.value.length || 1
  const percent = ((hoveredIndex.value + 0.5) / total) * 100
  return `${Math.min(88, Math.max(12, percent))}%`
})

function showTooltip(bar: Bar, index: number) {
  hoveredBar.value = bar
  hoveredIndex.value = index
}

function hideTooltip() {
  hoveredBar.value = null
}
</script>

<style scoped>
.monitor-timeline {
  position: relative;
  margin-top: auto;
  padding-top: 0.68rem;
}

.monitor-timeline__meta {
  display: flex;
  justify-content: space-between;
  margin-bottom: 0.42rem;
  color: #7d776f;
  font-size: 0.58rem;
  font-weight: 650;
  letter-spacing: 0;
}

.monitor-timeline__idle-key {
  display: inline-flex;
  align-items: center;
  gap: 0.24rem;
}

.monitor-timeline__idle-key i {
  width: 0.42rem;
  height: 0.42rem;
  background: #d8d3ca;
}

.monitor-timeline__maintenance {
  display: flex;
  width: 100%;
  height: 1rem;
  align-items: center;
  justify-content: center;
  border: 1px dashed rgba(148, 163, 184, 0.56);
  border-radius: 0.45rem;
  color: rgba(107, 114, 128, 0.66);
  font-size: 0.62rem;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.monitor-timeline__bars {
  display: flex;
  width: 100%;
  height: 1rem;
  align-items: flex-end;
  gap: 2px;
}

.monitor-timeline__bar {
  min-width: 2px;
  flex: 1 1 0;
  border-radius: 1px;
  cursor: help;
  transition: filter 120ms ease, transform 120ms ease;
}

.monitor-timeline__bar:hover {
  filter: saturate(1.18) brightness(0.9);
  transform: scaleY(1.12);
  transform-origin: bottom;
}

.monitor-timeline__bar--ok {
  background: #10b981;
}

.monitor-timeline__bar--warn {
  background: #f5b74a;
}

.monitor-timeline__bar--bad {
  background: #ef6b64;
}

.monitor-timeline__tooltip {
  position: absolute;
  z-index: 12;
  bottom: 1.28rem;
  max-width: calc(100% - 0.5rem);
  transform: translateX(-50%);
  border: 1px solid #d9d4cb;
  border-radius: 5px;
  background: #2f2c28;
  box-shadow: 0 5px 14px rgba(47, 44, 40, 0.18);
  color: #fff;
  padding: 0.3rem 0.45rem;
  font-size: 0.58rem;
  font-weight: 650;
  line-height: 1.35;
  pointer-events: none;
  white-space: nowrap;
}

.monitor-timeline__tooltip::after {
  position: absolute;
  bottom: -0.25rem;
  left: 50%;
  width: 0.45rem;
  height: 0.45rem;
  background: #2f2c28;
  content: '';
  transform: translateX(-50%) rotate(45deg);
}

:global(.dark) .monitor-timeline {
  border-color: rgb(51 65 85 / 0.55);
}

:global(.dark) .monitor-timeline__meta,
:global(.dark) .monitor-timeline__maintenance {
  color: rgb(156 163 175 / 0.72);
}
</style>
