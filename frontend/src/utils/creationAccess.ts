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

/**
 * A generation channel represents a group, not an individual key. When a user
 * owns multiple active keys for the same group, use the first preferred key so
 * the canvas does not render duplicate group/model choices.
 */
export function getTextCreationGroupKeys<T extends Pick<ApiKey, 'status' | 'group'>>(keys: readonly T[]): T[] {
  const seenGroupIds = new Set<number>()

  return getTextCreationKeys(keys).filter((key) => {
    const groupId = key.group?.id
    if (!groupId || seenGroupIds.has(groupId)) return false
    seenGroupIds.add(groupId)
    return true
  })
}

/** Extract a stable, deduplicated model list from an OpenAI-compatible response. */
export function getOpenAIModelIds(payload: unknown): string[] {
  if (!payload || typeof payload !== 'object') return []
  const data = (payload as { data?: unknown }).data
  if (!Array.isArray(data)) return []

  const seen = new Set<string>()
  const models: string[] = []
  for (const item of data) {
    if (!item || typeof item !== 'object') continue
    const id = typeof (item as { id?: unknown }).id === 'string'
      ? (item as { id: string }).id.trim()
      : ''
    if (!id || seen.has(id)) continue
    seen.add(id)
    models.push(id)
  }
  return models
}
