import { describe, expect, it } from 'vitest'
import accountShareSource from '../AccountShareView.vue?raw'
import storeSource from '../../StoreView.vue?raw'
import conversationsSource from '../ConversationsView.vue?raw'

const workspaceViews = [
  ['account sharing', accountShareSource],
  ['store', storeSource],
  ['support conversations', conversationsSource],
] as const

describe('user workspace layout consistency', () => {
  it.each(workspaceViews)('keeps %s in the user workspace shell', (_, source) => {
    expect(source).toMatch(/<template>\s*<UserWorkspaceLayout(?:\s|>)/)
    expect(source).not.toContain('<AppLayout')
    expect(source).toContain(
      "import UserWorkspaceLayout from '@/components/layout/UserWorkspaceLayout.vue'",
    )
  })
})
