import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'
import layoutSource from '@/components/layout/UserWorkspaceLayout.vue?raw'
import routerSource from '@/router/index.ts?raw'
import studioSource from '../VideoStudioView.vue?raw'
import historySource from '../VideoHistoryView.vue?raw'
import docsSource from '../VideoAPIDocsView.vue?raw'
import tabsSource from '@/components/video/VideoWorkspaceTabs.vue?raw'

const testDirectory = dirname(fileURLToPath(import.meta.url))
const themeSource = readFileSync(resolve(testDirectory, '../../../styles/user-themes.css'), 'utf8')

describe('video user workspace navigation', () => {
  it('keeps one video workspace entry directly after the image workshop in both themes', () => {
    const expectedOrder = /path: '\/images'[\s\S]*?path: '\/videos'[\s\S]*?path: '\/monitor'/g
    expect(layoutSource.match(expectedOrder)).toHaveLength(2)
    expect(layoutSource.match(/label: '视频工作台', icon: 'play'/g)).toHaveLength(2)
    expect(layoutSource).toContain("{ path: '/videos/history', label: '历史任务' }")
    expect(layoutSource).toContain('aria-label="视频工作台导航"')
  })

  it('registers authenticated user routes for all three pages', () => {
    expect(routerSource).toMatch(/path: '\/videos'[\s\S]*?requiresAuth: true[\s\S]*?requiresAdmin: false/)
    expect(routerSource).toMatch(/path: '\/videos\/history'[\s\S]*?requiresAuth: true[\s\S]*?requiresAdmin: false/)
    expect(routerSource).toMatch(/path: '\/docs\/video-api'[\s\S]*?requiresAuth: true[\s\S]*?requiresAdmin: false/)
  })

  it('uses the user workspace shell, shared tabs and current platform origin', () => {
    expect(studioSource).toMatch(/<template>\s*<UserWorkspaceLayout>/)
    expect(historySource).toMatch(/<template>\s*<UserWorkspaceLayout>/)
    expect(docsSource).toMatch(/<template>\s*<UserWorkspaceLayout>/)
    expect(studioSource).toContain('<VideoWorkspaceTabs />')
    expect(historySource).toContain('<VideoWorkspaceTabs />')
    expect(docsSource).toContain('<VideoWorkspaceTabs />')
    expect(tabsSource).toContain("path: '/videos/history'")
    expect(docsSource).toContain('window.location.origin')
    expect(docsSource).not.toContain('sub2.pokexiao.com')
  })

  it('keeps business-theme layout rules outside scoped page styles', () => {
    expect(studioSource).not.toContain(':global(.user-workspace')
    expect(docsSource).not.toContain(':global(.user-workspace')
    expect(themeSource).toContain(".user-workspace[data-user-theme='business'] .video-studio")
    expect(themeSource).toContain(".user-workspace[data-user-theme='business'] .video-docs")
    expect(themeSource).toContain(".user-workspace[data-user-theme='business'] .video-history")
  })

  it('keeps the Key above a directly visible model matrix', () => {
    expect(studioSource.indexOf('class="video-key-bar"')).toBeLessThan(studioSource.indexOf('class="video-model-shelf"'))
    expect(studioSource).toContain('class="video-model-grid"')
    expect(studioSource).toContain('role="radiogroup"')
    expect(studioSource).toContain('class="video-model-card"')
    expect(studioSource).not.toContain('role="combobox"')
    expect(studioSource).not.toContain('class="video-model-menu"')
    expect(studioSource).not.toContain('aria-label="最近任务"')
  })

  it('moves jobs to history and warns that output is not permanent storage', () => {
    expect(historySource).toContain('videoAPI.listJobs(key.key, 100)')
    expect(historySource).toContain('videoAPI.fetchContent')
    expect(historySource).toContain('当前接口没有承诺固定保存天数')
    expect(historySource).toContain('请立即下载')
    expect(docsSource).toContain('当前接口不承诺固定保存天数')
  })

  it('shows model limits before generation and detailed diagnostics for failed jobs', () => {
    expect(studioSource).toContain('模型限制')
    expect(studioSource).not.toContain('XiaoAPI 规则')
    expect(studioSource).toContain('提交前确认以下参数和素材要求')
    expect(studioSource).toContain('素材要求')
    expect(studioSource.indexOf('class="video-model-shelf"')).toBeLessThan(studioSource.indexOf('video-capability--shelf'))
    expect(studioSource.indexOf('video-capability--shelf')).toBeLessThan(studioSource.indexOf('class="video-studio__workspace"'))
    expect(studioSource).toContain('AIStartLab 的素材必须是公网 HTTP(S) URL')
    expect(studioSource).toContain('参考音频必须搭配参考图片或参考视频。')
    expect(historySource).toContain('生成失败 · 可排障信息')
    expect(historySource).toContain('任务编号')
    expect(historySource).toContain('错误编号')
    expect(historySource).toContain('请求追踪号')
    expect(historySource).toContain('copyDiagnostics(job)')
  })
})
