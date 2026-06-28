<template>
  <div class="auth-anime-shell relative min-h-screen overflow-hidden p-4 sm:p-6 lg:p-8">
    <div class="pointer-events-none absolute inset-0 auth-anime-shell__grid"></div>
    <div class="pointer-events-none absolute -left-20 top-16 h-56 w-56 rotate-[-12deg] border-[5px] border-comic-ink bg-[#ff7aae]/35 shadow-[10px_10px_0_rgba(33,31,28,0.86)] dark:border-white/15"></div>
    <div class="pointer-events-none absolute -right-28 bottom-20 h-72 w-72 rotate-[10deg] border-[5px] border-comic-ink bg-[#4ee9ff]/30 shadow-[10px_10px_0_rgba(33,31,28,0.86)] dark:border-white/15"></div>

    <div class="relative z-10 mx-auto grid min-h-[calc(100vh-2rem)] w-full max-w-6xl items-center gap-8 lg:grid-cols-[1.1fr_0.9fr]">
      <section class="hidden lg:block">
        <div class="auth-poster">
          <div class="mb-5 flex items-center gap-3">
            <div class="flex h-12 w-12 items-center justify-center overflow-hidden rounded-[18px] border-[3px] border-comic-ink bg-white shadow-[5px_5px_0_rgba(33,31,28,0.9)]">
              <img :src="siteLogo || '/brand/lingqu-ai-logo.svg'" alt="" class="h-full w-full object-contain" />
            </div>
            <div class="min-w-0">
              <h1 class="truncate text-2xl font-black text-comic-ink dark:text-white">
                {{ siteName }}
              </h1>
              <p class="text-xs font-black uppercase tracking-[0.22em] text-comic-ink/50 dark:text-white/45">
                Universal Model Key
              </p>
            </div>
          </div>
          <h2 class="auth-poster__title">一个 Key，用所有模型。</h2>
          <p class="auth-poster__desc">
            登录后复制你的 Key，应用就能稳定调用 GPT、Claude、Gemini、DeepSeek 等模型。
          </p>
          <div class="auth-poster__image-wrap">
            <img
              src="/illustrations/anime-gateway-hero.svg"
              alt="动漫风格 AI 模型网关插画"
              class="auth-poster__image"
            />
          </div>
        </div>
      </section>

      <div class="w-full">
        <div class="mb-6 text-center lg:hidden">
          <template v-if="settingsLoaded">
            <div class="mb-4 inline-flex h-16 w-16 items-center justify-center overflow-hidden rounded-[18px] border-[3px] border-comic-ink bg-white shadow-[6px_6px_0_rgba(33,31,28,0.9)]">
              <img :src="siteLogo || '/brand/lingqu-ai-logo.svg'" alt="Logo" class="h-full w-full object-contain" />
            </div>
            <h1 class="comic-display mb-2 text-3xl font-black text-comic-ink dark:text-white">
              {{ siteName }}
            </h1>
            <p class="text-sm font-semibold text-comic-ink/65 dark:text-white/55">
              {{ siteSubtitle }}
            </p>
          </template>
        </div>

        <div class="auth-panel">
          <slot />
        </div>

        <div class="mt-6 text-center text-sm">
          <slot name="footer" />
        </div>

        <div class="mt-8 text-center text-xs font-semibold text-comic-ink/[0.45] dark:text-white/40">
          &copy; {{ currentYear }} {{ siteName }}. All rights reserved.
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { resolveBrandLogo, resolveBrandName, resolveBrandSubtitle } from '@/constants/brand'
import { sanitizeUrl } from '@/utils/url'

const appStore = useAppStore()

const siteName = computed(() => resolveBrandName(appStore.siteName))
const siteLogo = computed(() => sanitizeUrl(resolveBrandLogo(appStore.siteLogo), { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => resolveBrandSubtitle(appStore.cachedPublicSettings?.site_subtitle))
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.auth-anime-shell {
  background:
    radial-gradient(circle at 14% 14%, rgba(255, 122, 174, 0.24), transparent 32%),
    radial-gradient(circle at 86% 16%, rgba(78, 233, 255, 0.24), transparent 30%),
    linear-gradient(135deg, #fff9ed 0%, #edfbff 48%, #fff1f8 100%);
}

:global(.dark) .auth-anime-shell {
  background:
    radial-gradient(circle at 14% 14%, rgba(255, 122, 174, 0.12), transparent 30%),
    radial-gradient(circle at 86% 16%, rgba(78, 233, 255, 0.12), transparent 30%),
    linear-gradient(135deg, #12110f 0%, #18182a 52%, #241722 100%);
}

.auth-anime-shell__grid {
  background-image:
    linear-gradient(rgba(33, 31, 28, 0.055) 1px, transparent 1px),
    linear-gradient(90deg, rgba(33, 31, 28, 0.055) 1px, transparent 1px);
  background-size: 24px 24px;
  mask-image: linear-gradient(to bottom, #000 0%, transparent 92%);
}

.auth-poster,
.auth-panel {
  border: 4px solid #211f1c;
  background: rgba(255, 255, 255, 0.78);
  box-shadow: 12px 12px 0 rgba(33, 31, 28, 0.92);
  backdrop-filter: blur(18px);
}

:global(.dark) .auth-poster,
:global(.dark) .auth-panel {
  background: rgba(24, 24, 32, 0.78);
}

.auth-poster {
  overflow: hidden;
  border-radius: 34px;
  padding: 2rem;
}

.auth-poster__title {
  max-width: 10ch;
  font-family: theme('fontFamily.display');
  font-size: clamp(3rem, 6vw, 5.6rem);
  font-weight: 950;
  letter-spacing: 0;
  line-height: 0.92;
  color: #ff4f7b;
  text-shadow: 4px 4px 0 #211f1c;
}

.auth-poster__desc {
  margin-top: 1.25rem;
  max-width: 32rem;
  color: rgba(33, 31, 28, 0.68);
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.8;
}

:global(.dark) .auth-poster__desc {
  color: rgba(255, 250, 240, 0.68);
}

.auth-poster__image-wrap {
  margin-top: 1.5rem;
  overflow: hidden;
  border: 3px solid #211f1c;
  border-radius: 28px;
  background: #fff;
  box-shadow: 7px 7px 0 rgba(33, 31, 28, 0.85);
}

.auth-poster__image {
  display: block;
  width: 100%;
  aspect-ratio: 16 / 10;
  object-fit: cover;
}

.auth-panel {
  border-radius: 30px;
  padding: clamp(1.35rem, 4vw, 2rem);
}
</style>
