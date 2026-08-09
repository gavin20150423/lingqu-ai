import type { VideoModel } from '@/api/video'

export type VideoCreationMode = 'text' | 'frames' | 'references'
export type ReferenceKind = 'image' | 'video' | 'audio'

export const videoMediaLimits = {
  image: { formats: 'PNG / JPG / WEBP', maxMiB: 10, minWidth: 300, maxWidth: 6000, minAspectRatio: 0.4, maxAspectRatio: 2.5 },
  video: { formats: 'MP4 / MOV', maxMiB: 99, minDuration: 2, maxDuration: 15, maxTotalDuration: 15 },
  audio: { formats: 'MP3 / WAV', maxMiB: 15, minDuration: 2, maxDuration: 15 },
} as const

export interface VideoModelCapability {
  id: string
  label: string
  resolutions: string[]
  durations: number[]
  aspectRatios: Record<string, string[]>
  defaultResolution: string
  defaultDuration: number
  defaultAspectRatio: string
  promptLimit: number
  supportsAudio: boolean
  supportsStartFrame: boolean
  requiresStartFrame: boolean
  supportsEndFrame: boolean
  supportsPromptEnhance: boolean
  maxReferences: Record<ReferenceKind, number>
  capabilitySource: 'xiaoapi' | 'aistartlab' | 'mixed' | 'unknown'
  usesXiaoAPIRules: boolean
}

type KnownVideoModelCapability = Omit<VideoModelCapability, 'capabilitySource' | 'usesXiaoAPIRules'>

const seedanceRatios = ['16:9', '9:16', '1:1', '4:3', '3:4', '21:9', '9:21']
const range = (start: number, end: number, step = 1) =>
  Array.from({ length: Math.floor((end - start) / step) + 1 }, (_, index) => start + index * step)

const knownCapabilities: Record<string, KnownVideoModelCapability> = {
  'seedance-2.0': {
    id: 'seedance-2.0', label: 'Seedance 2.0',
    resolutions: ['480p', '720p', '1080p'], durations: range(4, 15),
    aspectRatios: { '480p': seedanceRatios, '720p': seedanceRatios.slice(0, -1), '1080p': seedanceRatios },
    defaultResolution: '720p', defaultDuration: 8, defaultAspectRatio: '16:9', promptLimit: 5000,
    supportsAudio: true, supportsStartFrame: true, requiresStartFrame: false, supportsEndFrame: true,
    supportsPromptEnhance: false, maxReferences: { image: 4, video: 3, audio: 1 },
  },
  'seedance-2.0-fast': {
    id: 'seedance-2.0-fast', label: 'Seedance 2.0 Fast',
    resolutions: ['480p', '720p'], durations: range(4, 15),
    aspectRatios: { '480p': seedanceRatios, '720p': seedanceRatios.slice(0, -1) },
    defaultResolution: '720p', defaultDuration: 8, defaultAspectRatio: '16:9', promptLimit: 5000,
    supportsAudio: true, supportsStartFrame: true, requiresStartFrame: false, supportsEndFrame: true,
    supportsPromptEnhance: false, maxReferences: { image: 4, video: 3, audio: 1 },
  },
  'seedance-2.0-mini': {
    id: 'seedance-2.0-mini', label: 'Seedance 2.0 Mini',
    resolutions: ['480p', '720p'], durations: range(4, 15),
    aspectRatios: { '*': ['16:9', '1:1', '9:16'] },
    defaultResolution: '720p', defaultDuration: 8, defaultAspectRatio: '16:9', promptLimit: 5000,
    supportsAudio: true, supportsStartFrame: true, requiresStartFrame: false, supportsEndFrame: true,
    supportsPromptEnhance: false, maxReferences: { image: 4, video: 3, audio: 1 },
  },
  'happy-horse-1.1': {
    id: 'happy-horse-1.1', label: 'Happy Horse 1.1',
    resolutions: ['720p', '1080p'], durations: range(3, 15),
    aspectRatios: { '*': ['16:9', '4:3', '1:1', '3:4', '9:16'] },
    defaultResolution: '720p', defaultDuration: 5, defaultAspectRatio: '16:9', promptLimit: 2500,
    supportsAudio: true, supportsStartFrame: true, requiresStartFrame: false, supportsEndFrame: false,
    supportsPromptEnhance: true, maxReferences: { image: 9, video: 0, audio: 0 },
  },
  'grok-imagine-1.5': {
    id: 'grok-imagine-1.5', label: 'Grok Imagine 1.5',
    resolutions: ['400p', '544p', '720p', '960p'], durations: range(3, 15),
    aspectRatios: { '400p': ['16:9', '9:16'], '544p': ['1:1'], '720p': ['16:9', '9:16'], '960p': ['1:1'] },
    defaultResolution: '720p', defaultDuration: 6, defaultAspectRatio: '16:9', promptLimit: 5000,
    supportsAudio: true, supportsStartFrame: true, requiresStartFrame: true, supportsEndFrame: false,
    supportsPromptEnhance: false, maxReferences: { image: 0, video: 0, audio: 0 },
  },
  'ltx-2.3-pro': {
    id: 'ltx-2.3-pro', label: 'LTX 2.3 Pro',
    resolutions: ['1080p', '1440p', '2160p'], durations: [6, 8, 10], aspectRatios: { '*': ['16:9'] },
    defaultResolution: '1080p', defaultDuration: 6, defaultAspectRatio: '16:9', promptLimit: 5000,
    supportsAudio: true, supportsStartFrame: true, requiresStartFrame: false, supportsEndFrame: true,
    supportsPromptEnhance: true, maxReferences: { image: 0, video: 0, audio: 0 },
  },
  'ltx-2.3-fast': {
    id: 'ltx-2.3-fast', label: 'LTX 2.3 Fast',
    resolutions: ['1080p', '1440p', '2160p'], durations: range(6, 20, 2), aspectRatios: { '*': ['16:9'] },
    defaultResolution: '1080p', defaultDuration: 6, defaultAspectRatio: '16:9', promptLimit: 5000,
    supportsAudio: true, supportsStartFrame: true, requiresStartFrame: false, supportsEndFrame: true,
    supportsPromptEnhance: true, maxReferences: { image: 0, video: 0, audio: 0 },
  },
}

const capabilityAliases: Record<string, string> = {
  'happyhorse-1.1': 'happy-horse-1.1',
  'happy-horse': 'happy-horse-1.1',
  'grok-imagine-video-1.5': 'grok-imagine-1.5',
  'grok-imagine-video': 'grok-imagine-1.5',
}

function capabilitySource(model: VideoModel): VideoModelCapability['capabilitySource'] {
  if (!model.capability_source || model.capability_source === 'native') return 'xiaoapi'
  if (model.capability_source === 'openai_sora') return 'aistartlab'
  if (model.capability_source === 'mixed') return 'mixed'
  return 'unknown'
}

function fallbackCapability(model: VideoModel): VideoModelCapability {
  const resolutions = model.resolutions.length
    ? model.resolutions
    : model.default_resolution ? [model.default_resolution] : []
  const modelDurations = (model.durations ?? []).filter((value) => Number.isInteger(value) && value > 0)
  const defaultDuration = Number(model.default_duration) > 0 ? Number(model.default_duration) : modelDurations[0] || 0
  const durations = modelDurations.length ? [...new Set(modelDurations)].sort((a, b) => a - b) : (defaultDuration > 0 ? [defaultDuration] : [])
  const ratios = (model.aspect_ratios ?? []).filter((value) => typeof value === 'string' && value.trim())
  const maxReferences = {
    image: Math.max(0, Number(model.max_references?.image) || 0),
    video: Math.max(0, Number(model.max_references?.video) || 0),
    audio: Math.max(0, Number(model.max_references?.audio) || 0),
  }
  return {
    id: model.id,
    label: model.id,
    resolutions,
    durations,
    aspectRatios: { '*': ratios.length ? ratios : [model.default_aspect_ratio || '16:9'] },
    defaultResolution: model.default_resolution || resolutions[0],
    defaultDuration,
    defaultAspectRatio: model.default_aspect_ratio || '16:9',
    promptLimit: 5000,
    supportsAudio: model.supports_audio === true,
    supportsStartFrame: model.supports_start_frame === true,
    requiresStartFrame: model.requires_start_frame === true,
    supportsEndFrame: model.supports_end_frame === true,
    supportsPromptEnhance: false,
    maxReferences: Object.values(maxReferences).some((count) => count > 0)
      ? maxReferences
      : { image: model.supports_guidances ? 1 : 0, video: 0, audio: 0 },
    capabilitySource: capabilitySource(model),
    usesXiaoAPIRules: false,
  }
}

export function resolveVideoCapability(model: VideoModel): VideoModelCapability {
  const source = capabilitySource(model)
  const known = source === 'xiaoapi'
    ? knownCapabilities[model.id] || knownCapabilities[capabilityAliases[model.id]]
    : undefined
  if (!known) return fallbackCapability(model)
  const available = new Set(model.resolutions)
  const resolutions = known.resolutions.filter((resolution) => available.size === 0 || available.has(resolution))
  const defaultResolution = resolutions.includes(model.default_resolution || '')
    ? model.default_resolution!
    : resolutions.includes(known.defaultResolution) ? known.defaultResolution : resolutions[0]
  return {
    ...known,
    id: model.id,
    resolutions,
    defaultResolution,
    defaultDuration: model.default_duration || known.defaultDuration,
    defaultAspectRatio: model.default_aspect_ratio || known.defaultAspectRatio,
    supportsAudio: model.supports_audio === true || known.supportsAudio,
    capabilitySource: 'xiaoapi',
    usesXiaoAPIRules: true,
  }
}

export function aspectRatiosFor(capability: VideoModelCapability, resolution: string): string[] {
  return capability.aspectRatios[resolution] || capability.aspectRatios['*'] || [capability.defaultAspectRatio]
}

export function durationsFor(capability: VideoModelCapability, resolution: string): number[] {
  if (capability.id === 'seedance-2.0' && resolution === '1080p') {
    return capability.durations.filter((duration) => duration <= 12)
  }
  return capability.durations
}

export function createIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return `video-${crypto.randomUUID()}`
  }
  return `video-${Date.now()}-${Math.random().toString(36).slice(2, 12)}`
}
