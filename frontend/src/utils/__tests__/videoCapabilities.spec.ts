import { describe, expect, it } from 'vitest'
import type { VideoModel } from '@/api/video'
import { aspectRatiosFor, durationsFor, resolveVideoCapability, videoMediaLimits } from '../videoCapabilities'

function model(id: string, resolutions: string[], overrides: Partial<VideoModel> = {}): VideoModel {
  return { id, object: 'model', owned_by: 'video', resolutions, ...overrides }
}

describe('video capabilities', () => {
  it('keeps Seedance generated audio enabled and limits 1080p to 12 seconds', () => {
    const capability = resolveVideoCapability(model('seedance-2.0', ['480p', '720p', '1080p']))

    expect(capability.supportsAudio).toBe(true)
    expect(durationsFor(capability, '1080p')).toEqual([4, 5, 6, 7, 8, 9, 10, 11, 12])
    expect(aspectRatiosFor(capability, '720p')).not.toContain('9:21')
    expect(aspectRatiosFor(capability, '480p')).toContain('9:21')
  })

  it('intersects documented capabilities with resolutions enabled by platform pricing', () => {
    const capability = resolveVideoCapability(model('ltx-2.3-fast', ['1440p']))

    expect(capability.resolutions).toEqual(['1440p'])
    expect(capability.defaultResolution).toBe('1440p')
    expect(capability.durations).toEqual([6, 8, 10, 12, 14, 16, 18, 20])
  })

  it('preserves safe runtime values for newly configured upstream models', () => {
    const capability = resolveVideoCapability(model('future-video-model', ['900p'], {
      default_duration: 7,
      default_aspect_ratio: '4:3',
      supports_audio: true,
      supports_guidances: true,
    }))

    expect(capability.label).toBe('future-video-model')
    expect(capability.durations).toEqual([7])
    expect(capability.supportsAudio).toBe(true)
    expect(capability.maxReferences.image).toBe(1)
  })

  it('keeps upload boundaries aligned with the user-facing capability panel', () => {
    expect(videoMediaLimits.image).toMatchObject({ maxMiB: 10, minWidth: 300, maxWidth: 6000 })
    expect(videoMediaLimits.video).toMatchObject({ maxMiB: 99, minDuration: 2, maxTotalDuration: 15 })
    expect(videoMediaLimits.audio).toMatchObject({ maxMiB: 15, minDuration: 2, maxDuration: 15 })
  })

  it('maps pricing aliases to the same documented user-facing capabilities', () => {
    const capability = resolveVideoCapability(model('grok-imagine-video-1.5', ['400p', '720p']))
    expect(capability.label).toBe('Grok Imagine 1.5')
    expect(capability.id).toBe('grok-imagine-video-1.5')
    expect(capability.defaultDuration).toBe(6)
    expect(capability.requiresStartFrame).toBe(true)
    expect(capability.usesXiaoAPIRules).toBe(true)
  })

  it('does not apply XiaoAPI constraints to AIStartLab models with the same ID', () => {
    const capability = resolveVideoCapability(model('seedance-2.0', ['720p'], {
      capability_source: 'openai_sora',
      default_duration: 10,
      default_aspect_ratio: '16:9',
      supports_audio: false,
    }))

    expect(capability.capabilitySource).toBe('aistartlab')
    expect(capability.usesXiaoAPIRules).toBe(false)
    expect(capability.durations).toEqual([10])
    expect(capability.supportsStartFrame).toBe(false)
    expect(capability.maxReferences).toEqual({ image: 0, video: 0, audio: 0 })
  })

  it('does not invent a five-second duration when upstream metadata is missing', () => {
    const capability = resolveVideoCapability(model('future-video-model', ['720p'], {
      capability_source: 'openai_sora',
    }))

    expect(capability.defaultDuration).toBe(0)
    expect(capability.durations).toEqual([])
  })
})
