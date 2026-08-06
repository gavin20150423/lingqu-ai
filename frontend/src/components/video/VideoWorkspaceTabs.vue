<template>
  <nav class="video-workspace-tabs" aria-label="视频工作台导航">
    <router-link
      v-for="item in items"
      :key="item.path"
      :to="item.path"
      :class="{ 'video-workspace-tabs__item--active': isActive(item) }"
      class="video-workspace-tabs__item"
    >
      <span class="video-workspace-tabs__icon"><Icon :name="item.icon" size="sm" /></span>
      <span>
        <strong>{{ item.label }}</strong>
        <small>{{ item.description }}</small>
      </span>
    </router-link>
  </nav>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'

const route = useRoute()
const items = [
  { path: '/videos', exact: true, label: '视频创作', description: '配置并生成视频', icon: 'play' },
  { path: '/videos/history', exact: false, label: '历史任务', description: '查看、播放与下载', icon: 'clock' },
  { path: '/docs/video-api', exact: false, label: 'API 文档', description: '接入与参数说明', icon: 'book' },
] as const

function isActive(item: (typeof items)[number]) {
  return item.exact ? route.path === item.path : route.path === item.path || route.path.startsWith(`${item.path}/`)
}
</script>

<style scoped>
.video-workspace-tabs {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  min-width: 0;
  overflow: hidden;
  border: 1px solid #d5e0e5;
  border-radius: 10px;
  background: #fff;
  box-shadow: 0 8px 24px rgba(18, 36, 51, .05);
}
.video-workspace-tabs__item {
  position: relative;
  display: flex;
  min-width: 0;
  min-height: 4.2rem;
  align-items: center;
  gap: .65rem;
  border-right: 1px solid #dce5e9;
  padding: .72rem .9rem;
  color: #647984;
  text-decoration: none;
}
.video-workspace-tabs__item:last-child { border-right: 0; }
.video-workspace-tabs__item::after {
  position: absolute;
  right: .9rem;
  bottom: 0;
  left: .9rem;
  height: 3px;
  border-radius: 3px 3px 0 0;
  background: transparent;
  content: '';
}
.video-workspace-tabs__item:hover { background: #f7fafb; color: #122433; }
.video-workspace-tabs__item--active { background: #f3f9fb; color: #122433; }
.video-workspace-tabs__item--active::after { background: #24a8af; }
.video-workspace-tabs__icon {
  display: grid;
  width: 2rem;
  height: 2rem;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 7px;
  background: #e8f0f3;
  color: #49616e;
}
.video-workspace-tabs__item--active .video-workspace-tabs__icon { background: #122433; color: #58cbd0; }
.video-workspace-tabs__item > span:last-child { display: grid; min-width: 0; gap: .16rem; }
.video-workspace-tabs strong { font-size: .7rem; font-weight: 850; }
.video-workspace-tabs small { overflow: hidden; color: #8a9aa2; font-size: .56rem; text-overflow: ellipsis; white-space: nowrap; }
.video-workspace-tabs__item:focus-visible { z-index: 1; outline: 2px solid rgba(36, 168, 175, .45); outline-offset: -2px; }

@media (max-width: 620px) {
  .video-workspace-tabs__item { min-height: 3.35rem; justify-content: center; padding: .55rem .35rem; }
  .video-workspace-tabs__icon { width: 1.65rem; height: 1.65rem; }
  .video-workspace-tabs small { display: none; }
  .video-workspace-tabs strong { font-size: .62rem; }
}
</style>
