<template>
  <UserWorkspaceLayout>
    <div class="video-studio">
      <VideoWorkspaceTabs />

      <div v-if="!selectedKey && !loadingKeys" class="video-studio__empty">
        <span><Icon name="key" size="xl" /></span>
        <strong>{{ videoKeys.length ? '选择一个视频 Key 开始创作' : '还没有可用的视频 Key' }}</strong>
        <p>{{ videoKeys.length ? '模型和历史任务会按照所选 Key 隔离。' : '先创建 XiaoAPI 分组 Key，再回到这里使用视频工作台。' }}</p>
        <router-link v-if="videoKeys.length === 0" to="/keys?create=1">创建 Key</router-link>
      </div>

      <div v-else class="video-studio__shell">
        <section class="video-creator">
          <section class="video-key-bar" aria-label="创作 Key">
            <div class="video-key-bar__title">
              <span><Icon name="key" size="sm" /></span>
              <div><strong>创作 Key</strong><small>模型和任务按 Key 独立</small></div>
            </div>
            <div class="video-key-select">
              <select id="video-key-select" v-model="selectedKeyId" :disabled="loadingKeys || videoKeys.length === 0">
                <option value="">选择 XiaoAPI 分组 Key</option>
                <option v-for="key in videoKeys" :key="key.id" :value="String(key.id)">
                  {{ key.name }} · {{ key.group?.name || '视频分组' }}
                </option>
              </select>
              <button type="button" class="video-icon-button" title="刷新 Key" :disabled="loadingKeys" @click="loadKeys">
                <Icon name="refresh" size="sm" :class="{ 'animate-spin': loadingKeys }" />
              </button>
            </div>
            <span class="video-key-bar__count">{{ videoKeys.length }} 个可用</span>
          </section>

          <section class="video-model-shelf" aria-labelledby="video-model-heading">
            <div class="video-model-shelf__header">
              <div>
                <span id="video-model-heading">选择模型</span>
                <strong>{{ selectedCapability?.label || '请选择一个视频模型' }}</strong>
              </div>
              <button type="button" class="video-icon-button" title="刷新模型" :disabled="loadingModels" @click="loadWorkspace">
                <Icon name="refresh" size="sm" :class="{ 'animate-spin': loadingModels }" />
              </button>
            </div>
            <div v-if="loadingModels && capabilities.length === 0" class="video-model-shelf__loading">正在读取可用模型…</div>
            <div v-else class="video-model-grid" role="radiogroup" aria-label="视频模型">
              <button
                v-for="item in capabilities"
                :key="item.id"
                type="button"
                role="radio"
                class="video-model-card"
                :class="{ 'video-model-card--active': selectedModelId === item.id }"
                :aria-checked="selectedModelId === item.id"
                @click="selectModel(item)"
              >
                <span class="video-model-card__mark"><Icon name="play" size="sm" /></span>
                <span class="video-model-card__copy"><strong>{{ item.label }}</strong><small>{{ item.id }}</small></span>
                <span class="video-model-card__spec">{{ item.resolutions.join(' / ') }}</span>
                <span class="video-model-card__features">
                  <em v-if="item.supportsAudio">音轨</em>
                  <em v-if="Object.values(item.maxReferences).some((count) => count > 0)">素材</em>
                </span>
                <Icon v-if="selectedModelId === item.id" name="checkCircle" size="sm" />
              </button>
            </div>
          </section>

          <div class="video-studio__workspace">

        <main class="video-composer">
          <div class="video-composer__topline">
            <div>
              <span>创作内容</span>
              <strong>{{ selectedCapability?.label || '请选择模型' }} · {{ selectedModeLabel }}</strong>
            </div>
            <span v-if="retryingSameRequest" class="video-retry-badge">可安全重试</span>
          </div>

          <div class="video-mode-tabs" role="tablist" aria-label="创作方式">
            <button
              v-for="item in availableModes"
              :key="item.value"
              type="button"
              role="tab"
              :aria-selected="creationMode === item.value"
              :class="{ 'video-mode-tab--active': creationMode === item.value }"
              @click="creationMode = item.value"
            >
              <Icon :name="item.icon" size="sm" />
              <span>{{ item.label }}</span>
            </button>
          </div>

          <section v-if="creationMode === 'frames'" class="video-media-band" aria-label="首尾帧">
            <MediaInput
              label="首帧"
              :hint="isAIStartLab ? '粘贴公开的 HTTP(S) 图片链接' : 'PNG / JPEG / WebP，最大 10 MiB'"
              accept="image/png,image/jpeg,image/webp"
              :remote="isAIStartLab"
              :required="selectedCapability?.requiresStartFrame"
              :item="startFrame"
              @select="setSingleMedia('start', $event)"
              @select-url="setSingleMediaUrl('start', $event)"
              @remove="clearSingleMedia('start')"
            />
            <MediaInput
              v-if="selectedCapability?.supportsEndFrame"
              label="尾帧"
              hint="让镜头自然过渡到目标构图"
              accept="image/png,image/jpeg,image/webp"
              :remote="isAIStartLab"
              :item="endFrame"
              @select="setSingleMedia('end', $event)"
              @select-url="setSingleMediaUrl('end', $event)"
              @remove="clearSingleMedia('end')"
            />
          </section>

          <section v-if="creationMode === 'references'" class="video-reference-section" aria-label="参考素材">
            <div class="video-reference-row" v-if="referenceLimit('image') > 0">
              <div><strong>参考图片</strong><span>最多 {{ referenceLimit('image') }} 张</span></div>
              <RemoteMediaInput
                v-if="isAIStartLab"
                kind="image"
                placeholder="粘贴公开图片 URL"
                @select-url="addReferenceUrl('image', $event)"
              />
              <label v-else class="video-add-media">
                <Icon name="plus" size="sm" /><span>添加图片</span>
                <input type="file" multiple accept="image/png,image/jpeg,image/webp" @change="addReferenceFiles('image', $event)" />
              </label>
            </div>
            <div v-if="referenceImages.length" class="video-media-list">
              <MediaChip v-for="item in referenceImages" :key="item.id" :item="item" @remove="removeReference('image', item.id)" />
            </div>

            <div class="video-reference-row" v-if="referenceLimit('video') > 0">
              <div><strong>参考视频</strong><span>MP4 / MOV，最大 100 MiB</span></div>
              <RemoteMediaInput
                v-if="isAIStartLab"
                kind="video"
                placeholder="粘贴公开视频 URL"
                @select-url="addReferenceUrl('video', $event)"
              />
              <label v-else class="video-add-media">
                <Icon name="plus" size="sm" /><span>添加视频</span>
                <input type="file" multiple accept="video/mp4,video/quicktime" @change="addReferenceFiles('video', $event)" />
              </label>
            </div>
            <div v-if="referenceVideos.length" class="video-media-list">
              <MediaChip v-for="item in referenceVideos" :key="item.id" :item="item" @remove="removeReference('video', item.id)" />
            </div>

            <div class="video-reference-row" v-if="referenceLimit('audio') > 0">
              <div><strong>参考音频</strong><span>MP3 / WAV，需同时提供图片或视频</span></div>
              <RemoteMediaInput
                v-if="isAIStartLab"
                kind="audio"
                placeholder="粘贴公开音频 URL"
                @select-url="addReferenceUrl('audio', $event)"
              />
              <label v-else class="video-add-media">
                <Icon name="plus" size="sm" /><span>添加音频</span>
                <input type="file" accept="audio/mpeg,audio/wav,audio/x-wav" @change="addReferenceFiles('audio', $event)" />
              </label>
            </div>
            <div v-if="referenceAudios.length" class="video-media-list">
              <MediaChip v-for="item in referenceAudios" :key="item.id" :item="item" @remove="removeReference('audio', item.id)" />
            </div>
          </section>

          <label class="video-prompt">
            <span>
              <strong>画面描述</strong>
              <small>{{ prompt.length }} / {{ selectedCapability?.promptLimit || 5000 }}</small>
            </span>
            <textarea
              v-model="prompt"
              :maxlength="selectedCapability?.promptLimit || 5000"
              rows="5"
              placeholder="描述主体、动作、环境、镜头运动、光线和整体风格…"
            ></textarea>
          </label>

          <section class="video-shot-language" aria-label="镜头语言">
            <span><Icon name="sparkles" size="xs" /> 镜头语言</span>
            <div>
              <button v-for="item in promptCues" :key="item.label" type="button" @click="appendPromptCue(item.value)">
                {{ item.label }}
              </button>
            </div>
          </section>

          <div v-if="formError" class="video-form-error">
            <Icon name="exclamationCircle" size="sm" />
            <span>{{ formError }}</span>
          </div>

          <footer class="video-composer__actions">
            <span>{{ uploading ? '正在上传参考素材…' : submitting ? '正在提交任务…' : '创建后可在历史任务中查看进度' }}</span>
            <button type="button" :disabled="!canSubmit || submitting || uploading" @click="submitVideo">
              <Icon :name="submitting || uploading ? 'refresh' : 'play'" size="sm" :class="{ 'animate-spin': submitting || uploading }" />
              {{ retryingSameRequest ? '安全重试' : '开始生成' }}
            </button>
          </footer>
            </main>

            <aside class="video-parameters" aria-label="生成参数">
            <div class="video-preview-stage">
              <div class="video-preview-stage__hud">
                <span><i aria-hidden="true"></i> REC / MONITOR</span>
                <span>{{ resolution }} · {{ aspectRatio }}</span>
              </div>
              <div class="video-preview-stage__empty">
                <span><Icon name="play" size="lg" /></span>
                <strong>输出画布</strong>
                <small>完成后可在历史任务中播放和下载</small>
              </div>
            </div>
            <div class="video-monitor-meta">
              <div><span>MODEL</span><strong>{{ selectedCapability?.label || '未选择' }}</strong></div>
              <div><span>LENGTH</span><strong>{{ duration }}s</strong></div>
              <div><span>OUTPUT</span><strong>{{ resolution }}</strong></div>
            </div>
            <div class="video-sidebar__title">
              <div><strong>输出设置</strong><small>按模型自动适配</small></div>
            </div>
            <section class="video-settings">
              <div class="video-setting video-setting--resolution">
                <span>分辨率</span>
                <div class="video-segments">
                  <button
                    v-for="item in selectedCapability?.resolutions || []"
                    :key="item"
                    type="button"
                    :class="{ 'video-segment--active': resolution === item }"
                    @click="resolution = item"
                  >{{ item }}</button>
                </div>
              </div>

              <div class="video-setting">
                <span>时长</span>
                <div class="video-duration">
                  <input v-model.number="durationIndex" type="range" min="0" :max="Math.max(0, durationOptions.length - 1)" step="1" />
                  <strong>{{ duration }} 秒</strong>
                </div>
              </div>

              <div class="video-setting video-setting--ratio">
                <span>画面比例</span>
                <div class="video-segments video-segments--ratios">
                  <button
                    v-for="item in aspectRatioOptions"
                    :key="item"
                    type="button"
                    :class="{ 'video-segment--active': aspectRatio === item }"
                    @click="aspectRatio = item"
                  >{{ item }}</button>
                </div>
              </div>

              <label v-if="selectedCapability?.supportsPromptEnhance" class="video-setting">
                <span>提示词增强</span>
                <select v-model="promptEnhance">
                  <option value="AUTO">自动</option>
                  <option value="ON" :disabled="selectedModelId === 'happy-horse-1.1' && Boolean(startFrame)">开启</option>
                  <option value="OFF">关闭</option>
                </select>
              </label>

              <label v-if="selectedCapability?.supportsAudio" class="video-audio-toggle">
                <span>
                  <strong>生成音轨</strong>
                  <small>随视频生成匹配的声音</small>
                </span>
                <input v-model="audio" type="checkbox" />
                <i aria-hidden="true"></i>
              </label>
            </section>

            <section v-if="selectedCapability" class="video-capability" aria-labelledby="video-capability-title">
              <header>
                <span><Icon name="infoCircle" size="sm" /></span>
                <div><strong id="video-capability-title">{{ selectedCapability.usesXiaoAPIRules ? '当前模型限制' : '当前可用参数' }}</strong><small>{{ selectedCapability.label }}</small></div>
                <em>{{ capabilitySourceLabel }}</em>
              </header>
              <dl>
                <div><dt>输出</dt><dd>{{ capabilityOutput }}</dd></div>
                <div><dt>画面比例</dt><dd>{{ aspectRatioOptions.join(' / ') }}</dd></div>
                <div><dt>成品音频</dt><dd>{{ selectedCapability.supportsAudio ? '支持生成同步音轨' : '不支持生成音轨' }}</dd></div>
                <div><dt>首尾帧</dt><dd>{{ frameSupportLabel }}</dd></div>
                <div><dt>参考素材</dt><dd>{{ referenceSupportLabel }}</dd></div>
                <div><dt>提示词</dt><dd>{{ selectedCapability.usesXiaoAPIRules ? `最多 ${selectedCapability.promptLimit} 字` : `本站输入最多 ${selectedCapability.promptLimit} 字，上游限制以 AIStartLab 为准` }}</dd></div>
              </dl>
              <div v-if="selectedCapability.usesXiaoAPIRules" class="video-capability__section">
                <strong>素材边界</strong>
                <p>图片 / 首尾帧：{{ videoMediaLimits.image.formats }}，单个 ≤ {{ videoMediaLimits.image.maxMiB }} MiB，宽高 {{ videoMediaLimits.image.minWidth }}-{{ videoMediaLimits.image.maxWidth }} px，比例 {{ videoMediaLimits.image.minAspectRatio }}-{{ videoMediaLimits.image.maxAspectRatio }}</p>
                <p v-if="selectedCapability.maxReferences.video > 0">视频：{{ videoMediaLimits.video.formats }}，单段 {{ videoMediaLimits.video.minDuration }}-{{ videoMediaLimits.video.maxDuration }} 秒，合计 ≤ {{ videoMediaLimits.video.maxTotalDuration }} 秒，单个 ≤ {{ videoMediaLimits.video.maxMiB }} MiB</p>
                <p v-if="selectedCapability.maxReferences.audio > 0">音频：{{ videoMediaLimits.audio.formats }}，时长 {{ videoMediaLimits.audio.minDuration }}-{{ videoMediaLimits.audio.maxDuration }} 秒，单个 ≤ {{ videoMediaLimits.audio.maxMiB }} MiB</p>
              </div>
              <div v-if="selectedCapability.usesXiaoAPIRules" class="video-capability__section">
                <strong>组合提醒</strong>
                <p v-for="note in capabilityNotes" :key="note">{{ note }}</p>
              </div>
              <div v-else class="video-capability__section">
                <strong>{{ selectedCapability.capabilitySource === 'aistartlab' ? 'AIStartLab 参数说明' : '上游参数说明' }}</strong>
                <p>这里只展示上游模型元数据与本站价格配置中明确提供的参数；AIStartLab 的素材必须是公网 HTTP(S) URL，XiaoAPI 的本地文件边界不适用于它。</p>
                <p v-if="selectedCapability.capabilitySource === 'mixed'">该模型同时来自多种上游，提交时仅使用双方都能安全支持的基础参数。</p>
              </div>
            </section>
            </aside>
          </div>
        </section>
      </div>
    </div>
  </UserWorkspaceLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onBeforeUnmount, onMounted, ref, watch, type PropType } from 'vue'
import { useRoute } from 'vue-router'
import { keysAPI, videoAPI } from '@/api'
import type { ApiKey } from '@/types'
import { VideoAPIError, type UploadedVideoMedia, type VideoModel } from '@/api/video'
import UserWorkspaceLayout from '@/components/layout/UserWorkspaceLayout.vue'
import VideoWorkspaceTabs from '@/components/video/VideoWorkspaceTabs.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import {
  aspectRatiosFor,
  createIdempotencyKey,
  durationsFor,
  resolveVideoCapability,
  videoMediaLimits,
  type ReferenceKind,
  type VideoCreationMode,
  type VideoModelCapability,
} from '@/utils/videoCapabilities'

interface MediaSelection {
  id: string
  kind: ReferenceKind
  file: File | null
  remoteUrl?: string
  previewUrl: string
  uploaded: UploadedVideoMedia | null
  status: 'ready' | 'uploading' | 'uploaded' | 'error'
  error: string
}

const MediaInput = defineComponent({
  props: {
    label: { type: String, required: true }, hint: { type: String, required: true },
    accept: { type: String, required: true }, required: Boolean, remote: Boolean,
    item: { type: Object as PropType<MediaSelection | null>, default: null },
  },
  emits: ['select', 'select-url', 'remove'],
  setup(props, { emit }) {
    const remoteUrl = ref('')
    const selectRemote = () => {
      const value = remoteUrl.value.trim()
      if (!/^https?:\/\/[^\s]+$/i.test(value)) return
      emit('select-url', value)
      remoteUrl.value = ''
    }
    return () => h('div', { class: 'video-media-input' }, [
      h('div', { class: 'video-media-input__copy' }, [
        h('strong', props.label), props.required ? h('em', '必需') : null, h('span', props.hint),
      ]),
      props.item
        ? h('div', { class: 'video-media-input__selected' }, [
            h('span', props.item.file?.name || props.item.remoteUrl || ''),
            h('button', { type: 'button', title: '移除', onClick: () => emit('remove') }, [h(Icon, { name: 'x', size: 'sm' })]),
          ])
        : props.remote
          ? h('div', { class: 'video-remote-media-input' }, [
              h('input', {
                value: remoteUrl.value,
                type: 'url',
                placeholder: 'https://…',
                onInput: (event: Event) => { remoteUrl.value = (event.target as HTMLInputElement).value },
                onKeydown: (event: KeyboardEvent) => { if (event.key === 'Enter') { event.preventDefault(); selectRemote() } },
              }),
              h('button', { type: 'button', onClick: selectRemote }, '使用链接'),
            ])
        : h('label', { class: 'video-media-input__add' }, [
            h(Icon, { name: 'upload', size: 'sm' }), h('span', '选择文件'),
            h('input', {
              type: 'file', accept: props.accept,
              onChange: (event: Event) => {
                const input = event.target as HTMLInputElement
                if (input.files?.[0]) emit('select', input.files[0])
                input.value = ''
              },
            }),
          ]),
    ])
  },
})

const RemoteMediaInput = defineComponent({
  props: {
    kind: { type: String as PropType<ReferenceKind>, required: true },
    placeholder: { type: String, required: true },
  },
  emits: ['select-url'],
  setup(props, { emit }) {
    const value = ref('')
    const submit = () => {
      const url = value.value.trim()
      if (!/^https?:\/\/[^\s]+$/i.test(url)) return
      emit('select-url', url)
      value.value = ''
    }
    return () => h('div', { class: 'video-remote-reference-input' }, [
      h('input', {
        value: value.value,
        type: 'url',
        placeholder: props.placeholder,
        onInput: (event: Event) => { value.value = (event.target as HTMLInputElement).value },
        onKeydown: (event: KeyboardEvent) => { if (event.key === 'Enter') { event.preventDefault(); submit() } },
      }),
      h('button', { type: 'button', onClick: submit }, [h(Icon, { name: 'plus', size: 'xs' }), h('span', '添加')]),
    ])
  },
})

const MediaChip = defineComponent({
  props: { item: { type: Object as PropType<MediaSelection>, required: true } },
  emits: ['remove'],
  setup(props, { emit }) {
    return () => h('div', { class: ['video-media-chip', `video-media-chip--${props.item.status}`] }, [
      props.item.kind === 'image'
        ? h('img', { src: props.item.previewUrl, alt: '' })
        : h('span', { class: 'video-media-chip__type' }, props.item.kind === 'video' ? '视频' : '音频'),
      h('span', { title: props.item.file?.name || props.item.remoteUrl }, props.item.file?.name || props.item.remoteUrl || ''),
      h('button', { type: 'button', title: '移除', onClick: () => emit('remove') }, [h(Icon, { name: 'x', size: 'xs' })]),
    ])
  },
})

const route = useRoute()
const appStore = useAppStore()
const apiKeys = ref<ApiKey[]>([])
const models = ref<VideoModel[]>([])
const selectedKeyId = ref('')
const selectedModelId = ref('')
const creationMode = ref<VideoCreationMode>('text')
const prompt = ref('')
const promptCues = [
  { label: '缓慢推进', value: '镜头缓慢向主体推进' },
  { label: '环绕运镜', value: '镜头围绕主体平稳环绕' },
  { label: '手持跟拍', value: '手持镜头跟随主体移动' },
  { label: '升降航拍', value: '航拍镜头逐渐升高并展开环境' },
  { label: '浅景深', value: '浅景深，背景柔和虚化' },
  { label: '电影光影', value: '电影级光影与自然明暗层次' },
]
const resolution = ref('')
const aspectRatio = ref('')
const durationIndex = ref(0)
const audio = ref(true)
const promptEnhance = ref<'AUTO' | 'ON' | 'OFF'>('AUTO')
const startFrame = ref<MediaSelection | null>(null)
const endFrame = ref<MediaSelection | null>(null)
const referenceImages = ref<MediaSelection[]>([])
const referenceVideos = ref<MediaSelection[]>([])
const referenceAudios = ref<MediaSelection[]>([])
const loadingKeys = ref(false)
const loadingModels = ref(false)
const submitting = ref(false)
const uploading = ref(false)
const formError = ref('')
const pendingIdempotencyKey = ref('')
const pendingRequestBody = ref('')
let workspaceRequest = 0

const videoKeys = computed(() => apiKeys.value.filter((key) => key.status === 'active' && key.group?.platform === 'xiaoapi'))
const selectedKey = computed(() => videoKeys.value.find((key) => String(key.id) === selectedKeyId.value) || null)
const capabilities = computed(() => models.value.map(resolveVideoCapability).filter((item) => item.resolutions.length > 0 && item.durations.length > 0))
const selectedCapability = computed<VideoModelCapability | null>(() => capabilities.value.find((item) => item.id === selectedModelId.value) || null)
const isAIStartLab = computed(() => selectedCapability.value?.capabilitySource === 'aistartlab')
const durationOptions = computed(() => selectedCapability.value ? durationsFor(selectedCapability.value, resolution.value) : [5])
const duration = computed(() => durationOptions.value[durationIndex.value] || durationOptions.value[0] || 5)
const aspectRatioOptions = computed(() => selectedCapability.value ? aspectRatiosFor(selectedCapability.value, resolution.value) : ['16:9'])
const availableModes = computed(() => {
  const capability = selectedCapability.value
  const modes: Array<{ value: VideoCreationMode; label: string; icon: 'sparkles' | 'image' | 'grid' }> = []
  if (!capability?.requiresStartFrame) modes.push({ value: 'text', label: '文生视频', icon: 'sparkles' })
  if (capability?.supportsStartFrame) modes.push({ value: 'frames', label: capability.supportsEndFrame ? '首尾帧' : '首帧驱动', icon: 'image' })
  if (capability && Object.values(capability.maxReferences).some((count) => count > 0)) {
    modes.push({ value: 'references', label: '参考素材', icon: 'grid' })
  }
  return modes
})
const selectedModeLabel = computed(() => availableModes.value.find((item) => item.value === creationMode.value)?.label || '文生视频')
const canSubmit = computed(() => Boolean(
  selectedKey.value
  && selectedCapability.value
  && prompt.value.trim()
  && (!selectedCapability.value.requiresStartFrame || startFrame.value),
))
const retryingSameRequest = computed(() => Boolean(pendingIdempotencyKey.value && pendingRequestBody.value))
const capabilitySourceLabel = computed(() => {
  switch (selectedCapability.value?.capabilitySource) {
    case 'xiaoapi': return 'XiaoAPI 规则'
    case 'aistartlab': return 'AIStartLab 元数据'
    case 'mixed': return '混合上游'
    default: return '上游元数据'
  }
})
const capabilityOutput = computed(() => {
  const capability = selectedCapability.value
  if (!capability) return ''
  const durations = durationOptions.value
  const consecutive = durations.every((value, index) => index === 0 || value === durations[index - 1] + 1)
  const durationText = consecutive && durations.length > 2
    ? `${durations[0]}-${durations[durations.length - 1]} 秒`
    : `${durations.join(' / ')} 秒`
  const evenOnly = durations.length > 2 && durations.every((value) => value % 2 === 0) && !consecutive
  return `分辨率：${capability.resolutions.join(' / ')} · 当前分辨率时长：${durationText}${evenOnly ? '（偶数）' : ''}`
})
const frameSupportLabel = computed(() => {
  const capability = selectedCapability.value
  if (!capability?.supportsStartFrame) return '不支持首尾帧'
  if (capability.requiresStartFrame) return '必须上传首帧'
  if (capability.supportsEndFrame) return '支持首帧和尾帧，尾帧需先传首帧'
  return '支持首帧'
})
const referenceSupportLabel = computed(() => {
  const limits = selectedCapability.value?.maxReferences
  if (!limits || !Object.values(limits).some((count) => count > 0)) return '不支持参考图片、视频或音频'
  const parts = [
    limits.image > 0 ? `图片最多 ${limits.image} 张` : '图片不支持',
    limits.video > 0 ? `视频最多 ${limits.video} 个` : '视频不支持',
    limits.audio > 0 ? `音频最多 ${limits.audio} 个` : '音频不支持',
  ]
  return parts.join(' · ')
})
const capabilityNotes = computed(() => {
  const capability = selectedCapability.value
  if (!capability) return []
  const notes = ['分辨率、画面比例和时长以上方当前可选项为准；提示词里尽量不要重复写横屏、竖屏或分辨率。']
  if (capability.supportsEndFrame) notes.push('首帧或尾帧不能与参考素材同时使用；上传尾帧前必须先上传首帧。')
  if (capability.maxReferences.audio > 0) notes.push('参考音频必须搭配参考图片或参考视频，且参考视频和参考音频不能同时使用。')
  if (capability.requiresStartFrame) notes.push('该模型不支持纯文本直出，提交前必须上传首帧。')
  return notes
})

function selectedIdStorageKey() { return 'lingqu:video-studio:selected-key-id' }
function appendPromptCue(value: string) {
  const current = prompt.value.trim()
  const separator = current && !/[，。！？；]$/.test(current) ? '，' : ''
  const limit = selectedCapability.value?.promptLimit || 5000
  prompt.value = `${current}${separator}${value}`.slice(0, limit)
}
function selectModel(item: VideoModelCapability) {
  selectedModelId.value = item.id
}
function shortJobId(id: string) { return id.length > 16 ? `${id.slice(0, 9)}…${id.slice(-5)}` : id }
function errorMessage(error: unknown) {
  if (error instanceof VideoAPIError) {
    const requestId = error.requestId ? `（请求 ID：${error.requestId}）` : ''
    return `${error.message}${requestId}`
  }
  return error instanceof Error ? error.message : '请求失败，请稍后重试'
}

function createMedia(file: File, kind: ReferenceKind): MediaSelection {
  return {
    id: `${kind}-${Date.now()}-${Math.random().toString(36).slice(2)}`,
    kind, file, previewUrl: URL.createObjectURL(file), uploaded: null, status: 'ready', error: '',
  }
}
function createRemoteMedia(url: string, kind: ReferenceKind): MediaSelection {
  return {
    id: `${kind}-remote-${Date.now()}-${Math.random().toString(36).slice(2)}`,
    kind, file: null, remoteUrl: url, previewUrl: kind === 'image' ? url : '', uploaded: null, status: 'uploaded', error: '',
  }
}
function releaseMedia(item: MediaSelection | null) { if (item?.previewUrl) URL.revokeObjectURL(item.previewUrl) }
function releaseAllMedia() {
  releaseMedia(startFrame.value); releaseMedia(endFrame.value)
  ;[...referenceImages.value, ...referenceVideos.value, ...referenceAudios.value].forEach(releaseMedia)
  startFrame.value = null; endFrame.value = null
  referenceImages.value = []; referenceVideos.value = []; referenceAudios.value = []
}
function validateMediaFile(file: File, kind: ReferenceKind): string {
  const allowed = kind === 'image'
    ? ['image/png', 'image/jpeg', 'image/webp']
    : kind === 'video' ? ['video/mp4', 'video/quicktime'] : ['audio/mpeg', 'audio/wav', 'audio/x-wav']
  const maxMiB = videoMediaLimits[kind].maxMiB
  const max = maxMiB << 20
  if (!allowed.includes(file.type)) return `${file.name} 的格式不受支持`
  if (file.size > max) return `${file.name} 超出 ${maxMiB} MiB 限制`
  return ''
}
function setSingleMedia(target: 'start' | 'end', file: File) {
  const error = validateMediaFile(file, 'image')
  if (error) { formError.value = error; return }
  const current = target === 'start' ? startFrame.value : endFrame.value
  releaseMedia(current)
  if (target === 'start') startFrame.value = createMedia(file, 'image')
  else endFrame.value = createMedia(file, 'image')
  formError.value = ''
}
function setSingleMediaUrl(target: 'start' | 'end', url: string) {
  const current = target === 'start' ? startFrame.value : endFrame.value
  releaseMedia(current)
  const selection = createRemoteMedia(url, 'image')
  if (target === 'start') startFrame.value = selection
  else endFrame.value = selection
  formError.value = ''
}
function clearSingleMedia(target: 'start' | 'end') {
  const current = target === 'start' ? startFrame.value : endFrame.value
  releaseMedia(current)
  if (target === 'start') startFrame.value = null
  else endFrame.value = null
}
function referenceList(kind: ReferenceKind) {
  return kind === 'image' ? referenceImages : kind === 'video' ? referenceVideos : referenceAudios
}
function referenceLimit(kind: ReferenceKind) { return selectedCapability.value?.maxReferences[kind] || 0 }
function addReferenceFiles(kind: ReferenceKind, event: Event) {
  const input = event.target as HTMLInputElement
  const list = referenceList(kind)
  const remaining = Math.max(0, referenceLimit(kind) - list.value.length)
  const files = Array.from(input.files || []).slice(0, remaining)
  const errors = files.map((file) => validateMediaFile(file, kind)).filter(Boolean)
  if (errors.length) formError.value = errors[0]
  list.value.push(...files.filter((file) => !validateMediaFile(file, kind)).map((file) => createMedia(file, kind)))
  input.value = ''
}
function addReferenceUrl(kind: ReferenceKind, url: string) {
  const list = referenceList(kind)
  if (list.value.length >= referenceLimit(kind)) {
    formError.value = `${kind === 'image' ? '参考图片' : kind === 'video' ? '参考视频' : '参考音频'} 已达到数量上限`
    return
  }
  list.value.push(createRemoteMedia(url, kind))
  formError.value = ''
}
function removeReference(kind: ReferenceKind, id: string) {
  const list = referenceList(kind)
  const target = list.value.find((item) => item.id === id) || null
  releaseMedia(target)
  list.value = list.value.filter((item) => item.id !== id)
}

async function loadKeys() {
  loadingKeys.value = true
  try {
    const response = await keysAPI.list(1, 100, { status: 'active', sort_by: 'created_at', sort_order: 'desc' })
    apiKeys.value = response.items
    const ids = new Set(videoKeys.value.map((key) => String(key.id)))
    const queryId = typeof route.query.key_id === 'string' ? route.query.key_id : ''
    const storedId = window.localStorage.getItem(selectedIdStorageKey()) || ''
    selectedKeyId.value = [queryId, selectedKeyId.value, storedId, String(videoKeys.value[0]?.id || '')].find((id) => ids.has(id)) || ''
  } catch (error) {
    appStore.showError(errorMessage(error))
  } finally {
    loadingKeys.value = false
  }
}
async function loadWorkspace() {
  const key = selectedKey.value
  const requestId = ++workspaceRequest
  const previousModelId = selectedModelId.value
  models.value = []; selectedModelId.value = ''
  if (!key) return
  loadingModels.value = true
  try {
    const nextModels = await videoAPI.listModels(key.key)
    if (requestId !== workspaceRequest) return
    models.value = nextModels
    const preferred = nextModels.find((model) => model.id === previousModelId) || nextModels[0]
    selectedModelId.value = preferred?.id || ''
  } catch (error) {
    if (requestId === workspaceRequest) appStore.showError(errorMessage(error))
  } finally {
    if (requestId === workspaceRequest) loadingModels.value = false
  }
}

async function ensureUploaded(item: MediaSelection): Promise<UploadedVideoMedia> {
  if (item.uploaded) return item.uploaded
  if (item.remoteUrl) {
    return { media_id: `external-${item.id}`, url: item.remoteUrl, type: item.kind, expires_at: '' }
  }
  const key = selectedKey.value
  if (!key) throw new Error('请先选择视频 Key')
  if (!item.file) throw new Error('素材文件不存在，请重新选择')
  item.status = 'uploading'; item.error = ''
  try {
    item.uploaded = await videoAPI.upload(key.key, item.file)
    item.status = 'uploaded'
    return item.uploaded
  } catch (error) {
    item.status = 'error'; item.error = errorMessage(error)
    throw error
  }
}
async function buildRequest() {
  const capability = selectedCapability.value
  if (!capability) throw new Error('请选择可用模型')
  if (!prompt.value.trim()) throw new Error('请填写画面描述')
  if (capability.requiresStartFrame && !startFrame.value) {
    throw new Error(`${capability.label} 必须提供一张首帧`)
  }
  if (creationMode.value === 'references' && referenceAudios.value.length > 0 && referenceImages.value.length + referenceVideos.value.length === 0) {
    throw new Error('参考音频必须同时搭配至少一张参考图片或一个参考视频')
  }
  if (creationMode.value === 'references' && referenceAudios.value.length > 0 && referenceVideos.value.length > 0) {
    throw new Error('参考视频和参考音频不能同时使用')
  }
  if (creationMode.value === 'frames' && endFrame.value && !startFrame.value) {
    throw new Error('上传尾帧前必须先上传首帧')
  }
  uploading.value = true
  try {
    const request: import('@/api/video').CreateVideoRequest = {
      model: capability.id,
      prompt: prompt.value.trim(),
      resolution: resolution.value,
      duration: duration.value,
      aspect_ratio: aspectRatio.value,
      audio: capability.supportsAudio ? audio.value : false,
    }
    if (capability.supportsPromptEnhance) request.prompt_enhance = promptEnhance.value
    if (creationMode.value === 'frames') {
      if (startFrame.value) request.start_frame_url = (await ensureUploaded(startFrame.value)).url
      if (endFrame.value && capability.supportsEndFrame) request.end_frame_url = (await ensureUploaded(endFrame.value)).url
    }
    if (creationMode.value === 'references') {
      const images = await Promise.all(referenceImages.value.map(ensureUploaded))
      const videos = await Promise.all(referenceVideos.value.map(ensureUploaded))
      const audios = await Promise.all(referenceAudios.value.map(ensureUploaded))
      request.guidances = {}
      if (images.length) request.guidances.image_reference = images.map((item, index) => ({
        image: { url: item.url, type: 'UPLOADED' }, strength: 'MID', order: index + 1,
      }))
      if (videos.length) request.guidances.video_reference_base = videos.map((item) => ({ video: { url: item.url, type: 'UPLOADED' } }))
      if (audios.length) request.guidances.audio_reference = audios.map((item) => ({ audio: { url: item.url, type: 'UPLOADED' } }))
      if (Object.keys(request.guidances).length === 0) delete request.guidances
    }
    return request
  } finally {
    uploading.value = false
  }
}
async function submitVideo() {
  const key = selectedKey.value
  if (!key) return
  formError.value = ''
  try {
    const request = await buildRequest()
    const body = JSON.stringify(request)
    if (!pendingIdempotencyKey.value || pendingRequestBody.value !== body) {
      pendingIdempotencyKey.value = createIdempotencyKey()
      pendingRequestBody.value = body
    }
    submitting.value = true
    const created = await videoAPI.create(key.key, request, pendingIdempotencyKey.value)
    pendingIdempotencyKey.value = ''; pendingRequestBody.value = ''
    appStore.showSuccess(`视频任务 ${shortJobId(created.job_id)} 已提交`)
  } catch (error) {
    formError.value = errorMessage(error)
  } finally {
    submitting.value = false
  }
}
watch(selectedKeyId, () => {
  try { if (selectedKeyId.value) window.localStorage.setItem(selectedIdStorageKey(), selectedKeyId.value) }
  catch { /* The studio still works without persisted selection. */ }
  releaseAllMedia(); pendingIdempotencyKey.value = ''; pendingRequestBody.value = ''
  loadWorkspace()
})
watch(selectedCapability, (capability) => {
  if (!capability) {
    return
  }
  resolution.value = capability.defaultResolution
  const durationValue = capability.defaultDuration
  durationIndex.value = Math.max(0, durationsFor(capability, resolution.value).indexOf(durationValue))
  aspectRatio.value = capability.defaultAspectRatio
  audio.value = true; promptEnhance.value = 'AUTO'; formError.value = ''
  if (capability.requiresStartFrame) creationMode.value = 'frames'
  else if (!availableModes.value.some((item) => item.value === creationMode.value)) creationMode.value = 'text'
  releaseAllMedia()
})
watch(resolution, () => {
  const durationValues = durationOptions.value
  if (durationIndex.value >= durationValues.length) durationIndex.value = Math.max(0, durationValues.length - 1)
  if (!aspectRatioOptions.value.includes(aspectRatio.value)) aspectRatio.value = aspectRatioOptions.value[0]
})
watch([prompt, resolution, aspectRatio, durationIndex, audio, promptEnhance, creationMode], () => {
  formError.value = ''
})
watch(startFrame, (value) => {
  if (selectedModelId.value === 'happy-horse-1.1' && value && promptEnhance.value === 'ON') promptEnhance.value = 'AUTO'
})

onMounted(loadKeys)
onBeforeUnmount(releaseAllMedia)
</script>

<style scoped media="not all">
.video-studio { display: grid; gap: 1rem; color: #211f1c; }
.video-studio__header { display: flex; align-items: end; justify-content: space-between; gap: 1.5rem; border: 2px solid #211f1c; border-radius: 8px; background: #fffdf5; box-shadow: 5px 5px 0 #211f1c; padding: 1.1rem 1.25rem; }
.video-studio__eyebrow { display: inline-flex; align-items: center; gap: .4rem; color: #08799a; font-size: .72rem; font-weight: 900; }
.video-studio__header h1 { margin: .3rem 0 .18rem; font-family: Georgia, 'Songti SC', serif; font-size: clamp(1.35rem, 2vw, 2rem); line-height: 1.1; }
.video-studio__header p { margin: 0; color: #6e675f; font-size: .84rem; }
.video-studio__keybar { display: flex; align-items: end; gap: .45rem; }
.video-studio__keybar label { display: grid; gap: .25rem; }
.video-studio__keybar label > span, .video-setting > span, .video-prompt > span strong { font-size: .7rem; font-weight: 900; }
.video-studio select { min-height: 2.5rem; border: 1px solid #b8aea3; border-radius: 6px; background: #fff; padding: 0 .75rem; color: inherit; font-weight: 700; }
.video-studio__keybar select { width: min(22rem, 34vw); }
.video-icon-button, .video-doc-link { display: inline-flex; min-width: 2.5rem; min-height: 2.5rem; align-items: center; justify-content: center; gap: .4rem; border: 1px solid #211f1c; border-radius: 6px; background: #fff; color: inherit; font-size: .76rem; font-weight: 900; text-decoration: none; }
.video-doc-link { padding: 0 .75rem; background: #ffd447; }
.video-studio__workspace { display: grid; grid-template-columns: 13.5rem minmax(0, 1fr) 18rem; align-items: start; gap: 1rem; }
.video-studio__models, .video-composer, .video-jobs { min-width: 0; border: 1px solid rgba(33,31,28,.22); border-radius: 8px; background: rgba(255,253,245,.95); box-shadow: 0 8px 24px rgba(72,58,45,.08); }
.video-studio__models, .video-jobs { display: grid; gap: .5rem; padding: .8rem; }
.video-panel-title { display: flex; align-items: center; justify-content: space-between; gap: .5rem; padding-bottom: .55rem; border-bottom: 1px solid rgba(33,31,28,.12); }
.video-panel-title > div { display: grid; }
.video-panel-title span { color: #756d65; font-size: .67rem; font-weight: 800; }
.video-panel-title strong { font-size: .86rem; }
.video-panel-title .video-icon-button { min-width: 2rem; min-height: 2rem; border-color: #c9c0b7; }
.video-model-option { display: grid; grid-template-columns: 2rem minmax(0,1fr) auto; align-items: center; gap: .55rem; width: 100%; min-height: 3.4rem; border: 1px solid transparent; border-radius: 6px; background: transparent; padding: .5rem; color: inherit; text-align: left; }
.video-model-option:hover { border-color: #c3b7aa; background: #fff; }
.video-model-option--active { border-color: #211f1c; background: #fff; box-shadow: 2px 2px 0 #211f1c; }
.video-model-option__icon { display: grid; width: 2rem; height: 2rem; place-items: center; border-radius: 5px; background: #dff7ff; color: #08799a; }
.video-model-option strong, .video-model-option small { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.video-model-option strong { font-size: .75rem; }.video-model-option small { margin-top: .15rem; color: #857c73; font-size: .61rem; }
.video-studio__loading, .video-jobs__empty { display: grid; min-height: 8rem; place-items: center; color: #817970; font-size: .74rem; text-align: center; }
.video-composer { padding: 1rem; }
.video-composer__topline { display: flex; align-items: center; justify-content: space-between; gap: 1rem; }
.video-composer__topline > div { display: grid; }.video-composer__topline span { color: #7d746b; font-size: .67rem; font-weight: 800; }.video-composer__topline strong { font-size: .95rem; }
.video-retry-badge { border: 1px solid #15926f; border-radius: 999px; background: #e7faf3; padding: .25rem .55rem; color: #0b6b50 !important; }
.video-mode-tabs { display: grid; grid-template-columns: repeat(3, minmax(0,1fr)); gap: .35rem; margin: .8rem 0; border-radius: 7px; background: #eee9e2; padding: .25rem; }
.video-mode-tabs button { display: flex; min-height: 2.35rem; align-items: center; justify-content: center; gap: .35rem; border: 0; border-radius: 5px; background: transparent; color: #736b63; font-size: .72rem; font-weight: 900; }
.video-mode-tabs .video-mode-tab--active { background: #fff; box-shadow: 0 2px 8px rgba(33,31,28,.11); color: #211f1c; }
.video-media-band { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: .55rem; margin-bottom: .8rem; }
.video-media-input { display: flex; min-height: 4.2rem; align-items: center; justify-content: space-between; gap: .7rem; border: 1px dashed #a69b90; border-radius: 7px; background: #fbfaf7; padding: .65rem; }
.video-media-input__copy { display: flex; min-width: 0; flex-wrap: wrap; align-items: center; gap: .3rem; }.video-media-input__copy strong { font-size: .73rem; }.video-media-input__copy em { border-radius: 3px; background: #ffe4eb; padding: .1rem .25rem; color: #a92852; font-size: .58rem; font-style: normal; }.video-media-input__copy span { width: 100%; color: #7e756c; font-size: .62rem; }
.video-media-input__add, .video-add-media { display: inline-flex; flex: 0 0 auto; cursor: pointer; align-items: center; gap: .28rem; border: 1px solid #b8aea3; border-radius: 5px; background: #fff; padding: .48rem .58rem; font-size: .66rem; font-weight: 900; }.video-media-input input, .video-add-media input { display: none; }
.video-remote-media-input, .video-remote-reference-input { display: flex; min-width: 0; align-items: center; gap: .3rem; }
.video-remote-media-input input, .video-remote-reference-input input { width: min(16rem, 100%); min-height: 2rem; border: 1px solid #b8aea3; border-radius: 5px; background: #fff; padding: 0 .5rem; color: inherit; font-size: .63rem; }
.video-remote-media-input button, .video-remote-reference-input button { display: inline-flex; min-height: 2rem; flex: 0 0 auto; align-items: center; gap: .2rem; border: 1px solid #08799a; border-radius: 5px; background: #e1f8ff; padding: 0 .5rem; color: #076582; font-size: .62rem; font-weight: 900; }
.video-media-input__selected { display: flex; min-width: 0; align-items: center; gap: .35rem; }.video-media-input__selected span { max-width: 8rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: .65rem; }.video-media-input__selected button { display: grid; width: 1.8rem; height: 1.8rem; place-items: center; border: 0; background: transparent; }
.video-reference-section { display: grid; gap: .48rem; margin-bottom: .8rem; border-block: 1px solid #ddd5cc; padding: .7rem 0; }
.video-reference-row { display: flex; align-items: center; justify-content: space-between; gap: .7rem; }.video-reference-row > div { display: grid; }.video-reference-row strong { font-size: .72rem; }.video-reference-row span { color: #80776e; font-size: .61rem; }
.video-media-list { display: flex; flex-wrap: wrap; gap: .35rem; }
.video-media-chip { display: grid; max-width: 10rem; grid-template-columns: 1.7rem minmax(0,1fr) 1.3rem; align-items: center; gap: .3rem; border: 1px solid #d1c7bc; border-radius: 5px; background: #fff; padding: .22rem; }.video-media-chip img, .video-media-chip__type { width: 1.7rem; height: 1.7rem; border-radius: 3px; object-fit: cover; }.video-media-chip__type { display: grid; place-items: center; background: #e7f7fb; color: #08799a; font-size: .48rem; font-weight: 900; }.video-media-chip > span:nth-child(2) { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: .58rem; }.video-media-chip button { display: grid; place-items: center; border: 0; background: transparent; }.video-media-chip--uploading { opacity: .55; }.video-media-chip--error { border-color: #d93963; }
.video-prompt { display: grid; gap: .35rem; }.video-prompt > span { display: flex; justify-content: space-between; }.video-prompt small { color: #817870; font-size: .63rem; }.video-prompt textarea { width: 100%; resize: vertical; border: 1px solid #b8aea3; border-radius: 7px; background: #fff; padding: .75rem; color: inherit; font: inherit; font-size: .82rem; line-height: 1.65; }
.video-settings { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: .75rem; margin-top: .85rem; border-top: 1px solid #ddd5cc; padding-top: .8rem; }.video-setting { display: grid; align-content: start; gap: .35rem; }.video-setting--wide { grid-column: 1 / -1; }
.video-segments { display: flex; flex-wrap: wrap; gap: .3rem; }.video-segments button { min-height: 2rem; border: 1px solid #c9bfb5; border-radius: 5px; background: #fff; padding: 0 .65rem; color: inherit; font-size: .66rem; font-weight: 900; }.video-segments .video-segment--active { border-color: #08799a; background: #e1f8ff; color: #076582; box-shadow: inset 0 0 0 1px #08799a; }
.video-duration { display: grid; grid-template-columns: minmax(0,1fr) 3.5rem; align-items: center; gap: .65rem; min-height: 2.5rem; }.video-duration input { accent-color: #08799a; }.video-duration strong { border: 1px solid #ccc2b8; border-radius: 5px; background: #fff; padding: .45rem; font-size: .7rem; text-align: center; }
.video-audio-toggle { position: relative; display: flex; align-items: center; justify-content: space-between; gap: .6rem; cursor: pointer; }.video-audio-toggle > span { display: grid; }.video-audio-toggle strong { font-size: .72rem; }.video-audio-toggle small { color: #7e756c; font-size: .61rem; }.video-audio-toggle input { position: absolute; opacity: 0; }.video-audio-toggle i { position: relative; width: 2.6rem; height: 1.4rem; border: 1px solid #a99f94; border-radius: 999px; background: #d9d3cc; }.video-audio-toggle i::after { position: absolute; top: .16rem; left: .17rem; width: .95rem; height: .95rem; border-radius: 50%; background: #fff; box-shadow: 0 1px 3px rgba(0,0,0,.2); content: ''; transition: transform .18s; }.video-audio-toggle input:checked + i { border-color: #08799a; background: #08a9d6; }.video-audio-toggle input:checked + i::after { transform: translateX(1.16rem); }
.video-form-error { display: flex; align-items: start; gap: .4rem; margin-top: .75rem; border: 1px solid #edb3c3; border-radius: 6px; background: #fff0f4; padding: .55rem .65rem; color: #a7244b; font-size: .68rem; line-height: 1.45; }
.video-composer__actions { display: flex; align-items: center; justify-content: space-between; gap: 1rem; margin-top: .85rem; border-top: 1px solid #ddd5cc; padding-top: .8rem; }.video-composer__actions > span { color: #7b7269; font-size: .65rem; }.video-composer__actions button { display: inline-flex; min-height: 2.6rem; align-items: center; gap: .45rem; border: 1px solid #211f1c; border-radius: 6px; background: #ff5f8f; box-shadow: 3px 3px 0 #211f1c; padding: 0 1rem; color: #211f1c; font-size: .75rem; font-weight: 900; }.video-composer__actions button:disabled { cursor: not-allowed; opacity: .45; }
.video-jobs { max-height: calc(100vh - 10rem); overflow-y: auto; }.video-preview { position: relative; overflow: hidden; border-radius: 6px; background: #111; }.video-preview video { display: block; width: 100%; max-height: 20rem; }.video-preview button { position: absolute; top: .35rem; right: .35rem; display: grid; width: 1.8rem; height: 1.8rem; place-items: center; border: 0; border-radius: 50%; background: rgba(0,0,0,.65); color: #fff; }
.video-jobs__empty { gap: .5rem; }.video-job { position: relative; border: 1px solid #d3c9be; border-radius: 6px; background: #fff; padding: .65rem; }.video-job__head, .video-job__meta { display: flex; align-items: center; justify-content: space-between; gap: .5rem; }.video-job__head time { color: #8c8379; font-size: .56rem; }.video-job > strong { display: block; margin-top: .48rem; font-size: .72rem; }.video-job > p { margin: .18rem 0 .52rem; color: #756c63; font-size: .62rem; }.video-job__meta { border-top: 1px solid #eee8e2; padding-top: .45rem; }.video-job__meta span { color: #08799a; font-size: .68rem; font-weight: 900; }.video-job__meta small { max-width: 7rem; overflow: hidden; color: #948a80; font-family: monospace; font-size: .54rem; }.video-status { border-radius: 999px; padding: .18rem .38rem; font-size: .56rem; font-weight: 900; }.video-status--pending { background: #fff4cf; color: #846200; }.video-status--running,.video-status--settling { background: #dff6ff; color: #086884; }.video-status--completed { background: #dcf7ed; color: #087255; }.video-status--failed { background: #ffe4ec; color: #a72249; }.video-status--canceled { background: #ebe8e5; color: #6d655e; }
.video-job__actions { position: absolute; right: .45rem; bottom: 2.25rem; display: flex; gap: .2rem; }.video-job__actions button { display: grid; width: 1.75rem; height: 1.75rem; place-items: center; border: 1px solid #c9c0b7; border-radius: 4px; background: #fff; color: #4f4841; }
.video-studio__empty { display: grid; min-height: 24rem; place-items: center; align-content: center; gap: .5rem; border: 1px dashed #aa9e92; border-radius: 8px; background: rgba(255,253,245,.85); text-align: center; }.video-studio__empty > span { display: grid; width: 4rem; height: 4rem; place-items: center; border-radius: 8px; background: #e1f8ff; color: #08799a; }.video-studio__empty p { margin: 0; color: #746b62; font-size: .75rem; }.video-studio__empty a { margin-top: .4rem; border: 1px solid #211f1c; border-radius: 5px; background: #ffd447; padding: .5rem .8rem; color: inherit; font-size: .7rem; font-weight: 900; text-decoration: none; }
button:not(:disabled), select:not(:disabled), input[type='range'] { cursor: pointer; }

@media (max-width: 1180px) { .video-studio__workspace { grid-template-columns: 12rem minmax(0,1fr); }.video-jobs { grid-column: 1 / -1; max-height: none; grid-template-columns: repeat(3,minmax(0,1fr)); }.video-jobs .video-panel-title,.video-jobs .video-preview,.video-jobs .video-jobs__empty,.video-jobs .video-studio__loading { grid-column: 1 / -1; } }
@media (max-width: 760px) { .video-studio__header { align-items: stretch; flex-direction: column; }.video-studio__keybar { flex-wrap: wrap; }.video-studio__keybar label { width: 100%; }.video-studio__keybar select { width: 100%; }.video-studio__workspace { grid-template-columns: 1fr; }.video-studio__models { grid-template-columns: repeat(2,minmax(0,1fr)); }.video-studio__models .video-panel-title,.video-studio__models .video-studio__loading { grid-column: 1 / -1; }.video-media-band,.video-settings { grid-template-columns: 1fr; }.video-setting--wide { grid-column: auto; }.video-mode-tabs { grid-template-columns: 1fr; }.video-jobs { grid-column: auto; grid-template-columns: 1fr; }.video-composer__actions { align-items: stretch; flex-direction: column; }.video-composer__actions button { justify-content: center; }.video-studio__header h1 { font-size: 1.35rem; } }

/* Compact creator workspace */
.video-studio { gap: .75rem; }
.video-studio__header {
  align-items: center;
  border-width: 1px;
  box-shadow: 3px 3px 0 #211f1c;
  padding: .7rem .9rem;
}
.video-studio__heading { display: flex; align-items: center; gap: .7rem; min-width: 0; }
.video-studio__header h1 { margin: 0; font-size: 1.25rem; letter-spacing: 0; }
.video-studio__eyebrow { border-right: 1px solid #d3c9be; padding-right: .7rem; white-space: nowrap; }
.video-doc-link { min-height: 2.25rem; padding: 0 .65rem; }
.video-studio__shell { display: grid; gap: .75rem; min-width: 0; }
.video-studio__setup {
  position: relative;
  z-index: 4;
  display: grid;
  grid-template-columns: minmax(13rem, .8fr) minmax(22rem, 1.35fr) 2.3rem;
  align-items: start;
  gap: .75rem;
  min-width: 0;
  border: 1px solid rgba(33,31,28,.22);
  border-radius: 8px;
  background: rgba(255,253,245,.96);
  padding: .7rem;
  box-shadow: 0 6px 18px rgba(72,58,45,.06);
}
.video-control-field { display: grid; min-width: 0; gap: .3rem; }
.video-control-label { display: flex; align-items: center; justify-content: space-between; gap: .5rem; min-height: 1rem; }
.video-control-label label { font-size: .7rem; font-weight: 900; }
.video-control-label span { color: #817970; font-size: .61rem; font-weight: 700; }
.video-key-select { display: grid; grid-template-columns: minmax(0,1fr) 2.3rem; gap: .35rem; }
.video-key-select select { width: 100%; min-width: 0; min-height: 2.3rem; }
.video-key-select .video-icon-button,
.video-refresh-workspace { min-width: 2.3rem; min-height: 2.3rem; }
.video-refresh-workspace { align-self: end; }
.video-model-picker { position: relative; }
.video-model-combobox {
  display: grid;
  grid-template-columns: 1rem minmax(0,1fr) 2rem;
  align-items: center;
  min-height: 2.3rem;
  border: 1px solid #b8aea3;
  border-radius: 6px;
  background: #fff;
  padding-left: .65rem;
  color: #817970;
}
.video-model-combobox:focus-within,
.video-model-combobox--open { border-color: #08799a; box-shadow: 0 0 0 2px rgba(8,121,154,.14); }
.video-model-combobox input {
  width: 100%;
  min-width: 0;
  height: 2.2rem;
  border: 0;
  outline: 0;
  background: transparent;
  padding: 0 .55rem;
  color: #211f1c;
  font-size: .75rem;
  font-weight: 800;
}
.video-model-combobox input::-webkit-search-cancel-button { display: none; }
.video-model-combobox button {
  display: grid;
  width: 2rem;
  height: 2.2rem;
  place-items: center;
  border: 0;
  background: transparent;
  color: #59524c;
}
.video-model-menu {
  position: absolute;
  top: calc(100% - 1.2rem);
  right: 0;
  left: 0;
  z-index: 12;
  display: grid;
  max-height: min(22rem, 55vh);
  overflow-y: auto;
  border: 1px solid #8f857b;
  border-radius: 7px;
  background: #fff;
  padding: .3rem;
  box-shadow: 0 14px 32px rgba(33,31,28,.2);
}
.video-model-menu__empty { display: grid; min-height: 5rem; place-items: center; color: #817970; font-size: .7rem; }
.video-model-option {
  grid-template-columns: 2rem minmax(0,1fr) auto 1rem;
  min-height: 3.25rem;
  padding: .45rem .5rem;
}
.video-model-option--highlighted { border-color: #c3b7aa; background: #f8f6f2; }
.video-model-option--active { border-color: #08799a; box-shadow: inset 3px 0 0 #08799a; }
.video-model-option__copy { min-width: 0; }
.video-model-option__copy strong { font-size: .73rem; }
.video-model-option__copy small { font-family: ui-monospace, SFMono-Regular, Consolas, monospace; }
.video-model-option__meta { display: flex; max-width: 12rem; flex-wrap: wrap; align-items: center; justify-content: flex-end; gap: .2rem; }
.video-model-option__meta small { width: 100%; text-align: right; }
.video-model-option__meta em {
  border-radius: 3px;
  background: #e7f7fb;
  padding: .08rem .24rem;
  color: #08799a;
  font-size: .52rem;
  font-style: normal;
  font-weight: 900;
}
.video-model-summary { display: flex; min-width: 0; flex-wrap: wrap; align-items: center; gap: .25rem; min-height: 1rem; }
.video-model-summary span { max-width: 15rem; overflow: hidden; color: #817970; font-family: ui-monospace, SFMono-Regular, Consolas, monospace; font-size: .56rem; text-overflow: ellipsis; white-space: nowrap; }
.video-model-summary small { border-radius: 3px; background: #eee9e2; padding: .08rem .23rem; color: #655e57; font-size: .52rem; font-weight: 800; }
.video-studio__workspace { grid-template-columns: minmax(0,1fr) minmax(18rem,20rem); gap: .75rem; }
.video-composer,
.video-jobs { border-color: rgba(33,31,28,.22); background: rgba(255,253,245,.96); }
.video-composer { padding: .85rem; }
.video-jobs { position: sticky; top: .75rem; gap: .45rem; max-height: calc(100vh - 7rem); padding: .7rem; }
.video-composer__topline strong { font-size: .86rem; }
.video-mode-tabs { grid-template-columns: repeat(3,minmax(0,1fr)); margin: .65rem 0; }
.video-prompt textarea { min-height: 7.5rem; }
.video-settings { grid-template-columns: repeat(3,minmax(0,1fr)); gap: .65rem; margin-top: .7rem; padding-top: .7rem; }
.video-setting,
.video-audio-toggle { min-width: 0; border: 1px solid #e0d8cf; border-radius: 6px; background: rgba(255,255,255,.58); padding: .55rem; }
.video-setting--resolution { grid-column: span 2; }
.video-setting--ratio { grid-column: span 2; }
.video-setting > span,
.video-prompt > span strong { font-size: .68rem; font-weight: 900; }
.video-setting select { min-height: 2.15rem; }
.video-composer__actions { margin-top: .7rem; padding-top: .7rem; }
.video-composer__actions button { min-height: 2.45rem; }

@media (max-width: 1280px) {
  .video-studio__workspace { grid-template-columns: 1fr; }
  .video-jobs { position: static; grid-column: auto; grid-template-columns: repeat(3,minmax(0,1fr)); max-height: none; }
  .video-jobs .video-panel-title,
  .video-jobs .video-preview,
  .video-jobs .video-jobs__empty,
  .video-jobs .video-studio__loading { grid-column: 1 / -1; }
}
@media (max-width: 900px) {
  .video-studio__setup { grid-template-columns: minmax(0,1fr) minmax(0,1.25fr) 2.3rem; }
  .video-settings { grid-template-columns: repeat(2,minmax(0,1fr)); }
  .video-setting--resolution,
  .video-setting--ratio { grid-column: span 1; }
}
@media (max-width: 680px) {
  .video-studio__header { flex-direction: row; align-items: center; gap: .5rem; }
  .video-studio__heading { gap: .5rem; }
  .video-studio__eyebrow { display: none; }
  .video-studio__header h1 { font-size: 1.1rem; }
  .video-doc-link { min-width: 2.25rem; padding: 0 .55rem; }
  .video-studio__setup { grid-template-columns: minmax(0,1fr); }
  .video-key-control,
  .video-model-picker { grid-column: auto; }
  .video-refresh-workspace { display: none; }
  .video-model-menu { top: calc(100% - 1.1rem); }
  .video-model-option { grid-template-columns: 2rem minmax(0,1fr) 1rem; }
  .video-model-option__meta { display: none; }
  .video-mode-tabs { grid-template-columns: repeat(3,minmax(0,1fr)); }
  .video-mode-tabs button { padding: 0 .2rem; }
  .video-media-band,
  .video-settings { grid-template-columns: 1fr; }
  .video-jobs { grid-template-columns: 1fr; }
  .video-composer__actions { align-items: stretch; flex-direction: column; }
  .video-composer__actions button { justify-content: center; }
}

/* Product layout: keep the prompt primary and group supporting controls by task. */
.video-studio {
  --video-ink: #17232b;
  --video-muted: #71808a;
  --video-line: #dbe3e6;
  --video-surface: #ffffff;
  --video-soft: #f5f8f8;
  --video-accent: #0f7f90;
  max-width: 1260px;
  margin: 0 auto;
  gap: .75rem;
  color: var(--video-ink);
}
.video-studio__header {
  min-height: 3.65rem;
  align-items: center;
  border: 0;
  border-bottom: 1px solid var(--video-line);
  border-radius: 0;
  background: transparent;
  box-shadow: none;
  padding: .15rem 0 .7rem;
}
.video-studio__heading { align-items: center; gap: .8rem; }
.video-studio__heading > div { display: grid; gap: .16rem; }
.video-studio__eyebrow {
  display: inline-flex;
  width: 2.35rem;
  height: 2.35rem;
  justify-content: center;
  border: 1px solid #b9dfe5;
  border-radius: 7px;
  background: #eaf8fa;
  color: var(--video-accent);
  font-size: 0;
}
.video-studio__eyebrow svg { width: 1rem; }
.video-studio__header h1 { margin: 0; color: var(--video-ink); font-family: inherit; font-size: 1.35rem; font-weight: 850; }
.video-studio__header p { margin: 0; color: var(--video-muted); font-size: .72rem; }
.video-studio__header-actions { display: inline-flex; align-items: center; gap: .55rem; }
.video-studio__status { display: inline-flex; align-items: center; gap: .35rem; color: #4d6c62; font-size: .66rem; font-weight: 800; }
.video-studio__status i { width: .42rem; height: .42rem; border-radius: 50%; background: #20aa73; box-shadow: 0 0 0 3px #def5e9; }
.video-doc-link { min-height: 2.2rem; border-color: #b9dfe5; border-radius: 6px; background: #eaf8fa; color: #0c6876; padding: 0 .7rem; }
.video-doc-link:hover { border-color: var(--video-accent); background: #dff3f5; }
.video-studio__shell { gap: .8rem; }
.video-studio__setup {
  grid-template-columns: minmax(11rem, .7fr) minmax(15rem, 1fr) minmax(24rem, 1.65fr) 2.3rem;
  gap: .8rem;
  border: 1px solid var(--video-line);
  border-radius: 8px;
  background: var(--video-surface);
  box-shadow: 0 3px 14px rgba(29, 55, 61, .055);
  padding: .65rem .75rem;
}
.video-setup__intro { display: flex; min-width: 0; align-items: center; gap: .55rem; }
.video-setup__intro > div { display: grid; min-width: 0; gap: .12rem; }
.video-setup__intro strong { font-size: .72rem; font-weight: 850; }
.video-setup__intro small { color: var(--video-muted); font-size: .6rem; line-height: 1.35; }
.video-setup__step { display: grid; width: 1.65rem; height: 1.65rem; flex: 0 0 auto; place-items: center; border-radius: 5px; background: var(--video-ink); color: #fff; font-family: ui-monospace, monospace; font-size: .6rem; font-weight: 900; }
.video-control-label label { color: var(--video-ink); font-size: .66rem; }
.video-control-label span { color: var(--video-muted); font-size: .58rem; }
.video-studio select { min-height: 2.25rem; border-color: #c7d3d6; border-radius: 6px; background: #fbfdfd; font-size: .72rem; }
.video-key-select { grid-template-columns: minmax(0, 1fr) 2.15rem; }
.video-key-select select { min-height: 2.25rem; }
.video-icon-button { min-width: 2.15rem; min-height: 2.15rem; border-color: #c7d3d6; border-radius: 6px; background: #fbfdfd; color: var(--video-muted); }
.video-icon-button:hover { border-color: var(--video-accent); color: var(--video-accent); }
.video-model-combobox { min-height: 2.25rem; border-color: #c7d3d6; border-radius: 6px; background: #fbfdfd; }
.video-model-combobox:focus-within, .video-model-combobox--open { border-color: var(--video-accent); box-shadow: 0 0 0 3px rgba(15, 127, 144, .1); }
.video-model-combobox input { height: 2.15rem; font-size: .72rem; }
.video-model-menu { top: calc(100% + .45rem); border-color: #b5c7ca; border-radius: 7px; box-shadow: 0 16px 34px rgba(24, 48, 53, .16); }
.video-model-option { min-height: 3rem; border-radius: 5px; }
.video-model-option--active { border-color: #8fcbd2; background: #effbfc; box-shadow: inset 3px 0 0 var(--video-accent); }
.video-model-option__icon { width: 1.8rem; height: 1.8rem; border-radius: 5px; background: #eaf8fa; color: var(--video-accent); }
.video-model-option__meta em { background: #eaf8fa; color: var(--video-accent); }
.video-model-summary { min-height: .9rem; }
.video-model-summary span { color: var(--video-muted); }
.video-model-summary small { background: #eef4f5; color: #52666d; }
.video-refresh-workspace { align-self: end; }
.video-studio__workspace { grid-template-columns: minmax(0, 1fr) minmax(18rem, 21rem); gap: .8rem; }
.video-composer, .video-parameters, .video-jobs { border: 1px solid var(--video-line); border-radius: 8px; background: var(--video-surface); box-shadow: 0 5px 18px rgba(29, 55, 61, .055); }
.video-composer { padding: 1rem 1.05rem; }
.video-composer__topline { min-height: 2.15rem; border-left: 3px solid var(--video-accent); padding-left: .7rem; }
.video-composer__topline span { color: var(--video-muted); font-size: .62rem; }
.video-composer__topline strong { color: var(--video-ink); font-size: .88rem; }
.video-retry-badge { border-color: #a9dec8; background: #effaf5; color: #167451 !important; font-size: .62rem; }
.video-mode-tabs { gap: .2rem; margin: .8rem 0 1rem; border: 1px solid var(--video-line); border-radius: 7px; background: var(--video-soft); padding: .2rem; }
.video-mode-tabs button { min-height: 2.2rem; color: #6d7d83; font-size: .68rem; }
.video-mode-tabs .video-mode-tab--active { background: var(--video-surface); box-shadow: 0 2px 7px rgba(31, 55, 60, .1); color: var(--video-ink); }
.video-prompt { gap: .45rem; }
.video-prompt > span strong { color: var(--video-ink); font-size: .72rem; }
.video-prompt small { color: var(--video-muted); font-size: .6rem; }
.video-prompt textarea { min-height: 13rem; border-color: #c7d3d6; border-radius: 7px; background: #fbfdfd; padding: .9rem; font-size: .8rem; line-height: 1.65; outline: 0; }
.video-prompt textarea:focus { border-color: var(--video-accent); box-shadow: 0 0 0 3px rgba(15, 127, 144, .1); }
.video-sidebar { display: grid; min-width: 0; align-content: start; gap: .8rem; }
.video-parameters { padding: .85rem; }
.video-sidebar__title { display: flex; align-items: start; justify-content: space-between; gap: .6rem; padding-bottom: .65rem; border-bottom: 1px solid var(--video-line); }
.video-sidebar__title > div { display: flex; align-items: center; gap: .5rem; }
.video-sidebar__title span { display: grid; width: 1.5rem; height: 1.5rem; place-items: center; border-radius: 4px; background: var(--video-ink); color: #fff; font-family: ui-monospace, monospace; font-size: .55rem; font-weight: 900; }
.video-sidebar__title strong { font-size: .78rem; }
.video-sidebar__title small { color: var(--video-muted); font-size: .58rem; text-align: right; }
.video-settings { grid-template-columns: 1fr; gap: .55rem; margin-top: .7rem; border-top: 0; padding-top: 0; }
.video-setting, .video-audio-toggle { border: 1px solid #e0e8e9; border-radius: 6px; background: #fbfdfd; padding: .6rem; }
.video-setting { gap: .45rem; }
.video-setting > span { color: #50636a; font-size: .63rem; }
.video-setting--resolution, .video-setting--ratio { grid-column: auto; }
.video-segments { gap: .25rem; }
.video-segments button { min-height: 1.9rem; border-color: #cad8da; border-radius: 5px; padding: 0 .55rem; color: #50636a; font-size: .62rem; }
.video-segments .video-segment--active { border-color: var(--video-accent); background: #eaf8fa; color: #0b6d7b; box-shadow: inset 0 0 0 1px var(--video-accent); }
.video-duration { grid-template-columns: minmax(0, 1fr) 3.35rem; min-height: 2.1rem; gap: .5rem; }
.video-duration input { accent-color: var(--video-accent); }
.video-duration strong { border-color: #cad8da; background: #fff; padding: .38rem .25rem; color: #52666d; font-size: .62rem; }
.video-setting select { min-height: 2rem; border-color: #cad8da; font-size: .66rem; }
.video-audio-toggle { min-height: 3rem; }
.video-audio-toggle strong { color: var(--video-ink); font-size: .66rem; }
.video-audio-toggle small { color: var(--video-muted); font-size: .58rem; }
.video-audio-toggle i { width: 2.35rem; height: 1.25rem; }
.video-audio-toggle i::after { width: .85rem; height: .85rem; }
.video-audio-toggle input:checked + i { background: var(--video-accent); }
.video-audio-toggle input:checked + i::after { transform: translateX(1.05rem); }
.video-composer__actions { margin-top: 1rem; border-top: 1px solid var(--video-line); padding-top: .85rem; }
.video-composer__actions > span { color: var(--video-muted); font-size: .62rem; }
.video-composer__actions button { min-height: 2.55rem; border: 0; border-radius: 6px; background: var(--video-accent); box-shadow: 0 3px 8px rgba(15, 127, 144, .2); padding: 0 1.1rem; color: #fff; font-size: .7rem; }
.video-composer__actions button:hover:not(:disabled) { background: #0b6d7b; }
.video-jobs { position: static; max-height: none; gap: .55rem; padding: .85rem; overflow: visible; }
.video-panel-title { padding-bottom: .65rem; border-color: var(--video-line); }
.video-panel-title span { color: var(--video-muted); font-size: .61rem; }
.video-panel-title strong { color: var(--video-ink); font-size: .78rem; }
.video-panel-title .video-icon-button { min-width: 2rem; min-height: 2rem; }
.video-jobs__empty, .video-studio__loading { min-height: 6.5rem; color: var(--video-muted); font-size: .66rem; }
.video-job { border-color: #dbe3e6; border-radius: 6px; background: #fbfdfd; padding: .65rem; }
.video-job > strong { color: var(--video-ink); font-size: .68rem; }
.video-job > p { color: var(--video-muted); font-size: .59rem; }
.video-job__meta { border-color: #e6edef; }
.video-job__meta span { color: var(--video-accent); font-size: .64rem; }
.video-job__actions { bottom: 2.05rem; }
.video-job__actions button { border-color: #cbd8da; color: #52666d; }
.video-form-error { border-color: #f0bdca; background: #fff5f7; font-size: .65rem; }
.video-preview { border-radius: 6px; }

@media (max-width: 1080px) {
  .video-studio__setup { grid-template-columns: minmax(10rem, .6fr) minmax(14rem, 1fr) minmax(18rem, 1.3fr) 2.3rem; }
}
@media (max-width: 900px) {
  .video-studio__setup { grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) 2.3rem; }
  .video-setup__intro { display: none; }
  .video-studio__workspace { grid-template-columns: minmax(0, 1fr) 18rem; }
}
@media (max-width: 720px) {
  .video-studio__header { align-items: flex-start; }
  .video-studio__header-actions { align-items: flex-end; flex-direction: column; gap: .35rem; }
  .video-studio__status { display: none; }
  .video-studio__setup { grid-template-columns: minmax(0, 1fr); }
  .video-refresh-workspace { display: none; }
  .video-studio__workspace { grid-template-columns: 1fr; }
  .video-composer { padding: .8rem; }
  .video-prompt textarea { min-height: 10rem; }
}

/* Second pass: one continuous creator surface instead of stacked cards. */
.video-studio {
  --video-ink: #1d2a33;
  --video-muted: #73818a;
  --video-line: #e1e7ea;
  --video-surface: #fff;
  --video-soft: #f7f9fa;
  --video-accent: #2563b8;
  max-width: 1220px;
  gap: .85rem;
  color: var(--video-ink);
}
.video-creator {
  overflow: visible;
  border: 1px solid var(--video-line);
  border-radius: 10px;
  background: var(--video-surface);
  box-shadow: 0 8px 26px rgba(27, 51, 60, .055);
}
.video-creator__header {
  display: flex;
  min-height: 4.1rem;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: .8rem 1.1rem;
}
.video-creator__header > div { display: flex; min-width: 0; align-items: center; gap: .65rem; }
.video-creator__header > div > div { display: grid; gap: .15rem; }
.video-creator__header strong { color: var(--video-ink); font-size: .9rem; font-weight: 850; }
.video-creator__header small { color: var(--video-muted); font-size: .64rem; }
.video-creator__icon { display: grid; width: 2.1rem; height: 2.1rem; flex: 0 0 auto; place-items: center; border: 1px solid #c9daf2; border-radius: 7px; background: #eef5ff; color: var(--video-accent); }
.video-doc-link { min-height: 2.1rem; border: 1px solid #c8d9ef; border-radius: 6px; background: #f1f6fd; color: var(--video-accent); padding: 0 .65rem; font-size: .64rem; }
.video-doc-link:hover { border-color: var(--video-accent); background: #e7f0fc; }
.video-studio__setup {
  grid-template-columns: minmax(14rem, .8fr) minmax(22rem, 1.35fr) 2.2rem;
  align-items: end;
  gap: .85rem;
  border: 0;
  border-block: 1px solid var(--video-line);
  border-radius: 0;
  background: var(--video-soft);
  box-shadow: none;
  padding: .7rem 1.1rem;
}
.video-control-label { min-height: .9rem; }
.video-control-label label { color: #51616a; font-size: .64rem; letter-spacing: .01em; }
.video-control-label span { color: #8b979d; font-size: .57rem; }
.video-studio select { min-height: 2.25rem; border-color: #ccd8dd; border-radius: 6px; background: #fff; color: var(--video-ink); font-size: .7rem; }
.video-key-select { grid-template-columns: minmax(0, 1fr) 2.1rem; }
.video-key-select select { min-height: 2.25rem; }
.video-icon-button { min-width: 2.1rem; min-height: 2.1rem; border-color: #ccd8dd; border-radius: 6px; background: #fff; color: #6f7d85; }
.video-icon-button:hover { border-color: var(--video-accent); color: var(--video-accent); }
.video-model-combobox { min-height: 2.25rem; border-color: #ccd8dd; border-radius: 6px; background: #fff; }
.video-model-combobox:focus-within, .video-model-combobox--open { border-color: var(--video-accent); box-shadow: 0 0 0 3px rgba(37, 99, 184, .1); }
.video-model-combobox input { height: 2.15rem; color: var(--video-ink); font-size: .7rem; }
.video-model-menu { top: calc(100% + .4rem); border-color: #bdccd2; border-radius: 7px; box-shadow: 0 18px 36px rgba(27, 51, 60, .16); }
.video-model-option { min-height: 3rem; border-radius: 5px; }
.video-model-option--active { border-color: #adc8ea; background: #f2f7fe; box-shadow: inset 3px 0 0 var(--video-accent); }
.video-model-option__icon { width: 1.8rem; height: 1.8rem; border-radius: 5px; background: #eef5ff; color: var(--video-accent); }
.video-model-option__meta em { background: #edf4fd; color: var(--video-accent); }
.video-model-summary { min-height: .85rem; }
.video-model-summary span { color: #87949b; }
.video-model-summary small { background: #edf2f4; color: #60727b; }
.video-refresh-workspace { align-self: end; }
.video-studio__workspace {
  grid-template-columns: minmax(0, 1fr) minmax(17rem, 19.5rem);
  align-items: stretch;
  gap: 0;
}
.video-composer { min-height: 32rem; border: 0; border-right: 1px solid var(--video-line); border-radius: 0; background: #fff; box-shadow: none; padding: 1.1rem 1.25rem 1rem; }
.video-composer__topline { min-height: 2rem; align-items: center; border-left: 0; border-bottom: 1px solid var(--video-line); padding: 0 0 .8rem; }
.video-composer__topline span { color: #8a969c; font-size: .6rem; }
.video-composer__topline strong { color: var(--video-ink); font-size: .83rem; }
.video-mode-tabs { gap: 1.15rem; margin: .7rem 0 1.05rem; border: 0; border-bottom: 1px solid var(--video-line); border-radius: 0; background: transparent; padding: 0; }
.video-mode-tabs button { position: relative; min-height: 2.25rem; justify-content: flex-start; border-radius: 0; color: #849198; font-size: .67rem; }
.video-mode-tabs button::after { position: absolute; right: 0; bottom: -1px; left: 0; height: 2px; background: transparent; content: ''; }
.video-mode-tabs .video-mode-tab--active { background: transparent; box-shadow: none; color: var(--video-accent); }
.video-mode-tabs .video-mode-tab--active::after { background: var(--video-accent); }
.video-prompt { gap: .45rem; }
.video-prompt > span strong { color: var(--video-ink); font-size: .73rem; }
.video-prompt small { color: #8a969c; font-size: .59rem; }
.video-prompt textarea { min-height: 12.5rem; border: 1px solid #ccd8dd; border-radius: 7px; background: #fbfcfd; padding: .95rem; color: var(--video-ink); font-size: .8rem; line-height: 1.7; outline: 0; }
.video-prompt textarea:focus { border-color: var(--video-accent); box-shadow: 0 0 0 3px rgba(37, 99, 184, .1); }
.video-media-band { margin-bottom: .8rem; }
.video-media-input { border-color: #cbd9df; background: #fbfcfd; }
.video-reference-section { border-color: var(--video-line); }
.video-parameters { min-width: 0; border: 0; border-radius: 0; background: #fbfcfd; box-shadow: none; padding: 1.1rem 1rem; }
.video-sidebar__title { padding-bottom: .75rem; border-color: var(--video-line); }
.video-sidebar__title > div { display: grid; gap: .15rem; }
.video-sidebar__title strong { color: var(--video-ink); font-size: .78rem; }
.video-sidebar__title small { color: #8a969c; font-size: .58rem; }
.video-settings { grid-template-columns: 1fr; gap: 0; margin-top: .2rem; border-top: 0; padding-top: 0; }
.video-parameters .video-setting, .video-parameters .video-audio-toggle { min-height: 3.3rem; border: 0; border-bottom: 1px solid var(--video-line); border-radius: 0; background: transparent; padding: .75rem 0; }
.video-parameters .video-setting > span { color: #63727a; font-size: .63rem; }
.video-setting--resolution, .video-setting--ratio { grid-column: auto; }
.video-segments { gap: .25rem; }
.video-segments button { min-height: 1.85rem; border-color: #cad7dc; border-radius: 5px; background: #fff; padding: 0 .55rem; color: #53656e; font-size: .61rem; }
.video-segments .video-segment--active { border-color: var(--video-accent); background: #eef5ff; color: var(--video-accent); box-shadow: inset 0 0 0 1px var(--video-accent); }
.video-duration { min-height: 2.1rem; grid-template-columns: minmax(0, 1fr) 3.15rem; gap: .45rem; }
.video-duration input { accent-color: var(--video-accent); }
.video-duration strong { border-color: #cad7dc; background: #fff; padding: .35rem .2rem; color: #53656e; font-size: .6rem; }
.video-parameters .video-setting select { min-height: 2rem; border-color: #cad7dc; font-size: .64rem; }
.video-parameters .video-audio-toggle { min-height: 3rem; border-bottom: 0; }
.video-audio-toggle strong { color: var(--video-ink); font-size: .65rem; }
.video-audio-toggle small { color: #849198; font-size: .57rem; }
.video-audio-toggle i { width: 2.35rem; height: 1.25rem; }
.video-audio-toggle input:checked + i { background: var(--video-accent); border-color: var(--video-accent); }
.video-audio-toggle input:checked + i::after { transform: translateX(1.05rem); }
.video-form-error { margin-top: .75rem; }
.video-composer__actions { margin-top: 1rem; border-color: var(--video-line); padding-top: .85rem; }
.video-composer__actions > span { color: #8a969c; font-size: .6rem; }
.video-composer__actions button { min-height: 2.6rem; border: 0; border-radius: 6px; background: var(--video-accent); box-shadow: 0 4px 10px rgba(37, 99, 184, .2); padding: 0 1.15rem; color: #fff; font-size: .68rem; }
.video-composer__actions button:hover:not(:disabled) { background: #1e55a0; }
.video-jobs { display: grid; gap: .45rem; max-height: none; border: 0; border-top: 1px solid var(--video-line); border-radius: 0; background: #fff; box-shadow: none; padding: .9rem 1.1rem 1rem; overflow: visible; }
.video-panel-title { padding-bottom: .65rem; border-color: var(--video-line); }
.video-panel-title span { color: #8a969c; font-size: .59rem; }
.video-panel-title strong { color: var(--video-ink); font-size: .76rem; }
.video-jobs__empty, .video-studio__loading { min-height: 5rem; color: #8a969c; font-size: .64rem; }
.video-job { display: grid; grid-template-columns: 7rem minmax(8rem, 1.15fr) minmax(8rem, 1fr) 7rem auto; min-height: 3.7rem; align-items: center; gap: .75rem; border: 1px solid #e0e7ea; border-radius: 6px; background: #fbfcfd; padding: .55rem .7rem; }
.video-job:hover { border-color: #bed0df; background: #f8fbff; }
.video-job__head { display: grid; justify-items: start; gap: .28rem; }
.video-job__head time { color: #8a969c; font-size: .56rem; }
.video-job > strong { margin-top: 0; color: var(--video-ink); font-size: .67rem; }
.video-job > p { margin: 0; color: #78868d; font-size: .59rem; }
.video-job__meta { border-top: 0; padding-top: 0; }
.video-job__meta span { color: var(--video-accent); font-size: .63rem; }
.video-job__meta small { max-width: 6rem; font-size: .52rem; }
.video-job__actions { position: static; justify-content: flex-end; gap: .22rem; }
.video-job__actions button { border-color: #cbd8df; background: #fff; color: #58707b; }
.video-preview { max-width: 28rem; }

@media (max-width: 980px) {
  .video-studio__workspace { grid-template-columns: 1fr; }
  .video-composer { border-right: 0; }
  .video-parameters { border-top: 1px solid var(--video-line); padding: .9rem 1.1rem 1rem; }
  .video-parameters .video-settings { grid-template-columns: repeat(2, minmax(0, 1fr)); column-gap: 1rem; }
  .video-parameters .video-setting, .video-parameters .video-audio-toggle { border-bottom: 1px solid var(--video-line); }
  .video-parameters .video-audio-toggle { grid-column: 1 / -1; }
}
@media (max-width: 680px) {
  .video-creator__header { padding: .75rem .8rem; }
  .video-creator__header small { max-width: 12rem; }
  .video-studio__setup { grid-template-columns: 1fr; align-items: stretch; padding: .7rem .8rem; }
  .video-refresh-workspace { display: none; }
  .video-composer { min-height: 0; padding: .9rem .8rem 1rem; }
  .video-mode-tabs { gap: .4rem; }
  .video-mode-tabs button { justify-content: center; }
  .video-prompt textarea { min-height: 10.5rem; }
  .video-parameters { padding: .85rem .8rem 1rem; }
  .video-parameters .video-settings { grid-template-columns: 1fr; }
  .video-parameters .video-audio-toggle { grid-column: auto; }
  .video-jobs { padding: .85rem .8rem 1rem; }
  .video-job { grid-template-columns: minmax(0, 1fr) auto; gap: .4rem .6rem; padding: .65rem; }
  .video-job__head { grid-column: 1 / -1; display: flex; justify-content: space-between; }
  .video-job > p { grid-column: 1 / -1; }
  .video-job__meta { grid-column: 1; }
  .video-job__actions { grid-column: 2; grid-row: 3; }
}

/* Video-first visual direction: preview stage anchors the creator workspace. */
.video-studio { max-width: 1240px; }
.video-studio__workspace { grid-template-columns: minmax(0, 1fr) minmax(19rem, 21.5rem); }
.video-composer { min-height: 35rem; padding: 1.2rem 1.35rem 1.1rem; }
.video-parameters { padding: .75rem; background: #f5f7f9; }
.video-preview-stage {
  position: relative;
  overflow: hidden;
  width: 100%;
  aspect-ratio: 16 / 9;
  border: 1px solid #253540;
  border-radius: 8px;
  background: #14212a;
  box-shadow: 0 10px 24px rgba(11, 28, 38, .16);
}
.video-preview-stage__empty { display: grid; height: 100%; place-items: center; align-content: center; gap: .38rem; color: #dbe7ec; text-align: center; }
.video-preview-stage__empty > span { display: grid; width: 2.8rem; height: 2.8rem; place-items: center; border: 1px solid #52636e; border-radius: 50%; background: #1d2d37; color: #f3f8fa; }
.video-preview-stage__empty strong { margin-top: .12rem; color: #f4f8fa; font-size: .72rem; font-weight: 800; }
.video-preview-stage__empty small { color: #8fa1ab; font-size: .57rem; }
.video-preview-stage__media { position: relative; width: 100%; height: 100%; }
.video-preview-stage__media video { display: block; width: 100%; height: 100%; object-fit: contain; background: #0d171d; }
.video-preview-stage__media button { position: absolute; top: .45rem; right: .45rem; display: grid; width: 1.8rem; height: 1.8rem; place-items: center; border: 0; border-radius: 50%; background: rgba(7, 16, 21, .74); color: #fff; }
.video-parameters .video-sidebar__title { margin-top: .9rem; padding: 0 .15rem .7rem; }
.video-parameters .video-settings { padding: 0 .15rem; }
.video-parameters .video-setting, .video-parameters .video-audio-toggle { min-height: 3.5rem; }
.video-prompt textarea { min-height: 14.5rem; background: #fff; }
.video-composer__actions button:not(:disabled) { background: #1f5ea8; }
.video-composer__actions button:hover:not(:disabled) { background: #194f90; }

@media (max-width: 980px) {
  .video-studio__workspace { grid-template-columns: minmax(0, 1fr) 19rem; }
  .video-parameters .video-settings { grid-template-columns: 1fr; }
  .video-parameters .video-audio-toggle { grid-column: auto; }
}
@media (max-width: 820px) {
  .video-studio__workspace { grid-template-columns: 1fr; }
  .video-composer { min-height: 0; border-right: 0; }
  .video-parameters { border-top: 1px solid var(--video-line); }
  .video-preview-stage { width: min(100%, 30rem); margin: 0 auto; }
  .video-parameters .video-settings { grid-template-columns: repeat(2, minmax(0, 1fr)); column-gap: 1rem; }
  .video-parameters .video-audio-toggle { grid-column: 1 / -1; }
}
@media (max-width: 560px) {
  .video-composer { padding: .9rem .8rem 1rem; }
  .video-prompt textarea { min-height: 11rem; }
  .video-parameters .video-settings { grid-template-columns: 1fr; }
  .video-parameters .video-audio-toggle { grid-column: auto; }
}
</style>
<style scoped src="./VideoStudioDirector.css"></style>
