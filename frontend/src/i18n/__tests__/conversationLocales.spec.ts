import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

function hasMessage(messages: Record<string, unknown>, path: string): boolean {
  let value: unknown = messages
  for (const segment of path.split('.')) {
    if (!value || typeof value !== 'object' || !(segment in value)) return false
    value = (value as Record<string, unknown>)[segment]
  }
  return typeof value === 'string' && value.length > 0
}

const conversationKeys = [
  'title',
  'searchPlaceholder',
  'startTicket',
  'empty',
  'emptyDescription',
  'noSelection',
  'noSelectionDescription',
  'unread',
  'unreadCount',
  'createdAt',
  'markRead',
  'loadEarlier',
  'noMessages',
  'reopenPlaceholder',
  'replyPlaceholder',
  'closedPlaceholder',
  'send',
  'close',
  'subject',
  'type',
  'priority',
  'content',
  'allStatus',
  'allMessages',
  'unreadOnly',
  'defaultSubject',
  'createSuccess',
  'createFailed',
  'sent',
  'sendFailed',
  'markReadFailed',
  'closed',
  'closeFailed',
  'loadFailed',
  'loadMessagesFailed',
  ...['open', 'pending_user', 'pending_admin', 'resolved', 'closed'].map((key) => `statusLabels.${key}`),
  ...['low', 'normal', 'high', 'urgent'].map((key) => `priorityLabels.${key}`),
  ...['support', 'notice', 'billing', 'subscription', 'account', 'security'].map((key) => `typeLabels.${key}`),
  ...['user', 'admin', 'system'].map((key) => `senderLabels.${key}`),
]

const adminConversationKeys = [
  'title',
  'searchPlaceholder',
  'userId',
  'assignedAdminId',
  'unreadCount',
  'createConversation',
  'empty',
  'emptyDescription',
  'noSelection',
  'noSelectionDescription',
  'assign',
  'unassign',
  'assigneeValue',
  'replyPlaceholder',
  'source',
  'allPriority',
  'allType',
  'userIdValue',
  'loadFailed',
  'loadMessagesFailed',
  'createSuccess',
  'createFailed',
  'statusUpdated',
  'statusUpdateFailed',
  'assigneeUpdated',
  'assigneeUpdateFailed',
  ...['open', 'pending_user', 'pending_admin', 'resolved', 'closed'].map((key) => `statusLabels.${key}`),
  ...['user', 'admin', 'system'].map((key) => `senderLabels.${key}`),
]

describe.each([
  ['zh', zh],
  ['en', en],
] as const)('%s conversation locale completeness', (_locale, messages) => {
  it.each(conversationKeys)('has conversations.%s', (key) => {
    expect(hasMessage(messages, `conversations.${key}`)).toBe(true)
  })

  it.each(adminConversationKeys)('has admin.conversations.%s', (key) => {
    expect(hasMessage(messages, `admin.conversations.${key}`)).toBe(true)
  })
})
