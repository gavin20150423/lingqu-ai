import { describe, expect, it } from 'vitest'
import accountShareSource from '../AccountShareView.vue?raw'
import storeSource from '../../StoreView.vue?raw'
import conversationsSource from '../ConversationsView.vue?raw'
import keysSource from '../KeysView.vue?raw'
import usageSource from '../UsageView.vue?raw'
import paymentSource from '../PaymentView.vue?raw'
import subscriptionsSource from '../SubscriptionsView.vue?raw'
import channelStatusSource from '../ChannelStatusView.vue?raw'

const workspaceViews = [
  ['account sharing', accountShareSource],
  ['store', storeSource],
  ['support conversations', conversationsSource],
  ['API keys', keysSource],
  ['usage records', usageSource],
  ['payment', paymentSource],
  ['subscriptions', subscriptionsSource],
  ['channel status', channelStatusSource],
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
