import { describe, expect, it } from 'vitest'
import { isImageCreationKey } from '../creationAccess'

describe('image creation key access', () => {
  it('only accepts active OpenAI groups with image generation enabled', () => {
    expect(isImageCreationKey({ status: 'active', group: { allow_image_generation: true } as never })).toBe(false)
    expect(isImageCreationKey({ status: 'active', group: { status: 'active', platform: 'openai', allow_image_generation: true } as never })).toBe(true)
    expect(isImageCreationKey({ status: 'active', group: { status: 'inactive', platform: 'openai', allow_image_generation: true } as never })).toBe(false)
    expect(isImageCreationKey({ status: 'active', group: { status: 'active', platform: 'openai', allow_image_generation: false } as never })).toBe(false)
    expect(isImageCreationKey({ status: 'active', group: { status: 'active', platform: 'xiaoapi', allow_image_generation: true } as never })).toBe(false)
    expect(isImageCreationKey({ status: 'active', group: { status: 'active', platform: 'grok', allow_image_generation: true } as never })).toBe(false)
    expect(isImageCreationKey({ status: 'active', group: { status: 'active', platform: 'anthropic', allow_image_generation: true } as never })).toBe(false)
  })

  it('rejects ungrouped keys because their protocol is unknown', () => {
    expect(isImageCreationKey({ status: 'active', group: undefined })).toBe(false)
    expect(isImageCreationKey({ status: 'inactive', group: undefined })).toBe(false)
  })
})
