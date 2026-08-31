<template>
  <UserWorkspaceLayout :hide-announcement="true">
    <div class="ai-creation-route">
      <div v-if="loading" class="ai-creation-state">
        <Icon name="sparkles" size="xl" />
        <strong>正在准备 AI 创作工作区</strong>
        <span>正在接入当前登录态和可用 Key</span>
      </div>
      <div v-else-if="!hasCreationKey" class="ai-creation-state ai-creation-state--empty">
        <Icon name="key" size="xl" />
        <strong>{{ emptyStateTitle }}</strong>
        <span>{{ emptyStateDescription }}</span>
        <router-link to="/keys?create=1" class="ai-creation-link">
          <Icon name="plus" size="sm" />
          创建 Key
        </router-link>
      </div>
      <iframe
        v-else
        :key="frameKey"
        :src="frameSrc"
        title="AI 创作"
        class="ai-creation-frame"
        allow="clipboard-read; clipboard-write"
      />
    </div>
  </UserWorkspaceLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { keysAPI } from '@/api'
import type { ApiKey } from '@/types'
import Icon from '@/components/icons/Icon.vue'
import UserWorkspaceLayout from '@/components/layout/UserWorkspaceLayout.vue'
import {
  getOpenAIModelIds,
  getTextCreationGroupKeys,
  isImageCreationKey,
} from '@/utils/creationAccess'

const BRIDGE_STORAGE_KEY = 'lingqu:ai-creation:bridge'
const ENTRY_PATH = import.meta.env.DEV ? '/ai-creation/index.html' : '/ai-creation/'

const route = useRoute()
const keys = ref<ApiKey[]>([])
const loading = ref(true)
const frameSrc = ref('')
const frameKey = ref(0)

type TextCreationGroup = {
  key: ApiKey
  models: string[]
}

const textCreationGroups = ref<TextCreationGroup[]>([])

const activeKeys = computed(() => {
  return keys.value.filter((key) => key.status === 'active' && key.group)
})

const imageKeys = computed(() => {
  return keys.value.filter(isImageCreationKey)
})

const textGroupKeys = computed(() => {
  return getTextCreationGroupKeys(keys.value)
})

const imageKey = computed(() => {
  return imageKeys.value[0] || null
})

const videoKey = computed(() => {
  return activeKeys.value.find((key) => key.group?.platform === 'xiaoapi') || null
})

function requestedSection() {
  const querySection = typeof route.query.section === 'string' ? route.query.section : ''
  return querySection || (route.path === '/video-workbench' ? 'video' : 'canvas')
}

const creationSection = computed(() => requestedSection())
const requiredKey = computed(() => {
  if (creationSection.value === 'video') return videoKey.value
  if (creationSection.value === 'image') return imageKey.value
  return imageKey.value || textCreationGroups.value[0]?.key || null
})
const hasCreationKey = computed(() => Boolean(requiredKey.value))
const emptyStateTitle = computed(() => creationSection.value === 'video' ? '还没有可用的视频创作 Key' : creationSection.value === 'image' ? '还没有可用的生图 Key' : '还没有可用的创作 Key')
const emptyStateDescription = computed(() => creationSection.value === 'video' ? '先创建一个视频渠道 Key，再回来使用视频创作。' : creationSection.value === 'image' ? '先创建一个 OpenAI 分组且开启生图权限的 Key，再回来使用生图工作台。' : '先创建一个可用的 OpenAI 文本或生图 Key，再回来使用无限画布。')

function textGroupDisplayName(key: ApiKey) {
  const name = key.group?.name || key.name
  const duplicateName = textGroupKeys.value.some((candidate) => (
    candidate.group?.id !== key.group?.id &&
    (candidate.group?.name || candidate.name) === name
  ))
  return duplicateName ? `${name} (#${key.group?.id})` : name
}

async function loadTextCreationGroups() {
  const groups = await Promise.all(textGroupKeys.value.map(async (key): Promise<TextCreationGroup | null> => {
    try {
      const response = await fetch(`${window.location.origin}/v1/models`, {
        credentials: 'same-origin',
        headers: {
          Accept: 'application/json',
          Authorization: `Bearer ${key.key}`,
        },
      })
      if (!response.ok) return null
      const models = getOpenAIModelIds(await response.json())
      return models.length ? { key, models } : null
    } catch {
      return null
    }
  }))

  textCreationGroups.value = groups.filter((group): group is TextCreationGroup => Boolean(group))
}

function writeBridge() {
  const image = imageKey.value
  const textGroup = textCreationGroups.value[0] || null
  const text = textGroup?.key || null
  const video = videoKey.value
  const creationTheme = window.localStorage.getItem('theme') === 'dark' ? 'dark' : 'light'
  const payload = JSON.stringify({
    apiUrl: image || text ? `${window.location.origin}/v1` : undefined,
    videoApiUrl: video ? `${window.location.origin}/api/v1/video` : undefined,
    imageKeyId: image?.id,
    imageApiKey: image?.key || '',
    imageKeyName: image?.name || '',
    imageGroupName: image?.group?.name || '',
    imageKeys: imageKeys.value.map((key) => ({
      id: key.id,
      apiKey: key.key,
      keyName: key.name,
      groupName: key.group?.name || key.name,
      model: 'gpt-image-2',
    })),
    textKeyId: text?.id,
    textApiKey: text?.key || '',
    textKeyName: text?.name || '',
    textGroupName: text?.group?.name || '',
    textKeys: textCreationGroups.value.map(({ key, models }) => ({
      id: key.group?.id || key.id,
      keyId: key.id,
      apiKey: key.key,
      keyName: key.name,
      groupName: textGroupDisplayName(key),
      models,
    })),
    keyId: image?.id || text?.id || video?.id,
    apiKey: image?.key || text?.key || video?.key || '',
    keyName: image?.name || text?.name || video?.name || '',
    groupName: image?.group?.name || text?.group?.name || '',
    videoKeyId: video?.id,
    videoApiKey: video?.key || '',
    videoKeyName: video?.name || '',
    videoGroupName: video?.group?.name || '',
    model: 'gpt-image-2',
    userEmail: '',
    userTheme: window.localStorage.getItem('user-workspace-theme') || 'business',
    theme: creationTheme,
    launchedAt: Date.now(),
  })
  window.sessionStorage.setItem(BRIDGE_STORAGE_KEY, payload)
  window.localStorage.setItem(BRIDGE_STORAGE_KEY, payload)
  window.localStorage.setItem('infinite-canvas:theme_store', JSON.stringify({
    state: { theme: creationTheme },
    version: 0,
  }))
}

async function launch() {
  try {
    const allKeys = await keysAPI.listAll(100, {
      status: 'active',
      sort_by: 'created_at',
      sort_order: 'desc',
    })
    keys.value = allKeys
    await loadTextCreationGroups()
    if (!hasCreationKey.value) return
    writeBridge()
    const section = requestedSection()
    frameSrc.value = `${ENTRY_PATH}?lingqu=${Date.now()}${section ? `#/${section}` : ''}`
    frameKey.value += 1
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void launch()
})
</script>

<style scoped>
.ai-creation-route {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #fff;
}

.ai-creation-frame {
  display: block;
  flex: 1 1 auto;
  height: 100%;
  min-height: 0;
  width: 100%;
  border: 0;
  background: #fff;
}

.ai-creation-state {
  display: grid;
  min-height: 30rem;
  place-items: center;
  align-content: center;
  gap: 0.55rem;
  padding: 2rem;
  color: rgba(33, 31, 28, 0.62);
  text-align: center;
}

.ai-creation-state :deep(.icon) {
  color: #0f766e;
}

.ai-creation-state strong {
  color: #211f1c;
  font-size: 1.1rem;
}

.ai-creation-state span {
  max-width: 30rem;
  font-size: 0.9rem;
}

.ai-creation-link {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  margin-top: 0.45rem;
  border-radius: 10px;
  background: #211f1c;
  padding: 0.62rem 0.9rem;
  color: #fff;
  text-decoration: none;
}

</style>
