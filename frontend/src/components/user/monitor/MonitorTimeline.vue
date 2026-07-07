<template>
  <div class="monitor-timeline">
    <div class="monitor-timeline__meta">
      <span>{{ t('monitorCommon.history60pts', { n: length }) }}</span>
      <span class="tabular-nums">{{ t('monitorCommon.nextUpdateIn', { n: countdownSeconds }) }}</span>
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
      ></div>
    </div>

    <div
      class="monitor-timeline__range"
    >
      <span>{{ t('monitorCommon.past') }}</span>
      <span>{{ t('monitorCommon.now') }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
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

interface Bar {
  colorClass: string
  heightPct: number
  title: string
}

// 4 级高度 + 颜色双重编码：高=好+绿，短=坏+红，灰=未测试。
// 长绿(正常) > 中黄(降级) > 短红(失败/系统错误) > 很短灰(未测试)。
const STATUS_HEIGHT: Record<string, number> = {
  operational: 100,
  degraded: 65,
  failed: 35,
  error: 35,
  empty: 15,
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
      title: '',
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
</script>

<style scoped>
.monitor-timeline {
  margin-top: 0.98rem;
  border-top: 1px solid rgba(148, 163, 184, 0.14);
  padding-top: 0.72rem;
}

.monitor-timeline__meta {
  display: flex;
  justify-content: space-between;
  margin-bottom: 0.5rem;
  color: rgba(107, 114, 128, 0.64);
  font-size: 0.62rem;
  font-weight: 680;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.monitor-timeline__maintenance {
  display: flex;
  width: 100%;
  height: 1.25rem;
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
  height: 1.25rem;
  align-items: flex-end;
  gap: 2px;
}

.monitor-timeline__bar {
  min-width: 3px;
  flex: 1 1 0;
  border-radius: 999px;
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

.monitor-timeline__range {
  display: flex;
  justify-content: space-between;
  margin-top: 0.28rem;
  color: rgba(107, 114, 128, 0.5);
  font-size: 0.56rem;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

:global(.dark) .monitor-timeline {
  border-color: rgb(51 65 85 / 0.55);
}

:global(.dark) .monitor-timeline__meta,
:global(.dark) .monitor-timeline__range,
:global(.dark) .monitor-timeline__maintenance {
  color: rgb(156 163 175 / 0.72);
}
</style>
