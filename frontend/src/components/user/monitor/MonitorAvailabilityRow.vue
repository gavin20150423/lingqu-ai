<template>
  <div class="monitor-availability">
    <div class="monitor-availability__label">
      {{ windowLabel }}
    </div>
    <div class="monitor-availability__value">
      <span
        class="monitor-availability__number"
        :style="colorStyle"
      >
        {{ displayValue }}
      </span>
      <span
        class="monitor-availability__unit"
        :style="colorStyle"
      >%</span>
    </div>
  </div>
  <div
    v-if="samplesLabel"
    class="monitor-availability__samples"
  >
    {{ samplesLabel }}
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { hslForPct } from '@/composables/useChannelMonitorFormat'

const props = defineProps<{
  windowLabel: string
  value: number | null
  samplesLabel?: string
}>()

const { t } = useI18n()

const displayValue = computed(() => {
  if (props.value === null || Number.isNaN(props.value)) return t('monitorCommon.latencyEmpty')
  return props.value.toFixed(2)
})

const colorStyle = computed(() => {
  const colour = hslForPct(props.value)
  return colour ? { color: colour } : { color: 'rgb(156 163 175)' }
})
</script>

<style scoped>
.monitor-availability {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  margin-top: 0.95rem;
}

.monitor-availability__label {
  color: rgba(107, 114, 128, 0.64);
  font-size: 0.68rem;
  font-weight: 650;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.monitor-availability__value {
  display: flex;
  align-items: baseline;
  gap: 0.08rem;
}

.monitor-availability__number {
  font-size: 1.85rem;
  font-weight: 780;
  line-height: 1;
  font-variant-numeric: tabular-nums;
}

.monitor-availability__unit {
  font-size: 0.94rem;
  font-weight: 680;
  line-height: 1;
}

.monitor-availability__samples {
  margin-top: 0.35rem;
  color: rgba(107, 114, 128, 0.62);
  font-size: 0.7rem;
  text-align: right;
}

:global(.dark) .monitor-availability__label,
:global(.dark) .monitor-availability__samples {
  color: rgb(156 163 175 / 0.74);
}
</style>
