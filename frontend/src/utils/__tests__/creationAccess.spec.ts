import { describe, expect, it } from 'vitest'
import type { ApiKey } from '@/types'
import { getTextCreationKeys, isImageCreationKey, isTextCreationKey } from '../creationAccess'

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

describe('text creation key access', () => {
  it('accepts active OpenAI groups', () => {
    expect(isTextCreationKey({ status: 'active', group: { status: 'active', platform: 'openai' } as never })).toBe(true)
  })

  it('rejects inactive keys, inactive groups, and other protocols', () => {
    expect(isTextCreationKey({ status: 'inactive', group: { status: 'active', platform: 'openai' } as never })).toBe(false)
    expect(isTextCreationKey({ status: 'active', group: { status: 'inactive', platform: 'openai' } as never })).toBe(false)
    expect(isTextCreationKey({ status: 'active', group: { status: 'active', platform: 'anthropic' } as never })).toBe(false)
    expect(isTextCreationKey({ status: 'active', group: undefined })).toBe(false)
  })

  it('prefers general OpenAI keys before image-enabled keys', () => {
    const imageKey = {
      id: 1,
      status: 'active',
      group: { status: 'active', platform: 'openai', allow_image_generation: true },
    } as ApiKey
    const generalKey = {
      id: 2,
      status: 'active',
      group: { status: 'active', platform: 'openai', allow_image_generation: false },
    } as ApiKey

    expect(getTextCreationKeys([imageKey, generalKey]).map((key) => key.id)).toEqual([2, 1])
  })
})
