import { describe, expect, it } from 'vitest'
import { resolveImageUrl } from './imageApiShared'

describe('resolveImageUrl', () => {
  it('resolves provider-local image paths against the active API base URL', () => {
    expect(resolveImageUrl(
      '/v1/images/files/images/imgtask_example-0.png',
      'https://api.example.com/v1',
    )).toBe('https://api.example.com/v1/images/files/images/imgtask_example-0.png')
  })

  it('keeps absolute and data URLs unchanged', () => {
    expect(resolveImageUrl('https://cdn.example.com/image.png', 'https://api.example.com/v1'))
      .toBe('https://cdn.example.com/image.png')
    expect(resolveImageUrl('data:image/png;base64,abc', 'https://api.example.com/v1'))
      .toBe('data:image/png;base64,abc')
  })

  it('does not guess a host for unsupported relative values', () => {
    expect(resolveImageUrl('images/image.png', 'https://api.example.com/v1')).toBeNull()
  })
})
