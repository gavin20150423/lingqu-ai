import { describe, expect, it } from 'vitest'
import { formatCacheHitRate } from '../formatters'

describe('formatCacheHitRate', () => {
  it('uses input plus cache reads as the input-side denominator', () => {
    expect(formatCacheHitRate(100, 100)).toBe('50.0%')
  })

  it('handles empty and very small rates', () => {
    expect(formatCacheHitRate(0, 0)).toBe('0%')
    expect(formatCacheHitRate(100000, 1)).toBe('<0.1%')
  })

  it('normalizes missing and negative values', () => {
    expect(formatCacheHitRate(undefined, null)).toBe('0%')
    expect(formatCacheHitRate(-100, 50)).toBe('100.0%')
  })
})
