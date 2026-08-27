import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post } = vi.hoisted(() => ({
  post: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: { post }
}))

import { batchDelete } from '@/api/admin/accounts'

describe('admin account batch delete API', () => {
  beforeEach(() => {
    post.mockReset()
    post.mockResolvedValue({
      data: { total: 2, success: 2, failed: 0, success_ids: [1, 2], failed_ids: [], errors: [] }
    })
  })

  it('uses the dedicated long-running operation timeout', async () => {
    await batchDelete([1, 2])

    expect(post).toHaveBeenCalledWith('/admin/accounts/batch-delete', {
      account_ids: [1, 2]
    }, {
      timeout: 5 * 60 * 1000
    })
  })
})
