<template>
  <div v-if="homeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <div v-else v-html="homeContent"></div>
  </div>

  <div
    v-else
    class="lingqu-home min-h-screen overflow-x-hidden"
    @mousemove="handleHomePointer"
    @mouseleave="resetHomePointer"
  >
    <div class="lingqu-bg" aria-hidden="true">
      <div class="lingqu-bg__grid"></div>
      <div class="lingqu-bg__wash"></div>
      <div class="lingqu-cloud lingqu-cloud--left">
        <span></span>
        <span></span>
        <span></span>
      </div>
      <div class="lingqu-cloud lingqu-cloud--right">
        <span></span>
        <span></span>
        <span></span>
      </div>
      <div class="lingqu-spark lingqu-spark--one"></div>
      <div class="lingqu-spark lingqu-spark--two"></div>
      <div class="lingqu-comet lingqu-comet--one"></div>
      <div class="lingqu-comet lingqu-comet--two"></div>
    </div>

    <header class="lingqu-header">
      <nav class="lingqu-nav" aria-label="首页导航">
        <router-link to="/home" class="lingqu-brand" aria-label="灵渠AI 首页">
          <span class="lingqu-brand__mark">
            <img :src="siteLogo || '/brand/lingqu-ai-logo.svg'" alt="" />
          </span>
          <span class="lingqu-brand__text">
            <strong>{{ siteName }}</strong>
            <small>One Key · All Models</small>
          </span>
        </router-link>

        <div class="lingqu-nav-links" aria-label="页面分区">
          <a v-for="item in navItems" :key="item.href" :href="item.href">{{ item.label }}</a>
        </div>

        <div class="lingqu-nav-actions">
          <LocaleSwitcher />
          <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="lingqu-nav-cta">
            <span>{{ isAuthenticated ? '进入控制台' : '登录' }}</span>
            <Icon name="arrowRight" size="sm" />
          </router-link>
        </div>
      </nav>
    </header>

    <main class="lingqu-main">
      <section id="capabilities" class="lingqu-hero" aria-labelledby="home-title">
        <div class="lingqu-copy">
          <div class="lingqu-kicker">
            <Icon name="sparkles" size="sm" />
            <span>顶尖大模型统一入口</span>
          </div>

          <h1 id="home-title" class="lingqu-title">
            <span class="lingqu-title__brand">灵渠AI</span>
            <span class="lingqu-title__line lingqu-title__line--one">一个 Key 接入</span>
            <span class="lingqu-title__line lingqu-title__line--two">所有大模型</span>
          </h1>

          <p class="lingqu-lead">
            面向开发者和团队的 AI 网关。你的应用只接一个 OpenAI 兼容接口，
            灵渠AI 在后面完成模型路由、账号池调度、失败切换和用量统计。
          </p>

          <div class="lingqu-actions">
            <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="lingqu-action lingqu-action--primary">
              <span>{{ isAuthenticated ? '进入控制台' : '进入控制台' }}</span>
              <Icon name="arrowRight" size="md" />
            </router-link>
            <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="lingqu-action">
              <span>看看怎么接入</span>
            </a>
          </div>

          <div class="lingqu-tags" aria-label="核心优势">
            <span v-for="tag in heroTags" :key="tag">{{ tag }}</span>
          </div>
        </div>

        <div class="lingqu-stage" aria-label="灵渠AI 模型网关插画">
          <div class="lingqu-browser-card">
            <div class="lingqu-browser-card__bar">
              <span></span>
              <span></span>
              <span></span>
            </div>
            <div class="lingqu-browser-card__screen">
              <img src="/illustrations/anime-gateway-hero.svg" alt="一个 Key 连接 GPT、Claude、Gemini、DeepSeek 等模型" />
            </div>
          </div>

          <div class="lingqu-chat-card">
            <div class="lingqu-avatar">
              <span class="lingqu-avatar__cap"></span>
              <span class="lingqu-avatar__face">
                <i></i>
                <i></i>
              </span>
              <span class="lingqu-avatar__key">
                <Icon name="key" size="sm" />
              </span>
            </div>
            <div class="lingqu-chat-card__text">
              <strong>一个 Key，全部搞定</strong>
              <small>OpenAI Compatible</small>
            </div>
          </div>

          <div class="lingqu-model-pill lingqu-model-pill--gpt">GPT</div>
          <div class="lingqu-model-pill lingqu-model-pill--claude">Claude</div>
          <div class="lingqu-model-pill lingqu-model-pill--gemini">Gemini</div>
        </div>
      </section>

      <section id="integration" class="lingqu-flow" aria-label="接入流程">
        <article
          v-for="(item, index) in flowItems"
          :key="item.title"
          class="lingqu-flow-item"
          :style="{ '--item-delay': `${index * 90}ms` }"
        >
          <span>{{ item.step }}</span>
          <strong>{{ item.title }}</strong>
          <small>{{ item.desc }}</small>
        </article>
      </section>

      <section id="advantages" class="lingqu-section lingqu-benefits" aria-labelledby="benefits-title">
        <div class="lingqu-section-heading">
          <span>Why LingQu AI</span>
          <h2 id="benefits-title">用户只要会复制 Key，剩下的交给灵渠AI。</h2>
          <p>
            把模型选择、账号池、失败切换、用量统计这些复杂事情藏在后面，前台只保留最短路径。
          </p>
        </div>

        <div class="lingqu-benefit-grid">
          <article
            v-for="(item, index) in benefitCards"
            :key="item.title"
            class="lingqu-benefit-card"
            :class="`lingqu-benefit-card--${index + 1}`"
          >
            <span class="lingqu-benefit-card__icon">
              <Icon :name="item.icon" size="lg" />
            </span>
            <small>{{ item.tag }}</small>
            <strong>{{ item.title }}</strong>
            <p>{{ item.desc }}</p>
          </article>
        </div>
      </section>

      <section class="lingqu-section lingqu-theater" aria-labelledby="theater-title">
        <div class="lingqu-theater__copy">
          <div class="lingqu-section-heading lingqu-section-heading--compact">
            <span>Request Theater</span>
            <h2 id="theater-title">一次请求进来，后台自动把路跑完。</h2>
            <p>
              接口保持 OpenAI 兼容，模型可以在灵渠AI 后台统一编排。用户看到的是一个入口，系统做的是自动调度。
            </p>
          </div>

          <div class="lingqu-route-chips" aria-label="路由能力">
            <span v-for="chip in routeChips" :key="chip">{{ chip }}</span>
          </div>
        </div>

        <div class="lingqu-terminal" aria-label="请求调度示例">
          <div class="lingqu-terminal__bar">
            <span></span>
            <span></span>
            <span></span>
            <strong>lingqu.route</strong>
          </div>
          <div class="lingqu-terminal__body">
            <p
              v-for="(line, index) in terminalLines"
              :key="line"
              :style="{ '--line-delay': `${index * 120}ms` }"
            >
              <span>{{ String(index + 1).padStart(2, '0') }}</span>
              <code>{{ line }}</code>
            </p>
          </div>
        </div>
      </section>

      <section class="lingqu-section lingqu-models" aria-labelledby="models-title">
        <div class="lingqu-section-heading">
          <span>Model Map</span>
          <h2 id="models-title">把顶尖模型放进同一个入口。</h2>
          <p>
            不是让用户研究每个平台怎么接，而是让用户拿到 Key 后直接开始用。
          </p>
        </div>

        <div class="lingqu-model-grid">
          <article
            v-for="model in modelCards"
            :key="model.name"
            class="lingqu-model-card"
            :style="{ '--model-color': model.color }"
          >
            <span>{{ model.short }}</span>
            <strong>{{ model.name }}</strong>
            <small>{{ model.desc }}</small>
          </article>
        </div>
      </section>

      <section class="lingqu-final-cta" aria-label="开始使用灵渠AI">
        <div class="lingqu-final-cta__mascot" aria-hidden="true">
          <span></span>
          <i></i>
        </div>
        <div>
          <small>Ready</small>
          <h2>现在开始，用一个 Key 接入所有模型。</h2>
          <p>登录后创建 Key，复制 Base URL，现有 OpenAI SDK 基本不用重写。</p>
        </div>
        <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="lingqu-action lingqu-action--primary">
          <span>创建我的 Key</span>
          <Icon name="arrowRight" size="md" />
        </router-link>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted } from 'vue'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { DEFAULT_SITE_LOGO, resolveBrandLogo, resolveBrandName } from '@/constants/brand'

const authStore = useAuthStore()
const appStore = useAppStore()

const siteName = computed(() => resolveBrandName(appStore.cachedPublicSettings?.site_name || appStore.siteName))
const siteLogo = computed(() => resolveBrandLogo(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo) || DEFAULT_SITE_LOGO)
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => (isAdmin.value ? '/admin/dashboard' : '/dashboard'))

const navItems = [
  { label: '能力', href: '#capabilities' },
  { label: '优势', href: '#advantages' },
  { label: '接入', href: '#integration' }
] as const

const heroTags = ['稳定调度', '快速响应', '统一计量'] as const

const flowItems = [
  { step: '01', title: '创建 Key', desc: '登录后生成一个可控的调用 Key。' },
  { step: '02', title: '复制地址', desc: '使用 OpenAI 兼容 Base URL。' },
  { step: '03', title: '直接调用', desc: '模型切换和失败重试交给灵渠AI。' }
] as const

const benefitCards = [
  {
    icon: 'shield',
    tag: 'Stable',
    title: '稳定调度',
    desc: '账号池、通道状态和失败切换统一管理，尽量减少用户侧的接入波动。'
  },
  {
    icon: 'bolt',
    tag: 'Fast',
    title: '快速响应',
    desc: '按模型和场景自动选择可用通道，让请求更快落到合适的模型上。'
  },
  {
    icon: 'chart',
    tag: 'Clear',
    title: '统一计量',
    desc: '不同模型的用量集中记录，用户不用在多个平台之间来回对账。'
  },
  {
    icon: 'globe',
    tag: 'Compatible',
    title: '兼容接入',
    desc: '保持 OpenAI 风格调用方式，已有应用迁移成本更低。'
  }
] as const

const routeChips = ['模型路由', '失败切换', '账号池调度', '统一计量'] as const

const terminalLines = [
  'receive request: /v1/chat/completions',
  'read key policy: model + quota + group',
  'scan healthy channels: GPT / Claude / Gemini',
  'route to best available model',
  'stream response and record usage'
] as const

const modelCards = [
  { short: 'G', name: 'GPT', desc: '创作、代码、推理', color: '#ff4f7b' },
  { short: 'C', name: 'Claude', desc: '长文本、复杂任务', color: '#1ab6e8' },
  { short: 'Gm', name: 'Gemini', desc: '多模态、上下文', color: '#7e63ff' },
  { short: 'D', name: 'DeepSeek', desc: '推理、性价比', color: '#2fcb92' },
  { short: 'X', name: 'Grok', desc: '实时感、快响应', color: '#ff9c40' },
  { short: '+', name: '更多模型', desc: '后续统一扩展', color: '#ffd447' }
] as const

const latestPointer = { x: 50, y: 50 }
let pointerFrame = 0
let pointerRoot: HTMLElement | null = null

function applyHomePointer() {
  pointerFrame = 0
  if (!pointerRoot) return
  const x = latestPointer.x
  const y = latestPointer.y
  pointerRoot.style.setProperty('--pointer-x', `${x}%`)
  pointerRoot.style.setProperty('--pointer-y', `${y}%`)
  pointerRoot.style.setProperty('--hero-tilt-x', `${(50 - y) / 56}deg`)
  pointerRoot.style.setProperty('--hero-tilt-y', `${(x - 50) / 58}deg`)
  pointerRoot.style.setProperty('--float-x', `${(x - 50) * 0.12}px`)
  pointerRoot.style.setProperty('--float-y', `${(y - 50) * 0.12}px`)
}

function scheduleHomePointer() {
  if (pointerFrame) return
  pointerFrame = window.requestAnimationFrame(applyHomePointer)
}

function handleHomePointer(event: MouseEvent) {
  const target = event.currentTarget as HTMLElement
  pointerRoot = target
  const rect = target.getBoundingClientRect()
  latestPointer.x = Math.min(100, Math.max(0, ((event.clientX - rect.left) / rect.width) * 100))
  latestPointer.y = Math.min(100, Math.max(0, ((event.clientY - rect.top) / rect.height) * 100))
  scheduleHomePointer()
}

function resetHomePointer() {
  latestPointer.x = 50
  latestPointer.y = 50
  scheduleHomePointer()
}

onMounted(() => {
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})

onBeforeUnmount(() => {
  if (pointerFrame) {
    window.cancelAnimationFrame(pointerFrame)
    pointerFrame = 0
  }
})
</script>

<style scoped>
.lingqu-home {
  --ink: #211f1c;
  --paper: #fffdf5;
  --paper-soft: #fff8e8;
  --pink: #ff4f7b;
  --pink-dark: #e83263;
  --cyan: #1ab6e8;
  --yellow: #ffd447;
  --orange: #ff9c40;
  --mint: #2fcb92;
  --violet: #7e63ff;
  position: relative;
  min-height: 100vh;
  color: var(--ink);
  background:
    radial-gradient(circle at var(--pointer-x, 70%) var(--pointer-y, 16%), rgba(255, 212, 71, 0.2), transparent 28rem),
    linear-gradient(100deg, #fff3ec 0%, #fffdf6 43%, #e8fbff 100%);
  font-feature-settings: 'kern';
}

.lingqu-home::after {
  content: '';
  position: fixed;
  inset: 0;
  z-index: 1;
  pointer-events: none;
  background:
    linear-gradient(90deg, rgba(255, 79, 123, 0.1), transparent 36%, rgba(26, 182, 232, 0.13)),
    radial-gradient(circle at 46% 0%, rgba(255, 255, 255, 0.72), transparent 32rem);
}

.lingqu-bg,
.lingqu-bg__grid,
.lingqu-bg__wash {
  position: fixed;
  inset: 0;
  pointer-events: none;
}

.lingqu-bg {
  z-index: 0;
  overflow: hidden;
}

.lingqu-bg__grid {
  background-image:
    radial-gradient(circle at 1px 1px, rgba(33, 31, 28, 0.11) 1.2px, transparent 1.8px),
    linear-gradient(rgba(33, 31, 28, 0.045) 1px, transparent 1px),
    linear-gradient(90deg, rgba(33, 31, 28, 0.045) 1px, transparent 1px);
  background-size: 24px 24px, 48px 48px, 48px 48px;
  opacity: 0.72;
  animation: lingquGridDrift 18s linear infinite;
}

.lingqu-bg__wash {
  background:
    radial-gradient(circle at 15% 34%, rgba(255, 79, 123, 0.16), transparent 22rem),
    radial-gradient(circle at 88% 45%, rgba(26, 182, 232, 0.18), transparent 28rem),
    radial-gradient(circle at 70% 86%, rgba(126, 99, 255, 0.12), transparent 24rem);
}

.lingqu-header {
  position: relative;
  z-index: 20;
  padding: clamp(0.9rem, 2vw, 1.55rem) 1rem 0;
}

.lingqu-nav {
  display: flex;
  width: min(77.5rem, calc(100vw - 2rem));
  min-height: 4.85rem;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin: 0 auto;
  border: 4px solid var(--ink);
  border-radius: 28px;
  background: rgba(255, 255, 255, 0.86);
  box-shadow: 8px 8px 0 rgba(33, 31, 28, 0.94);
  padding: 0.65rem 0.72rem 0.65rem 0.8rem;
  backdrop-filter: blur(16px);
  animation: lingquDropIn 560ms cubic-bezier(0.16, 0.86, 0.28, 1.16) both;
}

.lingqu-brand {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 0.72rem;
  color: inherit;
}

.lingqu-brand__mark {
  display: grid;
  width: 3.25rem;
  height: 3.25rem;
  flex: none;
  place-items: center;
  overflow: hidden;
  border: 3px solid var(--ink);
  border-radius: 19px;
  background: #fff7d7;
  box-shadow: 4px 4px 0 rgba(33, 31, 28, 0.95);
  transition:
    transform 180ms ease,
    box-shadow 180ms ease;
}

.lingqu-brand:hover .lingqu-brand__mark {
  transform: translate(-1px, -2px) rotate(-4deg) scale(1.04);
  box-shadow: 6px 6px 0 rgba(33, 31, 28, 0.95);
}

.lingqu-brand__mark img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.lingqu-brand__text {
  display: grid;
  min-width: 0;
  gap: 0.08rem;
}

.lingqu-brand__text strong {
  overflow: hidden;
  color: var(--ink);
  font-family: theme('fontFamily.display');
  font-size: clamp(1.05rem, 1.6vw, 1.28rem);
  font-weight: 950;
  letter-spacing: 0;
  line-height: 1;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.lingqu-brand__text small {
  overflow: hidden;
  color: rgba(33, 31, 28, 0.52);
  font-size: 0.72rem;
  font-weight: 950;
  letter-spacing: 0;
  line-height: 1;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.lingqu-nav-links {
  display: flex;
  align-items: center;
  gap: clamp(1.15rem, 3vw, 2.35rem);
}

.lingqu-nav-links a {
  position: relative;
  color: rgba(33, 31, 28, 0.7);
  font-size: 0.95rem;
  font-weight: 950;
  transition:
    color 160ms ease,
    transform 160ms ease;
}

.lingqu-nav-links a::after {
  content: '';
  position: absolute;
  left: 50%;
  bottom: -0.42rem;
  width: 0.38rem;
  height: 0.38rem;
  border: 2px solid var(--ink);
  border-radius: 50%;
  background: var(--yellow);
  opacity: 0;
  transform: translateX(-50%) scale(0);
  transition:
    opacity 160ms ease,
    transform 160ms ease;
}

.lingqu-nav-links a:hover {
  color: var(--ink);
  transform: translateY(-1px);
}

.lingqu-nav-links a:hover::after {
  opacity: 1;
  transform: translateX(-50%) scale(1);
}

.lingqu-nav-actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.lingqu-nav-actions :deep(button) {
  border-radius: 999px;
  color: var(--ink);
  font-weight: 850;
}

.lingqu-nav-cta,
.lingqu-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 3px solid var(--ink);
  color: inherit;
  font-weight: 950;
  transition:
    transform 170ms ease,
    box-shadow 170ms ease,
    filter 170ms ease;
}

.lingqu-nav-cta {
  min-height: 3rem;
  gap: 0.42rem;
  border-radius: 999px;
  background: var(--ink);
  box-shadow: 5px 5px 0 rgba(33, 31, 28, 0.5);
  color: #fff;
  padding: 0 1.25rem;
}

.lingqu-nav-cta:hover,
.lingqu-action:hover {
  transform: translate(-2px, -2px);
  box-shadow: 7px 7px 0 rgba(33, 31, 28, 0.94);
}

.lingqu-main {
  position: relative;
  z-index: 10;
  padding-bottom: 4.5rem;
}

.lingqu-hero {
  display: grid;
  width: min(77rem, calc(100vw - 2rem));
  min-height: calc(100vh - 6.4rem);
  grid-template-columns: minmax(0, 0.86fr) minmax(22rem, 1.14fr);
  align-items: center;
  gap: clamp(2.2rem, 6vw, 5.5rem);
  margin: 0 auto;
  padding: clamp(2.5rem, 7vh, 5.2rem) 1rem clamp(4rem, 8vh, 6.5rem);
}

.lingqu-copy {
  max-width: 38.5rem;
  animation: lingquCopyIn 720ms cubic-bezier(0.18, 0.8, 0.28, 1) 80ms both;
}

.lingqu-kicker {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  border: 3px solid var(--ink);
  border-radius: 999px;
  background: #fff6bc;
  box-shadow: 4px 4px 0 rgba(33, 31, 28, 0.9);
  color: var(--ink);
  padding: 0.52rem 1rem;
  font-size: 0.84rem;
  font-weight: 950;
  transform: rotate(-0.8deg);
  animation: lingquWiggle 5.2s ease-in-out infinite;
}

.lingqu-title {
  margin-top: 1.08rem;
  color: var(--ink);
  font-family: theme('fontFamily.display');
  font-weight: 950;
  letter-spacing: 0;
  line-height: 0.9;
}

.lingqu-title__brand,
.lingqu-title__line {
  display: block;
}

.lingqu-title__brand {
  font-size: clamp(4.7rem, 9.3vw, 8rem);
  text-shadow: 0.045em 0.045em 0 rgba(33, 31, 28, 0.1);
}

.lingqu-title__line {
  color: var(--pink);
  font-size: clamp(2.6rem, 5.25vw, 4.72rem);
  text-shadow:
    4px 4px 0 var(--ink),
    0 7px 0 rgba(255, 255, 255, 0.8);
}

.lingqu-title__line--one {
  margin-top: -0.07em;
  animation: lingquPopText 620ms cubic-bezier(0.2, 1.35, 0.28, 1) 420ms both;
}

.lingqu-title__line--two {
  animation: lingquPopText 620ms cubic-bezier(0.2, 1.35, 0.28, 1) 520ms both;
}

.lingqu-lead {
  max-width: 34.5rem;
  margin-top: 1.65rem;
  color: rgba(33, 31, 28, 0.68);
  font-size: clamp(1rem, 1.42vw, 1.13rem);
  font-weight: 800;
  line-height: 1.8;
}

.lingqu-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.85rem;
  margin-top: 2.15rem;
}

.lingqu-action {
  min-height: 3.35rem;
  gap: 0.5rem;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.78);
  box-shadow: 6px 6px 0 rgba(33, 31, 28, 0.94);
  padding: 0 1.28rem;
  backdrop-filter: blur(10px);
}

.lingqu-action--primary {
  background: linear-gradient(135deg, #ff526f 0%, #ffbe45 100%);
  color: #211f1c;
}

.lingqu-action:active,
.lingqu-nav-cta:active {
  transform: translate(1px, 1px);
  box-shadow: 3px 3px 0 rgba(33, 31, 28, 0.94);
}

.lingqu-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.65rem;
  margin-top: 1.35rem;
}

.lingqu-tags span {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 2.3rem;
  border: 2px solid rgba(33, 31, 28, 0.18);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.7);
  box-shadow: 2px 3px 0 rgba(33, 31, 28, 0.18);
  padding: 0 1rem;
  font-size: 0.86rem;
  font-weight: 950;
  transition:
    transform 160ms ease,
    border-color 160ms ease,
    background 160ms ease;
}

.lingqu-tags span:hover {
  border-color: var(--ink);
  background: #fff;
  transform: translateY(-2px) rotate(-1deg);
}

.lingqu-stage {
  position: relative;
  min-height: clamp(25rem, 41vw, 35.5rem);
  perspective: 1200px;
  animation: lingquStageIn 760ms cubic-bezier(0.18, 0.8, 0.28, 1) 180ms both;
}

.lingqu-browser-card {
  position: absolute;
  inset: 8% 0 0 3%;
  overflow: hidden;
  border: 4px solid var(--ink);
  border-radius: 30px;
  background: #fff8c6;
  box-shadow:
    15px 15px 0 rgba(33, 31, 28, 0.92),
    0 34px 70px rgba(26, 182, 232, 0.2);
  transform:
    perspective(1200px)
    rotateX(var(--hero-tilt-x, 0deg))
    rotateY(var(--hero-tilt-y, 0deg))
    rotate(1.2deg)
    translate3d(var(--float-x, 0), var(--float-y, 0), 0);
  transform-style: preserve-3d;
  transition:
    box-shadow 220ms ease,
    filter 220ms ease;
  will-change: transform;
}

.lingqu-browser-card:hover {
  box-shadow:
    19px 19px 0 rgba(33, 31, 28, 0.92),
    0 40px 84px rgba(255, 79, 123, 0.22);
  filter: saturate(1.04);
}

.lingqu-browser-card__bar {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  height: 3.05rem;
  border-bottom: 3px solid var(--ink);
  padding: 0 1.05rem;
}

.lingqu-browser-card__bar span {
  width: 0.72rem;
  height: 0.72rem;
  border: 2px solid var(--ink);
  border-radius: 50%;
}

.lingqu-browser-card__bar span:nth-child(1) {
  background: var(--pink);
}

.lingqu-browser-card__bar span:nth-child(2) {
  background: var(--yellow);
}

.lingqu-browser-card__bar span:nth-child(3) {
  background: var(--cyan);
}

.lingqu-browser-card__screen {
  position: relative;
  overflow: hidden;
  background: #fff;
}

.lingqu-browser-card__screen::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(110deg, transparent 0%, rgba(255, 255, 255, 0.6) 42%, transparent 56%);
  transform: translateX(-120%);
  animation: lingquScreenShine 4.8s ease-in-out infinite;
  pointer-events: none;
}

.lingqu-browser-card__screen img {
  display: block;
  width: 100%;
  height: 100%;
  aspect-ratio: 16 / 9.4;
  object-fit: cover;
  object-position: center;
}

.lingqu-chat-card {
  position: absolute;
  left: -2.8%;
  bottom: 2.5%;
  z-index: 3;
  display: flex;
  width: min(17.8rem, 46vw);
  align-items: center;
  gap: 0.9rem;
  border: 4px solid var(--ink);
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.92);
  box-shadow: 8px 8px 0 rgba(33, 31, 28, 0.9);
  padding: 0.85rem 1rem;
  transform: rotate(-4.2deg);
  animation: lingquFloat 4.9s ease-in-out infinite;
}

.lingqu-chat-card__text {
  display: grid;
  min-width: 0;
  gap: 0.2rem;
}

.lingqu-chat-card__text strong {
  font-family: theme('fontFamily.display');
  font-size: clamp(0.95rem, 1.4vw, 1.2rem);
  font-weight: 950;
  line-height: 1.1;
}

.lingqu-chat-card__text small {
  color: rgba(33, 31, 28, 0.55);
  font-size: 0.72rem;
  font-weight: 900;
  letter-spacing: 0.04em;
}

.lingqu-avatar {
  position: relative;
  width: 4.5rem;
  height: 4.95rem;
  flex: none;
}

.lingqu-avatar__cap {
  position: absolute;
  left: 0.62rem;
  top: 0;
  width: 3.24rem;
  height: 1.28rem;
  border: 3px solid var(--ink);
  border-radius: 999px 999px 9px 9px;
  background: var(--pink);
}

.lingqu-avatar__face {
  position: absolute;
  left: 0.54rem;
  top: 0.76rem;
  display: flex;
  width: 3.35rem;
  height: 3.12rem;
  align-items: center;
  justify-content: center;
  gap: 0.68rem;
  border: 3px solid var(--ink);
  border-radius: 44%;
  background: #fff4bc;
}

.lingqu-avatar__face::after {
  content: '';
  position: absolute;
  left: 1.16rem;
  bottom: 0.72rem;
  width: 0.98rem;
  height: 0.48rem;
  border-bottom: 3px solid var(--ink);
  border-radius: 999px;
}

.lingqu-avatar__face i {
  width: 0.44rem;
  height: 0.56rem;
  border-radius: 999px;
  background: var(--ink);
  animation: lingquBlink 4.2s infinite;
}

.lingqu-avatar__key {
  position: absolute;
  right: 0.28rem;
  bottom: 0;
  display: grid;
  width: 2.02rem;
  height: 1.76rem;
  place-items: center;
  border: 3px solid var(--ink);
  border-radius: 0.8rem;
  background: var(--cyan);
  color: #fff;
}

.lingqu-model-pill {
  position: absolute;
  z-index: 4;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 3px solid var(--ink);
  border-radius: 999px;
  box-shadow: 4px 4px 0 rgba(33, 31, 28, 0.82);
  color: #fff;
  font-size: 0.72rem;
  font-weight: 950;
  padding: 0.32rem 0.75rem;
  animation: lingquFloat 5.5s ease-in-out infinite;
}

.lingqu-model-pill--gpt {
  right: 19%;
  top: 23%;
  background: var(--pink);
}

.lingqu-model-pill--claude {
  right: 5%;
  bottom: 29%;
  background: var(--cyan);
  animation-delay: 0.8s;
}

.lingqu-model-pill--gemini {
  right: 23%;
  bottom: 10%;
  background: var(--violet);
  animation-delay: 1.3s;
}

.lingqu-cloud {
  position: absolute;
  width: 11rem;
  height: 4.2rem;
  opacity: 0.64;
  animation: lingquCloud 10s ease-in-out infinite;
}

.lingqu-cloud span {
  position: absolute;
  bottom: 0;
  border: 3px solid rgba(33, 31, 28, 0.08);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.56);
  box-shadow: 0 8px 20px rgba(255, 255, 255, 0.4);
}

.lingqu-cloud span:nth-child(1) {
  left: 0;
  width: 4.5rem;
  height: 3.1rem;
}

.lingqu-cloud span:nth-child(2) {
  left: 2.2rem;
  bottom: 0.75rem;
  width: 4.6rem;
  height: 4.6rem;
}

.lingqu-cloud span:nth-child(3) {
  left: 5.35rem;
  width: 5.4rem;
  height: 2.55rem;
}

.lingqu-cloud--left {
  left: 6%;
  top: 20%;
}

.lingqu-cloud--right {
  right: 6.5%;
  top: 34%;
  transform: scale(0.9);
  animation-delay: 1.6s;
}

.lingqu-spark {
  position: absolute;
  width: 2.4rem;
  height: 2.4rem;
  background: var(--pink);
  clip-path: polygon(50% 0, 61% 37%, 100% 50%, 61% 63%, 50% 100%, 39% 63%, 0 50%, 39% 37%);
  filter: drop-shadow(2px 2px 0 rgba(33, 31, 28, 0.64));
  animation: lingquSpark 4.8s ease-in-out infinite;
}

.lingqu-spark--one {
  right: 18%;
  bottom: 22%;
}

.lingqu-spark--two {
  left: 18%;
  bottom: 12%;
  width: 1.4rem;
  height: 1.4rem;
  background: var(--yellow);
  animation-delay: 1.2s;
}

.lingqu-comet {
  position: absolute;
  height: 5px;
  border-radius: 999px;
  background: rgba(33, 31, 28, 0.16);
  transform: rotate(-8deg);
  animation: lingquComet 7s ease-in-out infinite;
}

.lingqu-comet--one {
  left: 6%;
  top: 48%;
  width: 23rem;
}

.lingqu-comet--two {
  right: 9%;
  top: 17%;
  width: 17rem;
  animation-delay: 2.1s;
}

.lingqu-flow {
  display: grid;
  width: min(77rem, calc(100vw - 2rem));
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1rem;
  margin: -3.7rem auto 0;
  padding: 0 1rem 4rem;
}

.lingqu-flow-item {
  min-height: 8.5rem;
  border: 3px solid var(--ink);
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.74);
  box-shadow: 6px 6px 0 rgba(33, 31, 28, 0.9);
  padding: 1rem;
  backdrop-filter: blur(12px);
  animation: lingquFlowIn 520ms ease both;
  animation-delay: calc(680ms + var(--item-delay));
  transition:
    transform 170ms ease,
    box-shadow 170ms ease,
    background 170ms ease;
}

.lingqu-flow-item:hover {
  background: #fff;
  transform: translate(-2px, -4px) rotate(-0.5deg);
  box-shadow: 9px 9px 0 rgba(33, 31, 28, 0.92);
}

.lingqu-flow-item span {
  display: inline-flex;
  width: 2.2rem;
  height: 2.2rem;
  align-items: center;
  justify-content: center;
  border: 2px solid var(--ink);
  border-radius: 0.78rem;
  background: #fff6bc;
  font-family: theme('fontFamily.display');
  font-weight: 950;
}

.lingqu-flow-item strong {
  display: block;
  margin-top: 0.7rem;
  font-family: theme('fontFamily.display');
  font-size: 1.2rem;
  font-weight: 950;
}

.lingqu-flow-item small {
  display: block;
  margin-top: 0.32rem;
  color: rgba(33, 31, 28, 0.62);
  font-size: 0.9rem;
  font-weight: 750;
  line-height: 1.55;
}

.lingqu-section {
  position: relative;
  width: min(77rem, calc(100vw - 2rem));
  margin: 0 auto;
  padding: clamp(3.5rem, 7vw, 6rem) 1rem;
}

.lingqu-section-heading {
  max-width: 48rem;
  margin-bottom: clamp(1.5rem, 3vw, 2.4rem);
}

.lingqu-section-heading span,
.lingqu-final-cta small {
  display: inline-flex;
  align-items: center;
  border: 3px solid var(--ink);
  border-radius: 999px;
  background: #fff6bc;
  box-shadow: 4px 4px 0 rgba(33, 31, 28, 0.9);
  padding: 0.38rem 0.82rem;
  color: var(--ink);
  font-size: 0.72rem;
  font-weight: 950;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.lingqu-section-heading h2,
.lingqu-final-cta h2 {
  margin-top: 0.8rem;
  color: var(--ink);
  font-family: theme('fontFamily.display');
  font-size: clamp(2.4rem, 5vw, 4.7rem);
  font-weight: 950;
  letter-spacing: 0;
  line-height: 1;
}

.lingqu-section-heading p,
.lingqu-final-cta p {
  margin-top: 1rem;
  color: rgba(33, 31, 28, 0.66);
  font-size: clamp(0.98rem, 1.45vw, 1.1rem);
  font-weight: 780;
  line-height: 1.8;
}

.lingqu-section-heading--compact {
  margin-bottom: 0;
}

.lingqu-benefits::before,
.lingqu-models::before {
  content: '';
  position: absolute;
  z-index: -1;
  border: 4px solid rgba(33, 31, 28, 0.08);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.24);
  filter: blur(0.2px);
}

.lingqu-benefits::before {
  right: -4rem;
  top: 6rem;
  width: 13rem;
  height: 13rem;
}

.lingqu-models::before {
  left: -3rem;
  top: 10rem;
  width: 10rem;
  height: 10rem;
}

.lingqu-benefit-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 1rem;
}

.lingqu-benefit-card,
.lingqu-model-card,
.lingqu-terminal,
.lingqu-final-cta {
  border: 4px solid var(--ink);
  background: rgba(255, 255, 255, 0.78);
  box-shadow: 8px 8px 0 rgba(33, 31, 28, 0.9);
  backdrop-filter: blur(13px);
}

.lingqu-benefit-card {
  position: relative;
  min-height: 18rem;
  overflow: hidden;
  border-radius: 24px;
  padding: 1.2rem;
  transition:
    transform 180ms ease,
    box-shadow 180ms ease,
    background 180ms ease;
  animation: lingquFlowIn 560ms ease both;
}

.lingqu-benefit-card::after {
  content: '';
  position: absolute;
  right: -2.2rem;
  top: -2rem;
  width: 6rem;
  height: 6rem;
  border: 3px dashed rgba(33, 31, 28, 0.14);
  border-radius: 50%;
  background: rgba(255, 212, 71, 0.28);
}

.lingqu-benefit-card--2 {
  transform: translateY(1.35rem) rotate(0.6deg);
}

.lingqu-benefit-card--3 {
  transform: translateY(-0.7rem) rotate(-0.7deg);
}

.lingqu-benefit-card--4 {
  transform: translateY(0.7rem) rotate(0.4deg);
}

.lingqu-benefit-card:hover {
  background: #fff;
  transform: translate(-3px, -5px) rotate(-0.8deg);
  box-shadow: 12px 12px 0 rgba(33, 31, 28, 0.92);
}

.lingqu-benefit-card__icon {
  display: grid;
  width: 3.35rem;
  height: 3.35rem;
  place-items: center;
  border: 3px solid var(--ink);
  border-radius: 18px;
  background: linear-gradient(135deg, #ff526f, #ffbe45);
  box-shadow: 4px 4px 0 rgba(33, 31, 28, 0.86);
  color: #fff;
}

.lingqu-benefit-card small {
  display: block;
  margin-top: 1rem;
  color: var(--pink);
  font-size: 0.74rem;
  font-weight: 950;
  letter-spacing: 0.14em;
  text-transform: uppercase;
}

.lingqu-benefit-card strong {
  display: block;
  margin-top: 0.45rem;
  font-family: theme('fontFamily.display');
  font-size: 1.55rem;
  font-weight: 950;
  line-height: 1;
}

.lingqu-benefit-card p {
  margin-top: 0.8rem;
  color: rgba(33, 31, 28, 0.66);
  font-size: 0.95rem;
  font-weight: 760;
  line-height: 1.75;
}

.lingqu-theater {
  display: grid;
  grid-template-columns: minmax(0, 0.86fr) minmax(22rem, 1.14fr);
  align-items: center;
  gap: clamp(1.6rem, 5vw, 4.4rem);
}

.lingqu-route-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.7rem;
  margin-top: 1.4rem;
}

.lingqu-route-chips span {
  border: 2px solid var(--ink);
  border-radius: 999px;
  background: #fff;
  box-shadow: 3px 3px 0 rgba(33, 31, 28, 0.8);
  padding: 0.46rem 0.9rem;
  font-size: 0.84rem;
  font-weight: 950;
}

.lingqu-terminal {
  position: relative;
  overflow: hidden;
  border-radius: 28px;
  background: #221f1b;
  color: #fffaf1;
  transform: rotate(1deg);
}

.lingqu-terminal::before {
  content: '';
  position: absolute;
  inset: 0;
  background:
    radial-gradient(circle at 20% 20%, rgba(255, 79, 123, 0.18), transparent 28%),
    radial-gradient(circle at 80% 70%, rgba(26, 182, 232, 0.18), transparent 32%);
  pointer-events: none;
}

.lingqu-terminal__bar {
  position: relative;
  display: flex;
  align-items: center;
  gap: 0.45rem;
  border-bottom: 3px solid rgba(255, 255, 255, 0.16);
  padding: 1rem;
}

.lingqu-terminal__bar span {
  width: 0.78rem;
  height: 0.78rem;
  border: 2px solid var(--ink);
  border-radius: 50%;
}

.lingqu-terminal__bar span:nth-child(1) {
  background: var(--pink);
}

.lingqu-terminal__bar span:nth-child(2) {
  background: var(--yellow);
}

.lingqu-terminal__bar span:nth-child(3) {
  background: var(--cyan);
}

.lingqu-terminal__bar strong {
  margin-left: auto;
  color: rgba(255, 255, 255, 0.58);
  font-family: theme('fontFamily.mono');
  font-size: 0.8rem;
}

.lingqu-terminal__body {
  position: relative;
  display: grid;
  gap: 0.72rem;
  padding: clamp(1rem, 2.4vw, 1.55rem);
}

.lingqu-terminal__body p {
  display: grid;
  grid-template-columns: 2rem minmax(0, 1fr);
  gap: 0.72rem;
  margin: 0;
  opacity: 0;
  transform: translateY(8px);
  animation: lingquTerminalLine 540ms ease forwards;
  animation-delay: calc(200ms + var(--line-delay));
}

.lingqu-terminal__body span {
  color: var(--yellow);
  font-family: theme('fontFamily.mono');
  font-weight: 950;
}

.lingqu-terminal__body code {
  color: rgba(255, 250, 241, 0.86);
  font-family: theme('fontFamily.mono');
  font-size: clamp(0.74rem, 1.2vw, 0.95rem);
  font-weight: 780;
  overflow-wrap: anywhere;
}

.lingqu-terminal__body p:last-child code::after {
  content: '';
  display: inline-block;
  width: 0.55em;
  height: 1em;
  margin-left: 0.18em;
  border-right: 3px solid var(--yellow);
  transform: translateY(0.16em);
  animation: lingquCaret 760ms step-end infinite;
}

.lingqu-model-grid {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 0.85rem;
}

.lingqu-model-card {
  position: relative;
  min-height: 12rem;
  overflow: hidden;
  border-radius: 24px;
  padding: 1rem;
  transition:
    transform 170ms ease,
    box-shadow 170ms ease,
    background 170ms ease;
}

.lingqu-model-card::before {
  content: '';
  position: absolute;
  right: -2rem;
  bottom: -2rem;
  width: 6rem;
  height: 6rem;
  border-radius: 50%;
  background: color-mix(in srgb, var(--model-color), transparent 62%);
}

.lingqu-model-card:hover {
  background: #fff;
  transform: translateY(-6px) rotate(-1deg);
  box-shadow: 11px 11px 0 rgba(33, 31, 28, 0.92);
}

.lingqu-model-card span {
  display: grid;
  width: 3rem;
  height: 3rem;
  place-items: center;
  border: 3px solid var(--ink);
  border-radius: 1rem;
  background: var(--model-color);
  box-shadow: 4px 4px 0 rgba(33, 31, 28, 0.84);
  color: #fff;
  font-family: theme('fontFamily.display');
  font-weight: 950;
}

.lingqu-model-card strong {
  display: block;
  margin-top: 1rem;
  font-family: theme('fontFamily.display');
  font-size: 1.25rem;
  font-weight: 950;
}

.lingqu-model-card small {
  display: block;
  margin-top: 0.35rem;
  color: rgba(33, 31, 28, 0.62);
  font-weight: 780;
  line-height: 1.5;
}

.lingqu-final-cta {
  position: relative;
  display: grid;
  width: min(77rem, calc(100vw - 2rem));
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: clamp(1rem, 3vw, 2rem);
  margin: 1rem auto 0;
  border-radius: 32px;
  background:
    radial-gradient(circle at 12% 20%, rgba(255, 212, 71, 0.35), transparent 18rem),
    linear-gradient(135deg, rgba(255, 255, 255, 0.9), rgba(232, 251, 255, 0.8));
  padding: clamp(1.4rem, 3vw, 2rem);
}

.lingqu-final-cta__mascot {
  position: relative;
  width: 6.5rem;
  height: 6.5rem;
  border: 4px solid var(--ink);
  border-radius: 28px;
  background: #fff4bc;
  box-shadow: 7px 7px 0 rgba(33, 31, 28, 0.9);
  transform: rotate(-4deg);
  animation: lingquFloat 5.2s ease-in-out infinite;
}

.lingqu-final-cta__mascot span {
  position: absolute;
  left: 1rem;
  top: 1.05rem;
  width: 4.2rem;
  height: 3.6rem;
  border: 4px solid var(--ink);
  border-radius: 45%;
  background: #fff;
}

.lingqu-final-cta__mascot span::before,
.lingqu-final-cta__mascot span::after {
  content: '';
  position: absolute;
  top: 1.25rem;
  width: 0.48rem;
  height: 0.62rem;
  border-radius: 50%;
  background: var(--ink);
}

.lingqu-final-cta__mascot span::before {
  left: 1rem;
}

.lingqu-final-cta__mascot span::after {
  right: 1rem;
}

.lingqu-final-cta__mascot i {
  position: absolute;
  right: 0.85rem;
  bottom: 0.75rem;
  width: 2.1rem;
  height: 1.85rem;
  border: 3px solid var(--ink);
  border-radius: 0.85rem;
  background: var(--cyan);
}

.lingqu-final-cta h2 {
  max-width: 48rem;
  font-size: clamp(2rem, 4vw, 3.8rem);
}

.lingqu-final-cta p {
  max-width: 42rem;
}

@keyframes lingquDropIn {
  from {
    opacity: 0;
    transform: translateY(-22px) scale(0.98);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

@keyframes lingquCopyIn {
  from {
    opacity: 0;
    transform: translateX(-26px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

@keyframes lingquStageIn {
  from {
    opacity: 0;
    transform: translateY(26px) rotate(2deg) scale(0.97);
  }
  to {
    opacity: 1;
    transform: translateY(0) rotate(0) scale(1);
  }
}

@keyframes lingquPopText {
  from {
    opacity: 0;
    transform: translateY(18px) scale(0.98);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

@keyframes lingquFloat {
  0%, 100% {
    translate: 0 0;
  }
  50% {
    translate: 0 -10px;
  }
}

@keyframes lingquWiggle {
  0%, 100% {
    transform: rotate(-0.8deg);
  }
  50% {
    transform: rotate(1deg) translateY(-2px);
  }
}

@keyframes lingquScreenShine {
  0%, 38% {
    transform: translateX(-130%);
  }
  62%, 100% {
    transform: translateX(130%);
  }
}

@keyframes lingquBlink {
  0%, 92%, 100% {
    transform: scaleY(1);
  }
  95% {
    transform: scaleY(0.12);
  }
}

@keyframes lingquCloud {
  0%, 100% {
    translate: 0 0;
  }
  50% {
    translate: 12px -7px;
  }
}

@keyframes lingquSpark {
  0%, 100% {
    opacity: 0.72;
    transform: scale(0.96) rotate(0deg);
  }
  50% {
    opacity: 1;
    transform: scale(1.08) rotate(12deg);
  }
}

@keyframes lingquComet {
  0%, 100% {
    opacity: 0.1;
    transform: translateX(0) rotate(-8deg);
  }
  50% {
    opacity: 0.26;
    transform: translateX(18px) rotate(-8deg);
  }
}

@keyframes lingquGridDrift {
  to {
    background-position: 24px 24px, 48px 48px, 48px 48px;
  }
}

@keyframes lingquFlowIn {
  from {
    opacity: 0;
    transform: translateY(14px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes lingquTerminalLine {
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes lingquCaret {
  0%, 50% {
    opacity: 1;
  }
  51%, 100% {
    opacity: 0;
  }
}

@media (max-width: 1080px) {
  .lingqu-nav-links {
    display: none;
  }

  .lingqu-hero {
    grid-template-columns: 1fr;
    gap: 1.5rem;
    padding-top: 3rem;
  }

  .lingqu-copy {
    max-width: 48rem;
  }

  .lingqu-stage {
    width: min(44rem, 100%);
    min-height: 27rem;
    justify-self: center;
  }

  .lingqu-flow {
    margin-top: 0;
  }

  .lingqu-benefit-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .lingqu-benefit-card,
  .lingqu-benefit-card--2,
  .lingqu-benefit-card--3,
  .lingqu-benefit-card--4 {
    transform: none;
  }

  .lingqu-theater {
    grid-template-columns: 1fr;
  }

  .lingqu-model-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .lingqu-final-cta {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .lingqu-final-cta .lingqu-action {
    grid-column: 2;
    justify-self: start;
  }
}

@media (max-width: 720px) {
  .lingqu-header {
    padding-top: 0.72rem;
  }

  .lingqu-nav {
    width: calc(100vw - 1rem);
    min-height: 4.3rem;
    border-width: 3px;
    border-radius: 22px;
    box-shadow: 5px 5px 0 rgba(33, 31, 28, 0.94);
  }

  .lingqu-brand__mark {
    width: 2.8rem;
    height: 2.8rem;
    border-radius: 16px;
  }

  .lingqu-brand__text small {
    display: none;
  }

  .lingqu-nav-actions {
    gap: 0.35rem;
  }

  .lingqu-nav-actions :deep(button) {
    padding-inline: 0.42rem;
  }

  .lingqu-nav-cta {
    min-height: 2.68rem;
    padding: 0 0.9rem;
  }

  .lingqu-nav-cta span {
    display: none;
  }

  .lingqu-hero {
    width: min(100vw - 1rem, 42rem);
    min-height: auto;
    padding: 2.2rem 0.5rem 3rem;
  }

  .lingqu-title__brand {
    font-size: clamp(3.7rem, 19vw, 5.5rem);
  }

  .lingqu-title__line {
    font-size: clamp(2.05rem, 11vw, 3rem);
    text-shadow:
      3px 3px 0 var(--ink),
      0 5px 0 rgba(255, 255, 255, 0.8);
  }

  .lingqu-lead {
    font-size: 0.98rem;
  }

  .lingqu-actions > * {
    flex: 1 1 12rem;
  }

  .lingqu-stage {
    min-height: 22rem;
  }

  .lingqu-browser-card {
    inset: 4% 0 8% 0;
    border-radius: 22px;
    box-shadow:
      8px 8px 0 rgba(33, 31, 28, 0.92),
      0 24px 52px rgba(26, 182, 232, 0.18);
  }

  .lingqu-chat-card {
    left: 1%;
    bottom: 0;
    width: min(17rem, 80vw);
  }

  .lingqu-model-pill--claude,
  .lingqu-model-pill--gemini {
    display: none;
  }

  .lingqu-cloud--left,
  .lingqu-comet--one,
  .lingqu-comet--two {
    display: none;
  }

  .lingqu-flow {
    grid-template-columns: 1fr;
    width: min(100vw - 1rem, 42rem);
    padding-inline: 0.5rem;
  }

  .lingqu-section {
    width: min(100vw - 1rem, 42rem);
    padding: 3.4rem 0.5rem;
  }

  .lingqu-section-heading h2,
  .lingqu-final-cta h2 {
    font-size: clamp(2rem, 10vw, 3rem);
  }

  .lingqu-benefit-grid,
  .lingqu-model-grid {
    grid-template-columns: 1fr;
  }

  .lingqu-benefit-card,
  .lingqu-model-card {
    min-height: auto;
  }

  .lingqu-terminal {
    border-width: 3px;
    border-radius: 22px;
    transform: none;
  }

  .lingqu-terminal__body p {
    grid-template-columns: 1.7rem minmax(0, 1fr);
  }

  .lingqu-final-cta {
    width: min(100vw - 1rem, 42rem);
    grid-template-columns: 1fr;
  }

  .lingqu-final-cta__mascot {
    width: 5.5rem;
    height: 5.5rem;
  }

  .lingqu-final-cta .lingqu-action {
    grid-column: auto;
    justify-self: stretch;
  }
}

@media (prefers-reduced-motion: reduce) {
  .lingqu-home *,
  .lingqu-home *::before,
  .lingqu-home *::after {
    animation: none !important;
    transition: none !important;
  }
}
</style>
