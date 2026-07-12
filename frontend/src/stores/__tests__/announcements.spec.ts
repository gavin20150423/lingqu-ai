import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { UserAnnouncement } from '@/types'

const { listMock, markReadMock } = vi.hoisted(() => ({
  listMock: vi.fn(),
  markReadMock: vi.fn()
}))

vi.mock('@/api', () => ({
  announcementsAPI: {
    list: listMock,
    markRead: markReadMock
  }
}))

import { useAnnouncementStore } from '@/stores/announcements'

function createAnnouncement(overrides: Partial<UserAnnouncement> = {}): UserAnnouncement {
  return {
    id: 1,
    title: '系统公告',
    content: '公告内容',
    notify_mode: 'silent',
    created_at: '2026-07-13T00:00:00Z',
    updated_at: '2026-07-13T00:00:00Z',
    ...overrides
  }
}

describe('useAnnouncementStore popup controls', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    listMock.mockReset()
    markReadMock.mockReset()
    markReadMock.mockResolvedValue(undefined)
  })

  it('opens a selected read announcement without marking it again when closed', async () => {
    const store = useAnnouncementStore()
    const announcement = createAnnouncement({ read_at: '2026-07-13T01:00:00Z' })

    store.openPopup(announcement)

    expect(store.currentPopup).toStrictEqual(announcement)

    await store.dismissPopup()

    expect(store.currentPopup).toBeNull()
    expect(markReadMock).not.toHaveBeenCalled()
  })

  it('marks an unread selected announcement as read when closed', async () => {
    const store = useAnnouncementStore()
    const announcement = createAnnouncement()

    store.openPopup(announcement)
    await store.dismissPopup()

    expect(markReadMock).toHaveBeenCalledWith(announcement.id)
  })
})
