<template>
  <section class="monitor-hero">
    <div class="monitor-hero__overview">
      <span class="monitor-hero__signal" :class="`monitor-hero__signal--${summaryTone}`">
        <Icon :name="summaryTone === 'ok' ? 'check' : 'bolt'" size="md" />
      </span>
      <div>
        <h1>{{ summaryTitle }}</h1>
        <p>共 {{ total }} 个渠道分组 · 实时监测</p>
      </div>
    </div>

    <div class="monitor-hero__counts" aria-label="渠道状态统计">
      <span class="monitor-hero__count monitor-hero__count--ok">
        <i></i><strong>{{ counts.operational }}</strong> 正常
      </span>
      <span class="monitor-hero__count monitor-hero__count--warn">
        <i></i><strong>{{ counts.degraded }}</strong> 波动
      </span>
      <span class="monitor-hero__count monitor-hero__count--bad">
        <i></i><strong>{{ counts.failed }}</strong> 异常
      </span>
    </div>

    <div class="monitor-hero__refresh-control" aria-label="自动刷新间隔">
      <button
        type="button"
        :class="{ 'monitor-hero__interval--active': !autoRefresh?.enabled.value }"
        :disabled="loading"
        @click="handleManual"
      >
        手动
      </button>
      <button
        v-for="seconds in autoRefresh?.intervals ?? []"
        :key="seconds"
        type="button"
        :class="{
          'monitor-hero__interval--active':
            autoRefresh?.enabled.value && autoRefresh.intervalSeconds.value === seconds,
        }"
        @click="setRefreshInterval(seconds)"
      >
        {{ seconds }}s
      </button>
      <button
        type="button"
        class="monitor-hero__refresh"
        :disabled="loading"
        title="立即刷新"
        @click="emit('refresh')"
      >
        <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Icon from '@/components/icons/Icon.vue'

export type MonitorWindow = '7d' | '15d' | '30d'
export type OverallStatus = 'operational' | 'degraded'

const props = defineProps<{
  overallStatus: OverallStatus
  total: number
  counts: {
    operational: number
    degraded: number
    failed: number
  }
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
  (e: 'refresh'): void
}>()

const summaryTone = computed(() => {
  if (props.counts.failed > 0) return 'bad'
  if (props.overallStatus === 'degraded') return 'warn'
  return 'ok'
})

const summaryTitle = computed(() => {
  if (props.counts.failed > 0) return '部分异常'
  if (props.overallStatus === 'degraded') return '部分波动'
  return '全部正常'
})

function handleManual() {
  props.autoRefresh?.setEnabled(false)
  emit('refresh')
}

function setRefreshInterval(seconds: number) {
  props.autoRefresh?.setInterval(seconds)
  props.autoRefresh?.setEnabled(true)
}
</script>

<style scoped>
.monitor-hero {
  display: grid;
  grid-template-columns: minmax(15rem, 1fr) auto minmax(17rem, 1fr);
  align-items: center;
  gap: 1.25rem;
  margin-bottom: 0.7rem;
  border: 1px solid #e1ded7;
  border-radius: 8px;
  background: #fff;
  padding: 0.85rem 1rem;
  box-shadow: 0 3px 12px rgba(61, 53, 43, 0.05);
}

.monitor-hero__overview {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.78rem;
}

.monitor-hero__signal {
  display: grid;
  width: 2.5rem;
  height: 2.5rem;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 7px;
}

.monitor-hero__signal--ok {
  background: #edf6ef;
  color: #3f8055;
}

.monitor-hero__signal--warn {
  background: #fbf3df;
  color: #b47a13;
}

.monitor-hero__signal--bad {
  background: #fae9e6;
  color: #bd453d;
}

.monitor-hero h1 {
  margin: 0;
  color: #272521;
  font-size: 1rem;
  font-weight: 780;
  line-height: 1.35;
}

.monitor-hero p {
  margin: 0.14rem 0 0;
  color: #777169;
  font-size: 0.7rem;
  line-height: 1.4;
}

.monitor-hero__counts {
  display: flex;
  align-items: center;
  gap: 0.45rem;
}

.monitor-hero__count {
  display: inline-flex;
  min-height: 1.8rem;
  align-items: center;
  gap: 0.3rem;
  border-radius: 6px;
  padding: 0 0.58rem;
  color: #716b63;
  font-size: 0.67rem;
  white-space: nowrap;
}

.monitor-hero__count i {
  width: 0.38rem;
  height: 0.38rem;
  border-radius: 50%;
}

.monitor-hero__count strong {
  font-size: 0.75rem;
}

.monitor-hero__count--ok {
  background: #f0f5f0;
}

.monitor-hero__count--ok i {
  background: #4c8a5d;
}

.monitor-hero__count--warn {
  background: #fbf5e8;
}

.monitor-hero__count--warn i {
  background: #c88b18;
}

.monitor-hero__count--bad {
  background: #faece9;
}

.monitor-hero__count--bad i {
  background: #ca4841;
}

.monitor-hero__refresh-control {
  display: flex;
  justify-self: end;
  align-items: center;
  gap: 0.14rem;
  border-radius: 6px;
  background: #f7f5f1;
  padding: 0.18rem;
}

.monitor-hero__refresh-control button {
  min-width: 2.35rem;
  height: 1.78rem;
  border-radius: 5px;
  color: #706a62;
  padding: 0 0.5rem;
  font-size: 0.67rem;
  font-weight: 700;
}

.monitor-hero__refresh-control button:hover {
  background: #fff;
  color: #3c3934;
}

.monitor-hero__refresh-control .monitor-hero__interval--active {
  background: #c96342;
  color: #fff;
}

.monitor-hero__refresh {
  display: grid;
  margin-left: 0.28rem;
  border: 1px solid #ded9d0;
  background: #fff;
  place-items: center;
}

@media (max-width: 980px) {
  .monitor-hero {
    grid-template-columns: 1fr auto;
  }

  .monitor-hero__counts {
    grid-column: 1 / -1;
    grid-row: 2;
  }
}

@media (max-width: 620px) {
  .monitor-hero {
    grid-template-columns: 1fr;
  }

  .monitor-hero__refresh-control {
    grid-row: 3;
    justify-self: stretch;
  }

  .monitor-hero__refresh-control button {
    flex: 1;
  }
}
</style>
