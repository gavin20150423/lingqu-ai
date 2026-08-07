import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { MonitorTimelinePoint } from '@/api/channelMonitor'
import MonitorTimeline from './MonitorTimeline.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string, params?: { n?: number }) => `${key}:${params?.n ?? ''}` }),
}))

function point(checkedAt: string): MonitorTimelinePoint {
  return {
    status: 'operational',
    latency_ms: 120,
    ping_latency_ms: 8,
    checked_at: checkedAt,
  }
}

describe('MonitorTimeline', () => {
  afterEach(() => vi.useRealTimers())

  it('does not render old history as activity in the last hour', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-05T10:00:00Z'))

    const wrapper = mount(MonitorTimeline, {
      props: {
        buckets: [point('2026-07-25T02:18:00Z')],
        countdownSeconds: 60,
        length: 3,
      },
    })

    const bars = wrapper.findAll('.monitor-timeline__bar')
    expect(bars).toHaveLength(3)
    expect(bars.every(bar => bar.classes().includes('bg-gray-300'))).toBe(true)
  })

  it('carries a sample forward across the remaining time buckets', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-05T10:00:00Z'))

    const wrapper = mount(MonitorTimeline, {
      props: {
        buckets: [point('2026-08-05T09:30:00Z')],
        countdownSeconds: 60,
        length: 3,
      },
    })

    const bars = wrapper.findAll('.monitor-timeline__bar')
    expect(bars).toHaveLength(3)
    expect(bars[0].classes()).toContain('bg-gray-300')
    expect(bars[1].classes()).toContain('monitor-timeline__bar--ok')
    expect(bars[2].classes()).toContain('monitor-timeline__bar--ok')
  })
})
