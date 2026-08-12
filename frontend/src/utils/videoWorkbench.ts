export type ReferenceImageStrength = 'LOW' | 'MID' | 'HIGH'

export interface VideoPromptParameters {
  aspectRatio: string
  duration: number
}

const aspectRatioPattern = /(^|[^\d])(21:9|16:9|9:21|9:16|4:3|3:4|1:1)(?!\d)/g
const durationPattern = /(?:总时长|视频时长|片长|时长)\s*[:：为是]?\s*(\d{1,3})\s*(?:秒|s\b)/gi

export function validateVideoPromptParameters(
  prompt: string,
  parameters: VideoPromptParameters,
): string {
  const ratios = Array.from(prompt.matchAll(aspectRatioPattern), (match) => match[2])
  const conflictingRatio = ratios.find((value) => value !== parameters.aspectRatio)
  if (conflictingRatio) {
    return `提示词中写了 ${conflictingRatio}，但当前选择框是 ${parameters.aspectRatio}`
  }

  const durations = Array.from(prompt.matchAll(durationPattern), (match) => Number(match[1]))
  const conflictingDuration = durations.find((value) => value !== parameters.duration)
  if (conflictingDuration) {
    return `提示词标题中写了 ${conflictingDuration} 秒，但当前选择框是 ${parameters.duration} 秒`
  }
  return ''
}

export function normalizeVideoPrompt(prompt: string, parameters: VideoPromptParameters): string {
  return prompt
    .replace(aspectRatioPattern, (full, prefix: string, ratio: string) => (
      ratio === parameters.aspectRatio ? prefix : full
    ))
    .replace(durationPattern, (full, value: string) => (
      Number(value) === parameters.duration ? '' : full
    ))
    .replace(/[，,；;]{2,}/g, '，')
    .replace(/^[，,；;\s]+/, '')
    .replace(/[，,；;]\s*([。！？!?])/g, '$1')
    .replace(/(^|[\n。！？!?])\s*[，,；;]\s*/g, '$1')
    .replace(/[ \t]{2,}/g, ' ')
    .trim()
}

export function imageReferenceGuidances(
  urls: string[],
  strength: ReferenceImageStrength,
) {
  return urls.map((url, order) => ({
    image: { url, type: 'UPLOADED' as const },
    strength,
    order,
  }))
}
