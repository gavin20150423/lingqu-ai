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
