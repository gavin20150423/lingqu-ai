import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const dir = dirname(fileURLToPath(import.meta.url))
const source = readFileSync(resolve(dir, '../VersionBadge.vue'), 'utf8')

describe('VersionBadge layout', () => {
  it('resets inherited sidebar nowrap styles inside the popover', () => {
    expect(source).toMatch(/\.version-popover\s*\{[\s\S]*?white-space:\s*normal;/)
    expect(source).toMatch(/\.version-update-card__copy\s*\{[\s\S]*?white-space:\s*normal;/)
  })

  it('allows long Docker update hints to wrap without horizontal overflow', () => {
    expect(source).toMatch(/\.version-update-card__hint\s*\{[\s\S]*?overflow-wrap:\s*anywhere;/)
    expect(source).toMatch(/\.version-update-card__hint\s*\{[\s\S]*?word-break:\s*break-word;/)
  })
})
