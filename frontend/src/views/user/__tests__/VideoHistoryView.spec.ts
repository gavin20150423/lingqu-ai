import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import VideoHistoryView from '../VideoHistoryView.vue'

const mocks = vi.hoisted(() => ({
  listKeys: vi.fn(),
  listJobs: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api', () => ({
  keysAPI: { list: mocks.listKeys },
  videoAPI: {
    listJobs: mocks.listJobs,
    cancelJob: vi.fn(),
    fetchContent: vi.fn(),
  },
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: mocks.showError, showSuccess: mocks.showSuccess }),
}))

describe('VideoHistoryView failure diagnostics', () => {
  beforeEach(() => {
    window.localStorage.clear()
    mocks.listKeys.mockResolvedValue({
      items: [{ id: 7, key: 'sk-video-test', name: 'Video', status: 'active', group: { name: 'Video', platform: 'xiaoapi' } }],
    })
    mocks.listJobs.mockResolvedValue([{
      job_id: 'vidjob_failure_123456',
      status: 'failed',
      model: 'seedance-2.0',
      resolution: '720p',
      duration: 10,
      aspect_ratio: '16:9',
      amount: '3.00000000',
      currency: 'USD',
      created_at: '2026-08-08T12:00:00Z',
      updated_at: '2026-08-08T12:01:00Z',
      finished_at: '2026-08-08T12:01:00Z',
      settlement_status: 'released',
      status_url: '/v1/videos/jobs/vidjob_failure_123456',
      error: {
        code: 'VIDEO_GENERATION_FAILED',
        message: 'resolution is not supported by this model',
        stage: 'processing',
        upstream_code: 'VIDEO_RESOLUTION_INVALID',
        request_id: 'req_failure_123',
        failed_at: '2026-08-08T12:01:00Z',
      },
    }])
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('auto-expands the newest failure and renders copyable troubleshooting fields', async () => {
    const wrapper = mount(VideoHistoryView, {
      global: {
        stubs: {
          UserWorkspaceLayout: { template: '<main><slot /></main>' },
          VideoWorkspaceTabs: true,
          Icon: { props: ['name'], template: '<i :data-icon="name"></i>' },
          RouterLink: { template: '<a><slot /></a>' },
        },
      },
    })

    await flushPromises()
    await flushPromises()

    expect(mocks.listJobs).toHaveBeenCalledWith('sk-video-test', 100)
    expect(wrapper.text()).toContain('生成失败 · 可排障信息')
    expect(wrapper.text()).toContain('vidjob_failure_123456')
    expect(wrapper.text()).toContain('VIDEO_GENERATION_FAILED')
    expect(wrapper.text()).toContain('VIDEO_RESOLUTION_INVALID')
    expect(wrapper.text()).toContain('req_failure_123')
    expect(wrapper.text()).toContain('预授权已释放，失败任务未扣费')
    expect(wrapper.find('button[title="复制全部诊断信息"]').exists()).toBe(true)

    wrapper.unmount()
  })
})
