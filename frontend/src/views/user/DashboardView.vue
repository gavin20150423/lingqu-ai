<template>
  <UserWorkspaceLayout>
    <div class="user-start-page">
      <section class="user-start-hero">
        <div class="user-start-hero__copy">
          <span class="user-start-hero__badge">欢迎使用灵渠AI</span>
          <h1>先创建一个 Key，就可以开始调用模型。</h1>
          <p>
            不需要先理解账号池、通道、调度这些后台概念。创建 Key、复制接入地址，
            然后像调用 OpenAI 一样调用灵渠AI。
          </p>

          <div class="user-start-hero__actions">
            <router-link to="/keys?create=1" class="user-start-primary">
              <Icon name="plus" size="md" />
              创建我的 Key
            </router-link>
            <button type="button" class="user-start-secondary" @click="copyBaseUrl">
              <Icon name="copy" size="md" />
              复制接入地址
            </button>
          </div>
        </div>

        <div class="user-start-card" aria-label="接入信息">
          <div class="user-start-card__mascot" aria-hidden="true">
            <div class="user-start-card__face">
              <span></span>
              <span></span>
              <i></i>
            </div>
            <Icon name="key" size="lg" />
          </div>
          <div class="user-start-card__line">
            <small>Base URL</small>
            <code>{{ baseUrl }}</code>
          </div>
          <div class="user-start-card__line">
            <small>API Key</small>
            <code>sk-lingqu-••••••••••••</code>
          </div>
        </div>
      </section>

      <section class="user-start-steps" aria-label="快速开始步骤">
        <article v-for="step in steps" :key="step.title" class="user-start-step">
          <span>{{ step.index }}</span>
          <h2>{{ step.title }}</h2>
          <p>{{ step.desc }}</p>
          <router-link v-if="step.to" :to="step.to">{{ step.action }}</router-link>
        </article>
      </section>

      <section class="user-start-lite">
        <div class="user-start-lite__stat">
          <small>余额</small>
          <strong>${{ formatBalance(user?.balance || 0) }}</strong>
        </div>
        <div class="user-start-lite__stat">
          <small>今天请求</small>
          <strong>{{ stats?.today_requests || 0 }}</strong>
        </div>
        <div class="user-start-lite__stat">
          <small>API Key</small>
          <strong>{{ stats?.total_api_keys || 0 }}</strong>
        </div>
        <router-link to="/usage" class="user-start-lite__link">
          查看详细用量
          <Icon name="arrowRight" size="sm" />
        </router-link>
      </section>
    </div>
  </UserWorkspaceLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import UserWorkspaceLayout from '@/components/layout/UserWorkspaceLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { usageAPI, type UserDashboardStats as UserStatsType } from '@/api/usage'
import { useClipboard } from '@/composables/useClipboard'

const authStore = useAuthStore()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const stats = ref<UserStatsType | null>(null)
const user = computed(() => authStore.user)
const baseUrl = computed(() => {
  const configured = appStore.cachedPublicSettings?.api_base_url || appStore.apiBaseUrl
  return configured || `${window.location.origin}/v1`
})

const steps = [
  {
    index: '01',
    title: '创建 Key',
    desc: '点一下创建，默认配置就能先用。',
    action: '去创建',
    to: '/keys?create=1'
  },
  {
    index: '02',
    title: '复制接入信息',
    desc: '把 Base URL 和 Key 填进你的项目。',
    action: '',
    to: ''
  },
  {
    index: '03',
    title: '直接调用模型',
    desc: '保持 OpenAI 兼容格式，请求会自动进入灵渠AI。',
    action: '',
    to: ''
  }
] as const

function formatBalance(value: number): string {
  return Number(value || 0).toFixed(2)
}

function copyBaseUrl() {
  copyToClipboard(baseUrl.value, '接入地址已复制')
}

async function loadStats() {
  try {
    await authStore.refreshUser()
    stats.value = await usageAPI.getDashboardStats()
  } catch (error) {
    console.warn('Failed to load simple user start stats:', error)
  }
}

onMounted(() => {
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
  loadStats()
})
</script>

<style scoped>
.user-start-page {
  display: grid;
  gap: 0.9rem;
  width: 100%;
}

.user-start-hero {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(17rem, 0.52fr);
  gap: clamp(0.75rem, 1.8vw, 1.2rem);
  align-items: center;
  overflow: hidden;
  border: 1px solid rgba(33, 31, 28, 0.11);
  border-radius: 18px;
  background:
    radial-gradient(circle at 90% 14%, rgba(72, 185, 200, 0.1), transparent 28%),
    linear-gradient(135deg, rgba(255, 255, 255, 0.95), rgba(255, 252, 243, 0.92));
  box-shadow: 0 12px 30px rgba(29, 42, 42, 0.07);
  padding: clamp(0.78rem, 1.8vw, 1rem);
  animation: userStartRise 520ms cubic-bezier(0.2, 0.78, 0.24, 1) both;
}

.user-start-hero::before {
  display: none;
}

.user-start-hero::after {
  display: none;
}

.user-start-hero > * {
  position: relative;
  z-index: 1;
}

:global(.dark) .user-start-hero {
  background: linear-gradient(135deg, rgba(24, 24, 32, 0.9), rgba(34, 42, 48, 0.82));
}

.user-start-hero__badge {
  display: inline-flex;
  width: fit-content;
  border: 1px solid rgba(33, 31, 28, 0.1);
  border-radius: 999px;
  background: #fff8df;
  box-shadow: none;
  padding: 0.24rem 0.58rem;
  color: #211f1c;
  font-size: 0.68rem;
  font-weight: 950;
  animation: userStartPop 460ms ease 160ms both;
}

.user-start-hero h1 {
  margin-top: 0.52rem;
  max-width: 22ch;
  font-family: theme('fontFamily.display');
  font-size: clamp(1.58rem, 2.75vw, 2.18rem);
  font-weight: 950;
  letter-spacing: 0;
  line-height: 1.02;
  color: #242321;
  text-shadow: none;
  animation: userStartSlide 560ms cubic-bezier(0.2, 0.78, 0.24, 1) 90ms both;
}

.user-start-hero p {
  margin-top: 0.48rem;
  max-width: 31rem;
  color: rgba(33, 31, 28, 0.58);
  font-size: 0.86rem;
  font-weight: 750;
  line-height: 1.48;
  animation: userStartSlide 560ms cubic-bezier(0.2, 0.78, 0.24, 1) 180ms both;
}

:global(.dark) .user-start-hero p {
  color: rgba(255, 250, 240, 0.66);
}

.user-start-hero__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.55rem;
  margin-top: 0.72rem;
  animation: userStartSlide 560ms cubic-bezier(0.2, 0.78, 0.24, 1) 260ms both;
}

.user-start-primary,
.user-start-secondary,
.user-start-lite__link {
  display: inline-flex;
  min-height: 2.35rem;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
  border: 1px solid rgba(33, 31, 28, 0.12);
  border-radius: 12px;
  box-shadow: 0 8px 18px rgba(29, 42, 42, 0.07);
  padding: 0 0.78rem;
  color: #211f1c;
  font-weight: 950;
  transition: transform 150ms ease, box-shadow 150ms ease, filter 150ms ease;
}

.user-start-primary:hover,
.user-start-secondary:hover,
.user-start-lite__link:hover {
  transform: translateY(-2px);
  box-shadow: 0 14px 28px rgba(29, 42, 42, 0.1);
  filter: none;
}

.user-start-primary:active,
.user-start-secondary:active,
.user-start-lite__link:active {
  transform: translateY(1px);
  box-shadow: 0 6px 12px rgba(33, 31, 28, 0.07);
}

.user-start-primary {
  background: linear-gradient(135deg, #f8e08a, #f4b4bd);
}

.user-start-secondary,
.user-start-lite__link {
  background: #fff;
}

.user-start-card {
  position: relative;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 0.55rem 0.65rem;
  align-items: center;
  overflow: hidden;
  border: 1px solid rgba(33, 31, 28, 0.1);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.8);
  box-shadow: 0 10px 24px rgba(29, 42, 42, 0.06);
  padding: 0.72rem;
  transform-origin: center;
  animation: userStartFloatIn 680ms cubic-bezier(0.2, 0.78, 0.24, 1) 230ms both;
  transition: transform 180ms ease, box-shadow 180ms ease;
}

.user-start-card::before {
  display: none;
}

.user-start-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 14px 28px rgba(29, 42, 42, 0.09);
}

.user-start-card > * {
  position: relative;
  z-index: 1;
}

.user-start-card__mascot {
  display: grid;
  grid-row: span 2;
  justify-items: center;
  gap: 0.12rem;
  color: #4eaad0;
  animation: none;
}

.user-start-card__face {
  position: relative;
  display: flex;
  width: 3.25rem;
  height: 3rem;
  align-items: center;
  justify-content: center;
  gap: 0.46rem;
  border: 1px solid rgba(33, 31, 28, 0.12);
  border-radius: 45% 45% 42% 42%;
  background: #fff6dc;
}

.user-start-card__face::before {
  content: '';
  position: absolute;
  top: -0.48rem;
  width: 2.55rem;
  height: 0.92rem;
  border: 1px solid rgba(33, 31, 28, 0.12);
  border-radius: 999px 999px 10px 10px;
  background: #f4b4bd;
}

.user-start-card__face span {
  width: 0.28rem;
  height: 0.44rem;
  border-radius: 999px;
  background: #211f1c;
  animation: userStartBlink 4.2s infinite;
}

.user-start-card__face i {
  position: absolute;
  bottom: 0.58rem;
  width: 0.78rem;
  height: 0.36rem;
  border-bottom: 2px solid #211f1c;
  border-radius: 999px;
}

.user-start-card__line {
  display: grid;
  gap: 0.2rem;
  min-width: 0;
  border: 1px solid rgba(33, 31, 28, 0.08);
  border-radius: 13px;
  background: #fffdf7;
  padding: 0.52rem 0.62rem;
  transition: transform 160ms ease, border-color 160ms ease, background-color 160ms ease;
}

.user-start-card__line:hover {
  transform: none;
  border-color: rgba(74, 198, 160, 0.35);
  background: #ffffff;
}

.user-start-card__line small {
  color: rgba(33, 31, 28, 0.48);
  font-size: 0.68rem;
  font-weight: 950;
}

.user-start-card__line code {
  min-width: 0;
  overflow: hidden;
  color: #211f1c;
  font-family: theme('fontFamily.mono');
  font-size: 0.78rem;
  font-weight: 900;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-start-steps {
  position: relative;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
}

.user-start-steps::before {
  display: none;
}

.user-start-step,
.user-start-lite {
  border: 1px solid rgba(33, 31, 28, 0.11);
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.84);
  box-shadow: 0 10px 24px rgba(29, 42, 42, 0.06);
  animation: userStartRise 520ms cubic-bezier(0.2, 0.78, 0.24, 1) 420ms both;
}

.user-start-step {
  position: relative;
  padding: 0.85rem;
  transition: transform 170ms ease, box-shadow 170ms ease, background-color 170ms ease;
  animation: userStartRise 520ms cubic-bezier(0.2, 0.78, 0.24, 1) both;
}

.user-start-step:nth-child(1) {
  animation-delay: 260ms;
}

.user-start-step:nth-child(2) {
  animation-delay: 340ms;
}

.user-start-step:nth-child(3) {
  animation-delay: 420ms;
}

.user-start-step:hover {
  transform: translateY(-2px);
  background: rgba(255, 255, 255, 0.9);
  box-shadow: 0 14px 28px rgba(29, 42, 42, 0.09);
}

.user-start-step span {
  display: inline-flex;
  border-radius: 999px;
  background: #fff8df;
  padding: 0.2rem 0.48rem;
  color: #211f1c;
  font-size: 0.72rem;
  font-weight: 950;
  transition: transform 160ms ease, background-color 160ms ease;
}

.user-start-step:hover span {
  transform: none;
  background: #eaf8f7;
}

.user-start-step h2 {
  margin-top: 0.58rem;
  font-family: theme('fontFamily.display');
  font-size: 1.28rem;
  font-weight: 950;
  letter-spacing: 0;
}

.user-start-step p {
  margin-top: 0.32rem;
  color: rgba(33, 31, 28, 0.62);
  font-size: 0.9rem;
  font-weight: 750;
  line-height: 1.45;
}

.user-start-step a {
  display: inline-flex;
  margin-top: 0.48rem;
  color: #2f8aa2;
  font-weight: 950;
  transition: transform 140ms ease;
}

.user-start-step a:hover {
  transform: translateX(2px);
}

.user-start-lite {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr)) auto;
  gap: 0.8rem;
  align-items: center;
  padding: 1rem;
}

.user-start-lite__stat {
  display: grid;
  gap: 0.15rem;
  min-width: 0;
  border-radius: 16px;
  padding: 0.65rem;
  transition: background-color 160ms ease, transform 160ms ease;
}

.user-start-lite__stat:hover {
  background: rgba(255, 247, 208, 0.45);
  transform: translateY(-1px);
}

.user-start-lite__stat small {
  color: rgba(33, 31, 28, 0.48);
  font-size: 0.78rem;
  font-weight: 950;
}

.user-start-lite__stat strong {
  font-size: 1.55rem;
  font-weight: 950;
}

.user-start-lite__link {
  min-height: 2.7rem;
  border-width: 2px;
}

@media (max-width: 900px) {
  .user-start-hero,
  .user-start-steps,
  .user-start-lite {
    grid-template-columns: 1fr;
  }

  .user-start-card {
    max-width: 28rem;
  }

  .user-start-steps::before {
    display: none;
  }
}

@media (max-width: 640px) {
  .user-start-hero__actions > *,
  .user-start-lite__link {
    width: 100%;
  }
}

@keyframes userStartRise {
  from {
    opacity: 0;
    transform: translateY(18px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes userStartSlide {
  from {
    opacity: 0;
    transform: translateX(-18px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

@keyframes userStartPop {
  from {
    opacity: 0;
    transform: scale(0.9) rotate(-3deg);
  }
  to {
    opacity: 1;
    transform: scale(1) rotate(0);
  }
}

@keyframes userStartFloatIn {
  from {
    opacity: 0;
    transform: translateY(20px) rotate(4deg);
  }
  to {
    opacity: 1;
    transform: translateY(0) rotate(0);
  }
}

@keyframes userStartFloat {
  0%, 100% {
    translate: 0 0;
  }
  50% {
    translate: 0 -4px;
  }
}

@keyframes userStartMascot {
  0%, 100% {
    transform: rotate(0deg);
  }
  50% {
    transform: rotate(1deg) translateY(-2px);
  }
}

@keyframes userStartBlink {
  0%, 92%, 100% {
    transform: scaleY(1);
  }
  95% {
    transform: scaleY(0.12);
  }
}

@keyframes userStartShine {
  0%, 58% {
    transform: translateX(-120%);
  }
  76%, 100% {
    transform: translateX(120%);
  }
}

@keyframes userStartPulse {
  0%, 100% {
    opacity: 0.65;
    transform: scale(0.96);
  }
  50% {
    opacity: 1;
    transform: scale(1.05);
  }
}

@keyframes userStartSpin {
  to {
    transform: rotate(360deg);
  }
}

@media (prefers-reduced-motion: reduce) {
  .user-start-page *,
  .user-start-page *::before,
  .user-start-page *::after {
    animation: none !important;
    transition: none !important;
  }
}
</style>
