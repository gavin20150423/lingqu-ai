import { describe, expect, it } from 'vitest'
import {
  imageReferenceGuidances,
  normalizeVideoPrompt,
  validateVideoPromptParameters,
} from '@/utils/videoWorkbench'

describe('video workbench request contract', () => {
  it('uses one explicit strength and zero-based order for every reference image', () => {
    expect(imageReferenceGuidances(['https://example.test/a.png', 'https://example.test/b.png'], 'HIGH')).toEqual([
      {
        image: { url: 'https://example.test/a.png', type: 'UPLOADED' },
        strength: 'HIGH',
        order: 0,
      },
      {
        image: { url: 'https://example.test/b.png', type: 'UPLOADED' },
        strength: 'HIGH',
        order: 1,
      },
    ])
    expect(imageReferenceGuidances([], 'MID')).toEqual([])
  })

  it('reports prompt aspect ratio and total duration conflicts', () => {
    expect(validateVideoPromptParameters('9:16 竖屏构图', { aspectRatio: '16:9', duration: 8 }))
      .toBe('提示词中写了 9:16，但当前选择框是 16:9')
    expect(validateVideoPromptParameters('总时长：15 秒，分镜 00:00-00:03', { aspectRatio: '16:9', duration: 8 }))
      .toBe('提示词标题中写了 15 秒，但当前选择框是 8 秒')
    expect(validateVideoPromptParameters('16:9，总时长 8 秒，分镜 00:00-00:03', { aspectRatio: '16:9', duration: 8 }))
      .toBe('')
  })

  it('removes matching duplicate parameters without changing shot timecodes', () => {
    expect(normalizeVideoPrompt('16:9，总时长 8 秒，分镜 00:00-00:03', { aspectRatio: '16:9', duration: 8 }))
      .toBe('分镜 00:00-00:03')
  })
})
