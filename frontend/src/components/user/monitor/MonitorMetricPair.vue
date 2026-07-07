<template>
  <div class="monitor-metrics">
    <div
      class="monitor-metric"
    >
      <div
        class="monitor-metric__label"
      >
        <Icon :name="primaryIcon" size="xs" />
        <span>{{ primaryLabel }}</span>
      </div>
      <div class="monitor-metric__value">
        {{ primaryValue }}<span>{{ primaryUnit }}</span>
      </div>
    </div>
    <div
      class="monitor-metric"
    >
      <div
        class="monitor-metric__label"
      >
        <Icon :name="secondaryIcon" size="xs" />
        <span>{{ secondaryLabel }}</span>
      </div>
      <div class="monitor-metric__value">
        {{ secondaryValue }}<span>{{ secondaryUnit }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  primaryLabel: string
  primaryValue: string
  primaryUnit: string
  primaryIcon: 'bolt' | 'globe' | 'clock' | 'link'
  secondaryLabel: string
  secondaryValue: string
  secondaryUnit: string
  secondaryIcon: 'bolt' | 'globe' | 'clock' | 'link'
}>()
</script>

<style scoped>
.monitor-metrics {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.55rem;
  margin-top: 1.25rem;
}

.monitor-metric {
  border: 1px solid rgba(148, 163, 184, 0.16);
  border-radius: 0.85rem;
  background: rgba(248, 250, 252, 0.56);
  padding: 0.78rem;
}

.monitor-metric__label {
  display: flex;
  align-items: center;
  gap: 0.38rem;
  color: rgba(107, 114, 128, 0.64);
  font-size: 0.64rem;
  font-weight: 720;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.monitor-metric__value {
  margin-top: 0.42rem;
  color: #111827;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 1.03rem;
  font-weight: 760;
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}

.monitor-metric__value span {
  margin-left: 0.12rem;
  color: rgba(107, 114, 128, 0.58);
  font-size: 0.72rem;
  font-weight: 520;
}

:global(.dark) .monitor-metric {
  border-color: rgb(51 65 85 / 0.65);
  background: rgb(15 23 42 / 0.28);
}

:global(.dark) .monitor-metric__label {
  color: rgb(156 163 175 / 0.78);
}

:global(.dark) .monitor-metric__value {
  color: rgb(243 244 246);
}
</style>
