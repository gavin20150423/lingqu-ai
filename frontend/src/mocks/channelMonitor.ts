import type {
  MonitorTimelinePoint,
  UserMonitorDetail,
  UserMonitorView,
} from '@/api/channelMonitor'
import type { MonitorStatus, Provider } from '@/api/admin/channelMonitor'

interface MockMonitorSeed {
  id: number
  name: string
  provider: Provider
  groupName: string
  primaryModel: string
  status: MonitorStatus
  latency: number | null
  ping: number | null
  availability: number
  extraModels: Array<{
    model: string
    status: MonitorStatus
    latency: number | null
  }>
  degradedAt?: number[]
  failedAt?: number[]
}

const seeds: MockMonitorSeed[] = [
  {
    id: 9101,
    name: '福利特价-1',
    provider: 'openai',
    groupName: '福利特价',
    primaryModel: 'gpt-5.4-mini',
    status: 'degraded',
    latency: null,
    ping: null,
    availability: 0,
    extraModels: [
      { model: 'gpt-4.1-mini', status: 'degraded', latency: null },
    ],
    degradedAt: [0, 4, 11],
  },
  {
    id: 9102,
    name: '福利特价-pro20x（plus兜底）-1',
    provider: 'openai',
    groupName: '福利特价',
    primaryModel: 'gpt-5.4',
    status: 'degraded',
    latency: null,
    ping: null,
    availability: 0,
    extraModels: [
      { model: 'gpt-5.4-mini', status: 'degraded', latency: null },
    ],
    degradedAt: [0, 7, 18],
  },
  {
    id: 9103,
    name: '福利特价！！！-1',
    provider: 'openai',
    groupName: '福利特价！！！',
    primaryModel: 'gpt-5.4',
    status: 'degraded',
    latency: 2892,
    ping: 86,
    availability: 98.6,
    extraModels: [
      { model: 'gpt-5.4-mini', status: 'operational', latency: 968 },
    ],
    degradedAt: [0, 2, 9, 15],
    failedAt: [5, 13],
  },
  {
    id: 9104,
    name: 'aws-1',
    provider: 'anthropic',
    groupName: 'aws',
    primaryModel: 'claude-sonnet-4-6',
    status: 'degraded',
    latency: null,
    ping: null,
    availability: 0,
    extraModels: [
      { model: 'claude-haiku-4-5', status: 'degraded', latency: null },
    ],
    degradedAt: [0, 6, 17],
  },
  {
    id: 9105,
    name: 'kiro-混池-稳定',
    provider: 'anthropic',
    groupName: 'kiro',
    primaryModel: 'claude-sonnet-4-6',
    status: 'degraded',
    latency: null,
    ping: null,
    availability: 0,
    extraModels: [
      { model: 'claude-haiku-4-5', status: 'degraded', latency: null },
    ],
    degradedAt: [0, 3, 8, 12, 16],
  },
  {
    id: 9106,
    name: 'kiro-企业版-稳定',
    provider: 'anthropic',
    groupName: 'kiro',
    primaryModel: 'claude-opus-4-6',
    status: 'degraded',
    latency: null,
    ping: null,
    availability: 0,
    extraModels: [
      { model: 'claude-sonnet-4-6', status: 'degraded', latency: null },
    ],
    degradedAt: [0, 5, 19],
  },
  {
    id: 9107,
    name: 'max-稳定',
    provider: 'anthropic',
    groupName: 'max',
    primaryModel: 'claude-opus-4-6',
    status: 'degraded',
    latency: null,
    ping: null,
    availability: 0,
    extraModels: [],
    degradedAt: [0, 10, 20],
  },
  {
    id: 9108,
    name: 'plus-稳定',
    provider: 'openai',
    groupName: 'plus',
    primaryModel: 'gpt-5.4',
    status: 'degraded',
    latency: null,
    ping: null,
    availability: 0,
    extraModels: [
      { model: 'gpt-5.4-mini', status: 'degraded', latency: null },
    ],
    degradedAt: [0, 9, 18],
  },
  {
    id: 9109,
    name: 'pro-稳定',
    provider: 'openai',
    groupName: 'pro',
    primaryModel: 'gpt-5.4',
    status: 'degraded',
    latency: null,
    ping: null,
    availability: 0,
    extraModels: [
      { model: 'gpt-5.4-mini', status: 'degraded', latency: null },
    ],
    degradedAt: [0, 8, 16],
  },
]

function createTimeline(seed: MockMonitorSeed): MonitorTimelinePoint[] {
  const now = Date.now()
  return Array.from({ length: 24 }, (_, index) => {
    let status: MonitorStatus = 'operational'
    if (seed.degradedAt?.includes(index)) status = 'degraded'
    if (seed.failedAt?.includes(index)) status = 'failed'
    if (index === 0) status = seed.status

    const baseLatency = seed.latency ?? 1780
    const latency = status === 'failed' ? null : Math.round(baseLatency * (1 + ((index % 5) - 2) * 0.035))

    return {
      status,
      latency_ms: latency,
      ping_latency_ms: seed.ping,
      checked_at: new Date(now - index * 60 * 60 * 1000).toISOString(),
    }
  })
}

export function createMockChannelMonitors(): UserMonitorView[] {
  return seeds.map((seed) => ({
    id: seed.id,
    name: seed.name,
    provider: seed.provider,
    group_name: seed.groupName,
    primary_model: seed.primaryModel,
    primary_status: seed.status,
    primary_latency_ms: seed.latency,
    primary_ping_latency_ms: seed.ping,
    availability_7d: seed.availability,
    extra_models: seed.extraModels.map((model) => ({
      model: model.model,
      status: model.status,
      latency_ms: model.latency,
    })),
    timeline: createTimeline(seed),
  }))
}

export function createMockChannelMonitorDetails(): Record<number, UserMonitorDetail> {
  return Object.fromEntries(seeds.map((seed) => {
    const models = [
      {
        model: seed.primaryModel,
        status: seed.status,
        latency: seed.latency,
        availability: seed.availability,
      },
      ...seed.extraModels.map((model) => ({
        model: model.model,
        status: model.status,
        latency: model.latency,
        availability: seed.availability === 0 ? 0 : Math.max(94.5, seed.availability - 0.18),
      })),
    ]

    return [
      seed.id,
      {
        id: seed.id,
        name: seed.name,
        provider: seed.provider,
        group_name: seed.groupName,
        models: models.map((model, index) => ({
          model: model.model,
          latest_status: model.status,
          latest_latency_ms: model.latency,
          availability_7d: model.availability,
          availability_15d: model.availability === 0
            ? 0
            : Math.max(93, model.availability - 0.22 - index * 0.04),
          availability_30d: model.availability === 0
            ? 0
            : Math.max(92, model.availability - 0.46 - index * 0.06),
          avg_latency_7d_ms: model.latency == null ? null : Math.round(model.latency * 1.08),
        })),
      },
    ]
  }))
}
