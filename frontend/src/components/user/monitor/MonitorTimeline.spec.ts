import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { MonitorTimelinePoint } from '@/api/channelMonitor'
import MonitorTimeline from './MonitorTimeline.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string, params?: { n?: number }) => `${key}:${params?.n ?? ''}` }),
}))

function point(checkedAt: string, status: MonitorTimelinePoint['status'] = 'operational'): MonitorTimelinePoint {
  return {
    status,
    latency_ms: 120,
    ping_latency_ms: 8,
    checked_at: checkedAt,
  }
}

describe('MonitorTimeline', () => {
  afterEach(() => vi.useRealTimers())

  it('keeps an old probe result visible instead of expiring it as idle', () => {
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
    expect(bars).toHaveLength(1)
    expect(bars[0].classes()).toContain('monitor-timeline__bar--ok')
    expect(wrapper.text()).not.toContain('空闲')
  })

  it('renders exactly one bar for each real probe result', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-05T10:00:00Z'))

    const wrapper = mount(MonitorTimeline, {
      props: {
        buckets: [
          point('2026-08-05T09:30:00Z'),
          point('2026-08-05T09:45:00Z', 'failed'),
        ],
        countdownSeconds: 60,
        length: 3,
      },
    })

    const bars = wrapper.findAll('.monitor-timeline__bar')
    expect(bars).toHaveLength(2)
    expect(bars[0].classes()).toContain('monitor-timeline__bar--ok')
    expect(bars[1].classes()).toContain('monitor-timeline__bar--bad')
  })

  it('shows a probe-specific empty state before the first check', () => {
    const wrapper = mount(MonitorTimeline, {
      props: {
        buckets: [],
        countdownSeconds: 60,
        length: 3,
      },
    })

    expect(wrapper.findAll('.monitor-timeline__bar')).toHaveLength(0)
    expect(wrapper.text()).toContain('monitorCommon.noProbeResults')
  })
})
