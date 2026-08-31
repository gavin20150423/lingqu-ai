import type { ApiKey } from '@/types'

/**
 * The image workbench is backed by the OpenAI images protocol. Keep unrelated
 * protocol groups (Grok, Claude, video/xiaoapi, etc.) out even when their
 * group happens to carry stale image pricing flags.
 */
export function isImageCreationKey(key: Pick<ApiKey, 'status' | 'group'>): boolean {
  return key.status === 'active' &&
    key.group?.status === 'active' &&
    key.group.platform === 'openai' &&
    key.group.allow_image_generation === true
}

/** Text generation uses the standard OpenAI-compatible group route. */
export function isTextCreationKey(key: Pick<ApiKey, 'status' | 'group'>): boolean {
  return key.status === 'active' &&
    key.group?.status === 'active' &&
    key.group.platform === 'openai'
}

/** Prefer general OpenAI groups before image-enabled groups for text generation. */
export function getTextCreationKeys<T extends Pick<ApiKey, 'status' | 'group'>>(keys: readonly T[]): T[] {
  const candidates = keys.filter((key) => isTextCreationKey(key))

  return [
    ...candidates.filter((key) => key.group?.allow_image_generation !== true),
    ...candidates.filter((key) => key.group?.allow_image_generation === true),
  ]
}
